package server

import (
	"net/http"
	"strings"

	"video-stream/channel"
	"video-stream/log"
)

func Start(ch *channel.Channel) {
	// For each channel, create a handleFunc

	// Hardcoded for 1 channel for now
	// HTTP handler: subscribe clients
	http.HandleFunc("/channel1.ts", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Client connect to channel1.ts", "requester", r.RemoteAddr)

		w.Header().Set("Content-Type", "video/MP2T")
		conn, cleanup := ch.Connections.Add()
		defer func() {
			log.Info("Client disconnect from channel1.ts", "requester", r.RemoteAddr)
			cleanup()
		}()

		// stream to this client
		for data := range conn {
			if _, err := w.Write(data); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})

	http.HandleFunc("/channel2.ts", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Client connect to channel2.ts", "requester", r.RemoteAddr)

		w.Header().Set("Content-Type", "video/MP2T")
		w.WriteHeader(http.StatusTeapot)
	})

	// Simple playlist
	http.HandleFunc("/playlist.m3u", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Client connect to playlist.m3u")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")

		var m3u = []string{
			"#EXTM3U",
			`#EXTINF:-1, tvg-id="chan1" group-title="Cartoons",Cartoons`,
			"http://192.168.1.35:8080/channel1.ts",
			// `#EXTINF:-1 tvg-id="chan2" tvg-logo="http://192.168.1.35/logo2.png" group-title="Sports",Channel 2`,
			// "http://192.168.1.35:8080/channel2.ts",
		}

		w.Write([]byte(strings.Join(m3u, "\n")))
	})
	log.Info("Serving on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
