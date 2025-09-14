package channel

import (
	"sync"
	"video-stream/log"
)

type connectionList struct {
	mu      sync.Mutex
	streams map[chan []byte]struct{}
}

func (cl *connectionList) add() (chan []byte, func() int) {
	ch := make(chan []byte, 4096)

	cl.mu.Lock()
	cl.streams[ch] = struct{}{}
	cl.mu.Unlock()

	cleanupFn := func() int {
		log.Info("removing stream from channel")
		cl.mu.Lock()
		delete(cl.streams, ch)
		close(ch)
		cl.mu.Unlock()
		return len(cl.streams)
	}

	return ch, cleanupFn
}

func (cl *connectionList) broadcast(data []byte) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	for ch := range cl.streams {
		select {
		case ch <- data:
		default:
			// drop if client is too slow
		}
	}
}
func (cl *connectionList) Count() int {
	cl.mu.Lock()
	count := len(cl.streams)
	cl.mu.Unlock()

	return count
}
