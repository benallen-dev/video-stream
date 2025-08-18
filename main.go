package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"video-stream/log"
	"os"
	"video-stream/schedule"
)

var (
	clients   = make(map[chan []byte]struct{})
	clientsMu sync.Mutex
)


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

// TODO:
// - Goroutine that hosts the ffmpeg command
// - Make a list of files
// - Figure out how to define channels etc
// - Maybe vibe code a frontend
// - Get Jellyfin to properly stream to the Jellyfin TV client (browser works!)
func main() {
		// Example: loop a file forever

// ffmpeg -re -stream_loop -1 -i input.mkv \
//   -c:v libx264 -preset veryfast -c:a aac -ar 48000 -b:a 128k \
//   -f mpegts pipe:1
	
	f, err := schedule.RandomFile()
	if err != nil {
		log.Fatal(err.Error())
	}


	cmd := exec.Command(
		"ffmpeg",
		"-re",                // throttle to realtime
		"-stream_loop", "-1", // loop input infinitely
		"-i", f,
		// "-i", "input.mkv",    // replace with your content

  		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a","aac",
		"-ar", "48000",
		"-b:a", "128k",

		// "-c:v", "copy",
		// "-c:a", "aac",
		"-f", "mpegts",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err.Error())
	}

	// Pump ffmpeg → broadcast
	go func() {
		buf := make([]byte, 4096) 



		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				broadcast(data)
			}
			if err != nil {
				log.Info("ffmpeg ended:", err)
				break
			}
		}
	}()

	go func() {
		for {
			clientsMu.Lock()
			log.Debug(fmt.Sprintf("%d clients connected", len(clients)))
			clientsMu.Unlock()
			time.Sleep(time.Duration(5 * time.Second))
		}
	}()

	// HTTP handler: subscribe clients
	http.HandleFunc("/channel1.ts", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Client connect to channel1.ts", "requester", r.RemoteAddr)

		w.Header().Set("Content-Type", "video/MP2T")
		ch := make(chan []byte, 1024)

		clientsMu.Lock()
		clients[ch] = struct{}{}
		clientsMu.Unlock()

		defer func() {
			log.Info("Client disconnected from channel.ts", "requester", r.RemoteAddr)
			clientsMu.Lock()
			delete(clients, ch)
			close(ch)
			clientsMu.Unlock()
		}()

		// stream to this client
		for data := range ch {
			if _, err := w.Write(data); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})

	http.HandleFunc("/channel2.ts", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Client connect to channel1.ts", "requester", r.RemoteAddr)

		w.Header().Set("Content-Type", "video/MP2T")
		w.WriteHeader(http.StatusTeapot)
	})

	// Simple playlist
	http.HandleFunc("/playlist.m3u", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Client connect to playlist.m3u")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")

		var m3u = []string{
			"#EXTM3U",
			`#EXTINF:-1, tvg-id="chan1" group-title="Cartoons",South Park`,
			"http://192.168.1.35:8080/channel1.ts",
			// `#EXTINF:-1 tvg-id="chan2" tvg-logo="http://192.168.1.35/logo2.png" group-title="Sports",Channel 2`,
			// "http://192.168.1.35:8080/channel2.ts",
		}

		w.Write([]byte(strings.Join(m3u, "\n")))
	})
	log.Info("Serving on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}

