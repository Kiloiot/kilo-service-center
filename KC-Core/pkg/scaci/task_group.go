package scaci

import (
	"sync"
	"time"
)

// taskGroup tracks goroutines whose starters are not all tracked themselves:
// once closed it refuses new tasks, so its wait cannot race a late start.
type taskGroup struct {
	mu     sync.Mutex
	closed bool
	wg     sync.WaitGroup
}

// start runs task unless the group is closed.
func (g *taskGroup) start(task func()) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return
	}
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		task()
	}()
}

// closeAndWait refuses new tasks and waits up to timeout for the running
// ones; it reports whether they all finished.
func (g *taskGroup) closeAndWait(timeout time.Duration) bool {
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()

	drained := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(drained)
	}()
	select {
	case <-drained:
		return true
	case <-time.After(timeout):
		return false
	}
}
