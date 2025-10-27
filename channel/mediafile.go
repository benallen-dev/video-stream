package channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"video-stream/log"
)

// TODO: Get all metadata once with ffprobe, parse JSON instead of having
// 3 different methods and CSV parsing and stuff.

type mediafile struct {
	name      string
	show      string
	path      string
	duration  time.Duration
	languages map[int]string
}

func (mf *mediafile) Name() string {
	if mf.name == "" {
		if err := mf.LoadMetadata(); err != nil {
			log.Warn("error loading metadata", "mediafile", mf.path, "error", err.Error())
		}
	}

	return mf.name
}

func (mf *mediafile) ShowName() string {
	return mf.show
}

// TODO: Do this for all metadata fields at once
func (mf *mediafile) LoadMetadata() error {
	cmd := exec.Command(
		"ffprobe",
		"-i", mf.path,
		"-show_entries", "stream=index:stream_tags=language:format_tags=title,show,season_number,episode_sort:format=duration",
		"-v", "quiet",
		"-of", "json",
	)

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ffprobe failed: %w", err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
			Tags     struct {
				Title   string `json:"title"`
				Show    string `json:"show"`
				Season  string `json:"season_number"`
				Episode string `json:"episode_sort"`
			} `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	pretty, _ := json.MarshalIndent(result, "", "  ")
	log.Info("ffprobe result", "pretty", string(pretty))

	// Deal with missing metadata
	if (result.Format.Tags.Title != "") {
		mf.name = result.Format.Tags.Title
	} else {
		mf.name = mf.path
	}

	if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
		mf.duration = time.Duration(duration * float64(time.Second))
	}

	return nil
}

func (mf *mediafile) Duration() (time.Duration, error) {
	if mf.duration != 0 {
		log.Debug("Using cached duration")
		return mf.duration, nil
	}
	cmd := exec.Command(
		"ffprobe",
		"-i", mf.path,
		"-show_entries", "format=duration",
		"-v", "quiet",
		"-of", "csv=p=0",
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	durationStr := strings.TrimSpace(string(out))
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	mf.duration = time.Duration(durationSec * float64(time.Second))
	return mf.duration, nil
}

// Can't we just replace this with .Duration().Format(time.DateTime) ?
func (mf *mediafile) DurationString() (string, error) {
	dd, err := mf.Duration()
	if err != nil {
		return "UNKNOWN", err
	}

	durationSec := dd.Seconds()

	minutes := int(durationSec) / 60
	seconds := int(durationSec) % 60
	return fmt.Sprintf("%02dm%02ds", minutes, seconds), nil
}

func (mf *mediafile) Languages() (map[int]string, error) {
	if mf.languages != nil {
		return mf.languages, nil
	}

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

	mf.languages = languages

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
