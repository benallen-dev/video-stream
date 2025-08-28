package main

import (
	"fmt"
	"sync"
	"time"

	"video-stream/channel"
	"video-stream/config"
	"video-stream/ffmpeg"
	"video-stream/log"
	"video-stream/schedule"
	"video-stream/server"
)

// TODO:
// - Logging:
//  - Add context for which channel the log originates from
// - Scheduling
//   - Generate schedule when starting
//   - Periodically extend schedule
//   - Use schedule when deciding what to play
// - Optimisation
//   - Don't start ffmpeg streaming media to stdout until a client connects
//   - Add cancellation to ffmpeg goroutine
//   - Graceful shutdown on quit
//   - Dockerise so I can run this on unraid
// - User Interface
//   - Add static HTTP routes for channel icons etc
//   - Add EPG support
//   - Frontend for monitoring/configuration
//   - Support skipping episodes via web UI

func main() {
	var wg sync.WaitGroup

	// Read config and build "channels"
	cfg, err := config.Read()
	if err != nil {
		log.Fatal("Could not read config", "msg", err.Error())
	}

	channels := make([]*channel.Channel, 0, len(cfg.Channels))
	for name, dirs := range cfg.Channels {
		channels = append(channels, channel.New(name, dirs))
	}

	// Stream files into Go channels
	for _, channel := range channels {
		wg.Go(func() {

			for {
				log.Info("Starting new file")
				f, err := schedule.RandomFile(channel)
				if err != nil {
					log.Error("error getting random file", "msg", err.Error(), "channel", channel.Name())
					continue
				}

				ffmpeg.StreamFile(f, channel.Broadcast)

				// Space out new files a little bit so clients can catch up
				var DELAY = 2
				for i := range DELAY {
					log.Info(fmt.Sprintf("Waiting %d", DELAY-i))
					time.Sleep(time.Second) // just a hunch
				}
			}
		})
	}

	// Run the webserver
	wg.Go(func() {
		log.Debug("Starting http server")
		server.Start(channels)
		wg.Done()
	})

	// Periodically print how many clients are connected
	go func() {
		watchClientCount(channels)
		wg.Done()
	}()

	wg.Wait()
}

func watchClientCount(chs []*channel.Channel) {
	w := 0 // Find width of longest channel name
	for _, ch := range chs {
		if len(ch.Name()) > w {
			w = len(ch.Name())
		}
	}
	w = w + 4

	for {
		out := "\n"
		for _, ch := range chs {
			s := ""
			if ch.Count() != 1 {
				s = "s"
			}
			out += fmt.Sprintf("\t"+ch.Name()+": \x1b[%dG%d client%s\n", w+8, ch.Count(), s) // TODO: Maybe we can get rid of count
		}
		log.Debug(out)

		time.Sleep(time.Duration(60 * time.Second))
	}
}
