package bssci

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Message Encoding Constants
// BSSCI Section 1 - Dual encoding support (JSON and MessagePack).
// Canonical definitions live in KC-DB/storage/mioty; re-exported here so
// existing bssci call sites remain unchanged.
const (
	// EncodingMessagePack is the default MIOTY message encoding (binary)
	EncodingMessagePack = mioty.EncodingMessagePack

	// EncodingJSON is the alternative MIOTY message encoding (text-based)
	EncodingJSON = mioty.EncodingJSON
)

// Base station location columns a statusRsp geoLocation writes.
const (
	locationColumnLatitude  = "latitude"
	locationColumnLongitude = "longitude"
	locationColumnAltitude  = "altitude"
	locationColumnSource    = "location_source"
	locationColumnUpdatedAt = "location_updated_at"
)

// Timing defaults applied when the corresponding protocol configuration is
// absent (tests and minimal setups); production values come from
// protocol.ack_timeout, protocol.duplicate_window, and
// protocol.bsci_certificate_poll_interval.
const (
	// defaultOperationAckTimeout bounds handshake waits (conCmp/errorAck after conRsp or a connect-stage error)
	defaultOperationAckTimeout = 30 * time.Second

	// defaultConnectionEstablishmentTimeout bounds a fresh connection before its con arrives
	defaultConnectionEstablishmentTimeout = 30 * time.Second

	// defaultStatusRequestInterval is how often the SC polls a base station for status
	defaultStatusRequestInterval = 30 * time.Second
	// defaultStatusRequestInitialDelay delays the first status poll after connect
	defaultStatusRequestInitialDelay = 5 * time.Second
	// defaultDLRXQueryTimeout expires an unanswered dlRxStatQry
	defaultDLRXQueryTimeout = 300 * time.Second
	// defaultDLRXCleanupInterval is the dlRxStatQry expiry sweep cadence
	defaultDLRXCleanupInterval = 60 * time.Second
)

// sessionTeardownTimeout bounds the persistence a lost connection runs; it is
// detached from the server context so it also completes during Stop.
const sessionTeardownTimeout = 5 * time.Second

// Exact float integer bounds for wire numeric coercion.
// IEEE 754 floating-point values represent integers exactly only up to their
// mantissa width; values beyond these bounds silently lose precision and must
// be rejected when converting to integer protocol fields (e.g. EUI-64).
const (
	// maxExactFloat64Integer is the largest integer exactly representable in a float64 (2^53)
	maxExactFloat64Integer = uint64(1) << 53

	// maxExactFloat32Integer is the largest integer exactly representable in a float32 (2^24)
	maxExactFloat32Integer = uint64(1) << 24
)

// Propagate Status Constants
// BSSCI Sections 5.6-5.7, 5.9 - Endpoint telemetry propagation status values
// These are re-exported from the endpoint package to avoid import cycles
const (
	// PropagateStatusDetachReceived indicates endpoint has received detach message
	PropagateStatusDetachReceived = endpoint.PropagateStatusDetachReceived

	// PropagateStatusAttached indicates attach propagate completed successfully
	PropagateStatusAttached = endpoint.PropagateStatusAttached

	// PropagateStatusDetaching indicates detach propagate is in-flight (awaiting BS response)
	PropagateStatusDetaching = endpoint.PropagateStatusDetaching

	// EndpointStatusAttached is the ep_status value for attached endpoints
	EndpointStatusAttached = endpoint.EndpointStatusAttached
)

// Event severities; the canonical definitions live with the persisted event
// model in KC-DB/storage/models.
const (
	SeverityInfo    = models.EventSeverityInfo
	SeverityWarning = models.EventSeverityWarning
	SeverityError   = models.EventSeverityError
)

// Event Status Constants
// System event lifecycle status values
const (
	// EventStatusNew indicates a newly created event not yet processed
	EventStatusNew = "new"
)

