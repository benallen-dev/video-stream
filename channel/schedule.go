package channel

import (
	"bytes"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"maps"
	"strings"
	"slices"
	"time"

	"video-stream/log"
)

type schedule struct{
	media map[string][]mediafile
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
			out[showName][i] = mediafile{ path: f }
		}
	}

	return out, nil
}

// Returns a copy of the generated schedule or an error
func (s *schedule) generate() ([]mediafile, error)  {
	var c = 0
	for {
		// Doing this for every loop is so insanely expensive
		rem, err := s.timeRemaining()
		if err != nil {
			return nil, err
		}

		if rem > time.Duration(24 * time.Hour) || c > 100 { // 24 hours or 100 files is enough
			return s.scheduled, nil
		}

		s.scheduled = append(s.scheduled, s.randomFile())
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
		fileDuration, err := mf.Duration() // caching this duration would improve performance
		if err != nil {
			return 0, err
		}

		dur, err := time.ParseDuration(fileDuration)
		if err != nil {
			return 0, err
		}

		total += dur
	}

	return total, nil
}
