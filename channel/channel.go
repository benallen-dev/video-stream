package channel

import (
	"fmt"
)

// TODO: Flesh this guy out with things like a name and where to find media, icons, etc
// Idk maybe context for ffmpeg or whatever idk sky's the limit

// TODO:
// - Roll schedule into channel struct
// - Add some method of extracting metadata

type Channel struct {
	connections ConnectionList
	name        string
	Shows       []string
}

func New(name string, shows []string) *Channel {
	strMap := make(map[chan []byte]struct{})

	return &Channel{
		name:  name,
		Shows: shows,
		connections: ConnectionList{
			streams: strMap,
		},
	}
}

func (c *Channel) Name() string {
	return c.name
}

// Why is this on Channel when everything else is on Connections
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

func (c *Channel) String() string {
	s := ""
	if c.connections.Count() != 1 {
		s = "s"
	}
	
	return fmt.Sprintf("Channel: %s - %d client%s", c.name, c.connections.Count(), s)
}

func (c *Channel) Add() (chan []byte, func()) {
	// Are we the first?
	//   If so, get the schedule
	//   Figure out the offset
	//   Start ffmpeg
	// In all cases
	//   Register a stream with the connectionlist

	return c.connections.Add()
}

func (c *Channel) Count() int {
	return c.connections.Count()
}
