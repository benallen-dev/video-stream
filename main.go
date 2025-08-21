package main

import (
	"bytes"
	"fmt"
	"path"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"video-stream/log"
	"video-stream/schedule"
)

var (
	clients   = make(map[chan []byte]struct{})
	clientsMu sync.Mutex
)

// TODO:
// - Graceful shutdown on quit
// - Refactor sections of main as independent functions/routines
// - Use ffprobe to find the english audio track or otherwise default to the first track
// - Define channel object
// - Add cancellation to ffmpeg goroutine
// - Create advance schedule on boot
// - Update schedule when media file ends
// - Add static HTTP routes for channel icons etc
// - Add EPG support
// - Add support for multiple channels
// - Keep in mind potential REST endpoints for manipulating schedule
// - Dockerise so I can run this on unraid

// Returns audio langues for a given file
func getAudioLanguages(input string) (map[int]string, error) {
	// Run ffprobe to extract audio stream indexes and language tags
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=index:stream_tags=language",
		"-of", "csv=p=0",
		input,
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	languages := make(map[int]string)

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) == 2 {
			// parts[0] = stream index, parts[1] = language
			// (e.g., "1,eng")
			idx := strings.TrimSpace(parts[0])
			lang := strings.TrimSpace(parts[1])
			if idx != "" && lang != "" {
				// convert index string to int
				var streamIdx int
				fmt.Sscanf(idx, "%d", &streamIdx)
				languages[streamIdx] = lang
			}
		}
	}

	return languages, nil
}

// getDuration returns the duration of a media file formatted as mm:ss
func getDuration(file string) (string, error) {
	cmd := exec.Command(
		"ffprobe",
		"-i", file,
		"-show_entries", "format=duration",
		"-v", "quiet",
		"-of", "csv=p=0",
	)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run ffprobe: %w", err)
	}

	durationStr := strings.TrimSpace(string(out))
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse duration: %w", err)
	}

	minutes := int(durationSec) / 60
	seconds := int(durationSec) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds), nil
}

func broadcast(data []byte) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for ch := range clients {
		select {
		case ch <- data:
		default:
			// drop if client is too slow
		}
	}
}

func main() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for {
			f, err := schedule.RandomFile()
			if err != nil {
				log.Fatal(err.Error())
			}

			dur, err := getDuration(f)
			if err != nil {
				log.Warn("Couldn't get file duration", "error", err.Error())
			}

			// // Find if english stream exists using ffprobe
			// langs, err := getAudioLanguages(f)
			// if err != nil {
			// 	log.Fatal(err.Error())
			// }

			// log.Info("Languages:")
			// hasEng := false
			// for _, lang := range langs {
			// 	log.Info(lang)
			// 	if lang == "eng" {
			// 		hasEng = true
			// 	}
			// }

			// ffmpegArgs := []string{
			// 	"-re", // Realtime
			// 	"-i", f, // Input from file
			// 	"-c:v", "libx264", // h264 video
			// 	"-preset", "veryfast",
			// 	// "-c:a","aac",
			// 	"-ar", "48000",
			// 	"-b:a", "128k",
			// 	"-map", "0:v", // Use first video stream
			// }

			// if hasEng {
			// 	ffmpegArgs = append(ffmpegArgs, "-map", "0:a:m:language:eng")
			// } else {
			// 	ffmpegArgs = append(ffmpegArgs, "-map", "0:a?")
			// }

			// ffmpegArgs = append(ffmpegArgs,
			// 	"-f", "mpegts", // format into mpegts so we can just dump it over http
			// 	"pipe:1", // use stdout so we can pipe it into our go program
			// 	)

			// cmd := exec.Command("ffmpeg", ffmpegArgs...)
			// log.Info(cmd.String())

			cmd := exec.Command(
				"ffmpeg",
				"-re",                // throttle to realtime
				// "-stream_loop", "-1", // loop input infinitely
				"-i", f,

				"-c:v", "libx264",
				"-preset", "veryfast",
				"-c:a","aac",
				"-ar", "48000",
				"-b:a", "128k",
				"-map", "0:v",
				"-map", "0:a?",
				"-map", "0:a:m:language:eng",
				// "-c", "copy",

				// "-c:v", "copy",
				// "-c:a", "aac",
				"-f", "mpegts",
				"pipe:1",
			)

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				log.Fatal(err.Error())
			}

			stderr, err := cmd.StderrPipe()
			if err != nil {
				log.Fatal(err.Error())
			}

			log.Info("Running ffmpeg", "file", path.Base(f), "duration", dur)
			if err := cmd.Start(); err != nil {
				log.Fatal(err.Error())
			}

			var innerWg sync.WaitGroup
			// Pump ffmpeg → broadcast
			innerWg.Add(1)
			go func() {
				buf := make([]byte, 4096) 

				for {
					n, err := stdout.Read(buf)
					if n > 0 {
						data := make([]byte, n)
						copy(data, buf[:n])
						broadcast(data)
					}
					if err != nil {
						log.Info("ffmpeg ended:", err)
						log.Info(cmd.String())

						for {
							_, err := stderr.Read(buf)
							log.Info(string(buf))
							if err != nil {
								log.Info(err.Error())
								break
							}
						}

						break
					}
				}
					innerWg.Done()
			}()
			innerWg.Wait()
		}
	}()

	go func() {
		for {
			clientsMu.Lock()
			log.Debug(fmt.Sprintf("%d client(s)", len(clients)))
			clientsMu.Unlock()
			time.Sleep(time.Duration(300 * time.Second))
		}
	}()

	wg.Add(1)
	go func() {
		// HTTP handler: subscribe clients
		http.HandleFunc("/channel1.ts", func(w http.ResponseWriter, r *http.Request) {
			log.Info("Client connect to channel1.ts", "requester", r.RemoteAddr)

			w.Header().Set("Content-Type", "video/MP2T")
			ch := make(chan []byte, 1024)

			clientsMu.Lock()
			clients[ch] = struct{}{}
			clientsMu.Unlock()

			defer func() {
				log.Info("Client disconnected from channel.ts", "requester", r.RemoteAddr)
				clientsMu.Lock()
				delete(clients, ch)
				close(ch)
				clientsMu.Unlock()
			}()

			// stream to this client
			for data := range ch {
				if _, err := w.Write(data); err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		})

		http.HandleFunc("/channel2.ts", func(w http.ResponseWriter, r *http.Request) {
			log.Info("Client connect to channel1.ts", "requester", r.RemoteAddr)

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

		wg.Done()
	}()

	wg.Wait()
}

