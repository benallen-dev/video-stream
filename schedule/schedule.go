package schedule

import (
	"bytes"
	"maps"
	"math/rand"
	"os/exec"
	"path"
	"slices"
	"strings"

	"video-stream/channel"
	"video-stream/log"
)

// Returns a list of media files contained in the root path
func findShowFiles(root string) ([]string, error) {

	cmd := exec.Command(
		"find",
		root,
		"-type", "f",
		"-iregex", ".*\\.\\(mp4\\|mkv\\|mov\\|avi\\|flv\\|wmv\\|webm\\)$",
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Debug(cmd.String())
		log.Error(out.String())
		return nil, err
	}

	// Split output by newlines to get individual file paths
	files := strings.Split(strings.TrimSpace(out.String()), "\n")

	return files, nil
}

func findMedia(chnl *channel.Channel) (map[string][]string, error) {

	out := make(map[string][]string, 0)

	for _, show := range chnl.Shows {
		files, err := findShowFiles(show)
		if err != nil {
			return nil, err
		}

		showName := path.Base(show)

		out[showName] = files
	}

	return out, nil
}

// Returns the absolute path to the next video file
// We're assuming there's only one channel for now
func RandomFile(ch *channel.Channel) (string, error) {
	media, err := findMedia(ch)
	if err != nil {
		log.Warn("Could not find media", "channel", ch.Name)
		return "", err
	}

	// Pick a random show
	randomIdx := rand.Intn(len(media))
	keys := slices.Collect(maps.Keys(media))
	key := keys[randomIdx]
	files := media[key]

	log.Info("Playing "+key, "channel", ch.Name)

	// Pick a random file
	randomIdx = rand.Intn(len(files))
	return files[randomIdx], nil
}
