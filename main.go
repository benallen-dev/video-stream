package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"video-stream/channel"
	"video-stream/ffmpeg"
	"video-stream/log"
	"video-stream/schedule"
	"video-stream/server"
)

// TODO:
// - Don't start streaming media to stdout until a client connects
// - Graceful shutdown on quit
// - Add cancellation to ffmpeg goroutine
// - Create advance schedule on boot?
// - Update schedule when media file ends?
// - Add static HTTP routes for channel icons etc
// - Add EPG support
// - Frontend
// - Keep in mind potential REST endpoints for manipulating schedule
// - Support skipping episodes via web UI
// - Dockerise so I can run this on unraid

func main() {
	var wg sync.WaitGroup

	cwd, err := os.Getwd()
	if err != nil {
		log.Error(err.Error())
	}

	channels := []*channel.Channel{
		channel.New("Testing", []string{
			cwd+"/test-data",
		}),
	}

	// Stream files forever
	for _, channel := range channels {
		wg.Go(func() {

			for {
				log.Info("Starting new file")
				f, err := schedule.RandomFile(channel)
				if err != nil {
					log.Fatal("error getting random file", "msg", err.Error())
				}

				log.Info("File", "path", f)

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
		for {
			for i, ch := range channels {
				log.Debug(fmt.Sprintf("Channel%d - %s: %d client(s)", i+1, ch.Name, ch.Connections.Count()))
			}
			time.Sleep(time.Duration(60 * time.Second))
		}
	}()

	wg.Wait()
}
