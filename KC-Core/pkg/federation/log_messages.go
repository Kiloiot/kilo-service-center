package federation

import "errors"

// Relay transport errors.
var (
	// ErrRelayCACertParse reports a CA certificate file that could not be
	// parsed for the relay's TLS configuration.
	ErrRelayCACertParse = errors.New("federation: failed to parse the relay CA certificate")
	// ErrEndpointIndexPrewarm reports a failed enumeration while pre-warming
	// the ingress disposition index; the resolver falls back to lazy warming.
	ErrEndpointIndexPrewarm = errors.New("federation: endpoint index pre-warm failed")
	// ErrRelayStopped reports a start attempt on a relay client that was
	// permanently stopped by an explicit Stop or by an ECE revocation.
	ErrRelayStopped = errors.New("federation: relay client is stopped")
	// ErrRelayRevoked reports a relay stream terminated by an ECE revocation
	// notice; the client latches stopped and never reconnects.
	ErrRelayRevoked = errors.New("federation: CE instance revoked by ECE")
	// ErrRelayOnboardingIncomplete reports a start attempt made before CE
	// onboarding completed; the relay stays idle until onboarding finishes.
	ErrRelayOnboardingIncomplete = errors.New("federation: CE onboarding not complete")
)

// Relay log messages, centralized so relay logging stays consistent between
// the community client and enterprise tooling.
const (
	// LogRelayTokenPersistFailed reports a newly issued federation token that
	// could not be stored.
	LogRelayTokenPersistFailed = "Failed to persist new federation token"
	// LogRelayOutboxReplayReadFailed reports an outbox read failure during
	// reconnect replay.
	LogRelayOutboxReplayReadFailed = "Failed to read outbox for reconnect replay"
	// LogRelayOutboxReadFailed reports an outbox read failure during normal
	// draining.
	LogRelayOutboxReadFailed = "Failed to read outbox"
)
