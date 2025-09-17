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

type scheduleItem struct {
	mediafile mediafile
	start     time.Time
	end       time.Time
}

type schedule struct {
	media     map[string][]mediafile
	scheduled []scheduleItem
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
func (s *schedule) generate(ctx context.Context) ([]scheduleItem, error) {
	for {
		select {
		case <-ctx.Done():
			return []scheduleItem{}, errors.New("context canceled")
		default:
			var endTime time.Time

			if len(s.scheduled) > 0 {
				endTime = s.scheduled[len(s.scheduled)-1].end
			} else {
				endTime = time.Now()
			}

			if endTime.After(time.Now().Add(config.Current.ScheduleHorizon)) || len(s.scheduled) >= 100 { // Either longer than configured, or 100 files
				return s.scheduled, nil
			}

			rf := s.randomFile()
			dur, _ := rf.Duration() // don't care about errors here

			log.Debug("appending new file to schedule", "file", rf.path)
			si := scheduleItem{
				mediafile: rf,
				start:     endTime,
				end:       endTime.Add(dur).Add(time.Second), // add a little margin
			}

			s.scheduled = append(s.scheduled, si)
		}
	}
}

func (s *schedule) nextFile() mediafile {
	// TODO:
	// - determine if time.Now() exists inside s.scheduled
	// - if not, call generate() before continuing
	// - play the correct file at the correct start time

	next := s.scheduled[0]
	s.scheduled = s.scheduled[0:] // because scheduled is max 100 items it's not worth using a linked list or whatever

	return next.mediafile
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

// Unused?
func (s schedule) timeRemaining() (time.Duration, error) {

	// get last item
	if len(s.scheduled) == 0 {
		return time.Duration(0), nil
	}

	last := s.scheduled[len(s.scheduled)-1]
	// get its end time
	return time.Until(last.end), nil
}
