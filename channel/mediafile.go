package channel

import (
	"bytes"
	"fmt"
	"strings"
	"strconv"
 	"os/exec"

	"video-stream/log"
)

type mediafile struct {
	path string
}

func (mf mediafile) Duration() (string, error) {

	cmd := exec.Command(
		"ffprobe",
		"-i", mf.path,
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


func (mf mediafile) Languages() (map[int]string, error) {
	// Run ffprobe to extract audio stream indexes and language tags
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=index:stream_tags=language",
		"-of", "csv=p=0",
		mf.path,
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

func (mf *mediafile) hasEnglishAudio() bool {

	langs, err := mf.Languages()
	if err != nil {
		log.Fatal("could not get audio languages", "msg", err.Error())
	}

	hasEng := false
	l := []string{}
	for _, lang := range langs {
		l = append(l, lang)
		if lang == "eng" {
			hasEng = true
		}
	}

	log.Debugf("Languages: %v", l)
	return hasEng
}
