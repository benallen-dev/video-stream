// Package channel provides a TV-like streaming abstraction where media files
// are broadcast continuously to multiple clients. Instead of analogue signals,
// it uses ffmpeg to transcode media into MPEG-TS streams and pipes them into Go,
// where they can be consumed by many clients simultaneously.
//
// A Channel maintains a schedule of media files, manages client connections,
// and automatically starts or stops playback based on client demand.
package channel

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"sync"
	"time"

	"video-stream/log"
)

//
// --- Request Types ---
//

type playRequest struct{ reqTime time.Time }

func (p playRequest) String() string {
	return fmt.Sprintf("Play request -> %s", p.reqTime.Format(time.DateTime))
}

type stopRequest struct{ reqTime time.Time }

func (s stopRequest) String() string {
	return fmt.Sprintf("Stop request -> %s", s.reqTime.Format(time.DateTime))
}

//
// --- Channel ---
//

// Channel behaves like an old school TV channel, except it's streaming MPEG-TS
// instead of analogue TV signals.
//
// Also, the fact that Go has the concept of a 'channel' makes this name super
// inconvenient but TV got there first.
type Channel struct {
	name        string
	schedule    *schedule
	connections *connectionList
	ffmpegCmd   *exec.Cmd
	playChan    chan playRequest
	stopChan    chan stopRequest
}

// New creates a new Channel with the given name and a list of show file paths.
// The channel maintains its own client connection list and a schedule to pick
// media files from.
func New(name string, shows []string) *Channel {
	strMap := make(map[chan []byte]struct{})
	playChan := make(chan playRequest)
	stopChan := make(chan stopRequest)

	return &Channel{
		name:     name,
		schedule: newSchedule(shows),
		connections: &connectionList{
			streams: strMap,
		},
		playChan: playChan,
		stopChan: stopChan,
	}
}

func (c *Channel) Name() string {
	return c.name
}

// AddClient registers a new client for the channel. It returns a byte stream
// channel for receiving MPEG-TS data and a cleanup function to call when the
// client disconnects.
//
// If this is the first client, the channel automatically issues a play request.
// When the last client disconnects, a stop request is issued.
func (c *Channel) AddClient() (chan []byte, func()) {
	if c.connections.Count() == 0 {
		c.playChan <- playRequest{reqTime: time.Now()}
	}

	conn, cleanup := c.connections.add()
	return conn, func() {
		cleanup()
		if c.connections.Count() == 0 {
			c.stopChan <- stopRequest{reqTime: time.Now()}
		}
	}
}

// Start runs the channel’s event loop. It listens for play and stop requests
// and launches or terminates the player accordingly.
//
// Start blocks until the provided context is canceled. Only one goroutine
// should call Start for a given Channel instance.
func (c *Channel) Start(ctx context.Context) error {
	childCtx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	var cancelPlayer func()

	for {
		select {
		case startReq := <-c.playChan:
			log.Debug("Start request recieved, starting ffmpeg", "channel", c.Name(), "request", startReq.String())
			cancelPlayer = c.StartPlayer(childCtx)
		case stopReq := <-c.stopChan:
			log.Debug("Stop request recieved, starting ffmpeg", "channel", c.Name(), "request", stopReq.String())
			cancelPlayer()
		case <-ctx.Done():
			log.Debug("[Start] outer context canceled, exiting channel", "channel", c.Name())
			return nil
		}
	}
}

// StartPlayer launches a background goroutine that continuously streams files
// from the channel's schedule until the provided context is canceled.
//
// Each iteration picks a random file from the channel's schedule and streams it
// with streamFile. After a file finishes streaming, the loop waits briefly
// (currently two seconds, logged as a countdown) to give clients time to catch up
// before starting the next file.
//
// The returned function is a cancel function that, when called, will stop the
// background goroutine and exit the player loop gracefully.
//
// Typical usage:
//
//	cancel := channel.StartPlayer(ctx)
//	defer cancel() // ensure cleanup
//
// The caller is responsible for invoking the returned cancel function to
// terminate playback, otherwise the goroutine will continue running until the
// parent context expires.
func (c *Channel) StartPlayer(ctx context.Context) func() {
	childCtx, cancelCtx := context.WithCancel(ctx)

	go func() {
		for {
			log.Debug("[StartPlayer] Starting stream", "channel", c.Name())
			c.streamFile(c.schedule.randomFile(), childCtx)

			select {
			case <-childCtx.Done():
				log.Debug("[StartPlayer] context is canceled, exiting")
				return
			default:
				// Space out new files a little bit so clients can catch up
				var DELAY = 2
				for i := range DELAY {
					log.Info(fmt.Sprintf("Waiting %d", DELAY-i))
					time.Sleep(time.Second) // just a hunch
				}
			}
		}
	}()

	return cancelCtx
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

// streamFile runs ffmpeg to stream a single media file to all connected
// clients. It blocks until either the file finishes streaming or the provided
// context is canceled.
//
// ffmpeg’s stdout is piped into Go and broadcast to all connections. If the
// context is canceled, the ffmpeg process is killed. Any errors while reading
// from ffmpeg’s output are logged but not returned.
func (c *Channel) streamFile(f mediafile, ctx context.Context) {

	var audioMap string
	if f.hasEnglishAudio() {
		log.Debug("Mapping eng audio stream")
		audioMap = "0:a:m:language:eng"
	} else {
		log.Debug("Mapping all audio streams")
		audioMap = "0:a"
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
	c.ffmpegCmd = cmd

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
	innerWg.Go(func() {
		buf := make([]byte, 4096)

		for {
			select {
			case <-ctx.Done():
				// kill ffmpeg command and return
				log.Debug("streamFile context canceled, killing ffmpeg and returning", "channel", c.Name())
				cmd.Process.Kill()
				return
			default:
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
		}
	})

	innerWg.Wait()

}
