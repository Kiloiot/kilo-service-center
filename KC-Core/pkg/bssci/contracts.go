package bssci

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// VersionNegotiator selects the BSSCI protocol version for a connection per
// rev1 §4.1-§4.3 and the §5.3.2 conRsp arbitration rule: the base station
// requests its newest supported version; the service center answers with the
// version it will speak (always an exact member of its supported set), and the
// base station agrees by completing the operation or rejects it with an error.
type VersionNegotiator interface {
	Negotiate(ctx context.Context, requested string) (selected string, err error)
}

// ResumeDisposition classifies the outcome of a session resume attempt so the
// connect flow never silently degrades an infrastructure failure or an
// inconsistent counter state into a fresh session.
type ResumeDisposition int

const (
	// ResumeNoMatch means no resumable session exists; start a fresh session.
	ResumeNoMatch ResumeDisposition = iota
	// ResumeCompatible means a resumable session was found and its constraints
	// hold; Previous carries the authoritative persisted session state.
	ResumeCompatible
	// ResumeInconsistent means a resumable session exists but the reported
	// counters or negotiated version are incompatible. The caller must
	// atomically terminate it (can_resume=false, pending ops removed) before
	// starting a fresh session.
	ResumeInconsistent
	// ResumeInfrastructureFailure means the resume lookup itself failed (e.g. a
	// database outage). The caller must reject the connect, never proceed
	// with a fresh session that would strand the old resumable state.
	ResumeInfrastructureFailure
)

// ResumeOutcome is the typed result of HandleResume.
type ResumeOutcome struct {
	Disposition ResumeDisposition
	// Previous is the resumable session (ResumeCompatible) or the stale
	// session to terminate (ResumeInconsistent); nil otherwise.
	Previous *Session
	// Err carries the underlying cause for ResumeInfrastructureFailure and
	// the mismatch detail for ResumeInconsistent.
	Err error
}

// SessionService handles connect/resume with REAL persistence
// Preserves: sessionsByUUID map, DbSessionId, HandshakeComplete, DB persistence
type SessionService interface {
	// HandleResume evaluates a session resume request (BSSCI §5.3.1) against
	// the DB-authoritative resumable-session lookup (tenant + base station EUI
	// + snBsUuid + disconnected + can_resume). The optional snBsOpId/snScOpId
	// constraints are pointers - absent means the constraint is not asserted.
	// It never publishes the hydrated session into any live registry; the
	// caller activates it. Returns a typed ResumeOutcome.
	HandleResume(ctx context.Context, session *Session, bsUUID []byte, bsOpId, scOpId *int64, bsEUI uint64) ResumeOutcome

	// PersistSession writes to basestation_sessions table
	// Uses real *sql.DB, updates DbSessionId, handles resume UPDATE vs new INSERT
	// BSSCI §5.3: Accepts connectInfo to persist arbitrary key-value pairs from connect message
	PersistSession(ctx context.Context, session *Session, baseStation *basestation.BaseStation, isResume bool, connectInfo json.RawMessage) error

	// StoreSessionByUUID adds to sessionsByUUID map
	StoreSessionByUUID(session *Session)

	// MarkHandshakeComplete sets HandshakeComplete=true
	MarkHandshakeComplete(session *Session)

	// RemoveSession cleans sessionsByUUID map on disconnect
	RemoveSession(session *Session)

	// TerminateSession marks a session as terminated in the database
	// Called during disconnect cleanup per BSSCI §3 session lifecycle
	TerminateSession(ctx context.Context, session *Session) error

	// UpdateEncoding persists negotiated message encoding to database (BSSCI Section 1)
	// Called when encoding is detected on first message; tenant-scoped
	UpdateEncoding(ctx context.Context, tenantID, sessionID int64, encoding string) error

	// MarkDisconnected marks an active session disconnected and resumable
	// after unexpected connection loss, guarded by the session's connection
	// ID so a newer connection is never marked offline by stale cleanup
	MarkDisconnected(ctx context.Context, session *Session) error

	// UpdateSessionCounters persists operation ID counters to database (BSSCI §5.2)
	// Called immediately after successful SC-initiated operations.
	// Ensures resume correctness by maintaining authoritative SC operation state.
	UpdateSessionCounters(ctx context.Context, session *Session) error

	// UpdatePingTimestamp persists last ping timestamp to database (BSSCI §5.4)
	// Called immediately after successful pingCmp reception
	// Ensures ping monitoring correctness by maintaining authoritative ping state
	UpdatePingTimestamp(ctx context.Context, session *Session) error
}

// DownlinkService owns downlink queue operations over the canonical MIOTY
// types; implementations persist through the message store helpers.
type DownlinkService interface {
	// ProcessDLDataResult handles complete downlink result processing (BSSCI §5.14)
	// Orchestrates: tenant resolution, DB update, SCACI broadcast, audit logging, cleanup
	// Returns response message for handler to send via sendMessage
	ProcessDLDataResult(ctx context.Context, session *Session, result *mioty.DLDataResult) (responseMsg map[string]interface{}, err error)

	// ProcessRevokeResponse records a base station's dlDataRevRsp (BSSCI
	// §3.13) and returns the dlDataRevCmp to send; revoked is false for a
	// downlink that expired, being revoked for its lifetime, and for one that
	// had already ended, which keeps its outcome.
	ProcessRevokeResponse(ctx context.Context, session *Session, opId int64, queueID int64, endpointEUI uint64) (responseMsg map[string]interface{}, revoked bool, err error)

	// ProcessRevokeRefusal records a base station's error answer to a
	// dlDataRev (BSSCI §3.17). A refusal saying the station does not hold the
	// downlink ends it as a confirmed revoke would; any other refusal leaves
	// it in flight. revoked reports whether it ended revoked.
	ProcessRevokeRefusal(ctx context.Context, session *Session, refusal RevokeRefusal) (revoked bool, err error)

	// ProcessQueueAck records the base station that answered a dlDataQue
	// with dlDataQueRsp as the holder of the downlink (BSSCI §3.12).
	ProcessQueueAck(ctx context.Context, session *Session, ack QueueAcknowledgement) error

	// ProcessQueueError fails a downlink whose dlDataQue the base station
	// answered with error (BSSCI §3.17) and reports it discarded (SCACI §3.12).
	ProcessQueueError(ctx context.Context, session *Session, rejection QueueRejection) error
}

