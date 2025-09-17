package channel

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"

	"video-stream/config"
	"video-stream/log"
)

type schedule struct {
	media     map[string][]mediafile
	scheduled []mediafile
}

func newSchedule(shows []string) *schedule {
	media, err := findMedia(shows)
	if err != nil {
		log.Error("could not find media", "msg", err.Error())
		return nil
	}

	return &schedule{
		media: media,
	}
}

func findMedia(dirs []string) (map[string][]mediafile, error) {

	out := make(map[string][]mediafile, 0)

	for _, dir := range dirs {
		cmd := exec.Command(
			"find",
			os.ExpandEnv(dir),
			"-type", "f",
			"-iregex", ".*\\.\\(mp4\\|mkv\\|mov\\|avi\\|flv\\|wmv\\|webm\\)$",
		)

		var buf bytes.Buffer
		cmd.Stdout = &buf

		err := cmd.Run()
		if err != nil {
			log.Debug(cmd.String())
			log.Error(buf.String())
			return nil, err
		}

		// Split output by newlines to get individual file paths
		files := strings.Split(strings.TrimSpace(buf.String()), "\n")

		showName := path.Base(dir)
		out[showName] = make([]mediafile, len(files))
		for i, f := range files {
			out[showName][i] = mediafile{path: f, show: showName}
		}
	}

	return out, nil
}

// Returns a copy of the generated schedule or an error
func (s *schedule) generate(ctx context.Context) ([]mediafile, error) {
	var c = 0

	rem, err := s.timeRemaining()
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			return []mediafile{}, errors.New("context canceled")
		default:

			if rem > config.Current.ScheduleHorizon || c > 100 { // Either longer than configured, or 100 files
				return s.scheduled, nil
			}

			rf := s.randomFile()

			log.Debug("appending new file to schedule", "file", rf.path)

			s.scheduled = append(s.scheduled, rf)
			dur, _ := rf.Duration()
			rem += dur
		}
	}
}

func (s *schedule) nextFile() {
	// pop next file off
}

func (s schedule) randomFile() mediafile {
	// Pick a random show
	randomIdx := rand.Intn(len(s.media))
	keys := slices.Collect(maps.Keys(s.media))
	key := keys[randomIdx]
	files := s.media[key]

	// Pick a random file
	randomIdx = rand.Intn(len(files))
	return files[randomIdx]
}

func (s schedule) timeRemaining() (time.Duration, error) {
	var total = time.Duration(0)

	for _, mf := range s.scheduled {
		// fileDuration, err := mf.DurationString() // caching this duration would improve performance
		// if err != nil {
		// 	return 0, err
		// }

		// dur, err := time.ParseDuration(fileDuration)
		// if err != nil {
		// 	return 0, err
		// }

		// If there's an error getting the duration, dur is 0
		dur, err := mf.Duration()
		if err != nil {
			log.Warn("Could not get duration", "mf", mf)
		}

		total += dur
	}

	return total, nil
}
