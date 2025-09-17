package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
		streamRoute := fmt.Sprintf("/%s.ts", strings.ToLower(strings.ReplaceAll(ch.Name(), " ", "-")))

		playlist = append(playlist,
			fmt.Sprintf(`#EXTINF:-1, %s`, ch.Name()),
			fmt.Sprintf(`http://%s:8080%s`, ip, streamRoute),
		)

		http.HandleFunc(streamRoute, func(w http.ResponseWriter, r *http.Request) {
			log.Info("[HTTP Server] client connected", "route", streamRoute, "channelName", ch.Name(), "client", r.RemoteAddr)

			w.Header().Set("Content-Type", "video/MP2T")

			// Add Connection, get datastream and cleanup fn
			stream, cleanup := ch.AddClient()

			defer func() {
				log.Info("[HTTP Server] client disconnected", "route", streamRoute, "channelName", ch.Name(), "client", r.RemoteAddr)
				cleanup()
			}()

			// stream to this client
			for data := range stream {
				select {
				case <-ctx.Done(): // If the server is shutting down
					// Close TCP connection
					hj, ok := w.(http.Hijacker)
					if !ok {
						log.Error("[HandleFunc] webserver doesn't support hijacking", "route", streamRoute)
						return
					}
					conn, _, err := hj.Hijack()
					if err != nil {
						log.Error("[HandleFunc] failed to hijack TCP connection", "route", streamRoute)
					}
					conn.Close()

					return
				default:
					if _, err := w.Write(data); err != nil {
						return
					}
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			}
		})

		http.HandleFunc(streamRoute+"/skip", func(w http.ResponseWriter, r *http.Request) {
			log.Info("[HTTP Server] /skip", "channel", ch.Name(), "client", r.RemoteAddr)
			success := ch.SkipFile()

			if !success {
				w.WriteHeader(400)
				w.Write([]byte("Not playing\n"))
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
