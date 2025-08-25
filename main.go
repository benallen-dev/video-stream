package main

import (
	"fmt"
	"sync"
	"time"

	"video-stream/log"
	"video-stream/ffmpeg"
	"video-stream/schedule"
	"video-stream/server"
)

var (
	clients   = make(map[chan []byte]struct{})
	clientsMu sync.Mutex
)

// TODO:
// - Graceful shutdown on quit
// - Don't start streaming media to stdout until a client connects
// - Use ffprobe to find the english audio track or otherwise default to the first track
// - Define channel object
// - Add cancellation to ffmpeg goroutine
// - Create advance schedule on boot
// - Update schedule when media file ends
// - Add static HTTP routes for channel icons etc
// - Add EPG support
// - Add support for multiple channels
// - Keep in mind potential REST endpoints for manipulating schedule
// - Dockerise so I can run this on unraid

func broadcast(data []byte) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for ch := range clients {
		select {
		case ch <- data:
		default:
			// drop if client is too slow
		}
	}
}

func main() {
	var wg sync.WaitGroup

	// Stream files forever
	wg.Go(func() {

		for {
			log.Info("Starting new file")
			f, err := schedule.RandomFile()
			if err != nil {
				log.Fatal(err.Error())
			}

			log.Info("File", "path", f)

			ffmpeg.StreamFile(f, broadcast)

			var DELAY = 5
				// time.Sleep(time.Duration(5) * time.Second) // just a hunch

			for i := range DELAY {
				// // Go up a line
				// fmt.Print("\033[F")
				// // Clear the line
				// fmt.Print("\033[K")

				log.Info(fmt.Sprintf("Waiting %d", DELAY-i))
				time.Sleep(time.Second) // just a hunch
			}
		}

	})

	// Run the webserver
	wg.Go(func() {
		server.Start(&clientsMu, clients)
		wg.Done()
	})

	wg.Wait()

	// Periodically print how many clients are connected
	go func() {
		for {
			clientsMu.Lock()
			log.Debug(fmt.Sprintf("%d client(s)", len(clients)))
			clientsMu.Unlock()
			time.Sleep(time.Duration(300 * time.Second))
		}
	}()
}
