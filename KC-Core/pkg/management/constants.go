// Package management provides HTTP management interfaces for BSSCI operations
package management

import "time"

// HTTP server timeouts for management API (BSSCI propagation endpoints)
const (
	// HTTPReadHeaderTimeout prevents Slowloris DoS attacks (gosec G112)
	// BSSCI management API serves internal-only attach/detach propagation
	// operations to all connected base station sessions.
	HTTPReadHeaderTimeout = 30 * time.Second

	// HTTPIdleTimeout closes idle keep-alive connections
	// Matches KC-DB/common/config DefaultIdleTimeout (60s) for consistency
	// across HTTP servers in the KiloCenter stack.
	HTTPIdleTimeout = 60 * time.Second
)
