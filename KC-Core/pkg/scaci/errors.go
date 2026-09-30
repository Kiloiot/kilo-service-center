// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"errors"
	"fmt"
)

// POSIX Error Codes for SCACI protocol messages per MIOTY SCACI v1.0.0 §3.14
//
// These error codes are used in:
//   - Status responses (§3.5.2)
//   - Error messages (§3.14.1)
//
// Error codes follow standard POSIX errno values for cross-platform compatibility.
//
//revive:disable:var-naming
const (
	// POSIX_OK indicates successful operation (no error)
	POSIX_OK = 0

	// POSIX_EPERM indicates operation not permitted (permission denied for privileged operation)
	POSIX_EPERM = 1

	// POSIX_ENOENT indicates no such file or directory (requested resource does not exist)
	// In SCACI context: endpoint/session/operation not found
	POSIX_ENOENT = 2

	// POSIX_EIO indicates input/output error (generic I/O failure)
	// In SCACI context: database write failure, network I/O error
	POSIX_EIO = 5

	// POSIX_EAGAIN indicates resource temporarily unavailable (try again later)
	// In SCACI context: no suitable base station available for transmission
	POSIX_EAGAIN = 11

	// POSIX_ENOMEM indicates out of memory (system resource exhaustion)
	POSIX_ENOMEM = 12

	// POSIX_EINVAL indicates invalid argument (malformed request, invalid field values)
	// In SCACI context: missing required fields, invalid opId, type mismatch
	POSIX_EINVAL = 22

	// POSIX_ERANGE indicates numerical result out of range
	// In SCACI context: queue ID exceeds max int64, invalid numeric field value
	POSIX_ERANGE = 34

	// POSIX_EEXIST indicates file exists (resource already exists, duplicate entry)
	// In SCACI context: duplicate queue ID, conflicting endpoint registration
	POSIX_EEXIST = 17

	// POSIX_EPROTO indicates protocol error (protocol violation, unexpected message)
	// In SCACI context: wrong direction of operation, invalid handshake sequence
	POSIX_EPROTO = 71

	// POSIX_ENOTSUP indicates operation not supported (feature not implemented)
	// In SCACI context: unsupported SCACI command, unimplemented optional feature
	POSIX_ENOTSUP = 95

	// POSIX_ETIMEDOUT indicates connection timed out (operation timeout)
	// In SCACI context: operation timeout, no response within expected window
	POSIX_ETIMEDOUT = 110
)

//revive:enable:var-naming

