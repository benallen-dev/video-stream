package stream

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"video-stream/channel"
	"video-stream/log"
)

func NewHandler(ctx context.Context, chs []*channel.Channel) http.Handler {

	mux := http.NewServeMux()

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

		mux.HandleFunc(streamRoute, func(w http.ResponseWriter, r *http.Request) {
			log.Info("[HTTP Server] client connected", "route", streamRoute, "channelName", ch.Name(), "client", r.RemoteAddr)

			w.Header().Set("Content-Type", "video/MP2T")

			// Add Connection, get datastream and cleanup fn
			stream, cleanup := ch.AddClient()

			defer func() {
				log.Info("[HTTP Server] client disconnected", "route", streamRoute, "channelName", ch.Name(), "client", r.RemoteAddr)
				cleanup()
			}()

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
				default: // stream to this client
					if _, err := w.Write(data); err != nil {
						return
					}
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			}
		})

		mux.HandleFunc(streamRoute+"/skip", func(w http.ResponseWriter, r *http.Request) {
			log.Info("[HTTP Server] /skip", "channel", ch.Name(), "client", r.RemoteAddr)
			success := ch.SkipFile()

			if !success {
				w.WriteHeader(400)
				w.Write([]byte("Not playing\n"))
			}
		})

		mux.HandleFunc(streamRoute+"/keepPlaying", func(w http.ResponseWriter, r *http.Request) {
			qCancel := r.URL.Query().Get("cancel")
			if qCancel != "" {
				log.Info("[HTTP Server] /keepPlaying", "qCancel", qCancel, "channel", ch.Name(), "client", r.RemoteAddr)
				err := ch.ClearKeepPlaying()

				if err != nil {
					log.Warn("[HTTP Server] /keepPlaying channel not playing", "qCancel", qCancel, "channel", ch.Name(), "client", r.RemoteAddr)

					w.WriteHeader(500)
					w.Write([]byte("Error clearing keepPlaying: not playing\n"))
					return
				}

				w.WriteHeader(200)
				w.Write([]byte("keepPlaying cleared"))
			} else {

				log.Info("[HTTP Server] /keepPlaying", "channel", ch.Name(), "client", r.RemoteAddr)
				err := ch.SetKeepPlaying()

				if err != nil {
					log.Warn("[HTTP Server] /keepPlaying channel not playing", "channel", ch.Name(), "client", r.RemoteAddr)

					w.WriteHeader(500)
					w.Write([]byte("Error setting keepPlaying: not playing\n"))
					return
				}

				w.WriteHeader(200)
				w.Write([]byte("keepPlaying set"))
			}
		})
	}

	// Simple playlist
	mux.HandleFunc("/playlist.m3u", func(w http.ResponseWriter, r *http.Request) {
		log.Info("[HTTP Server] client requested playlist.m3u", "client", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Write([]byte(strings.Join(playlist, "\n")))
	})

	return mux
}