// QueueAcknowledgement is a base station's dlDataQueRsp: the downlink it
// refers to, its owner, and the organization it was enqueued under.
type QueueAcknowledgement struct {
	QueueID        int64
	OwnerTenant    string
	OrganizationID *uuid.UUID
}

// QueueRejection is a base station's error answer to a dlDataQue: the
// downlink it refers to, its owner, and the station's POSIX code and message.
type QueueRejection struct {
	QueueID     int64
	EndpointEUI uint64
	OwnerTenant string
	Code        int
	Message     string
}

// RevokeRefusal is a base station's error answer to a dlDataRev: the
// downlink it names and the station's POSIX code.
type RevokeRefusal struct {
	QueueID     int64
	EndpointEUI uint64
	Code        int
}

// StatusService manages pendingOps map + DB persistence.
// Preserves: PendingOperation struct, bssci_pending_operations table.
// All methods accept a session parameter for composite key support (BSSCI §5.11-5.12.3).
type StatusService interface {
	// RecordPendingOperation stores in map + DB using SessionOpKey composite key
	RecordPendingOperation(ctx context.Context, session *Session, opId int64, op *PendingOperation, dbSessionID int64) error

	// RecordPendingOperations durably records several operations in one
	// repository transaction (all-or-nothing) and mirrors them into the cache
	// only after the transaction commits. Used for multi-frame sequences such
	// as the dlRxStatQry/dlDataQue pair whose recovery records must never be
	// partially persisted.
	RecordPendingOperations(ctx context.Context, session *Session, ops []*PendingOperation, dbSessionID int64) error

	// RestorePendingOperation hydrates the in-memory cache from an already
	// authoritative DB row without writing it back (session resume path).
	RestorePendingOperation(session *Session, opId int64, op *PendingOperation)

	// GetPendingOperation retrieves from map using SessionOpKey composite key
	GetPendingOperation(session *Session, opId int64) (*PendingOperation, error)

	// RemovePendingOperation cleans map + DB using SessionOpKey composite key
	RemovePendingOperation(ctx context.Context, session *Session, opId int64) error

	// ExtractQueueMetadata retrieves endpoint EUI, queue ID, tenant ID and owner
	// organization from a pending operation.
	// Used by downlink handlers to correlate responses with original requests.
	// Returns tenantID for proper roaming tenant isolation (BSSCI §5.12).
	// Session parameter required for SessionOpKey composite key lookup.
	ExtractQueueMetadata(session *Session, opId int64) (endpointEUI uint64, queueID int64, tenantID string, organizationID *uuid.UUID)

	// UpdatePendingOperationMetadata persists new metadata for an existing
	// pending row (metadataJSON is the pre-marshaled form of metadata) and,
	// on success, mirrors it into the cached operation. The DB write comes
	// first so the cache never runs ahead of the durable state.
	UpdatePendingOperationMetadata(ctx context.Context, session *Session, opId int64, metadata map[string]interface{}, metadataJSON json.RawMessage) error

	// PersistedOperations returns the raw persisted rows for session resume
	// hydration; strict decoding stays with the caller.
	PersistedOperations(ctx context.Context, sessionID int64) ([]PersistedOperation, error)

	// DeletePendingOperations removes every persisted row of the session and,
	// only after the DB deletion succeeds, evicts the session's cached
	// operations. A failed deletion leaves the cache untouched.
	DeletePendingOperations(ctx context.Context, session *Session) (int64, error)

	// EvictCachedOperations removes the session's cached operations without
	// touching the persisted rows. Called on every connection teardown: the
	// runtime session ID dies with the connection, so its cache entries are
	// unreachable afterwards, while the DB rows remain the durable source for
	// a later resume.
	EvictCachedOperations(session *Session)

	// SessionOperations lists the operations the session has in flight.
	SessionOperations(ctx context.Context, session *Session) []*PendingOperation
}

// PersistedOperation is a raw persisted pending-operation row returned for
// resume hydration. It is BSSCI-owned so the storage row model does not leak
// through the service boundary; payloads stay encoded.
type PersistedOperation struct {
	OperationID   int64
	OperationType string
	EndpointEUI   []byte
	OperationData []byte
	Metadata      []byte
	CreatedAt     time.Time
}

// AbandonedSessionReconciler hands the sessions a previous process of this
// service center left active back resumable, so a base station can resume
// after a crash (BSSCI §3.3). It runs once when the server starts, before any
// connection is accepted.
type AbandonedSessionReconciler interface {
	ReconcileAbandonedSessions(ctx context.Context) error
}

// BaseStationConnectionRegistry owns the live-connection operations the
// connect flow consumes: registration lookup across tenants (the tenant is
// not yet authenticated during the handshake) and live-connection
// registration. The concrete connection manager is captured by the adapter at
// construction, never passed per call.
type BaseStationConnectionRegistry interface {
	// GetBaseStationGlobal retrieves a base station by EUI across all tenants.
	GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*basestation.BaseStation, error)

	// RegisterConnection publishes the session's live connection and marks the
	// base station online.
	RegisterConnection(ctx context.Context, session *Session, baseStation *basestation.BaseStation) error

	// DisconnectBaseStationIfCurrent marks the base station offline only while
	// the given connection is still its current one (a reconnect that already
	// replaced this connection keeps the station online).
	DisconnectBaseStationIfCurrent(ctx context.Context, eui [8]byte, connectionID string) error

	// UpdateLastSeen refreshes the base station's last-seen timestamp.
	UpdateLastSeen(ctx context.Context, eui [8]byte) error
}

// CertificateIdentity is the tenant/organization identity resolved from a
// base station's TLS client certificate. SubjectEUI is set only when the
// certificate CN encodes a base station EUI-64 (the CE issuance scheme);
// legacy org-<UUID> CNs carry no station identity.
type CertificateIdentity struct {
	OrganizationID uuid.UUID
	TenantID       int64
	SubjectEUI     *uint64
}

