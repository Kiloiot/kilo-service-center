// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"time"

	"github.com/kilocenter/KC-Core/pkg/mioty"
)

// Session persistence timeouts per SCACI operational requirements
// These are exported as variables (not constants) to allow runtime configuration
// via environment variables or config files.
//
// Default values are production-tested; override only if you understand the
// SCACI handshake timing implications per §3.3.
var (
	// SessionPersistTimeout - DB writes for session state updates (§3.3)
	// Default: 2 seconds (tested with PostgreSQL 17.5 on local/networked DB)
	SessionPersistTimeout = 2 * time.Second

	// ConnectPersistTimeout - DB writes for connect completion (§3.3.3)
	// Default: 5 seconds (accounts for initial session creation overhead)
	ConnectPersistTimeout = 5 * time.Second

	// OperationRecordTimeout - Operation log persistence (§3.6-3.12)
	// Default: 2 seconds (async logging, non-critical path)
	OperationRecordTimeout = 2 * time.Second

	// ReplayOperationTimeout - DB lookup timeout for pending operations on session resume
	// Per SCACI §1: "operations which had not been completed before the connection loss are reissued"
	// Default: 5 seconds (matches ConnectPersistTimeout for consistency with session establishment)
	ReplayOperationTimeout = 5 * time.Second
)

// Future enhancement: Add Init(cfg *config.SCAConfig) to override from config file
// Example:
//   if cfg.SessionPersistTimeoutMs > 0 {
//       SessionPersistTimeout = time.Duration(cfg.SessionPersistTimeoutMs) * time.Millisecond
//   }

// Protocol version support per SCACI §2.1-2.3
const (
	// SupportedMajorVersion - SCACI protocol major version (§2.1)
	// Connection terminates if client requests different major version
	SupportedMajorVersion = 1

	// SupportedMinorVersion - SCACI protocol minor version (§2.2)
	// Connection terminates if client requests higher minor version
	SupportedMinorVersion = 0

	// ProtocolVersionString - Full version string for responses
	// Patch is always 0 per §2.3 (patch ignored in compatibility)
	ProtocolVersionString = "1.0.0"
)

// Operation ID constants per SCACI §3.2
const (
	// OpIDConnect is the operation ID used for the Connect handshake (§3.3)
	// Connect, ConnectResponse, and ConnectComplete MUST all use opId=0
	OpIDConnect int64 = 0
)

// ============================================================================
// DLDataResult.Result enum values per SCACI §3.12.1
// ============================================================================
const (
	ResultSent    = "sent"    // Downlink transmitted successfully
	ResultExpired = "expired" // Downlink expired before transmission
	ResultInvalid = "invalid" // Downlink payload invalid
	ResultRevoked = "revoked" // Internal-only: Downlink revoked by SC (NOT valid on SCACI wire per §3.12.1)
)

// ValidDLDataResults is the set of valid Result enum values per SCACI §3.12.1.
// Note: "revoked" is NOT a valid wire enum per spec - it is internal DB/BSSCI status only.
var ValidDLDataResults = map[string]bool{
	ResultSent:    true,
	ResultExpired: true,
	ResultInvalid: true,
}

// ============================================================================
// EPStatus.EpStatus enum values per SCACI §3.13.1
// Canonical constants are in pkg/mioty/format.go to avoid import cycles
// ============================================================================

// EPStatus enum aliases - re-exported from pkg/mioty to avoid import cycles in tests
// Canonical definitions remain in pkg/mioty/format.go per centralization rules
const (
	EPStatusAttached = mioty.EPStatusAttached // "attached" per SCACI §3.13.1
	EPStatusDetached = mioty.EPStatusDetached // "detached" per SCACI §3.13.1
)

// ValidEPStatuses is the set of valid EpStatus enum values per §3.13.1
// Uses mioty package constants to avoid duplication
var ValidEPStatuses = map[string]bool{
	EPStatusAttached: true,
	EPStatusDetached: true,
}

// ============================================================================
// Event Type Constants for system event logging
// Used by gRPC handlers and KC-DB storage for SCACI error event filtering
// ============================================================================
const (
	// EventTypeSCACIError - Event type for SCACI protocol errors
	// Used in system_events table for error event categorization and filtering
	EventTypeSCACIError = "scaciErr"
)

// ============================================================================
// Status Response Message Constants per SCACI §3.5.2
// Used by handleStatus and gRPC status endpoint for consistent messaging
// ============================================================================
const (
	// StatusMessageOperational - SC status message when all dependencies healthy per §3.5.2
	StatusMessageOperational = "Service Center operational"

	// StatusMessageDegraded - SC status message when dependencies unavailable per §3.5.2
	// Used when GetBaseStations or other status queries fail; indicates partial functionality
	StatusMessageDegraded = "Service Center degraded"
)

// ============================================================================
// Operation Metadata Keys for scaci_operation_log RequestData/ResponseData
// Exported for cross-module access (gRPC handlers use these for operation extraction)
// ============================================================================
const (
	// MetadataKeyEpEui is the JSON key for endpoint EUI in operation metadata.
	// Used in scaci_operation_log RequestData/ResponseData for §3.6 Register and §3.7 Deregister.
	// Exported for use by gRPC handlers (extractEpEuiFromRequestData).
	MetadataKeyEpEui = "epEui"

	// MetadataKeyCleanupSource indicates the source of epEui resolution for deregister cleanup.
	// Values: "cache" (in-memory from handleDeregister) or "db_fallback" (from scaci_operation_log).
	// Used in §3.7.3 DeregisterComplete cleanup flow.
	MetadataKeyCleanupSource = "cleanupSource"

	// MetadataKeyRevokedCount is the number of downlinks revoked during deregister cleanup per §3.7.3.
	MetadataKeyRevokedCount = "revokedCount"

	// MetadataKeyDetachErrorCount is the number of detach propagation errors during deregister cleanup per §3.7.3.
	MetadataKeyDetachErrorCount = "detachErrorCount"

	// MetadataKeyCleanupStatus indicates overall cleanup result for deregister operations per §3.7.3.
	// Values: "success" (all cleanup succeeded), "partial_failure" (some cleanup failed).
	MetadataKeyCleanupStatus = "cleanupStatus"

	// MetadataKeyErrorToken stores the error token for failed operations.
	// Used in operation ResponseData to record the cause of failure.
	MetadataKeyErrorToken = "errorToken"

	// MetadataKeyErrorDetail stores additional error context for failed operations.
	// Used in operation ResponseData alongside errorToken for debugging.
	MetadataKeyErrorDetail = "errorDetail"
)
