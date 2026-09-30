// Package clock abstracts wall-clock reads so time-dependent behavior can be
// driven deterministically in tests.
package clock

import "time"

// Clock supplies the current time.
type Clock interface {
	Now() time.Time
}

// SystemClock reads the process wall clock.
type SystemClock struct{}

// Now returns time.Now().
func (SystemClock) Now() time.Time { return time.Now() }
