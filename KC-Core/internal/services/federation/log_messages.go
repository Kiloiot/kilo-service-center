package federation

// Log messages for the federation domain services. Relay transport messages
// belong to the relay client adapter and KC-Core/pkg/federation.
const (
	// LogInstallationFetchFailed reports a CE installation record that could
	// not be read while serving a status request.
	LogInstallationFetchFailed = "Failed to fetch CE installation"
	// LogOnboardingCompleted reports a newly completed CE onboarding.
	LogOnboardingCompleted = "CE onboarding completed"
	// LogRelayStartFailed reports a relay client that could not start after
	// onboarding; onboarding itself still succeeds.
	LogRelayStartFailed = "Failed to start federation relay after onboarding"
	// LogRelayStartDeferredOnboarding reports a relay start attempt made
	// while CE onboarding is still incomplete; the relay stays idle.
	LogRelayStartDeferredOnboarding = "Federation relay start deferred: CE onboarding not complete"
	// LogUplinkEnqueued reports an uplink frame accepted into the relay outbox.
	LogUplinkEnqueued = "Uplink enqueued for federation relay"
)
