package ffmpeg

import (
	"os/exec"
	"path"
	"sync"

	"video-stream/log"
)

func StreamFile(f string, broadcast func([]byte)) {

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
		"-re", // throttle to realtime
		"-i", f,

		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a", "aac",
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
