// Package streamwake fans a stored-row announcement out to every live stream
// waiting on it.
package streamwake

import "sync"

// Signal wakes every stream waiting on it; the notifications between two
// reads of a stream wake it once.
type Signal struct {
	mu   sync.Mutex
	wake chan struct{}
}

// NewSignal returns a Signal no notification has woken yet.
func NewSignal() *Signal {
	return &Signal{wake: make(chan struct{})}
}

// Wake returns the channel the next Notify closes.
func (s *Signal) Wake() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wake
}

// Notify wakes every stream waiting on the signal without blocking.
func (s *Signal) Notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	close(s.wake)
	s.wake = make(chan struct{})
}
