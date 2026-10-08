// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
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

	// ShutdownPersistDrainTimeout bounds how long Stop waits for detached
	// session-state writes to finish before cancelling their context.
	ShutdownPersistDrainTimeout = 5 * time.Second

	// ConnectPersistTimeout - DB writes for connect completion (§3.3.3)
	// Default: 5 seconds (accounts for initial session creation overhead)
	ConnectPersistTimeout = 5 * time.Second

	// ReplayOperationTimeout - DB lookup timeout for pending operations on session resume
	// Per SCACI §1: "operations which had not been completed before the connection loss are reissued"
	// Default: 5 seconds (matches ConnectPersistTimeout for consistency with session establishment)
	ReplayOperationTimeout = 5 * time.Second
)

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

// validDLDataResults is the set of valid Result enum values per SCACI §3.12.1.
// Note: "revoked" is NOT a valid wire enum per spec - it is internal DB/BSSCI status only.
var validDLDataResults = map[string]bool{
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

// validEPStatuses is the set of valid EpStatus enum values per §3.13.1
// Uses mioty package constants to avoid duplication
var validEPStatuses = map[string]bool{
	EPStatusAttached: true,
	EPStatusDetached: true,
}

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
	// MetadataKeyEpEui is the JSON key for endpoint EUI in operation metadata,
	// recorded as the hex text mioty.FormatEUI64 writes.
	MetadataKeyEpEui = "epEui"

	// MetadataKeyRevokedCount is the number of downlinks revoked during deregister cleanup per §3.7.3.
	MetadataKeyRevokedCount = "revokedCount"

	// MetadataKeyDetachErrorCount is the number of detach propagation errors during deregister cleanup per §3.7.3.
	MetadataKeyDetachErrorCount = "detachErrorCount"

	// MetadataKeyCleanupStatus indicates overall cleanup result for deregister operations per §3.7.3.
	// Values: opStatusSuccess (all cleanup succeeded), opStatusPartialFailure (some cleanup failed).
	MetadataKeyCleanupStatus = "cleanupStatus"

	// MetadataKeyErrorToken stores the error token for failed operations.
	// Used in operation ResponseData to record the cause of failure.
	MetadataKeyErrorToken = "errorToken"

	// MetadataKeyErrorDetail stores additional error context for failed operations.
	// Used in operation ResponseData alongside errorToken for debugging.
	MetadataKeyErrorDetail = "errorDetail"
)

// JSON keys of epStatus fields persisted in scaci_operation_log RequestData
// and replayed after session resume (SCACI §3.13).
const (
	metadataKeyEpStatus   = "epStatus"
	metadataKeyNonce      = "nonce"
	metadataKeySign       = "sign"
	metadataKeySubpackets = "subpackets"
)

// metadataKeyBsEui is the JSON key of a base station EUI in operation
// metadata, recorded as the hex text mioty.FormatEUI64 writes.
const metadataKeyBsEui = "bsEui"

// SCACI specification section identifiers referenced in assembly-validation logs.
const (
	specSectionConnectRsp      = "§3.3.2"
	specSectionPingRsp         = "§3.4.2"
	specSectionStatusRsp       = "§3.5.2"
	specSectionRegisterRsp     = "§3.6.2"
	specSectionDeregisterRsp   = "§3.7.2"
	specSectionULData          = "§3.8.1"
	specSectionULDataCmp       = "§3.8.3"
	specSectionULDataTxRsp     = "§3.9.2"
	specSectionDLDataQueRsp    = "§3.10.2"
	specSectionDLDataRevRsp    = "§3.11.2"
	specSectionDLDataResult    = "§3.12.1"
	specSectionDLDataResultCmp = "§3.12.3"
	specSectionEPStatus        = "§3.13.1"
	specSectionEPStatusCmp     = "§3.13.3"
	specSectionError           = "§3.14.1"
)

// certPollRetryInterval is how often the server re-checks for TLS certificates
// before starting the listener.
const certPollRetryInterval = 10 * time.Second

// downlinkPathInternal marks dlDataQue operations that entered through the
// internal gRPC path rather than the SCACI socket.
const downlinkPathInternal = "internal"

// Downlink queue entry defaults for entries created via SCACI dlDataQue.
const (
	dlQueueInitialAttempts = 0
	dlQueueMaxAttempts     = 3
)

// initialOpIDCounter is the pre-handshake value of the per-session operation
// ID counters; the first real message on each side overwrites it.
const initialOpIDCounter = 0

