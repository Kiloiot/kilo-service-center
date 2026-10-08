package bssci

import "errors"

// Sentinel errors for structured error handling with errors.Is pattern

// ErrResumeCounterMismatch is returned when resume operation counters do not match persisted session state (BSSCI §5.2)
var ErrResumeCounterMismatch = errors.New("resume rejected: operation counters do not match persisted session state")

// ErrResumeAlreadyClaimed is returned when the resumable session was activated or retired by another connection before this one could claim it
var ErrResumeAlreadyClaimed = errors.New("resume rejected: session already claimed by another connection")

// ErrDetachValidationEndpointNotFound is returned by detach validator when endpoint not found in database.
// Enables typed error detection via errors.Is in server.go detach handler.
var ErrDetachValidationEndpointNotFound = errors.New("detach validation endpoint not found")

// ErrDetachSignatureInvalid is returned by detach validator when signature cryptographic check fails.
// Maps to errDetachSignatureInvalid catalog token and POSIX error code from internal validator.
var ErrDetachSignatureInvalid = errors.New("detach signature validation failed")

// ErrAttachCounterStale is returned by the attach persistence when, under the
// endpoint's row lock, the attach counter no longer advances the stored one:
// another attach of the endpoint with that counter was served first.
var ErrAttachCounterStale = errors.New("attach counter does not advance the stored counter")

// CatalogError wraps error catalog tokens for structured error handling
// This enables services to return tokens that transports resolve via ResolveErrorMessage
type CatalogError struct {
	Token string // Exact token from errors_catalog.go (e.g., errVersionIncompatible)
	Posix int    // POSIX error code for wire protocol
}

// Error implements the error interface
// Returns the token exactly - no prefixes or modifications
func (e *CatalogError) Error() string {
	return e.Token
}

// NewCatalogError creates a new catalog error with a token and POSIX code
func NewCatalogError(token string, posix int) *CatalogError {
	return &CatalogError{
		Token: token,
		Posix: posix,
	}
}

// catalogErrorOf returns the catalog error a service reported; any other
// failure is answered as a failed persistence update (POSIX EIO).
func catalogErrorOf(err error) *CatalogError {
	var catalogErr *CatalogError
	if errors.As(err, &catalogErr) {
		return catalogErr
	}
	return NewCatalogError(errDatabaseUpdateFailed, POSIX_EIO)
}

// Sentinel errors returned by this package; callers match them with errors.Is.
var (
	errCryptoSignatureMismatch      = errors.New("signature mismatch")
	errKeyMustBeExactly16           = errors.New("key must be exactly 16 bytes")
	errSignatureMustBeExactly4Bytes = errors.New("signature must be exactly 4 bytes")
	errNonceMustBeExactly4Bytes     = errors.New("nonce must be exactly 4 bytes")
	errNoValidSubpacketArraysFound  = errors.New("no valid subpacket arrays found")
)

