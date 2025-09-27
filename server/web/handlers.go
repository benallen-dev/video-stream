package web

import (
	"context"
	"html/template"
	"net/http"

	"video-stream/channel"
	"video-stream/log"
)

func NewHandler(ctx context.Context, chs []*channel.Channel) http.Handler {

	mux := http.NewServeMux()

	staticFs := http.FileServer(http.Dir("server/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFs))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Info("/web", "path", r.URL.Path)

		// because "/" matches everything
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			// Read template on req so we can develop fast
			tmpl := template.Must(template.ParseFiles("server/web/templates/index.html"))
			_ = tmpl.Execute(w, nil)

			return
		}

		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte{})
	})

	return mux
}
