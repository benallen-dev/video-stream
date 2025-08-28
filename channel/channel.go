package channel

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"time"

	"video-stream/ffmpeg"
	"video-stream/log"
)

// TODO: Flesh this guy out with things like a name and where to find media, icons, etc
// Idk maybe context for ffmpeg or whatever idk sky's the limit

// TODO:
// - Add some method of extracting metadata

// Channel behaves like an old school TV channel, except it's streaming MPEG-TS
// instead of analogue TV signals.
//
// Also, the fact that Go has the concept of a 'channel' makes this name super
// inconvenient but TV got there first.
type Channel struct {
	name        string
	schedule    *schedule
	connections *connectionList
}

func New(name string, shows []string) *Channel {
	strMap := make(map[chan []byte]struct{})

	return &Channel{
		name:     name,
		schedule: newSchedule(shows),
		connections: &connectionList{
			streams: strMap,
		},
	}
}

func (c *Channel) Name() string {
	return c.name
}

func (c *Channel) AddClient() (chan []byte, func()) {
	// Are we the first?
	//   If so, get the schedule
	//   Figure out the offset
	//   Start ffmpeg
	// In all cases
	//   Register a stream with the connectionlist

	return c.connections.Add()
}

func (c *Channel) Broadcast(data []byte) {
	c.connections.mutex.Lock()
	defer c.connections.mutex.Unlock()
	for ch := range c.connections.streams {
		select {
		case ch <- data:
		default:
			// drop if client is too slow
		}
	}
}

func (c *Channel) Start() {
	for {
		log.Info("Starting new file", "channel", c.name)
		f, err := c.RandomFile()
		if err != nil {
			log.Error("error getting random file", "msg", err.Error(), "channel", c.name)
			continue
		}

		ffmpeg.StreamFile(f, c.Broadcast)

		// Space out new files a little bit so clients can catch up
		var DELAY = 2
		for i := range DELAY {
			log.Info(fmt.Sprintf("Waiting %d", DELAY-i))
			time.Sleep(time.Second) // just a hunch
		}
	}
}

// Useful for debugging but not something I actually want to expose

func (c *Channel) String() string {
	s := ""
	if c.connections.Count() != 1 {
		s = "s"
	}

	return fmt.Sprintf("Channel: %s - %d client%s", c.name, c.connections.Count(), s)
}

func (c *Channel) Count() int {
	return c.connections.Count()
}

func (c *Channel) RandomFile() (string, error) {

	// Pick a random show
	randomIdx := rand.Intn(len(c.schedule.media))
	keys := slices.Collect(maps.Keys(c.schedule.media))
	key := keys[randomIdx]
	files := c.schedule.media[key]

	log.Info("Playing "+key, "channel", c.name)

	// Pick a random file
	randomIdx = rand.Intn(len(files))
	return files[randomIdx], nil
}
