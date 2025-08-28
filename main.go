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
// - Graceful shutdown on quit
// - Don't start streaming media to stdout until a client connects
// - Define channel object
// - Add cancellation to ffmpeg goroutine
// - Create advance schedule on boot
// - Update schedule when media file ends
// - Add static HTTP routes for channel icons etc
// - Add EPG support
// - Add support for multiple channels
// - Keep in mind potential REST endpoints for manipulating schedule
// - Dockerise so I can run this on unraid

func main() {
	var wg sync.WaitGroup

	cwd, err := os.Getwd()
	if err != nil {
		log.Error(err.Error())
	}
	// build one channel for testing
	var channel1 = channel.New("Test", []string{cwd+"/test-data"})

	// Stream files forever
	wg.Go(func() {

		for {
			log.Info("Starting new file")
			f, err := schedule.RandomFile(channel1)
			if err != nil {
				log.Fatal("error getting random file", "msg", err.Error())
			}

			log.Info("File", "path", f)

			ffmpeg.StreamFile(f, channel1.Broadcast)

			// Space out new files a little bit so clients can catch up
			var DELAY = 2
			for i := range DELAY {
				log.Info(fmt.Sprintf("Waiting %d", DELAY-i))
				time.Sleep(time.Second) // just a hunch
			}
		}

	})

	// Run the webserver
	wg.Go(func() {
		server.Start(channel1)
		wg.Done()
	})

	// Periodically print how many clients are connected
	go func() {
		for {
			log.Debug(fmt.Sprintf("Channel1: %d client(s)", channel1.Connections.Count()))
			time.Sleep(time.Duration(60 * time.Second))
		}
	}()

	wg.Wait()
}
