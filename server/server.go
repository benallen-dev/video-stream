package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"strings"

	"video-stream/channel"
	"video-stream/log"
)

func Start(ctx context.Context, chs []*channel.Channel) {
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
		route := fmt.Sprintf("/%s.ts", strings.ToLower(strings.ReplaceAll(ch.Name(), " ", "-")))

		playlist = append(playlist,
			fmt.Sprintf(`#EXTINF:-1, %s`, ch.Name()),
			fmt.Sprintf(`http://%s:8080%s`, ip, route),
		)


		// TODO: Handle dropped connections to avoid leaving ffmpeg running when clients time out
		http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			log.Info("[HTTP Server] client connected", "route", route, "channelName", ch.Name(), "client", r.RemoteAddr)

			w.Header().Set("Content-Type", "video/MP2T")

			// Add Connection, get datastream and cleanup fn
			stream, cleanup := ch.AddClient() 

			// TODO: cleanup doesn't appear to be working anymore

			defer func() {
			log.Info("[HTTP Server] client disconnected", "route", route, "channelName", ch.Name(), "client", r.RemoteAddr)
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
		log.Info("[HTTP Server] client requested playlist.m3u", "client", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Write([]byte(strings.Join(playlist, "\n")))
	})

	server := &http.Server{
		Addr: ":8080",
	}

	log.Info("[HTTP Server] starting http server", "address", server.Addr)
	go func() {
        if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
            log.Fatalf("[HTTP Server] server error: %v", err)
        }
        log.Info("[HTTP Server] stopped serving new connections")
    }()
    <-ctx.Done()

    shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownRelease()

    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Fatalf("[HTTP Server] shutdown error: %v", err)
    }
    log.Info("[HTTP Server] graceful shutdown complete.")
}