// Operation Constants
// Endpoint attachment operation tracking
const (
	// OperationStatusInitiated indicates an async operation has been started
	OperationStatusInitiated = "initiated"

	// OperationIDPrefixAttach is the prefix for attach operation IDs
	OperationIDPrefixAttach = "attach"

	// OperationIDPrefixDetach is the prefix for detach operation IDs
	OperationIDPrefixDetach = "detach"

	// OperationIDFormat is the format string for operation IDs
	// Use with fmt.Sprintf(OperationIDFormat, prefix, endpointID, timestamp)
	OperationIDFormat = "%s-%d-%d"

	// OperationTypeAttach identifies attach operations
	OperationTypeAttach = "attach"

	// OperationTypeDetach identifies detach operations
	OperationTypeDetach = "detach"

	// OperationLookbackHours is the default lookback window for endpoint operations
	OperationLookbackHours = 24
)

// Event detail keys; the canonical definitions live with the persisted event
// model in KC-DB/storage/models.
const (
	EventKeyOperationID    = models.EventDetailKeyOperationID
	EventKeyOperationType  = models.EventDetailKeyOperationType
	EventKeyEndpointID     = models.EventDetailKeyEndpointID
	EventKeyEpEui          = models.EventDetailKeyEpEui
	EventKeyQueID          = models.EventDetailKeyQueID
	EventKeyTargetBS       = models.EventDetailKeyTargetBS
	EventKeyTargetBSList   = models.EventDetailKeyTargetBSList
	EventKeyTargetBSCount  = models.EventDetailKeyTargetBSCount
	EventKeyError          = models.EventDetailKeyError
	EventKeyBsEui          = models.EventDetailKeyBsEui
	EventKeyBaseStationEui = models.EventDetailKeyBaseStationEui
)

// Event Field Values
// Centralized values for system event fields
const (
	// EventCategoryEndpoint is the category for endpoint-related events
	EventCategoryEndpoint = "endpoint"

	// EventSourceTypeEndpoint is the source type for endpoint-originated events
	EventSourceTypeEndpoint = "endpoint"

	// BaseStationNone indicates no base station is associated with an event
	BaseStationNone = "none"
)

// Event Type Constants
// System event types for BSSCI error handling per §5.17-5.17.2 and propagation per §5.8-5.8.3
const (

	// EventTypeAttachPropagateInitiated indicates service center initiated attach propagate to base station
	// BSSCI §5.8-5.8.3 Attach Propagate operation
	EventTypeAttachPropagateInitiated = "attach_propagate_initiated"

	// EventTypeAttachPropagateCompleted indicates attach propagate three-way handshake completed
	// BSSCI §3.8.3 Attach Propagate Complete
	EventTypeAttachPropagateCompleted = "attach_propagate_completed"

	// EventTypeDetachPropagateInitiated indicates service center initiated detach propagate to base station
	// BSSCI §5.9 Detach Propagate operation
	EventTypeDetachPropagateInitiated = "detach_propagate_initiated"

	// EventTypeAttachPropagateFailed indicates attach propagate operation failed
	EventTypeAttachPropagateFailed = "attach_propagate_failed"

	// EventTypeDetachPropagateFailed indicates detach propagate operation failed
	EventTypeDetachPropagateFailed = "detach_propagate_failed"

	// EventTypeEndpointAttachFailed is the SystemEvent.EventType for per-endpoint
	// failure events raised when a base station rejects an attach-propagate.
	// Matches the SQL filter `EventFilterAttachOps` in
	// KC-DB/storage/queries/operation_status_queries.go.
	EventTypeEndpointAttachFailed = "endpoint_attach_failed"

	// EventTypeEndpointDetachFailed is the symmetric counterpart for
	// detach-propagate rejection events.
	EventTypeEndpointDetachFailed = "endpoint_detach_failed"

	// EventTypeVMActivateSuccess indicates VM MAC type activation succeeded
	EventTypeVMActivateSuccess = "vm_activate_success"

	// EventTypeVMActivateFailed indicates VM MAC type activation failed
	EventTypeVMActivateFailed = "vm_activate_failed"

	// EventTypeVMDeactivateSuccess indicates VM MAC type deactivation succeeded
	EventTypeVMDeactivateSuccess = "vm_deactivate_success"

	// EventTypeVMStatusReceived indicates VM status response received from base station
	EventTypeVMStatusReceived = "vm_status_received"

	// EventTypeDLRxStatusReceived indicates downlink RX status received from endpoint
	EventTypeDLRxStatusReceived = "dl_rx_status_received"

	// EventTypeAttachOperationFailed indicates attach propagate operation failed
	EventTypeAttachOperationFailed = "attach_operation_failed"

	// EventTypeDetachOperationFailed indicates detach propagate operation failed
	EventTypeDetachOperationFailed = "detach_operation_failed"
)