// CertificateIdentityResolver resolves the identity asserted by a TLS client
// certificate at connection accept. The CE composite implementation resolves
// dashed-EUI CNs against the registered base stations and delegates other CN
// forms (org-<UUID>) to the deployment's organization resolver.
type CertificateIdentityResolver interface {
	ResolveCertificateIdentity(ctx context.Context, cert *x509.Certificate) (CertificateIdentity, error)
}

// RegisteredBaseStation is the persistent registration identity of a base
// station as the BSSCI connect path consumes it. It deliberately carries no
// organization ID - organization context is resolved separately - and no
// live-connection state, which belongs to the connection registry.
type RegisteredBaseStation struct {
	ID                 int64
	TenantID           int64
	EUI                uint64
	Name               string
	TLSCertificate     string
	TLSCertFingerprint string
	// TLSCertExpiresAt is the bound certificate's expiry; nil until recorded.
	TLSCertExpiresAt *time.Time
}

// RegisteredBaseStationDirectory reads registered station identity for
// certificate enforcement during connect and backfills the certificate
// fingerprint and expiry for rows that do not store them yet.
type RegisteredBaseStationDirectory interface {
	// GetGlobal returns the registration for an EUI across all tenants
	// (tenant is not yet authenticated at TLS accept).
	GetGlobal(ctx context.Context, eui uint64) (RegisteredBaseStation, error)

	// BackfillFingerprintIfBlank persists the fingerprint only while the
	// stored value is still blank; reports whether a row was updated (false
	// signals a concurrent writer - reload and compare).
	BackfillFingerprintIfBlank(ctx context.Context, tenantID, id int64, fingerprint string) (bool, error)

	// BackfillCertExpiryIfBlank persists the bound certificate's expiry only
	// while none is stored; an existing expiry is never overwritten.
	BackfillCertExpiryIfBlank(ctx context.Context, tenantID, id int64, expiresAt time.Time) (bool, error)
}

// StationEventRecorder records an event in a base station's activity feed,
// the same seam its online and offline events are recorded through;
// occurredAt is when the exchange happened on the wire.
type StationEventRecorder interface {
	RecordEvent(ctx context.Context, eui [8]byte, eventType string, occurredAt time.Time, data map[string]interface{}) error
}

// StationCertificateClaim is what a connecting base station asserts: the
// station it claims to be, the client certificate it presented and the
// station EUI that certificate names (nil for one naming no station, such as
// an organization certificate).
type StationCertificateClaim struct {
	BaseStationEUI uint64
	Certificate    *x509.Certificate
	SubjectEUI     *uint64
}

// StationCertificateBinder refuses a connecting base station whose client
// certificate does not belong to the registered station it claims to be.
type StationCertificateBinder interface {
	BindStationCertificate(ctx context.Context, claim StationCertificateClaim) error
}

// SCACIBroadcaster forwards via real scaciBroadcaster interface
// Matches the exact signatures from scaciBroadcaster to enable proper delegation
type SCACIBroadcaster interface {
	// BroadcastULData forwards uplink data to SCACI clients
	BroadcastULData(ctx context.Context, tenantID int64, data *mioty.ULDataMessage) error
}

// MQTTEventPublisher publishes device events to MQTT (no KC-MQTT imports in pkg/bssci)
type MQTTEventPublisher interface {
	PublishUplink(ctx context.Context, orgUUID string, msg *mioty.ULDataMessage) error
}

// AttachmentDecision is the service center's decision that an endpoint is
// attached or detached, taken for an operator, an application center, or a
// base station that heard the endpoint attach or detach.
type AttachmentDecision struct {
	// TenantID owns the endpoint.
	TenantID   int64
	EndpointID int64
	EpEUI      uint64
	// Status is EndpointStatusAttached or EndpointStatusDetached.
	Status string
	// OverTheAir is the base station's report; nil for a decision taken in
	// the service center.
	OverTheAir *OverTheAirReport
}

// OverTheAirReport is what a base station reported of an attach or detach it
// heard.
type OverTheAirReport struct {
	BaseStationEUI uint64
	// Status carries the over-the-air epStat fields (SCACI §3.13.1).
	Status *EPStatusData
	// Telemetry is recorded with a detachment.
	Telemetry *endpoint.DetachTelemetry
}

// AttachmentDecider records an attachment decision, tells the endpoint
// owner's application centers and MQTT subscribers about it, and reports
// whether it changed the endpoint's status.
type AttachmentDecider interface {
	Decide(ctx context.Context, decision AttachmentDecision) (bool, error)
}

// EPStatusData mirrors scaci.EPStatusData to avoid import cycle
// BSSCI constructs this, SCACI receives it through the attachment decider.
type EPStatusData struct {
	EpEui      uint64            // Endpoint EUI (required)
	EpStatus   string            // "attached" or "detached" (required)
	AttachCnt  *uint32           // OTA attach only
	Nonce      *mioty.Numeric4   // OTA attach only
	Sign       *mioty.Numeric4   // OTA attach/detach only
	Snr        *float64          // OTA attach/detach only
	Rssi       *float64          // OTA attach/detach only
	EqSnr      *float64          // Optional
	Subpackets *mioty.Subpackets // Optional
}

// EPStatus enum values per SCACI §3.13
// Canonical constants are in pkg/mioty/format.go (import mioty.EPStatusAttached/Detached)
// This comment preserved to document the removal; actual values imported from mioty package

// DownlinkCommander sends downlink commands to base stations
type DownlinkCommander interface {
	SendDLRXStatusQuery(sessionID string, epEui uint64) error
}

// SessionDirectory provides session lookup and selection
type SessionDirectory interface {
	GetConnectedSessions() []map[string]interface{}
	GetSessionByEUI(bsEui uint64) interface{}
	SelectBidirectionalSession(tenantID int64, targetBsEui *uint64) (sessionID string, actualBsEui uint64, err error)
	FindSessionForEndpointAttachment(bsEui uint64) (sessionID string, err error)
}

