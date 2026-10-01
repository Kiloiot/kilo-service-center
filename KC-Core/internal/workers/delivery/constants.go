package delivery

import "errors"

// Log messages owned by the delivery worker.
const (
	LogDeliveryClaimFailed      = "delivery outbox claim failed"
	LogDeliveryMarkFailed       = "delivery outbox mark delivered failed"
	LogDeliveryAttemptFailed    = "delivery attempt failed, will retry"
	LogDeliveryRescheduleFailed = "delivery outbox reschedule failed"
	LogDeliveryParked           = "delivery parked after a permanent failure"
	LogDeliveryParkFailed       = "delivery outbox park failed"
	LogDeliveryParkEventFailed  = "delivery parked system event failed"
)

// System event vocabulary for parked deliveries.
const (
	eventTypeDeliveryParked           = "message.delivery.parked"
	eventTitleDeliveryParkedFmt       = "Message delivery to %s parked"
	eventDescriptionDeliveryParkedFmt = "Message %s could not be delivered to %s after %d attempts: %v"
)

// backoffGrowth is the factor the retry delay grows by per failed attempt.
const backoffGrowth = 2

// Construction failures of the worker and its retry policy.
const (
	errFmtRetryPolicyBounds = "%w: base %v, max %v"
	errFmtInvalidConfig     = "%w: poll interval %v, batch size %d, lease %v"
	errFmtUnpublishableAck  = "%w: queue id %d, window %d"
)

// Sentinel errors for delivery attempts; each attempt wraps the channel's
// own failure so the outbox row keeps the cause.
var (
	errLoadMessage          = errors.New("delivery worker: load message")
	errLoadDownlink         = errors.New("delivery worker: load acknowledged downlink")
	errUnpublishableAck     = errors.New("delivery worker: acknowledgement names no publishable downlink")
	errChannelNotConfigured = errors.New("delivery worker: channel not configured")
	errUnknownChannel       = errors.New("delivery worker: unknown channel")
	errMissingDependency    = errors.New("delivery worker: missing dependency")
	errInvalidConfig        = errors.New("delivery worker: invalid config")
	errInvalidRetryPolicy   = errors.New("delivery worker: retry backoff must be positive and at most the maximum backoff")
)
