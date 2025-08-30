package channel

import (
	"fmt"
	"time"

	"video-stream/log"
)

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

	return c.connections.add()
}


func (c *Channel) Start() {
	for {
		streamFile(c.schedule.randomFile(), c.connections.broadcast)

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
