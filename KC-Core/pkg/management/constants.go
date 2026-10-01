// Package management provides HTTP management interfaces for BSSCI operations
package management

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
)

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

// Log messages for the management API. Unexported: this package is their only
// consumer, and routing them through the BSSCI catalog would imply they are
// part of the protocol vocabulary, which they are not.
const (
	logMgmtServerStarting             = "Starting BSSCI management HTTP server"
	logMgmtServerListening            = "BSSCI management HTTP server listening"
	logMgmtDecodeFailed               = "Failed to decode JSON"
	logMgmtAttachPropagateReceived    = "Received attach propagate request"
	logMgmtRepetitionEnabled          = "Repetition is enabled - may affect DL performance"
	logMgmtAttachPropagateAllStarting = "Starting SendAttachPropagateToAll"
	logMgmtAttachPropagateFailed      = "Attach propagate failed"
	logMgmtAttachPropagateAllOK       = "Attach propagate sent successfully to all sessions"
	logMgmtDetachPropagateAllStarting = "Starting SendDetachPropagateToAll"
	logMgmtDetachPropagateFailed      = "Detach propagate failed"
	logMgmtDetachPropagateAllOK       = "Detach propagate sent successfully to all sessions"
)

// Keys of the session maps returned by bssci.Server.GetConnectedSessions.
// The management API reads them by name, so they are named here rather than
// spelled out at each use site.
const (
	sessionKeyID                = bssci.SessionKeyID
	sessionKeyBaseStationEUI    = bssci.SessionKeyBaseStationEUI
	sessionKeyName              = bssci.SessionKeyName
	sessionKeyVendor            = bssci.SessionKeyVendor
	sessionKeyModel             = bssci.SessionKeyModel
	sessionKeyConnected         = bssci.SessionKeyConnected
	sessionKeyLastSeen          = bssci.SessionKeyLastSeen
	sessionKeyClientVersion     = bssci.SessionKeyClientVersion
	sessionKeyNegotiatedVersion = bssci.SessionKeyNegotiatedVersion
	sessionKeyBidirectional     = bssci.SessionKeyBidirectional
	sessionKeyHandshakeComplete = bssci.SessionKeyHandshakeComplete
	sessionKeyBsOpID            = "bsOpId"
	sessionKeyScOpID            = "scOpId"
	sessionKeyEncoding          = "encoding"
	sessionKeyCanResume         = "canResume"
	sessionKeyResolvedTenantID  = bssci.SessionKeyResolvedTenantID
	sessionKeyOrganizationID    = bssci.SessionKeyOrganizationID
	sessionKeySessionUUID       = "sessionUuid"
	sessionKeySnBsUUID          = "snBsUuid"
	sessionKeySnScUUID          = "snScUuid"
	sessionKeyGeoLocation       = "geoLocation"
	sessionKeyConnectInfo       = "connectInfo"
)
