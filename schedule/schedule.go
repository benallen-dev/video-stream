package schedule

import (
	"bytes"
	"os/exec"
	"strings"

	"math/rand"

	"video-stream/log"
)

var shows = []string{
}

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
		return nil, err
	}

	// Split output by newlines to get individual file paths
	files := strings.Split(strings.TrimSpace(out.String()), "\n")

	return files, nil
}

func findMedia() (*[]string, error) {

	// We don't know how long this list will be, so let's just use a slice and accept some reallocation
	// The average show runs for about 200 episodes so we'll initialise it with 200*len(shows)
	out := make([]string, 0, 200*len(shows))

	for _, show := range shows {
		files, err := findShowFiles(show)
		if err != nil {
			return nil, err
		}

		out = append(out, files...)
	}

	return &out, nil
}


func Example() error {
	res, err := findMedia()
	if err != nil {
		return err
	}

	for _, file := range *res {
		log.Info(file)
	}

	return nil
}

// Returns the absolute path to the next video file
// We're assuming there's only one channel for now
func RandomFile() (string, error) {
	files, err := findMedia()
	if err != nil {
		return "", err
	}

	// Pick a random file
	randomIdx := rand.Intn(len(*files))
	return (*files)[randomIdx], nil
}