// Event Title Constants
// Human-readable titles for BSSCI error events (BSSCI §5.17) and propagation events (BSSCI §5.8)
const (

	// TitleAttachPropagateInitiated is the title for attach propagate initiated events
	TitleAttachPropagateInitiated = "Attach Propagate Initiated"

	// TitleAttachPropagateCompleted is the title for attach propagate completion events
	TitleAttachPropagateCompleted = "Attach Propagate Completed"

	// TitleDetachPropagateInitiated is the title for detach propagate initiated events
	TitleDetachPropagateInitiated = "Detach Propagate Initiated"

	// TitleOperationFailed is the title for failed operation events
	TitleOperationFailed = "Operation Failed"

	// TitleAttachPropagateFailedForEndpointOnBS is the title format for
	// attach-propagate failure events raised when a base station rejects
	// the propagate. Format args: endpoint EUI (string), base station EUI (string).
	TitleAttachPropagateFailedForEndpointOnBS = "Attach propagate failed for endpoint %s on BS %s"

	// TitleDetachPropagateFailedForEndpointOnBS is the symmetric detach-side
	// title format. Format args: endpoint EUI (string), base station EUI (string).
	TitleDetachPropagateFailedForEndpointOnBS = "Detach propagate failed for endpoint %s on BS %s"

	// PropagateFailureReasonFormat is the short reason recorded in failure-event
	// details JSON when a base station rejects a propagate. Format args: result code.
	PropagateFailureReasonFormat = "Base station rejected with code %d"

	// PropagateFailureDescriptionFormat is the long description used on both the
	// endpoint and base-station system events raised on a rejected propagate.
	// Format args: result code.
	PropagateFailureDescriptionFormat = "Base station rejected operation with result code %d"

	// TitleRejectedOperationForEndpoint is the base-station-event title format for a
	// rejected propagate. Format args: operation type (string), endpoint EUI (string).
	TitleRejectedOperationForEndpoint = "Rejected %s for endpoint %s"
)

// Event Description Constants
// Human-readable descriptions for BSSCI propagation events (BSSCI §5.8-5.8.3)
const (
	// DescriptionAttachPropagateEndpoint is the description for endpoint-side attach propagate events
	DescriptionAttachPropagateEndpoint = "Attach propagate initiated"

	// DescriptionDetachPropagateEndpoint is the description for detach propagate initiated events
	DescriptionDetachPropagateEndpoint = "Detach propagate initiated"

	// DescriptionOperationFailedFormat is the format string for operation failure descriptions
	// Use with fmt.Sprintf(DescriptionOperationFailedFormat, targetBS, err)
	DescriptionOperationFailedFormat = "Failed to propagate to %s: %v"
)

// DL Data Revoke Operation Constants (BSSCI §5.13)
const (
	OperationDLDataRevoke = "dlDataRev"
)

