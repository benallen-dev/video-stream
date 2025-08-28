package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"video-stream/channel"
	"video-stream/log"
)

func getLocalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Error("Could not get IP", "msg", err.Error())
		return ""
	}

	if len(addrs) == 0 {
		log.Error("No network interfaces found")
		return ""
	}

	for _, addr := range addrs {
		log.Info(addr.String())

		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
    }

	log.Error("Could not find local IP address")
	return ""

}

func Start(chs []*channel.Channel) {
	// For each channel, create a handler that subscribes clients
	for i, ch := range chs {
		http.HandleFunc(fmt.Sprintf("/channel%d.ts", i+1), func(w http.ResponseWriter, r *http.Request) {
			log.Info(fmt.Sprintf("Client connect to channel%d.ts", i+1), "requester", r.RemoteAddr)

			w.Header().Set("Content-Type", "video/MP2T")
			conn, cleanup := ch.Connections.Add()
			defer func() {
				log.Info(fmt.Sprintf("Client disconnected from channel%d.ts", i+1), "requester", r.RemoteAddr)
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
	}

	// http.HandleFunc("/channel2.ts", func(w http.ResponseWriter, r *http.Request) {
	// 	log.Info("Client connect to channel2.ts", "requester", r.RemoteAddr)

	// 	w.Header().Set("Content-Type", "video/MP2T")
	// 	w.WriteHeader(http.StatusTeapot)
	// })

	// Simple playlist
	http.HandleFunc("/playlist.m3u", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Client connect to playlist.m3u")
		w.Header().Set("Content-Type", "application/x-mpegURL")

		ip := getLocalIp()

		var m3u = []string{
			"#EXTM3U",
			// `#EXTINF:-1, tvg-id="channel1" group-title="Cartoons",Cartoons`,
			// "http://192.168.1.35:8080/channel1.ts",
			// `#EXTINF:-1 tvg-id="channel2" tvg-logo="http://192.168.1.35/logo2.png" group-title="Sports",Channel 2`,
			// "http://192.168.1.35:8080/channel2.ts",
		}

		for i, ch := range chs {
			m3u = append(m3u,
				fmt.Sprintf(`#EXTINF:-1, tvg-logo="" tvg-id="channel%d" group-title="%s", %s`, i+1, ch.Name, ch.Name),
				fmt.Sprintf(`http://%s:8080/channel%d.ts`, ip, i+1),
			)
		}

		w.Write([]byte(strings.Join(m3u, "\n")))
	})

	log.Info("Serving on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error in HTTP server", "msg", err.Error())
	}
}