// ULTransmitter sends uplink transmit requests
type ULTransmitter interface {
	SendULDataTransmit(sessionID string, epEui uint64, nwkSnKey []byte,
		shAddr uint16, packetCnt uint32, userData []byte, profile string, format uint8) (int64, error)
}

// StatusRequester sends status requests to base stations
type StatusRequester interface {
	SendStatusRequest(session interface{}) (int64, error)
}

// PingCommander sends ping requests to base stations (BSSCI §5.4)
type PingCommander interface {
	// InitiatePing sends a ping request to the base station associated with the given EUI
	// Returns operation ID on success, error if session not found or handshake incomplete
	// tenantID parameter enables server-side tenant validation (defense-in-depth)
	InitiatePing(ctx context.Context, baseStationEUI uint64, tenantID int64) (int64, error)
}

// QueueSerializer builds canonical BSSCI downlink response frames (BSSCI §5.12-§5.14)
//
// This service encapsulates the construction of dlDataQue* response messages according
// to MIOTY specification requirements. All response maps follow MessagePack encoding rules.
//
// Spec References:
//   - §5.12: DL Data Queue Complete message format
//   - §5.13: DL Data Result Response message format
//   - §5.14: DL Data Result Complete message format
type QueueSerializer interface {
	// BuildDLDataQueueComplete constructs dlDataQueCmp response per BSSCI §5.12
	//
	// Returns map with:
	//   - command: dlDataQueCmp
	//   - opId: Operation ID from original dlDataQue request
	BuildDLDataQueueComplete(opId int64) map[string]interface{}

	// BuildDLDataResultResponse constructs dlDataResRsp response per BSSCI §5.13
	//
	// Returns map with:
	//   - command: dlDataResRsp
	//   - opId: Operation ID from original dlDataRes message
	//   - queId: Queue ID that identifies the downlink
	//   - success: Transmission success flag
	BuildDLDataResultResponse(opId int64, result *mioty.DLDataResult) map[string]interface{}

	// BuildDLDataResultComplete constructs dlDataResCmp response per BSSCI §5.14
	//
	// Returns map with:
	//   - command: dlDataResCmp
	//   - opId: Operation ID from original dlDataRes message
	BuildDLDataResultComplete(opId int64) map[string]interface{}

	// BuildDLDataRevokeComplete constructs dlDataRevCmp response per BSSCI §5.13
	//
	// Returns map with:
	//   - command: dlDataRevCmp
	//   - opId: Operation ID from original dlDataRev request
	BuildDLDataRevokeComplete(opID int64) map[string]interface{}
}

// AuditLogger records downlink audit events to system_events table (BSSCI §5.12, §5.14)
//
// This service provides centralized event logging for downlink operations, ensuring
// consistent event structure and tenant isolation. All events are persisted via
// interfaces.SystemEventStore for traceability and auditing.
//
// Spec References:
//   - §5.12: DL Data Queue acknowledgment tracking
//   - §5.14: DL Data Result event logging
type AuditLogger interface {
	// RecordQueueAck logs successful downlink queue acknowledgment (§5.12)
	//
	// Creates system event when base station acknowledges dlDataQue request.
	// Event captures queId, endpoint EUI, and base station details for audit trail.
	//
	// Parameters:
	//   - ctx: Request context
	//   - tenant: Tenant ID string (formatted via formatTenantID helper)
	//   - session: Active BSSCI session (provides base station identification)
	//   - epEui: Endpoint EUI (extracted from pending operation)
	//   - queueID: Queue ID assigned to downlink
	//   - opId: Operation ID for correlation
	//
	// Returns error only on critical event store failures (logging is best-effort)
	RecordQueueAck(ctx context.Context, tenant string, session *Session,
		epEui uint64, queueID int64, opId int64) error

	// RecordDLRevokeResponse logs a base station's acknowledgment of a
	// downlink revoke (§5.13).
	RecordDLRevokeResponse(ctx context.Context, tenant string, session *Session,
		epEui uint64, queueID int64, opId int64) error

	// RecordQueueRevoked logs a downlink revoked in the service center queue
	// before any base station held it (SCACI §3.11).
	RecordQueueRevoked(ctx context.Context, downlink *storage.DownlinkMessage) error
}

// TenantResolver resolves tenant ownership for downlink queues (BSSCI §5.12-§5.14)
//
// This interface replaces the old queueTenants map pattern with a proper collaborator
// that encapsulates the hot-path (in-memory cache) and cold-path (database fallback)
// tenant resolution strategy. The lifecycle is:
//  1. RegisterQueueTenant during SendDLDataQueue (§5.12) after successful queue
//  2. ResolveTenant during handleDLDataResult (§5.14), handleDLDataRevokeResponse (§5.13)
//  3. UnregisterQueueTenant when queue processing completes
//
// Spec References:
//   - §5.12: DL Data Queue - tenant must be tracked for queue ownership
//   - §5.13: DL Data Revoke - tenant required for authorization checks
//   - §5.14: DL Data Result - tenant required for result routing and cleanup
//
// Replaces: Server.queueTenants map + Server.resolveQueueTenant() method
type TenantResolver interface {
	// ResolveTenant resolves tenant ID for a queue ID (BSSCI §5.12-§5.14)
	//
	// Implementation uses two-tier strategy:
	//   - Hot path: Check in-memory cache (queueTenants map from old Server)
	//   - Cold path: Query database via getDownlinkTenantByQueueID
	//
	// Returns tenant ID string or error if queue not found.
	// Thread-safe for concurrent access.
	ResolveTenant(ctx context.Context, queueID int64) (tenantID string, err error)

	// RegisterQueueTenant registers queue-to-tenant mapping (BSSCI §5.12)
	//
	// Called by SendDLDataQueue after successful message send to cache
	// the queue ownership for fast lookup during result processing.
	// Replaces: s.queueTenants[queId] = tenantID pattern
	RegisterQueueTenant(queueID int64, tenantID string)

	// UnregisterQueueTenant removes queue-to-tenant mapping (BSSCI §5.14)
	//
	// Called when queue processing completes (result received or revoke complete)
	// to prevent unbounded cache growth.
	// Replaces: delete(s.queueTenants, queueID) pattern
	UnregisterQueueTenant(queueID int64)
}

