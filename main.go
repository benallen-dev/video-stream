package main

import (
	"context"
	"fmt"
	"sync"
	"time"
	"os"
	"os/signal"
	"syscall"

	"video-stream/channel"
	"video-stream/config"
	"video-stream/log"
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
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// Read config and build "channels"
	cfg, err := config.Read()
	if err != nil {
		log.Fatal("Could not read config", "msg", err.Error())
	}

	log.SetLevel(cfg.LogLevel)

	channels := make([]*channel.Channel, 0, len(cfg.Channels))
	for name, dirs := range cfg.Channels {
		channels = append(channels, channel.New(name, dirs))
	}

	// Asynchronous stuff starts here

	// Start channels
	for _, channel := range channels {
		wg.Go(func() {
			channel.Start(ctx)
			wg.Done()
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
		// just runs forever right now
		// TODO: put in wg.Go when building cancellation
		watchClientCount(channels)
	}()

	// Wait for interrupt
	go func() {
		done := make(chan os.Signal, 1)
  		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
  		fmt.Println("Blocking, press ctrl+c to continue...")
  		<-done  // Will block here until user hits ctrl+c
		log.Info("Received interrupt")
  		ctxCancel()
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
		out := ""
		for _, ch := range chs {
			s := ""
			if ch.Count() != 1 {
				s = "s"
			}
			out += fmt.Sprintf("\n\t"+ch.Name()+": \x1b[%dG%d client%s", w+8, ch.Count(), s) // TODO: Maybe we can get rid of count
		}
		log.Debug(out)

		time.Sleep(time.Duration(60 * time.Second))
	}
}