// Downlink Queue Status Constants (BSSCI §5.11-5.14)
// Queue lifecycle states for tracking downlink message progression.
// Canonical definitions live in KC-DB/storage/mioty; re-exported here.
const (
	DLQueueStatusPending  = mioty.DLQueueStatusPending
	DLQueueStatusReserved = mioty.DLQueueStatusReserved
	DLQueueStatusQueued   = mioty.DLQueueStatusQueued
)

// Validation Status Constants
// Detach signature validation tracking (BSSCI §5.7)
// These values indicate whether cryptographic validation was performed for a detach operation.
// Used in detachMetadata.ValidationStatus to track the detach signature validation state.
const (
	// ValidationStatusValidated indicates signature was cryptographically validated against endpoint.Sign
	ValidationStatusValidated = "validated"

	// ValidationStatusUnverified indicates the detach was recorded without a cryptographic
	// check: signature validation is disabled in server config, or no validator was configured.
	ValidationStatusUnverified = "unverified"

	// ValidationStatusInvalidSignature indicates signature cryptographically invalid (CMAC/equality check failed)
	ValidationStatusInvalidSignature = "invalid_signature"
)

// Log field values explaining rejected sessions and dropped values.
const (
	// reasonQueueIDOverflow marks a dlDataRevRsp queue ID that exceeds int64 range.
	reasonQueueIDOverflow = "uint64 overflow (> math.MaxInt64)"
	// reasonHandshakeIncomplete marks a session skipped because conCmp has not arrived.
	reasonHandshakeIncomplete = "handshake not complete"
	// reasonNotConnected marks a base station absent from the connected-session map.
	reasonNotConnected = "not in connected sessions"
	// expectedEnvelopeFields names the mandatory response envelope keys per BSSCI §2.4.
	expectedEnvelopeFields = "command,opId"
	// specSectionNormalization is the BSSCI section governing unknown-field handling.
	specSectionNormalization = "§2.4"
	// contextAttachPropagateFallback tags org-lookup fallbacks during attachPrpCmp handling.
	contextAttachPropagateFallback = "attach_propagate_complete_fallback"
)

// geoLocationComponents is the length of a geoLocation Numeric[3] (rev1 §5.3.1).
const geoLocationComponents = 3

// Event payload and pending-operation metadata keys shared by handlers.
const (
	// eventDataKeyStatus is the event-details key carrying an operation outcome.
	eventDataKeyStatus = models.EventDetailKeyStatus
	// metadataKeyFailed marks a pending operation as failed.
	metadataKeyFailed = "failed"
	// metadataKeyFailedAt records the failure timestamp (Unix ns) of a pending operation.
	metadataKeyFailedAt = "failedAt"
	// metadataKeyTenantID carries a propagate operation's endpoint owner tenant.
	metadataKeyTenantID = "tenantId"
	// metadataKeyEndpointID carries the database ID of the endpoint an attach
	// or detach names.
	metadataKeyEndpointID = "endpointID"
	// metadataKeyEndpointTenantID carries an attach's endpoint owner tenant.
	metadataKeyEndpointTenantID = "endpointTenantID"
	// metadataKeyOrganizationID carries a propagate operation's endpoint owner organization.
	metadataKeyOrganizationID = "organizationId"
	// metadataKeyOperatorRequested marks a status request an operator sent,
	// whose answer is announced even when it answers a resume reissue.
	metadataKeyOperatorRequested = "operatorRequested"
	// metadataKeyShortAddr carries the short address an attach propagate sent.
	metadataKeyShortAddr = "shortAddr"
	// metadataKeyBidirectional carries the bidi flag an attach propagate sent.
	metadataKeyBidirectional = "bidirectional"
	// metadataKeyLastPacketCnt carries the packet counter an attach propagate sent.
	metadataKeyLastPacketCnt = "lastPacketCnt"
	// metadataKeyDualChannel carries the dualChan flag an attach propagate sent.
	metadataKeyDualChannel = "dualChannel"
	// metadataKeyRepetition carries the repetition flag an attach propagate sent.
	metadataKeyRepetition = "repetition"
	// metadataKeyWideCarrOff carries the wideCarrOff flag an attach propagate sent.
	metadataKeyWideCarrOff = "wideCarrOff"
	// metadataKeyLongBlkDist carries the longBlkDist flag an attach propagate sent.
	metadataKeyLongBlkDist = "longBlkDist"
)