// MessageStore provides database persistence for MIOTY messages and downlink operations
//
// This interface enables testing by allowing mock implementations to replace the concrete
// postgres.DB type. It includes only the methods actually used by the BSSCI server,
// following the Interface Segregation Principle.
//
// Implementation: KC-DB/storage/postgres/postgres.go (*DB type)
// MessageStore interface removed - replaced by interfaces.MIOTYDownlinkRepository
// and interfaces.MIOTYMessageRepository.
// See KC-DB/storage/interfaces/mioty_*_repository.go for interface definitions.

// RoamingService manages endpoint roaming detection and event recording
//
// This service provides roaming detection, validation, and event logging for multi-tenant
// roaming scenarios. It wraps the roaming detector and storage layer to provide a cohesive
// roaming management API.
//
// Spec References:
//   - ../docs/architecture/roaming/: Cross-tenant roaming design
//
// Implementation: KC-Core/internal/services/bssci/roaming_adapter.go
type RoamingService interface {
	// DetectAndValidateRoaming checks if an endpoint is roaming and validates the operation
	//
	// Returns:
	//   - isRoaming: true if endpoint owner differs from serving tenant
	//   - ownerTenantID: the endpoint's home network tenant ID
	//   - err: error if detection failed or roaming not allowed
	DetectAndValidateRoaming(ctx context.Context, epEui []byte, servingTenantID int64) (isRoaming bool, ownerTenantID int64, err error)

	// RecordAttach records an attach event for a roaming endpoint
	//
	// Parameters:
	//   - epEui: Endpoint EUI bytes
	//   - bsEui: Base station EUI bytes
	//   - servingTenantID: Tenant ID of the serving network
	RecordAttach(ctx context.Context, epEui []byte, bsEui []byte, servingTenantID int64) error

	// RecordDetach records a detach event for a roaming endpoint
	RecordDetach(ctx context.Context, epEui []byte, bsEui []byte, servingTenantID int64) error

	// UpdateSessionRoaming updates session roaming state in the database
	//
	// Parameters:
	//   - sessionID: Database session ID
	//   - epEui: Endpoint EUI bytes
	//   - isAttach: true for attach, false for detach
	//   - servingTenantID: Tenant ID of the serving network
	UpdateSessionRoaming(ctx context.Context, sessionID int64, epEui []byte, isAttach bool, servingTenantID int64) error
}

// IngressDisposition classifies how an incoming uplink should be handled.
type IngressDisposition int

const (
	// DispositionLocal indicates the endpoint is owned by a local tenant and should be fully ingested.
	DispositionLocal IngressDisposition = iota
	// DispositionRelay indicates the endpoint is unknown locally and should be forwarded to ECE (CE mode only).
	DispositionRelay
	// DispositionDrop indicates the endpoint is unknown and no relay is configured; the uplink is silently accepted.
	DispositionDrop
)

// IngressDispositionResolver determines how an incoming uplink should be routed based on endpoint ownership.
//
// Implementations check the local endpoint index and relay configuration to classify each uplink
// before the BSSCI handler commits to processing it.
//
// Implementation: KC-Core/internal/services/federation/disposition.go
type IngressDispositionResolver interface {
	Resolve(ctx context.Context, epEUI uint64) (IngressDisposition, error)
}

// UplinkSource identifies the origin of an uplink being ingested.
type UplinkSource int

const (
	// UplinkSourceBSSCI indicates the uplink arrived directly over a BSSCI connection.
	UplinkSourceBSSCI UplinkSource = iota
	// UplinkSourceFederation indicates the uplink was relayed from a CE instance via the federation stream.
	UplinkSourceFederation
)

// UplinkIngestOptions carry per-call context for UplinkIngestService.Ingest.
type UplinkIngestOptions struct {
	// Source distinguishes direct BSSCI uplinks from CE-relayed federation uplinks.
	Source UplinkSource
	// ServingTenantID is the tenant resolved for the serving connection (BSSCI session or federation source).
	// Zero falls back to the ingest service's instance tenant (single-tenant CE default).
	ServingTenantID int64
}

// UplinkPayload is the canonical input to UplinkIngestService.
// Fields are the parsed, validated values extracted from the BSSCI ulData frame
// or the equivalent fields from a federation RelayedUplink.
type UplinkPayload struct {
	// OpID is the base station's ulData operation ID (BSSCI §3.2, §3.10.1).
	OpID        int64
	EpEUI       uint64
	BsEUI       uint64
	PacketCnt   uint32
	UserData    []byte
	SNR         float64
	RSSI        float64
	EqSNR       *float64
	RxTime      int64  // Unix UTC nanoseconds
	RxDuration  *int64 // First to last subpacket center in nanoseconds (optional)
	Profile     *string
	Mode        *string
	Format      *uint8
	Subpackets  *mioty.Subpackets
	DLOpen      bool
	ResponseExp bool
	DlAck       bool
}

// IngestResult carries the outcome of a successful UplinkIngestService.Ingest call.
// The BSSCI handler uses these values to drive downlink dispatch and response framing.
type IngestResult struct {
	// IsDuplicate reports whether this reception was merged into an existing message row.
	IsDuplicate bool
	// OwnerTenantID is the resolved tenant that owns this endpoint.
	OwnerTenantID int64
	// OwnerOrgUUID is the organization UUID of the owning tenant (uuid.Nil if unresolved).
	OwnerOrgUUID uuid.UUID
	// MessageID is the telegram's message row: this reception's own, or the first reception's for a merged duplicate.
	MessageID string
}

// UplinkIngestService handles the shared ingest pipeline for uplinks arriving via BSSCI or federation relay.
//
// It performs deduplication, tenant resolution, payload decoding, persistence, SCACI fan-out,
// and MQTT publishing. Downlink dispatch is intentionally excluded; callers with an active BSSCI
// session should invoke the downlink dispatcher independently using the returned IngestResult.
//
// Implementation: KC-Core/internal/services/bssci/uplink_ingest_service.go
type UplinkIngestService interface {
	Ingest(ctx context.Context, payload *UplinkPayload, opts UplinkIngestOptions) (*IngestResult, error)
}