// Operation initiator labels recorded in operation-log metadata.
const (
	initiatorSC = "sc"
	initiatorAC = "ac"
)

// Operation status values recorded in responses and operation metadata.
const (
	opStatusAck       = "ack"
	opStatusCompleted = "completed"
	opStatusSuccess   = "success"
	// opStatusPartialFailure records a deregistration whose downlink revoke or
	// detach propagation failed (SCACI §3.7.3).
	opStatusPartialFailure = "partial_failure"
)

// Error detail messages recorded in operation metadata for protocol and
// scheduling failures.
const (
	errDetailDLRevokeFailed        = "DL revoke failed"
	errDetailULTransmitSchedFailed = "UL transmit scheduling failed"
	errDetailFmtSendULDataCmp      = "Failed to send ulDataCmp: %v"
	errDetailFmtSendEPStatusCmp    = "Failed to send epStatCmp: %v"
	errDetailACSentULDataCmp       = "AC sent ulDataCmp but SC should send it"
	errDetailACSentULDataTxRsp     = "AC sent ulDataTxRsp but SC should send it"
	errDetailACSentTxDataResCmp    = "AC sent txDataResCmp but SC should send it"
	errDetailACSentEPStatusCmp     = "AC sent epStatCmp but SC should send it"
)

// TLS version display names for handshake logging.
const (
	tlsNameTLS10      = "TLS 1.0"
	tlsNameTLS11      = "TLS 1.1"
	tlsNameTLS12      = "TLS 1.2"
	tlsNameTLS13      = "TLS 1.3"
	tlsNameUnknownFmt = "Unknown (0x%04x)"
)

// A JSON-framed payload is an object, opened by jsonObjectStart after any
// jsonWhitespace (SCACI §3).
const (
	jsonObjectStart = '{'
	jsonWhitespace  = " \t\r\n"
)

// maxVersionComponent bounds each parsed protocol-version component; larger
// values indicate a malformed version string, not a real release.
const maxVersionComponent = 999

// unknownCertCN labels pre-handshake log entries when the peer certificate
// carries no common name.
const unknownCertCN = "unknown"

// Constructor dependency guard messages, one per required collaborator.
const (
	depMsgCfgRequired                     = "cfg is required"
	depMsgLoggerRequired                  = "logger is required"
	depMsgOperationRepoRequired           = "operationRepo is required (§3.2)"
	depMsgSessionRegistryRequired         = "sessionRegistry is required (§1)"
	depMsgHandshakeSvcRequired            = "handshakeSvc is required (§3.3)"
	depMsgEndpointSvcRequired             = "endpointSvc is required (§3.6-3.7)"
	depMsgULSvcRequired                   = "ulSvc is required (§3.9)"
	depMsgDLSvcRequired                   = "dlSvc is required (§3.8)"
	depMsgStatusSvcRequired               = "statusSvc is required (§3.5)"
	depMsgSessionValidatorRequired        = "sessionValidator is required (§3.3.1)"
	depMsgOperationRecorderRequired       = "operationRecorder is required"
	depMsgSessionPersistenceRequired      = "sessionPersistence is required"
	depMsgOrgResolverRequired             = "orgResolver is required (org context parity)"
	depMsgSessionSnapshotProviderRequired = "sessionSnapshotProvider is required (§3.3)"
	depMsgPropagationSvcRequired          = "propagationSvc is required (§3.6-3.7)"
)

// depMsgErrorRecorderRequired reports a missing error recorder dependency at
// server construction.
const depMsgErrorRecorderRequired = "errorRecorder is required (§3.14)"

// depMsgClockRequired reports a missing clock dependency at construction.
const depMsgClockRequired = "clock is required"

// depMsgSessionEventsRequired reports a missing session event store at
// server construction.
const depMsgSessionEventsRequired = "sessionEvents is required (§3.3)"

// depMsgPlatformTenantRequired reports a configuration without the tenant of
// server-level events.
const depMsgPlatformTenantRequired = "platform tenant is required"

// listenerName labels the SCACI listener in transport log lines.
const listenerName = "SCACI"

// Operation metadata keys for scaci_operation_log ResponseData/RequestData
// Most keys are now exported from constants.go for cross-module access (gRPC handlers).
// Only module-internal keys remain here.
const (
	metadataKeyCompletedAt = "completedAt"
)
