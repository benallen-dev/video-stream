package channel

import (
	"sync"
	"video-stream/log"
)

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
