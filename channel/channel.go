package channel

import (
	"sync"
	"video-stream/log"
)

// TODO: Flesh this guy out with things like a name and where to find media, icons, etc
// Idk maybe context for ffmpeg or whatever idk sky's the limit

type ConnectionList struct {
	streams map[chan []byte]struct{}
	mutex   sync.Mutex
}

func (cl *ConnectionList) Add() (chan []byte, func()) {
	ch := make(chan []byte, 4096)

	cl.mutex.Lock()
	cl.streams[ch] = struct{}{}
	cl.mutex.Unlock()

	cleanupFn := func() {
		log.Info("removing stream from channel")
		cl.mutex.Lock()
		delete(cl.streams, ch)
		close(ch)
		cl.mutex.Unlock()
	}

	return ch, cleanupFn
}

func (cl *ConnectionList) Count() int {
	cl.mutex.Lock()
	count := len(cl.streams)
	cl.mutex.Unlock()

	return count
}

type Channel struct {
	Connections ConnectionList
	name        string
	Shows       []string
}

func New(name string, shows []string) *Channel {
	strMap := make(map[chan []byte]struct{})

	return &Channel{
		name:  name,
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
