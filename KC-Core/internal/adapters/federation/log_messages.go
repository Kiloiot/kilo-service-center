package federation

// Relay client log messages. Cross-edition relay messages shared with
// enterprise tooling live in KC-Core/pkg/federation; these belong to the
// community relay client alone.
const (
	// LogOnboardingIncomplete reports a start attempt on an installation that
	// has not completed CE onboarding.
	LogOnboardingIncomplete = "CE onboarding not complete; federation relay will not start"
	// LogLoopRevokedExit reports the reconnect loop exiting because the CE was
	// revoked; the client never reconnects afterward.
	LogLoopRevokedExit = "CE revoked; relay loop exiting permanently"
	// LogStreamClosedReconnecting reports a dropped stream and the backoff
	// before the next reconnect attempt.
	LogStreamClosedReconnecting = "Federation relay stream closed; reconnecting"
	// LogTransportUnencrypted warns that the relay dials the ECE without TLS.
	LogTransportUnencrypted = "Federation relay transport is unencrypted; enable TLS for production use"
	// LogStreamEstablished reports a stream that authenticated successfully.
	LogStreamEstablished = "Federation relay stream established"
	// LogHeartbeatAcked reports an ECE heartbeat acknowledgement.
	LogHeartbeatAcked = "Heartbeat acknowledged"
	// LogRevokedByECE reports a revoke notice from the ECE; the relay halts
	// permanently.
	LogRevokedByECE = "CE instance revoked by ECE; relay permanently halted"
	// LogInvalidReceiptRelayID reports an uplink receipt whose relay_id did
	// not parse as a UUID.
	LogInvalidReceiptRelayID = "Invalid relay_id in receipt"
	// LogRelayRejected reports an uplink the ECE refused to accept.
	LogRelayRejected = "Relay rejected by ECE"
	// LogRelayReceiptNotRecorded reports an ECE receipt the outbox could not
	// record; the record stays sent and is replayed on reconnect.
	LogRelayReceiptNotRecorded = "Failed to record relay receipt"
	// LogRelayConnCloseFailed reports a relay connection that failed to close.
	LogRelayConnCloseFailed = "Failed to close federation relay connection"
)