// RelayOutboxWriter enqueues an uplink frame into the CE durable outbox for federation relay.
// It is called from handleULData when the disposition resolver returns DispositionRelay.
//
// Implementation: KC-Core/internal/services/federation/outbox.go
type RelayOutboxWriter interface {
	// Enqueue inserts a pending relay record and returns the assigned relay UUID.
	// The BSSCI handler must NOT send ulDataRsp until Enqueue returns nil.
	// On error, the handler must send a BSSCI error response instead.
	Enqueue(ctx context.Context, epEUI, bsEUI uint64, rawFrame []byte, receivedAtNs int64) (uuid.UUID, error)
}

// PropagationReconciler manages automatic endpoint propagation on base station connect.
//
// The reconciler is invoked asynchronously when a base station completes handshake
// (handleConnectComplete), eliminating the need for manual API-triggered propagation.
//
// Behavior:
//   - Streams endpoints across all tenants (cross-tenant roaming support)
//   - Resolves owner tenant per endpoint
//   - Processes in configurable batches with rate limiting
//   - Implements exponential backoff for retry on failures
//   - Persists progress to survive server restarts
//

// DetachSignatureValidator validates detach signatures for known and unknown
// endpoints when detach signature validation is enabled.
//
// The MIOTY spec (BSSCI §5.7.1) says the detach signature is "analogous to
// attach" but does not define its CMAC construction, so no authoritative
// validator ships with the community edition: enabling the feature requires
// injecting one, and startup refuses an enabled configuration without it.
// With validation disabled (the default) a well-formed detach is accepted and
// durably recorded as unverified.
//
// Spec References:
//   - BSSCI §5.7: Detach operation signature validation
type DetachSignatureValidator interface {
	// ValidateDetachSignature validates a detach signature and resolves the
	// endpoint owner. It is called for every detach while validation is
	// enabled - for unknown endpoints the returned tenant metadata routes the
	// record to its owner.
	//
	// Parameters:
	//   - ctx: Request context (for cancellation and tracing)
	//   - epEUI: Endpoint EUI (8 bytes as uint64)
	//   - detachSign: 4-byte detach signature from BSSCI message
	//
	// Returns:
	//   - result: Validation result with tenant/org metadata (nil on error)
	//   - error: Validation failed (endpoint not found, signature mismatch, or DB error)
	//     - ErrDetachValidationEndpointNotFound: Endpoint not found in database
	//     - ErrDetachSignatureInvalid: Signature mismatch
	//     - Other errors: Database failures
	ValidateDetachSignature(ctx context.Context, epEUI uint64, detachSign []byte) (*DetachValidationResult, error)
}

// DetachValidationResult captures endpoint signature validation outcome with tenant metadata
// DetachValidationResult is returned by DetachSignatureValidator for roaming-safe tenant assignment
type DetachValidationResult struct {
	Valid            bool   // Overall validation result (true = signature valid)
	TenantID         int64  // Endpoint owner tenant (for session tenant assignment)
	OwnerTenantID    int64  // Original owner tenant (roaming support - may differ from TenantID)
	ValidationStatus string // One of: bssci.ValidationStatusValidated | ValidationStatusInvalidSignature | ValidationStatusUnverified
}

// DownlinkDispatcher handles auto-dispatch on dlOpen=true (BSSCI §5.10.2)
//
// This interface abstracts downlink auto-dispatch when an endpoint signals it's ready
// to receive data (dlOpen=true in uplink). The dispatcher performs transactional
// reserve→send→mark-queued operations inside storage-owned transactions.
//
// Spec References:
//   - BSSCI §5.10.2: Downlink window opportunity
//
// Implementation: KC-Core/internal/services/bssci/downlink_dispatcher.go
type DownlinkDispatcher interface {
	// DispatchIfAvailable performs transactional reserve→send→mark-queued for pending downlinks.
	//
	// Parameters:
	//   - ownerCtx: Context with owner tenant/org metadata (NOT session tenant for roaming safety)
	//   - ownerTenantID: Owner tenant ID (endpoint owner, may differ from session tenant)
	//   - session: Active BSSCI session to send downlink via
	//   - epEUI: Endpoint EUI that signaled dlOpen=true
	//   - messageID: The telegram's message row, whose downlink window one dispatch claims
	//   - responseExp: Response expected flag from uplink
	//
	// The delivery organization is the reserved queue row's organization_id:
	// several organizations can share a tenant, and the row records which one
	// enqueued the downlink, so no resolver-derived organization is passed in.
	//
	// Returns:
	//   - dispatched: true if downlink was successfully dispatched, false otherwise
	//   - err: error if dispatch operation failed (caller should log but not fail uplink)
	//
	// Thread Safety:
	//   Uses FOR UPDATE SKIP LOCKED to prevent concurrent dispatchers from reserving
	//   the same downlink. Only the reception that created the uplink message
	//   dispatches, so several base stations hearing one telegram fill its window once.
	DispatchIfAvailable(
		ownerCtx context.Context,
		ownerTenantID int64,
		session *Session,
		epEUI uint64,
		messageID string,
		responseExp bool,
	) (dispatched bool, err error)

	// DispatchQueue reserves one exact pending queue row (by queue ID, tenant,
	// endpoint EUI, and the organization the downlink was enqueued under) and
	// dispatches it over the given session. Used for SCACI-initiated immediate
	// delivery (SCACI §3.10.1) so both delivery paths share the dispatcher's
	// pending→reserved→queued lifecycle. enqueueOrgUUID must be the enqueuing
	// caller's organization, never the base station session's organization.
	// Returns dispatched=false with nil error when no matching pending row
	// exists (already dispatched, revoked, or foreign).
	DispatchQueue(
		ownerCtx context.Context,
		ownerTenantID int64,
		enqueueOrgUUID uuid.UUID,
		session *Session,
		queueID uint64,
		epEUI uint64,
	) (dispatched bool, err error)
}