// Error format strings shared by this package's failure paths; verbs are filled at the point of failure.
const (
	errFmtTokenWithQueID                          = "%s: queId=%d" //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtResolveOrgForEndpointOwnerTenant        = "failed to resolve org for endpoint owner tenant %d: %w"
	errFmtFetchEndpoint                           = "failed to fetch endpoint: %w"
	errFmtTokenWithFormat                         = "%s: format=%d"     //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtTokenGotValue                           = "%s, got %d"        //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtTokenGotUnequal                         = "%s, got %d != %d"  //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtTokenForSession                         = "%s for session %s" //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtPayloadEntryTooLarge                    = "%s: entry %d is %d bytes (max %d)"
	errFmtCannotConvertTypeToByte                 = "cannot convert %T to byte"
	errFmtIntOutOfByteRange                       = "value %d out of byte range (0-255)"
	errFmtFloatOutOfByteRange                     = "value %f out of byte range (0-255)"
	errFmtNegativeValueCannotBeConvertedTo        = "negative value %d cannot be converted to byte"
	errFmtCannotConvertTypeToFloat64              = "cannot convert %T to float64"
	errFmtNonFiniteFieldValue                     = "non-finite value %v is not a valid field value"
	errFmtInvalidJSONNumber                       = "invalid JSON number %q: %w"
	errFmtIntOutOfRangeFor                        = "value %d out of range for %s"
	errFmtNegativeValueCoerce                     = "negative value cannot be coerced to %s: %v"
	errFmtFloatOutOfRangeFor                      = "value %f out of range for %s"
	errFmtNegativeIntCoerce                       = "negative value cannot be coerced to %s: %d"
	errFmtCannotConvertTypeTo                     = "cannot convert %T to %s"
	errFmtCannotConvertTypeToInt64                = "cannot convert %T to int64"
	errFmtValueOverflowsInt64                     = "value %d overflows int64"
	errFmtStringOverflowsInt64                    = "value %s overflows int64"
	errFmtFractionalInt64Coerce                   = "fractional value %s cannot be coerced to int64"
	errFmtValueOverflowsUint64                    = "value %s overflows uint64"
	errFmtNegativeUint64Coerce                    = "negative value cannot be coerced to uint64: %s"
	errFmtFractionalUint64Coerce                  = "fractional value %s cannot be coerced to uint64"
	errFmtInvalidJSONNumberPlain                  = "invalid JSON number %q"
	errFmtFractionalIntegerCoerce                 = "fractional value %v cannot be coerced to integer"
	errFmtNonFiniteCoerce                         = "non-finite value %v cannot be coerced to integer"
	errFmtUnsupportedFieldType                    = "unsupported field type: %s"
	errFmtExpectedByteArrayGot                    = "expected byte array, got %T"
	errFmtExpectedBytesGot                        = "expected %d bytes, got %d"
	errFmtArrayElementAtIndex                     = "array element at index %d: %w"
	errFmtExpectedBoolGot                         = "expected bool, got %T"
	errFmtExpectedStringGot                       = "expected string, got %T"
	errFmtWrapRadioSpecDualChannel                = "%w (MIOTY Radio Protocol §3.6.5.1)"
	errFmtFieldMustNotBePresent                   = "%w: %s must not be present (%s)"
	errFmtValidationFailedForOptionalFieldSpec    = "validation failed for optional field %s (spec: %s)"
	errFmtWrapOptionalFieldSpec                   = "%w for optional field %s: %v (spec: %s)"
	errFmtValidationFailedForFieldSpec            = "validation failed for field %s (spec: %s)"
	errFmtWrapFieldSpec                           = "%w for field %s: %v (spec: %s)"
	errFmtSpec                                    = "%w: %s (spec: %s)"
	errFmtTokenOpIDMetadata                       = "%s: opId %d metadata: %w" //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtTokenOpIDWrap                           = "%s: opId %d: %w"          //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtSessionWrap                             = "session %s: %w"
	errFmtTokenGotBytes                           = "%s, got %d bytes" //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtSessionNotFound                         = "session %s not found"
	errFmtEncodeRelayFrame                        = "encode relay frame: %w"
	errFmtNwkSnKeyWrongLength                     = "endpoint %d nwkSnKey length %d != 16"
	errFmtTrailingContentAfterJSONFrame           = "%s: trailing content after JSON frame"
	errFmtStationRejectedConnect                  = "base station rejected connect response: code=%d %s"
	errFmtTokenForOpID                            = "%s for opId %d" //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtBaseStationBelongsToDifferentTenant     = "base station %s belongs to different tenant (session: %d, caller: %d)"
	errFmtInvalidSessionTypeForEUI                = "invalid session type for EUI %s"
	errFmtReissuePendingOperationAfterResume      = "reissue pending operation %d after resume: %w"
	errFmtConnectionRegistrationFailedAfterConCmp = "connection registration failed after conCmp: %w"
	errFmtSessionPersistenceFailedAfterConCmp     = "session persistence failed after conCmp: %w"
	errFmtTokenWithEUI                            = "%s: EUI %s" //nolint:gosec // G101: catalog-token format string, not a credential
	errFmtRemoveInconsistentResumeOps             = "remove pending operations of inconsistent resume session: %w"
	errFmtTerminateInconsistentResumeSession      = "terminate inconsistent resume session: %w"
	errFmtOutboundValidationFailed                = "outbound validation failed: %w"
	errFmtDuplicateCommandSpec                    = "duplicate command spec: %s"
	errFmtCommandWithoutDirection                 = "command spec without direction: %s"
	errFmtBuildFrameCodec                         = "build frame codec: %w"
	errFmtCommandSpecRole                         = "%w: %s"
	errFmtWrapOutboundMessage                     = "failed to wrap outbound message: %w"
	errFmtServerWiringIncomplete                  = "server wiring incomplete: %w"
	errFmtReconcileAbandonedSessions              = "reconcile sessions left active by a previous process: %w"
	errFmtStationCertMismatchFingerprint          = "station %s presented certificate does not match registered fingerprint"
	errFmtRegisteredStationReload                 = "registered station reload: %w"
	errFmtStationFingerprintBackfill              = "station %s fingerprint backfill: %w"
	errFmtStationCertMismatchStored               = "station %s presented certificate does not match stored certificate"
	errFmtStationStoredCertificateUnparsable      = "station %s stored certificate unparsable: %w"
	errFmtStationHasNoStoredCertificateIdentity   = "station %s has no stored certificate identity"
	errFmtStationCertNamesAnotherStation          = "station %s presented a certificate naming another station"
	errFmtStationPresentedNoCertificate           = "station %s presented no client certificate"
	errFmtRegisteredStationLookup                 = "registered station lookup: %w"
	errFmtDependencyRequired                      = "%s is required"
	errFmtMessageMissingCommandField              = "message missing command/commandType field (type: %T)"
	errFmtCodeFieldUnexpectedType                 = "%s: code field has unexpected type %T"
	errFmtMessageFieldUnexpectedType              = "%s: message field has unexpected type %T"
	errFmtTimeFieldUnexpectedType                 = "%s: time field has unexpected type %T"
	errFmtMessageTypeNoOpID                       = "message type %T does not expose a BSSCI envelope"
	errFmtEncryptNetworkSessionKey                = "failed to encrypt network session key: %w"
)