// fallbackErrorMessage substitutes for a base-station error message when the
// inbound error payload omits the message field.
const fallbackErrorMessage = "unknown error"

// maxLoggedErrorMessageBytes bounds the free-text message of a base-station
// error where it is logged; the station chooses its length.
const maxLoggedErrorMessageBytes = 256

// Sublayer command prefixes (BSSCI-4-02): remote control and virtual machine.
const (
	sublayerPrefixRC = "rc."
	sublayerPrefixVM = "vm."
)

// BSSCI wire field names used when dispatching over raw payload maps.
const (
	wireFieldSign  = "sign"
	wireFieldRssi  = "rssi"
	wireFieldSnr   = "snr"
	wireFieldEqSnr = "eqSnr"
	wireFieldPhase = "phase"
	// wireFieldSubpackets is the optional per-subpacket reception object (BSSCI §3.10.1).
	wireFieldSubpackets = "subpackets"
)

// Go type names cited in wire-coercion range errors.
const (
	typeNameUint64 = "uint64"
	typeNameUint32 = "uint32"
	typeNameUint16 = "uint16"
)

// Attach-signature CMAC IV layout (MIOTY radio spec §3.7.1.3, Fig. 3-15):
// [EUI64 (8) | 0xFF | 0x00 | attach counter (4, big-endian) | 0xFF 0xFF].
const (
	attachIVPadFF         = 0xFF
	attachIVPad00         = 0x00
	attachIVSize          = 16
	attachIVPadFFOffset   = 8
	attachIVPad00Offset   = 9
	attachIVCounterOffset = 10
	attachIVTrailerOffset = 14
)

// MessagePack map markers used for encoding detection (fixmap range and the
// map16/map32 markers), plus the UTF-8 byte order mark JSON payloads may
// carry per RFC 8259.
const (
	msgpackFixmapMin byte = 0x80
	msgpackFixmapMax byte = 0x8f
	msgpackMap16     byte = 0xde
	msgpackMap32     byte = 0xdf
	utf8BOM               = "\xEF\xBB\xBF"
)

// Event status values recorded under eventDataKeyStatus.
const (
	eventStatusSuccess = "success"
	eventStatusFailed  = "failed"
)