// DownlinkReclaimer returns to pending the downlinks a base station no longer
// holds, so they are dispatched again, and reports how many it released. It
// reports no result for them, as they are still to be sent; only an overdue
// downlink the station was asked to drop and discarded ends expired.
type DownlinkReclaimer interface {
	// ReclaimReservations returns to pending every downlink the base station
	// holds reserved except the rows behind the dlDataQue operations reissued
	// on resume.
	ReclaimReservations(ctx context.Context, bsEUI uint64, reissuedQueIDs []int64) (released int64, err error)

	// ReclaimDiscardedQueue returns to pending every downlink queued at a
	// base station whose new session is not resumed and so discarded them
	// (BSSCI §1), and ends expired the overdue ones it was asked to drop; it
	// counts those returned to pending.
	ReclaimDiscardedQueue(ctx context.Context, bsEUI uint64) (released int64, err error)

	// ReclaimEndpointQueue returns to pending the owner tenant's downlinks
	// for the endpoint that were queued at the base station by the time an
	// attach propagate for the endpoint was issued: the station discards them
	// when it takes the attachment (BSSCI §3.8).
	ReclaimEndpointQueue(ctx context.Context, ownerTenantID int64, epEUI, bsEUI uint64, propagatedAt time.Time) (released int64, err error)
}

// BlueprintDecoder decodes MIOTY payloads using blueprint definitions (MIOTY App Layer Spec)
//
// This interface abstracts the payload decoding logic from the BSSCI server, enabling
// proper separation and testability. The decoder parses blueprint JSON specifications
// and applies them to raw payload bytes to extract typed field values.
//
// Spec References:
//   - MIOTY Application Layer Specification v1.0.0
//   - Blueprint JSON schema (version, typeEui, uplink[], downlink[])
//
// Implementation: KC-Core/internal/services/bssci/blueprint/decoder_service.go
type BlueprintDecoder interface {
	// Decode decodes raw payload using blueprint spec for given format ID.
	//
	// Parameters:
	//   - ctx: Request context for cancellation and tracing
	//   - bp: Blueprint model containing spec_json
	//   - userData: Raw payload bytes to decode
	//   - formatID: MIOTY format identifier from uplink message
	//   - calibration: Per-endpoint calibration data (can be nil)
	//
	// Returns:
	//   - DecodeResult: Success with decodedData, or failure with errorCode/errorDetail
	//   - error: Only for internal/infrastructure errors (not decode validation errors)
	//
	// Decode errors (invalid JSON, missing fields, etc.) are returned as DecodeResult
	// with Success=false and appropriate error token from errors_catalog.go.
	Decode(ctx context.Context, bp *models.Blueprint, userData []byte,
		formatID uint8, calibration map[string]interface{}) (*blueprint.DecodeResult, error)
}

// BlueprintResolver finds blueprints for endpoints based on TypeEUI and device model
//
// This interface implements the TypeEUI precedence rule:
//  1. If endpoint has device_model_id, get the default blueprint from that model
//  2. Otherwise, look up blueprint by Type EUI directly
//
// The resolver abstracts the database lookups and caching from the BSSCI server,
// enabling proper separation and testability.
//
// Spec References:
//   - MIOTY Application Layer Specification v1.0.0
//   - Blueprint device model association (TypeEUI precedence rule)
//
// Implementation: KC-Core/internal/services/bssci/blueprint/resolver.go
type BlueprintResolver interface {
	// ResolveBlueprint finds the applicable blueprint for an endpoint's type EUI.
	//
	// Parameters:
	//   - ctx: Request context for cancellation and tracing
	//   - tenantID: Tenant ID for multi-tenant isolation
	//   - typeEUI: 8-byte Type EUI from uplink message (can be nil)
	//   - formatID: Optional format ID hint (can be nil)
	//
	// Returns:
	//   - Blueprint model if found, nil if not found (not an error)
	//   - error: Only for infrastructure errors (DB failures, etc.)
	//
	// Returns nil (not error) if no blueprint matches - this allows the caller
	// to gracefully skip decoding when no blueprint is available.
	ResolveBlueprint(ctx context.Context, tenantID int64, typeEUI []byte,
		formatID *uint8) (*models.Blueprint, error)

	// ResolveBlueprintForEndpoint finds the applicable blueprint for an endpoint.
	//
	// This method implements the TypeEUI precedence rule:
	//   1. If endpoint has device_model_id, get the default blueprint from that model
	//   2. Otherwise, fall back to endpoint's TypeEUI for direct lookup
	//
	// Parameters:
	//   - ctx: Request context
	//   - tenantID: Tenant ID for multi-tenant isolation
	//   - endpoint: Endpoint model (contains device_model_id and type_eui)
	//   - formatID: Optional format ID hint
	//
	// Returns blueprint or nil (not error) if no blueprint found.
	ResolveBlueprintForEndpoint(ctx context.Context, tenantID int64,
		endpoint *models.EndPoint, formatID *uint8) (*models.Blueprint, error)

	// GetEndpointCalibration retrieves calibration data for an endpoint.
	//
	// Returns calibration data map, or empty map if none available.
	// Keys in the map correspond to $calibration.key references in blueprint func expressions.
	GetEndpointCalibration(ctx context.Context, tenantID int64,
		endpoint *models.EndPoint) map[string]interface{}
}

// ProtocolMessageStore records BSSCI protocol message rows (detach and
// propagate messages). Satisfied structurally by the MIOTY message repository.
type ProtocolMessageStore interface {
	CreateDetachMessage(ctx context.Context, msg *mioty.DetachMessage, structuredMsg map[string]interface{}) error
	CreateAttachPropagateMessage(ctx context.Context, msg *mioty.AttachPropagateMessage) error
	CreateDetachPropagateMessage(ctx context.Context, msg *mioty.DetachPropagateMessage) error
}

