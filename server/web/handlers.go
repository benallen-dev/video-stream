package web

import (
	"context"
	"net/http"
	
	"video-stream/channel"
)


func NewHandler(ctx context.Context, chs []*channel.Channel) http.Handler {

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello world, this is the web handler"))
	})

	return mux
}

