package main

import (
	"fmt"
	"sync"
	"time"

	"video-stream/channel"
	"video-stream/config"
	"video-stream/log"
	"video-stream/server"
)


// ░█████╗░██╗░░░██╗██████╗░██████╗░███████╗███╗░░██╗████████╗░░░████████╗██╗░░██╗██╗███╗░░██╗░██████╗░
// ██╔══██╗██║░░░██║██╔══██╗██╔══██╗██╔════╝████╗░██║╚══██╔══╝░░░╚══██╔══╝██║░░██║██║████╗░██║██╔════╝░
// ██║░░╚═╝██║░░░██║██████╔╝██████╔╝█████╗░░██╔██╗██║░░░██║░░░░░░░░░██║░░░███████║██║██╔██╗██║██║░░██╗░
// ██║░░██╗██║░░░██║██╔══██╗██╔══██╗██╔══╝░░██║╚████║░░░██║░░░░░░░░░██║░░░██╔══██║██║██║╚████║██║░░╚██╗
// ╚█████╔╝╚██████╔╝██║░░██║██║░░██║███████╗██║░╚███║░░░██║░░░░░░░░░██║░░░██║░░██║██║██║░╚███║╚██████╔╝
// ░╚════╝░░╚═════╝░╚═╝░░╚═╝╚═╝░░╚═╝╚══════╝╚═╝░░╚══╝░░░╚═╝░░░░░░░░░╚═╝░░░╚═╝░░╚═╝╚═╝╚═╝░░╚══╝░╚═════╝░

// Don't start stream until first client connects
// For this we need:
// - 1 To be able to start the ffmpeg goroutine from AddConnection
// - 2 To know where in the file to start
//   - 2.1 Needs schedule
// - To be able to stop the ffmpeg goroutine from the returned cleanup function
// - I think the ffmpeg goroutine needs to live in the Channel, as it's closely associated with it and not a seperate process anymore

// ███████╗███╗░░██╗██████╗░░░░░█████╗░██╗░░░██╗██████╗░██████╗░███████╗███╗░░██╗████████╗░░░████████╗██╗░░██╗██╗███╗░░██╗░██████╗░
// ██╔════╝████╗░██║██╔══██╗░░░██╔══██╗██║░░░██║██╔══██╗██╔══██╗██╔════╝████╗░██║╚══██╔══╝░░░╚══██╔══╝██║░░██║██║████╗░██║██╔════╝░
// █████╗░░██╔██╗██║██║░░██║░░░██║░░╚═╝██║░░░██║██████╔╝██████╔╝█████╗░░██╔██╗██║░░░██║░░░░░░░░░██║░░░███████║██║██╔██╗██║██║░░██╗░
// ██╔══╝░░██║╚████║██║░░██║░░░██║░░██╗██║░░░██║██╔══██╗██╔══██╗██╔══╝░░██║╚████║░░░██║░░░░░░░░░██║░░░██╔══██║██║██║╚████║██║░░╚██╗
// ███████╗██║░╚███║██████╔╝░░░╚█████╔╝╚██████╔╝██║░░██║██║░░██║███████╗██║░╚███║░░░██║░░░░░░░░░██║░░░██║░░██║██║██║░╚███║╚██████╔╝
// ╚══════╝╚═╝░░╚══╝╚═════╝░░░░░╚════╝░░╚═════╝░╚═╝░░╚═╝╚═╝░░╚═╝╚══════╝╚═╝░░╚══╝░░░╚═╝░░░░░░░░░╚═╝░░░╚═╝░░╚═╝╚═╝╚═╝░░╚══╝░╚═════╝░

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

	// Asynchronous stuff starts here

	// Start channels
	for _, channel := range channels {
		wg.Go(func() {
			channel.Start()
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
