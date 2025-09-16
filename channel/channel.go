package channel

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"strconv"
	"time"

	"video-stream/log"
)

type playRequest struct{ reqTime time.Time }

func (p playRequest) String() string {
	return fmt.Sprintf("Play request @ %s", p.reqTime.Format(time.DateTime))
}

type stopRequest struct{ reqTime time.Time }

func (s stopRequest) String() string {
	return fmt.Sprintf("Stop request @ %s", s.reqTime.Format(time.DateTime))
}

// Channel behaves like an old school TV channel, except it's streaming MPEG-TS
// instead of analogue TV signals.
//
// Also, the fact that Go has the concept of a 'channel' makes this name super
// inconvenient but TV got there first.
type Channel struct {
	name        string
	schedule    *schedule
	connections *connectionList
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

func (c *Channel) Count() int {
	return c.connections.Count()
}

func (c *Channel) AddClient() (chan []byte, func()) {
	if c.connections.Count() == 0 {
		c.playChan <- playRequest{reqTime: time.Now()}
	}

	conn, cleanup := c.connections.add()
	return conn, func() {
		conns := cleanup() // cleanup returns number of connections after removal
		log.Debug("[AddClient::cleanup] called cleanup", "remaining_connections", strconv.Itoa(conns))
		if conns == 0 {
			log.Debug("[AddClient::cleanup] sending stopRequest")
			c.stopChan <- stopRequest{reqTime: time.Now()}
		}
	}
}

// Start blocks until the provided context is canceled. Only one goroutine
// should call Start for a given Channel instance.
func (c *Channel) Start(ctx context.Context) error {
	// Generate the schedule
	generated, err := c.schedule.generate(ctx)
	if err != nil {
		log.Error("Could not generate schedule", "channel", c.Name(), "error", err.Error())
		return err
	}


	for _, g := range generated {
		log.Debug(g.path)
	}

	log.Info("Schedule generation complete", "channel", c.Name())

	childCtx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	var cancelPlayer func()

	for {
		select {
		case startReq := <-c.playChan:
			log.Info("[channel loop] start request recieved, starting player", "channel", c.Name(), "request", startReq.String())
			cancelPlayer = c.startPlayer(childCtx)
		case stopReq := <-c.stopChan:
			log.Info("[channel loop] stop request recieved, cancelling player", "channel", c.Name(), "request", stopReq.String())
			cancelPlayer()
		case <-ctx.Done():
			log.Info("[channel loop] outer context canceled, exiting channel", "channel", c.Name())
			return nil
		}
	}
}

// startPlayer launches a background goroutine that continuously streams files
// from the channel's schedule until the provided context is canceled.
//
// The returned function is a cancel function that, when called, will stop the
// background goroutine and exit the player loop gracefully.
//
// The caller is responsible for invoking the returned cancel function to
// terminate playback, otherwise the goroutine will continue running until the
// parent context expires.
func (c *Channel) startPlayer(ctx context.Context) func() {
	var DELAY = 2
	childCtx, cancelCtx := context.WithCancel(ctx)

	go func() {
		for {
			log.Debug("[startPlayer] Starting stream", "channel", c.Name())
			c.streamFile(c.schedule.randomFile(), childCtx)
			log.Debug("[startPlayer] Stream finished", "channel", c.Name())

			select {
			case <-ctx.Done():
				log.Debug("[startPlayer] parent context is canceled, exiting")
				return
			case <-childCtx.Done():
				log.Debug("[startPlayer] child context is canceled, exiting")
				return
			default:
				log.Debugf("[startPlayer] waiting %d seconds before starting next file", DELAY)
				time.Sleep(time.Duration(2*time.Second))
			}
		}
	}()

	return cancelCtx
}

// Starts an ffmpeg process that publishes mpeg-ts data to all connections
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
		"-sseof", "-10", // start N seconds from the end
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

	dur, err := f.DurationString()
	if err != nil {
		log.Warn("[streamFile] couldn't get file duration", "error", err.Error(), "channel", c.Name())
	}
	log.Info("[streamFile] Running ffmpeg", "file", path.Base(f.path), "duration", dur, "channel", c.Name())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("[streamFile] could not create stdout pipe", "error", err.Error(), "channel", c.Name())
	}

	// TODO: Figure out how to get output from stderr for debugging without blocking
	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }

	if err := cmd.Start(); err != nil {
		log.Fatal("[streamFile] could not run ffmpeg command", "error", err.Error(), "channel", c.Name())
	}

	streamDone := make(chan struct{})

	go func() {
		buf := make([]byte, 4096)

	streamloop:
		for {
			select {
			case <-ctx.Done():
				// kill ffmpeg command and return
				log.Debug("[streamFile] context canceled, killing ffmpeg and returning", "channel", c.Name())
				cmd.Process.Kill()
				break streamloop
			default:
				n, err := stdout.Read(buf)
				if err != nil {
					log.Info("[streamFile] ffmpeg ended:", "reason", err, "channel", c.Name())
					// log.Debug(cmd.String())

					// for {
					// 	n, err := stderr.Read(buf)
					// 	log.Debug("Stderr:", "contents", string(buf[:n]))
					// 	if err != nil {
					// 		log.Debug("Error reading stderr:", "error", err.Error())
					// 		break
					// 	}
					// }
					break streamloop
				}
				if n > 0 {
					data := make([]byte, n)
					copy(data, buf[:n])
					c.connections.broadcast(data)
				}
			}
		}

		streamDone <- struct{}{}
	}()

	log.Debug("[streamFile] waiting for streamDone signal", "channel", c.Name())
	<-streamDone
	log.Debug("[streamFile] received streamDone signal, end of streamFile", "channel", c.Name())
}