// Event titles, descriptions and reasons recorded in system events by the
// protocol handlers.
const (
	eventDescPendingOpRebuildSkipped  = "Persisted pending operation could not be rebuilt for session resume and was skipped"
	eventDescResumeRefused            = "The base station refused the offered session resume; the session was retired and its next connect starts a new session"
	eventDescEndpointAttached         = "Endpoint successfully attached to network"
	eventDescEndpointDetached         = "Endpoint successfully detached from network"
	eventReasonBSNotBidirectional     = "Base station does not support bidirectional operation"
	eventDescTargetBSNotBidirectional = "Target base station is not bidirectional"
	eventReasonNoConnectedBS          = "No connected base stations available"
	eventDescNoBSForAttachPropagate   = "No connected base stations available to propagate attach"
	eventDescNoBSForDetachPropagate   = "No connected base stations available to propagate detach"
	eventDescFmtAttachPropagateDone   = "Attach propagate to base station %s completed (opId %d)"
	// eventDescFmtEndpointPropagateDone names the endpoint of a completed attach propagate.
	eventDescFmtEndpointPropagateDone = "Attach propagate of endpoint %s to base station %s completed (opId %d)"
	eventTitleFmtAttachPropagate      = "Attach Propagate: EP %s to BS %s"
	eventDescFmtKeysPropagated        = "Endpoint %s keys propagated to base station %s with short address %04X"
	eventDescDetachPropagateDone      = "Detach propagate completed successfully"

	eventTitleFmtVMActivateSuccess   = "VM Activate Success - MAC Type %d"
	eventDescFmtVMActivateSuccess    = "Successfully activated VM MAC type %d for endpoint %s on base station %s"
	eventTitleFmtVMActivateFailed    = "VM Activate Failed - MAC Type %d"
	eventDescFmtVMActivateFailed     = "Failed to activate VM MAC type %d for endpoint %s on base station %s"
	eventErrVMActivateRejected       = "Base station rejected VM activate request"
	eventMsgFmtVMActivated           = "VM MAC type %d activated for endpoint"
	eventMsgFmtVMDeactivated         = "VM MAC type %d deactivated for endpoint"
	eventTitleFmtVMDeactivateSuccess = "VM Deactivate Success - MAC Type %d"
	eventDescFmtVMDeactivateSuccess  = "Successfully deactivated VM MAC type %d for endpoint %s on base station %s"
	eventTitleFmtVMStatus            = "VM Status - %d Active MAC Types"
	eventDescFmtVMStatus             = "Endpoint %s has %d active VM MAC types on base station %s"

	eventDescFmtRevokingQueuedDownlink = "Revoking downlink %d for endpoint %s at base station %s"
	eventDescFmtDLReceptionReported    = "Endpoint %s reported the reception of a downlink through base station %s: SNR=%.1f dB, RSSI=%.1f dBm"
	// stationLabelFmt renders a named base station as "name (EUI)".
	stationLabelFmt = "%s (%s)"
)

// Event title formats for propagate failures.
const (
	titleAttachPropagateFailedFormat = "Attach propagate failed for endpoint %s"
	titleDetachPropagateFailedFormat = "Detach propagate failed for endpoint %s"
)

// listenerName labels the BSSCI listener in transport log lines.
const listenerName = "BSSCI"

// Dependency names reported by Dependencies.missing when a required
// collaborator is absent.
const (
	depNameAttachPersistence           = "attach persistence"
	depNameAuditLogger                 = "audit logger"
	depNameStationCertificateBinder    = "station certificate binder"
	depNameBaseStationStatusStore      = "base station status store"
	depNameBaseStationStore            = "base station store"
	depNameBlueprintDecoder            = "blueprint decoder"
	depNameBlueprintResolver           = "blueprint resolver"
	depNameCertificateIdentityResolver = "certificate identity resolver"
	depNameClock                       = "clock"
	depNameConnectionRegistry          = "connection registry"
	depNameDLRXStatusStore             = "DL RX status store"
	depNameDispositionResolver         = "disposition resolver"
	depNameDownlinkQueueStore          = "downlink queue store"
	depNameDownlinkRevocationStore     = "downlink revocation store"
	depNamePendingDownlinkLister       = "pending downlink lister"
	depNameDownlinkService             = "downlink service"
	depNameAttachmentDecider           = "attachment decider"
	depNameEndpointDirectory           = "endpoint directory"
	depNameEndpointOwnerResolver       = "endpoint owner resolver"
	depNameEventStore                  = "event store"
	depNameOrganizationDirectory       = "organization directory"
	depNameProtocolMessageStore        = "protocol message store"
	depNameQueueSerializer             = "queue serializer"
	depNameServingStationLocator       = "endpoint serving station locator"
	depNameSessionService              = "session service"
	depNameSessionReconciler           = "session reconciler"
	depNameSessionKeySource            = "network session key source"
	depNameStatusService               = "status service"
	depNameStationEventRecorder        = "station event recorder"
	depNameTenantResolver              = "tenant resolver"
	depNameUplinkIngestService         = "uplink ingest service"
	depNameVersionNegotiator           = "version negotiator"
)

// fieldPingResult is the result a base station may add to pingRsp; BSSCI
// §5.4.2 defines only command and opId, so the field is optional.
const fieldPingResult = "result"
