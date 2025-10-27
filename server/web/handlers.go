package web

import (
	"sort"
	"context"
	"embed"
	"html/template"
	"net/http"

	"video-stream/channel"
	"video-stream/log"
)

// Maybe these could be a single embed.FS

////go:embed static/*
//var staticFS embed.FS

//go:embed templates/*.html
var templates embed.FS
var tmpl = template.Must(template.ParseFS(templates, "templates/*.html"))

func indexHandler(chs []*channel.Channel) func(w http.ResponseWriter, r *http.Request) {

	// Man I love closures
	data := struct{ Channels []*channel.Channel }{Channels: chs}

	sort.Slice(data.Channels, func(i, j int) bool {
		return data.Channels[i].PathName() < data.Channels[j].PathName()
	})

	return func(w http.ResponseWriter, r *http.Request) {

		log.Info("/web", "path", r.URL.Path, "rawUrl", r.URL.RawPath)

		// because "/" matches everything
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			// Read template on req so we can develop fast
			// tmpl := template.Must(template.ParseFiles("server/web/templates/index.html"))
			// err := tmpl.Execute(w, nil)
			err := tmpl.ExecuteTemplate(w, "index.html", data)
			if err != nil {
				log.Error("error parsing template", "error", err.Error())
				return
			}

			return
		}

		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte{})
	}
}

func statusHandler(chMap map[string]*channel.Channel) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		channelName := r.PathValue("channel")

		channel, ok := chMap[channelName]
		if !ok {
			http.Error(w, "Channel not found", http.StatusNotFound)
			return
		}

		err := tmpl.ExecuteTemplate(w, "channel-card", channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func skipHandler(chMap map[string]*channel.Channel) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		channelName := r.PathValue("channel")

		ch, ok := chMap[channelName]
		if !ok {
			http.Error(w, "Channel not found", http.StatusNotFound)
			return
		}

		if ch.SkipFile() {
			log.Info("skip request made successfully", "requester", r.RemoteAddr, "channel", ch.Name())
		} else {
			log.Warn("skip request failed", "requester", r.RemoteAddr, "channel", ch.Name())
		}

		err := tmpl.ExecuteTemplate(w, "channel-card", ch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func NewHandler(ctx context.Context, chs []*channel.Channel) http.Handler {

	mux := http.NewServeMux()

	// staticFs := http.FileServerFS(staticFS)
	staticFs := http.FileServer(http.Dir("server/web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFs))

	chMap := make(map[string]*channel.Channel)
	for _, ch := range chs {
		chMap[ch.PathName()] = ch
	}

	mux.HandleFunc("GET /channel/{channel}/status", statusHandler(chMap))
	mux.HandleFunc("POST /channel/{channel}/skip", skipHandler(chMap))
	mux.HandleFunc("/", indexHandler(chs))
	return mux
}
