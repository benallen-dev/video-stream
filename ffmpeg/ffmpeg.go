package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"

	"video-stream/log"
)

func englishAudioFfmpegCommand(f string) *exec.Cmd {

	// Find if english stream exists using ffprobe
	langs, err := getAudioLanguages(f)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Info("Languages:")
	hasEng := false
	for _, lang := range langs {
		log.Info(lang)
		if lang == "eng" {
			hasEng = true
		}
	}

	ffmpegArgs := []string{
		// Avoid timestamp funkiness
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",

		// Get input
		// "-sseof", "-10", // start N seconds from the end
		"-ss", "45", // skip the first 45 seconds
		"-re", // throttle to realtime
		"-i", f,

		// Map streams
		"-map", "0:v:0",

		// Re-encode video to h.264 1920x1080
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-vf", "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2", // letterbox 1080p

		// Re-encode audio to 48kHz stereo AAC
		"-c:a", "aac",
		"-ar", "48000",
		"-ac", "2",
		"-b:a", "128k",
	}

	if hasEng {
		log.Debug("Mapping eng audio stream")
		ffmpegArgs = append(ffmpegArgs, "-map", "0:a:m:language:eng")
	} else {
		log.Debug("Mapping all audio streams")
		ffmpegArgs = append(ffmpegArgs, "-map", "0:a")
	}

	ffmpegArgs = append(ffmpegArgs,
		"-f", "mpegts", // format into mpegts so we can just dump it over http
		"pipe:1", // use stdout so we can pipe it into our go program
		)

	// log.Info(cmd.String())
	return exec.Command("ffmpeg", ffmpegArgs...)
}

func fifoPlaylistFfmpegCommand() {
	playlist := "playlist.txt"

	// Create FIFO if it doesn't exist yet
	if _, err := os.Stat(playlist); os.IsNotExist(err) {
		if err := exec.Command("mkfifo", playlist).Run(); err != nil {
			log.Fatalf("failed to create fifo: %v", err)
		}
	}

	// Open FIFO for writing and KEEP it open
	// (ffmpeg exits if it hits EOF on this pipe)
	playlistWriter, err := os.OpenFile(playlist, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		log.Fatalf("failed to open fifo: %v", err)
	}
	defer playlistWriter.Close()

	// Start long-lived ffmpeg encoder
	cmd := exec.Command(
		"ffmpeg",
		"-re",
		"-f", "concat",
		"-safe", "0",
		"-i", playlist,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a", "aac",
		"-ar", "48000",
		"-b:a", "128k",
		"-f", "mpegts",
		"pipe:1",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start ffmpeg: %v", err)
	}

	// Dynamically append files to the playlist
	addFile := func(filepath string) {
		abs  := path.Dir(filepath)
		line := fmt.Sprintf("file '%s'\n", abs)
		if _, err := playlistWriter.WriteString(line); err != nil {
			log.Errorf("failed to write to fifo: %v", err)
		} else {
			log.Infof("queued: %s", abs)
		}
	}

	// Example: queue two videos
	addFile("video1.mp4")
	addFile("video2.mp4")

	// Wait for ffmpeg to exit (will run until playlist runs dry)
	if err := cmd.Wait(); err != nil {
		log.Fatalf("ffmpeg exited: %v", err)
	}
}

func basicFfmpegCommand(f string) *exec.Cmd {
	return exec.Command(
		"ffmpeg",
		// Avoid timestamp funkiness
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",

		// Get input
		// "-sseof", "-10", // start N seconds from the end
		"-re", // throttle to realtime
		"-i", f,

		// Map streams
		"-map", "0:v:0",
		"-map", "0:a:0?",

		// Re-encode video to h.264 1920x1080
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-vf", "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2", // letterbox 1080p

		// Re-encode audio to 48kHz stereo AAC
		"-c:a", "aac",
		"-ar", "48000",
		"-ac", "2",
		"-b:a", "128k",

		// "-map", "0:a:m:language:eng",
		"-f", "mpegts",
		"pipe:1",
	)
}

func StreamFile(f string, broadcast func([]byte)) {

	// cmd := basicFfmpegCommand(f)
	cmd := englishAudioFfmpegCommand(f)

	dur, err := getDuration(f)
	if err != nil {
		log.Warn("Couldn't get file duration", "error", err.Error())
	}
	log.Info("Running ffmpeg", "file", path.Base(f), "duration", dur)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err.Error())
	}

	// TODO: Figure out how to get output from stderr for debugging without blocking
	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }

	if err := cmd.Start(); err != nil {
		log.Fatal(err.Error())
	}

	var innerWg sync.WaitGroup

	// Pump ffmpeg → broadcast
	innerWg.Go( func() {
		buf := make([]byte, 4096)

		for {
			n, err := stdout.Read(buf)
			if err != nil {
				log.Info("ffmpeg ended:", "reason", err)
				log.Debug(cmd.String())

				// for {
				// 	n, err := stderr.Read(buf)
				// 	log.Debug("Stderr:", "contents", string(buf[:n]))
				// 	if err != nil {
				// 		log.Debug("Error reading stderr:", "error", err.Error())
				// 		break
				// 	}
				// }
				break
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				broadcast(data)
			}
			if n == 0 {
				log.Warn("Read zero bytes")
			}
		}
	})

	innerWg.Wait()
}
