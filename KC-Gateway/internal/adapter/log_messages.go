// Package adapter log message constants.
// Names structured log messages produced by the KC-Identity adapter so the
// verify-constants gate can enforce zero hardcoded log literals in this package.
package adapter

const (
	// LogGatewayPlatformEventViaIdentityRecordFailed is logged when the gateway's
	// outbound RPC to KC-Identity to record a platform event returns an error.
	// The event is dropped after this log (no retry).
	LogGatewayPlatformEventViaIdentityRecordFailed = "failed to record platform event via KC-Identity"
)
