package channel

import (
	"path"
	"sync"
	"fmt"
	"os/exec"
	"time"

	"video-stream/log"
)

// TODO:
// - Add some method of extracting metadata

// Channel behaves like an old school TV channel, except it's streaming MPEG-TS
// instead of analogue TV signals.
//
// Also, the fact that Go has the concept of a 'channel' makes this name super
// inconvenient but TV got there first.
type Channel struct {
	name        string
	schedule    *schedule
	connections *connectionList
}

func New(name string, shows []string) *Channel {
	strMap := make(map[chan []byte]struct{})

	return &Channel{
		name:     name,
		schedule: newSchedule(shows),
		connections: &connectionList{
			streams: strMap,
		},
	}
}

func (c *Channel) Name() string {
	return c.name
}

func (c *Channel) AddClient() (chan []byte, func()) {
	// Are we the first?
	//   If so, get the schedule
	//   Figure out the offset
	//   Start ffmpeg
	// In all cases
	//   Register a stream with the connectionlist

	return c.connections.add()
}

func (c *Channel) Start() {
	// Hang out until AddClient is called or parent context is canceled
	// Start streaming media until media context or parent context is canceled
	// GOTO 10
	for {
		c.streamFile(c.schedule.randomFile())

		// Space out new files a little bit so clients can catch up
		var DELAY = 2
		for i := range DELAY {
			log.Info(fmt.Sprintf("Waiting %d", DELAY-i))
			time.Sleep(time.Second) // just a hunch
		}
	}
}


// Useful for debugging but not something I actually want to expose

func (c *Channel) String() string {
	s := ""
	if c.connections.Count() != 1 {
		s = "s"
	}

	return fmt.Sprintf("Channel: %s - %d client%s", c.name, c.connections.Count(), s)
}

func (c *Channel) Count() int {
	return c.connections.Count()
}

func (c *Channel) streamFile(f mediafile) {

	var audioMap string	
	if f.hasEnglishAudio() {
		log.Debug("Mapping eng audio stream")
		audioMap = "0:a:m:language:eng"
	} else {
		log.Debug("Mapping all audio streams")
		audioMap= "0:a"
	}

	ffmpegArgs := []string{
		// Avoid timestamp funkiness
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",

		// Get input
		// "-sseof", "-10", // start N seconds from the end
		// "-ss", "45", // skip the first 45 seconds
		"-re", // throttle to realtime
		"-i", f.path,

		// Map streams
		"-map", "0:v:0",
		"-map", audioMap,

		// Re-encode video to h.264 1920x1080
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-vf", "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2", // letterbox 1080p

		// Re-encode audio to 48kHz stereo AAC
		"-c:a", "aac",
		"-ar", "48000",
		"-ac", "2",
		"-b:a", "128k",

		"-f", "mpegts", // format into mpegts so we can just dump it over http
		"pipe:1", // use stdout so we can pipe it into our go program
	}

	cmd := exec.Command("ffmpeg", ffmpegArgs...)

	dur, err := f.Duration()
	if err != nil {
		log.Warn("couldn't get file duration", "error", err.Error())
	}
	log.Info("Running ffmpeg", "file", path.Base(f.path), "duration", dur)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("could not create stdout pipe", "error", err.Error())
	}

	// TODO: Figure out how to get output from stderr for debugging without blocking
	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }

	if err := cmd.Start(); err != nil {
		log.Fatal("could not run ffmpeg command", "error", err.Error())
	}

	var innerWg sync.WaitGroup

	// Pump ffmpeg → broadcast
	innerWg.Go( func() {
		buf := make([]byte, 4096)

		for {
			n, err := stdout.Read(buf)
			if err != nil {
				log.Info("ffmpeg ended:", "reason", err)
				log.Debug(cmd.String())

				// for {
				// 	n, err := stderr.Read(buf)
				// 	log.Debug("Stderr:", "contents", string(buf[:n]))
				// 	if err != nil {
				// 		log.Debug("Error reading stderr:", "error", err.Error())
				// 		break
				// 	}
				// }
				break
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				c.connections.broadcast(data)
			}
			if n == 0 {
				log.Warn("Read zero bytes")
			}
		}
	})

	innerWg.Wait()

}