// Sentinel errors for server wiring and lookup failures; callers match them with errors.Is.
var (
	errCommandSpecRoleContradictsDirection      = errors.New("command spec role contradicts its direction")
	errCommandSpecWithoutRole                   = errors.New("command spec without role")
	errConnectOperationFailedAndWasAcknowledged = errors.New("connect operation failed and was acknowledged by the base station")
	errConnectStageErrorAcknowledged            = errors.New("connect-stage error acknowledged by the base station; closing")
	errDetachValidatorNotWired                  = errors.New("detach signature validation is enabled but no validator is wired")
	errDownlinkDispatcherRequired               = errors.New("downlink dispatcher is required")
	errDownlinkReclaimerRequired                = errors.New("downlink reclaimer is required")
	errNilRegisteredStationDirectory            = errors.New("station certificate binder: registered station directory is nil")
	errNilStationCertificateLogger              = errors.New("station certificate binder: logger is nil")
	errDownlinkEndpointNotFound                 = errors.New("endpoint not found")
	errEndpointRepositoryNotAvailable           = errors.New("endpoint repository not available")
	errMapMissingCommandOpIDEnvelope            = errors.New("map missing command/opId envelope")
	errPropagationServiceRequired               = errors.New("propagation service is required")
	errRuntimeAlreadyConfigured                 = errors.New("runtime already configured")
	errRuntimeReconfigureAfterStart             = errors.New("runtime cannot be reconfigured after Start")
	errServerAlreadyStarted                     = errors.New("server already started")
	errServerAlreadyStopped                     = errors.New("server already stopped: a stopped server cannot be restarted")
	errServerConfigRequired                     = errors.New("server configuration is required")
	errServerRuntimeNotConfigured               = errors.New("server runtime not configured: call ConfigureRuntime before Start")
	errStatusSvcRequired                        = errors.New("statusSvc is required for pending operation tracking")
	errUplinkIngestSvcRequired                  = errors.New("uplinkIngestSvc is required for handleULData")
)
