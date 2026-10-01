package streampoll

// Waker announces that rows may have been stored.
type Waker interface {
	// Wake returns a channel closed once rows may have been stored after the call.
	Wake() <-chan struct{}
}