// DLRXStatusStore owns dlRxStatQry correlation rows and dlRxStat reports
// (BSSCI rev1 §5.15-§5.16 / classic §3.15-§3.16). Satisfied structurally by
// the DL RX status repository.
type DLRXStatusStore interface {
	CreateDLRXStatus(ctx context.Context, status *mioty.DLRXStatus) error
	MarkDLRXStatusReceived(ctx context.Context, tenantID int64, epEui, bsEui []byte, bsOpID int64) (bool, error)
	CreateDLRXStatusQuery(ctx context.Context, tenantID int64, orgUUID *uuid.UUID, epEui, bsEui []byte, opId int64) error
	ExpireDLRXStatusQuery(ctx context.Context, cutoff time.Time) (int64, error)
}

// BaseStationStatusStore records base station status history rows (BSSCI
// rev1 §5.5 / classic §3.5). Satisfied structurally by the status repository.
type BaseStationStatusStore interface {
	Create(ctx context.Context, status *mioty.BaseStationStatusRecord) error
}

// DownlinkQueueStore reads the queue row a revocation targets, of the owner
// the revocation names. Satisfied structurally by the MIOTY downlink
// repository so error identity (sql.ErrNoRows) is preserved - never wrap it
// in a delegating adapter.
type DownlinkQueueStore interface {
	GetDownlinkByRevocation(ctx context.Context, revocation storage.DownlinkRevocation) (*storage.DownlinkMessage, error)
	// ListStationRevocations lists the downlinks the base station is asked to
	// drop because their lifetime ended while it held them.
	ListStationRevocations(ctx context.Context, bsEUI uint64) ([]*storage.DownlinkMessage, error)
}

// DownlinkRevocationStore revokes a downlink where it waits; a revocation
// without a station ends a downlink no base station holds yet, reporting
// false when the row is no longer pending.
type DownlinkRevocationStore interface {
	RevokeDownlink(ctx context.Context, revocation storage.DownlinkRevocation) (bool, error)
}

// PendingDownlinkLister lists the unexpired pending downlinks of every
// tenant, grouped by endpoint in the order a downlink window takes them.
type PendingDownlinkLister interface {
	ListPendingDownlinks(ctx context.Context) ([]storage.PendingDownlink, error)
}

// ServingStationLocator decides the base station serving a tenant's
// endpoint: known is false while no station heard or attached it, and
// storage.ErrNotFound means the tenant has no such endpoint.
type ServingStationLocator interface {
	ServingStation(ctx context.Context, tenantID int64, epEUI uint64) (bsEUI uint64, known bool, err error)
}

// EndpointDirectory is the endpoint repository surface the protocol server
// consumes: reads plus the attach/detach field updates. Satisfied
// structurally by the endpoint repository.
type EndpointDirectory interface {
	Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error)
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error)
	GetByID(ctx context.Context, id int64, tenantID int64) (*models.EndPoint, error)
	EndpointAttachmentStateUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointAttachmentStateParams) error
	EndpointDetachStateUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointDetachStateParams) error
	TransitionEndpointStatus(ctx context.Context, tenantID int64, endpointID int64, status string) (bool, error)
	UpdateRadioMetricsSelective(ctx context.Context, tenantID int64, eui models.EUI, update models.RadioMetricsUpdate) error
}

// EndpointOwner is an endpoint and the tenant that owns it
// (endpoints.owner_tenant_id): every station serves the endpoint for that
// tenant, whichever tenant the station belongs to (BSSCI §5.8.3).
type EndpointOwner struct {
	TenantID int64
	Endpoint *models.EndPoint
}

// EndpointOwnerResolver finds the owner of an endpoint by EUI;
// storage.ErrNotFound when no tenant owns it.
type EndpointOwnerResolver interface {
	ResolveOwner(ctx context.Context, eui models.EUI) (EndpointOwner, error)
}

// BaseStationStore is the registered-station repository surface the protocol
// server consumes. Satisfied structurally by the base station repository.
type BaseStationStore interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)
	Update(ctx context.Context, tenantID, id int64, updates map[string]interface{}) error
}

// OrganizationDirectory resolves organization identity for tenants and
// certificates. Satisfied structurally by org.Resolver implementations.
type OrganizationDirectory interface {
	ResolveCert(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, error)
	GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error)
}

// EventStore records system events emitted by the protocol server.
// Satisfied structurally by the system event repository.
type EventStore interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// AttachSessionRecord carries the attach-transaction inputs for the endpoint
// attachment persister (BSSCI rev1 §5.7 / classic §3.7).
type AttachSessionRecord struct {
	// TenantID is the endpoint owner tenant used for the endpoint update and
	// the session row.
	TenantID int64
	// BSLookupTenantID scopes the primary base station lookup (differs from
	// TenantID when the uplink arrived through a roaming station).
	BSLookupTenantID int64
	EndpointID       int64
	EndpointUpdates  models.EndpointAttachmentStateParams
	EncryptedKey     []byte
	// AttachCnt is handler-validated to fit 24 bits before persistence.
	AttachCnt      uint32
	ShAddr         uint16
	BaseStationEUI []byte
}

// AttachPropagateSessionRecord carries the attach-propagate transaction inputs
// (BSSCI rev1 §5.8 / classic §3.8); the owner tenant scopes every operation.
type AttachPropagateSessionRecord struct {
	TenantID        int64
	EndpointID      int64
	EndpointUpdates models.EndpointAttachSessionParams
	EncryptedKey    []byte
	ShAddr          uint16
	BaseStationEUI  []byte
}

// EndpointAttachmentPersistence owns the transactional attach and
// attach-propagate endpoint-session persistence: one transaction covering the
// endpoint field update and the endpoint-session upsert, with the primary
// base station looked up outside the transaction.
type EndpointAttachmentPersistence interface {
	PersistAttachSession(ctx context.Context, rec AttachSessionRecord) error
	PersistAttachPropagateSession(ctx context.Context, rec AttachPropagateSessionRecord) error
}

// NetworkSessionKeySource yields the network session key an endpoint currently
// operates on, the key an attach propagate hands to the base stations (radio
// spec §3.7.1.2-§3.7.1.3).
type NetworkSessionKeySource interface {
	NetworkSessionKey(ctx context.Context, endpoint *models.EndPoint) ([]byte, error)
}
