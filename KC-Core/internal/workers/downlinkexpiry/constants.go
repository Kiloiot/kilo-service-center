package downlinkexpiry

import "errors"

// Log messages owned by the downlink expiry worker.
const (
	LogDownlinkExpirySweepFailed   = "downlink expiry sweep failed"
	LogDownlinksExpired            = "downlinks expired in the queue"
	LogDownlinksRevoking           = "overdue downlinks held by base stations are being revoked"
	LogRevocationsAskedAgain       = "connected base stations that left a revoke unanswered are asked again"
	LogExpiredDownlinkUnidentified = "expired downlink not reported: its endpoint or queue id cannot be parsed"
	LogOverdueDownlinkNotRevoked   = "overdue downlink stays revoking: the base station holding it is not reachable now"
)

// errFmtInvalidConfig reports an unusable sweep configuration.
const errFmtInvalidConfig = "%w: interval %v, batch size %d"

// Sentinel errors of the expiry worker.
var (
	errMissingDependency = errors.New("downlink expiry worker: missing dependency")
	errInvalidConfig     = errors.New("downlink expiry worker: invalid config")
)
