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
	Connections ConnectionList
	Name        string
	Shows       []string
}

func New(name string, shows []string) *Channel {
	strMap := make(map[chan []byte]struct{})

	return &Channel{
		Name:  name,
		Shows: shows,
		Connections: ConnectionList{
			streams: strMap,
		},
	}
}

// Why is this on Channel when everything else is on Connections
func (c *Channel) Broadcast(data []byte) {
	c.Connections.mutex.Lock()
	defer c.Connections.mutex.Unlock()
	for ch := range c.Connections.streams {
		select {
		case ch <- data:
		default:
			// drop if client is too slow
		}
	}
}

func (c *Channel) String() string {
	s := ""
	if c.Connections.Count() != 1 {
		s = "s"
	}
	
	return fmt.Sprintf("Channel: %s - %d client%s", c.Name, c.Connections.Count(), s)
}

