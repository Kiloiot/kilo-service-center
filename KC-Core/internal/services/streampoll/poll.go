// Package streampoll turns a store read into a server stream of the rows
// stored after the stream opened. It reads in storage order, by the store's
// clock: every read covers the rows stored since the newest one it has read,
// reaching back by an overlap that catches the rows committed after a row
// stamped later, and sends each row it has not sent.
package streampoll

import (
	"context"
	"time"
)

// Fetch reads one page of at most the source's page size of the rows stored
// at or after since (every row when since is nil), newest stored first with
// rows stored at the same time always in the same order, skipping offset rows.
type Fetch[T any] func(ctx context.Context, since *time.Time, offset int) ([]T, error)

// Source describes the rows a stream delivers.
type Source[T any] struct {
	Fetch Fetch[T]
	// PageSize is the most rows one Fetch returns; a shorter page is the last.
	PageSize int
	// StoredAt is when the store stored the row; a row stored again later, as
	// an event moved forward in time is, is delivered again.
	StoredAt func(T) time.Time
	// Key identifies a row; with StoredAt it tells a row already sent apart.
	Key func(T) string
	// Overlap is how far each read reaches back behind the newest row read:
	// the longest a writer may take to commit a row after stamping it.
	Overlap time.Duration
	// OnError reports a failed read; the stream keeps polling.
	OnError func(error)
	// Wake announces stored rows so the stream reads before the next interval.
	Wake Waker
}

// Stream delivers, oldest stored first and once each, the rows stored after
// it opened, reading on every wake and every interval until ctx ends; the
// interval catches the rows whose announcement was lost.
func Stream[T any](ctx context.Context, interval time.Duration, buffer int, src Source[T]) <-chan T {
	out := make(chan T, buffer)
	go func() {
		defer close(out)
		c := &cursor[T]{src: src, seen: make(map[string]time.Time)}
		// The wake is taken before each read, so a row stored during the read wakes the next one.
		wake := src.Wake.Wake()
		c.baseline(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			case <-wake:
				wake = src.Wake.Wake()
			}
			if !c.deliver(ctx, out) {
				return
			}
		}
	}()
	return out
}

// cursor tracks the newest storage time a stream has read and the rows it has
// read within the overlap behind it; until a read succeeds it has no baseline
// and delivers nothing.
type cursor[T any] struct {
	src    Source[T]
	ready  bool
	latest *time.Time
	seen   map[string]time.Time
}

// baseline takes every row within the overlap of the newest one present as
// history, so only the rows stored after it are delivered.
func (c *cursor[T]) baseline(ctx context.Context) {
	rows, err := c.history(ctx)
	if err != nil {
		c.src.OnError(err)
		return
	}
	c.ready = true
	c.advance(rows)
}

// history reads the rows within the overlap of the newest row stored.
func (c *cursor[T]) history(ctx context.Context) ([]T, error) {
	newest, err := c.src.Fetch(ctx, nil, 0)
	if err != nil || len(newest) == 0 {
		return nil, err
	}
	since := c.src.StoredAt(newest[0]).Add(-c.src.Overlap)
	return c.read(ctx, &since)
}

// deliver sends the rows the cursor has not sent and reports false once ctx ended.
func (c *cursor[T]) deliver(ctx context.Context, out chan<- T) bool {
	if !c.ready {
		c.baseline(ctx)
		return ctx.Err() == nil
	}
	rows, err := c.read(ctx, c.since())
	if err != nil {
		c.src.OnError(err)
		return ctx.Err() == nil
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if c.known(rows[i]) {
			continue
		}
		select {
		case out <- rows[i]:
			c.seen[c.src.Key(rows[i])] = c.src.StoredAt(rows[i])
		case <-ctx.Done():
			return false
		}
	}
	c.advance(rows)
	return true
}

// since is where the next read starts: the overlap behind the newest row read.
func (c *cursor[T]) since() *time.Time {
	if c.latest == nil {
		return nil
	}
	since := c.latest.Add(-c.src.Overlap)
	return &since
}

// known reports whether the row, as stored now, was already read.
func (c *cursor[T]) known(row T) bool {
	storedAt, ok := c.seen[c.src.Key(row)]
	return ok && storedAt.Equal(c.src.StoredAt(row))
}

// advance records a complete read: the cursor moves to its newest row and
// forgets the rows the next read no longer reaches.
func (c *cursor[T]) advance(rows []T) {
	for _, row := range rows {
		storedAt := c.src.StoredAt(row)
		c.seen[c.src.Key(row)] = storedAt
		if c.latest == nil || storedAt.After(*c.latest) {
			c.latest = &storedAt
		}
	}
	since := c.since()
	if since == nil {
		return
	}
	for key, storedAt := range c.seen {
		if storedAt.Before(*since) {
			delete(c.seen, key)
		}
	}
}

// read reads every page of the rows stored since since, until a short page.
// A row stored during the read lands in a page already read or shifts rows
// into the next one, where they are read twice; its announcement wakes the
// next read, whose overlap reaches it.
func (c *cursor[T]) read(ctx context.Context, since *time.Time) ([]T, error) {
	var rows []T
	for {
		page, err := c.src.Fetch(ctx, since, len(rows))
		if err != nil {
			return nil, err
		}
		rows = append(rows, page...)
		if len(page) == 0 || len(page) < c.src.PageSize {
			return rows, nil
		}
	}
}
