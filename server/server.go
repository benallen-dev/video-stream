package server

import (
	"fmt"
	"net/http"
	"strings"

	"video-stream/channel"
	"video-stream/log"
)

func Start(chs []*channel.Channel) {
	// Static file hosting from ./public
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	ip := getLocalIp()

	// Set up m3u file
	var playlist = []string{
		"#EXTM3U",
		"#PLAYLIST Channels",
	}

	// For each channel, add to the m3u and create a handler that subscribes clients
	for _, ch := range chs {
		route := ch.Route()

		playlist = append(playlist,
			fmt.Sprintf(`#EXTINF:-1, %s`, ch.Name()),
			fmt.Sprintf(`http://%s:8080%s`, ip, route),
		)

		http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			log.Info("client connected", "route", route, "channelName", ch.Name(), "client", r.RemoteAddr)

			w.Header().Set("Content-Type", "video/MP2T")

			// Add Connection, get datastream and cleanup fn
			stream, cleanup := ch.AddClient()

			defer func() {
			log.Info("client disconnected", "route", route, "channelName", ch.Name(), "client", r.RemoteAddr)
				cleanup()
			}()

			// stream to this client
			for data := range stream {
				if _, err := w.Write(data); err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		})
	}

	// Simple playlist
	http.HandleFunc("/playlist.m3u", func(w http.ResponseWriter, r *http.Request) {
		log.Info("client requested playlist.m3u", "client", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Write([]byte(strings.Join(playlist, "\n")))
	})

	log.Info("Serving on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error in HTTP server", "msg", err.Error())
	}
}
