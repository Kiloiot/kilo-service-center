package downlinkexpiry

import "errors"

// Log messages owned by the downlink expiry worker.
const (
	LogDownlinkExpirySweepFailed   = "downlink expiry sweep failed"
	LogDownlinksExpired            = "downlinks expired in the queue"
	LogExpiredDownlinkUnidentified = "expired downlink not reported: its endpoint or queue id cannot be parsed"
	LogExpiredDownlinkNotRevoked   = "expired downlink not revoked at the base station holding it"
)

// errFmtInvalidConfig reports an unusable sweep configuration.
const errFmtInvalidConfig = "%w: interval %v, batch size %d"

// Sentinel errors of the expiry worker.
var (
	errMissingDependency = errors.New("downlink expiry worker: missing dependency")
	errInvalidConfig     = errors.New("downlink expiry worker: invalid config")
)
