package channel

import (
	"bytes"
	"math/rand"
	"os/exec"
	"path"
	"maps"
	"strings"
	"slices"

	"video-stream/log"
)

type schedule struct{
	media map[string][]mediafile
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
			dir,
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