// Error format strings shared by this package's failure paths; verbs are filled at the point of failure.
const (
	errFmtPersistConnectSyncFailed        = "persist connect sync failed: %w"
	errFmtHandshakeValidationFailed       = "handshake validation failed: %s"
	errFmtConnectValidationFailed         = "connect validation failed: %s"
	errFmtRevokeDownlinksPartial          = "failed to revoke %d/%d downlinks"
	errFmtQueryDownlinkQueue              = "failed to query downlink queue: %w"
	errFmtSCACIDlDataQueFailed            = "SCACI dlDataQue failed: %s"
	errFmtDLDataResultCmpCommandMismatch  = "DLDataResultComplete command mismatch: got %s, want %s"
	errFmtEPStatusCmpCommandMismatch      = "EPStatusComplete command mismatch: got %s, want %s"
	errFmtDLDataRevRspCommandMismatch     = "DLDataRevokeResponse command mismatch: got %s, want %s"
	errFmtDLDataQueRspCommandMismatch     = "DLDataQueueResponse command mismatch: got %s, want %s"
	errFmtULDataTxRspCommandMismatch      = "ULDataTransmitResponse command mismatch: got %s, want %s"
	errFmtULDataCmpCommandMismatch        = "ULDataComplete command mismatch: got %s, want %s"
	errFmtDeregisterRspCommandMismatch    = "DeregisterResponse command mismatch: got %s, want %s"
	errFmtRegisterRspCommandMismatch      = "RegisterResponse command mismatch: got %s, want %s"
	errFmtPingRspCommandMismatch          = "PingResponse command mismatch: got %s, want %s"
	errFmtErrorMessageValidationFailed    = "Error message validation failed: %s"
	errFmtULDataValidationFailed          = "ULData validation failed: %s"
	errFmtStatusResponseValidationFailed  = "StatusResponse validation failed: %s"
	errFmtConnectResponseValidationFailed = "ConnectResponse validation failed: %s"
	errFmtEPStatusValidationFailed        = "EPStatus validation failed: %s"
	errFmtDLDataResultValidationFailed    = "DLDataResult validation failed: %s"
	errFmtBroadcastSession                = "%w: %s (session %d): %w"
	errFmtBroadcastHeld                   = "%w: %s (held sessions): %w"
	errFmtDecodeUserDataForReplay         = "decode userData for replay: %w"
	errFmtSendFrame                       = "failed to send frame: %w"
	errFmtDuplicateCommandSpec            = "duplicate command spec: %s"
	errFmtStartListener                   = "SCACI listener: %w"
	errFmtMarshalResponse                 = "failed to marshal response: %w"
	errFmtScaciNewServer                  = "scaci.NewServer: %s"
	errFmtScaciNewServerCause             = "scaci.NewServer: %w"
	errFmtACOpIDMustIncrement             = "AC opId must increment (current: %d, got: %d)"
	errFmtACOpIDMustBePositive            = "AC opId must be positive, got %d"
	errFmtInvalidPatchVersion             = "invalid patch version: %s"
	errFmtInvalidMinorVersion             = "invalid minor version: %s"
	errFmtInvalidMajorVersion             = "invalid major version: %s"
	errFmtInvalidVersionFormat            = "version must be major.minor.patch, got: %s"
)

// Sentinel errors returned by this package; callers match them with errors.Is.
var (
	errInvalidConnectOpId     = errors.New("invalid connect opId")
	errOrgUUIDNotResolved     = errors.New("org enforcement: organization UUID required but not resolved from certificate")
	errOpIDReservedForConnect = errors.New("opId 0 reserved for connect")
	errNilULDataMessage       = errors.New("nil UL data message")
	errULDataMessageNotStored = errors.New("UL data message without its stored ID")
	errNilDLDataResult        = errors.New("nil DL data result")
	errNilEPStatusData        = errors.New("nil EPStatus data")
	errMissingStoredEpEui     = errors.New("missing/invalid epEui in stored RequestData")
	errMissingStoredEpStatus  = errors.New("missing/invalid epStatus in stored RequestData")
	errSubpacketsNotAMap      = errors.New("subpackets not a map")
	errTrailingJSONData       = errors.New("data after the JSON object")
)

// errDownlinkServiceUnavailable reports a deregister cleanup that could not
// revoke queued downlinks because no downlink service is wired.
var errDownlinkServiceUnavailable = errors.New("scaci downlink service unavailable")

// errBroadcastSessionFailed marks one Application Center session that did not
// receive a broadcast service-center operation.
var errBroadcastSessionFailed = errors.New("scaci broadcast to session failed")

// errRecordSCOperation marks a service-center operation that could not be
// recorded and therefore was not sent.
var errRecordSCOperation = errors.New("record service center operation")

// errResumedSessionNotHeld refuses a resume of a session this server neither
// serves nor holds for resumption (SCACI §1).
var errResumedSessionNotHeld = errors.New("resumed session is not held for resumption")

// errSessionSuperseded refuses to complete the connect operation of a session
// a newer session of its application center took the connection over from.
var errSessionSuperseded = errors.New("session was superseded before its connect operation completed")

// errMissingRegistryDependency refuses a session registry built without one of
// its collaborators.
var errMissingRegistryDependency = errors.New("session registry: missing dependency")

// DLDataQueueError is the failure of an internally initiated dlDataQue,
// carrying the catalog token and POSIX code the socket path would have sent
// so callers can classify the outcome without parsing text.
type DLDataQueueError struct {
	Token string
	POSIX int
}

func (e *DLDataQueueError) Error() string {
	return fmt.Sprintf(errFmtSCACIDlDataQueFailed, e.Token)
}
