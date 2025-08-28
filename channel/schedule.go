package channel

import (
	"bytes"
	"os/exec"
	"path"
	"strings"

	"video-stream/log"
)

type schedule struct{
	media map[string][]string
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

func findMedia(dirs []string) (map[string][]string, error) {

	out := make(map[string][]string, 0)

	// For each show
	for _, dir := range dirs {
		// Find files in this directory
		// files, err := findShowFiles(show)
		// if err != nil {
		// 	return nil, err
		// }

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

		out[showName] = files
	}

	return out, nil
}

