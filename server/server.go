package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"video-stream/channel"
	"video-stream/log"

	"video-stream/server/stream"
	"video-stream/server/web"
)


func Start(ctx context.Context, chs []*channel.Channel) {

	http.Handle("/web/", http.StripPrefix("/web", web.NewHandler(ctx, chs)))
	http.Handle("/stream/", http.StripPrefix("/stream", stream.NewHandler(ctx, chs)))

	http.Handle("/favicon.ico", http.RedirectHandler("/web/static/favicon.ico", http.StatusMovedPermanently))

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	// If we're in the root, and the client is a browser, redirect to web, otherwise some other HTTP status to indicate the server cannot respond.

	// 	http.Redirect(w,r,"/web",http.StatusFound)
	// })

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
