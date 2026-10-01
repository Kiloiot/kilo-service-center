package federation

import "errors"

// Domain sentinels for federation service failures. Each wraps the underlying
// cause via fmt.Errorf("%w: %w", ...); cross-edition sentinels shared with the
// gRPC adapter live in KC-Core/pkg/federation.
var (
	// ErrInstallationCheck reports a failed installation lookup while deciding
	// whether CE onboarding is still required.
	ErrInstallationCheck = errors.New("ce installation check")
	// ErrOutboxEnqueue reports an uplink frame that could not be inserted into
	// the durable relay outbox.
	ErrOutboxEnqueue = errors.New("outbox enqueue")
	// ErrBootstrapDependencyMissing reports a bootstrap service built without
	// its installation store, logger or clock.
	ErrBootstrapDependencyMissing = errors.New("ce bootstrap service: missing dependency")
)
