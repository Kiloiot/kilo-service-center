package downlinkid

const (
	// DefaultAttempts bounds the enqueue retries on a taken queue id.
	DefaultAttempts = 5
	// maxQueueID keeps service center queue ids within JavaScript's exact
	// integer range (2^53-1) so JSON consumers, MQTT events included, read them exactly.
	maxQueueID = int64(1)<<53 - 1
)
