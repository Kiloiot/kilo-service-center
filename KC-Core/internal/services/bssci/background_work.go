package bssciservices

import (
	"context"
	"sync"
)

// BackgroundWork runs work detached from the request that started it; the
// composition root builds one per process and stops it at shutdown.
type BackgroundWork struct {
	wg sync.WaitGroup
}

// NewBackgroundWork returns the process's background runner.
func NewBackgroundWork() *BackgroundWork {
	return &BackgroundWork{}
}

// Go runs fn with ctx on a goroutine Stop accounts for.
func (w *BackgroundWork) Go(ctx context.Context, fn func(context.Context)) {
	w.wg.Go(func() { fn(ctx) })
}

// Stop blocks until every goroutine started by Go has returned, or until ctx ends.
func (w *BackgroundWork) Stop(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
