package downlinkid

import "errors"

var (
	// ErrEntropyUnavailable wraps a generator failure: no queue id could be drawn.
	ErrEntropyUnavailable = errors.New("downlink queue id entropy unavailable")
	// ErrAttemptsExhausted wraps the last collision after every attempt was refused.
	ErrAttemptsExhausted = errors.New("downlink queue id attempts exhausted")
	// ErrNilGenerator rejects an allocator built without a queue id generator.
	ErrNilGenerator = errors.New("downlink queue id generator is nil")
	// ErrInvalidAttempts rejects an allocator built with no attempts to make.
	ErrInvalidAttempts = errors.New("downlink queue id attempts must be positive")
	// ErrNilEnqueue rejects an allocation with no enqueue to run.
	ErrNilEnqueue = errors.New("downlink enqueue is nil")
)
