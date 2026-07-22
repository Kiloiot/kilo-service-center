package bssci

import (
	"context"
	"encoding/json"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
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
	// ResumeNoMatch: no resumable session exists; start a fresh session.
	ResumeNoMatch ResumeDisposition = iota
	// ResumeCompatible: a resumable session was found and its constraints
	// hold; Previous carries the authoritative persisted session state.
	ResumeCompatible
	// ResumeInconsistent: a resumable session exists but the reported
	// counters or negotiated version are incompatible. The caller must
	// atomically terminate it (can_resume=false, pending ops removed) before
	// starting a fresh session.
	ResumeInconsistent
	// ResumeInfrastructureFailure: the resume lookup itself failed (e.g. a
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

// DownlinkService uses REAL postgres.DB methods (not storage.Storage)
// Preserves: CreateULDataMessage, EnqueueDownlink with MIOTY types
type DownlinkService interface {
	// EnqueueDownlink enqueues a downlink message using storage interface
	EnqueueDownlink(ctx context.Context, epEui uint64, userData []byte, priority float32, tenantID int64) (queId int64, err error)

	// UpdateDownlinkStatus updates downlink message status via storage interface
	UpdateDownlinkStatus(ctx context.Context, queId uint64, tenantIDStr string, status string) error

	// ProcessDLDataResult handles complete downlink result processing (BSSCI §5.14)
	// Orchestrates: tenant resolution, DB update, SCACI broadcast, audit logging, cleanup
	// Returns response message for handler to send via sendMessage
	ProcessDLDataResult(ctx context.Context, session *Session, result *mioty.DLDataResult) (responseMsg map[string]interface{}, err error)

	// ProcessRevokeResponse handles downlink revoke response processing (BSSCI §5.13)
	// Orchestrates: tenant resolution, DB update, event recording, cleanup
	// Returns response message for handler to send via sendMessage
	ProcessRevokeResponse(ctx context.Context, session *Session, opId int64, queueID int64, endpointEUI uint64) (responseMsg map[string]interface{}, err error)
}

// StatusService manages pendingOps map + DB persistence.
// Preserves: PendingOperation struct, bssci_pending_operations table.
// All methods accept a session parameter for composite key support (BSSCI §5.11-5.12.3).
type StatusService interface {
	// RecordPendingOperation stores in map + DB using SessionOpKey composite key
	RecordPendingOperation(ctx context.Context, session *Session, opId int64, op *PendingOperation, dbSessionID int64) error

	// GetPendingOperation retrieves from map using SessionOpKey composite key
	GetPendingOperation(session *Session, opId int64) (*PendingOperation, error)

	// RemovePendingOperation cleans map + DB using SessionOpKey composite key
	RemovePendingOperation(ctx context.Context, session *Session, opId int64) error

	// ExtractQueueMetadata retrieves endpoint EUI, queue ID, and tenant ID from pending operation.
	// Used by downlink handlers to correlate responses with original requests.
	// Returns tenantID for proper roaming tenant isolation (BSSCI §5.12).
	// Session parameter required for SessionOpKey composite key lookup.
	ExtractQueueMetadata(session *Session, opId int64) (endpointEUI uint64, queueID int64, tenantID string)

	// CleanupPendingOp removes pending operation from in-memory map only using SessionOpKey.
	// Enables handlers to clean up map without direct mutex access.
	// DB cleanup should use RemovePendingOperation for complete cleanup.
	CleanupPendingOp(session *Session, opId int64)
}

// ConnectionService wraps REAL basestation.ConnectionManager operations
// Preserves: GetBaseStation, UpdateConnectionStatus, EventRecorder
type ConnectionService interface {
	// GetBaseStation via connectionMgr (tenant-scoped via context)
	GetBaseStation(ctx context.Context, eui [8]byte, mgr *basestation.ConnectionManager) (*basestation.BaseStation, error)

	// GetBaseStationGlobal retrieves a base station by EUI across all tenants.
	// Used during BSSCI connect handshake when tenant is not yet resolved.
	GetBaseStationGlobal(ctx context.Context, eui [8]byte, mgr *basestation.ConnectionManager) (*basestation.BaseStation, error)

	// RegisterConnection updates basestation status
	RegisterConnection(ctx context.Context, session *Session, baseStation *basestation.BaseStation, mgr *basestation.ConnectionManager) error
}

// SCACIBroadcaster forwards via real scaciBroadcaster interface
// Matches the exact signatures from scaciBroadcaster to enable proper delegation
type SCACIBroadcaster interface {
	// BroadcastULData forwards uplink data to SCACI clients
	BroadcastULData(ctx context.Context, tenantID int64, data *mioty.ULDataMessage) error

	// BroadcastDLDataResult forwards downlink results to SCACI clients
	BroadcastDLDataResult(ctx context.Context, tenantID int64, result *mioty.DLDataResult) error
}

// MQTTEventPublisher publishes device events to MQTT (no KC-MQTT imports in pkg/bssci)
type MQTTEventPublisher interface {
	PublishUplink(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64,
		rssi float64, snr float64, rxTime int64, packetCnt uint32, userData []byte, decodedPayload []byte) error
	PublishAttach(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64) error
	PublishDetach(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64) error
	PublishDownlinkResult(ctx context.Context, orgUUID string, epEUI uint64, queID uint64, result string) error
}

// SCACIEPStatusBroadcaster forwards endpoint attach/detach status to SCACI ACs (SCACI §3.13)
//
// This interface allows BSSCI to notify SCACI clients when endpoints attach or detach
// without creating import cycles. Implemented by scaci.Server.BroadcastEPStatus().
//
// Spec References:
//   - SCACI §3.13: EPStatus operation (SC-initiated)
//   - BSSCI §5.6: Attach complete triggers EPStatus
//   - BSSCI §5.7: Detach complete triggers EPStatus
type SCACIEPStatusBroadcaster interface {
	// BroadcastEPStatus forwards endpoint status to all active ACs for tenant
	//
	// Parameters:
	//   - ctx: Context for operation
	//   - tenantID: Tenant ID to filter sessions
	//   - data: EPStatus data containing epEui, epStatus, and optional OTA fields
	//
	// Returns error if any sends fail. Returns nil if no active ACs (not an error).
	BroadcastEPStatus(ctx context.Context, tenantID int64, data *EPStatusData) error
}

// EPStatusData mirrors scaci.EPStatusData to avoid import cycle
// BSSCI constructs this, SCACI receives via SCACIEPStatusBroadcaster interface
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
	SendDLDataQueue(sessionID string, epEui uint64, payloads [][]byte, queId int64,
		prio float32, cntDepend bool, packetCnt []int64, format uint8,
		responseExp bool, responsePrio bool, dlWindReq bool, expOnly bool, tenantID int64) error
	SendDLDataRevoke(sessionID string, epEui uint64, queId uint64) error
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
	//   - command: "dlDataQueCmp"
	//   - opId: Operation ID from original dlDataQue request
	BuildDLDataQueueComplete(opId int64) map[string]interface{}

	// BuildDLDataResultResponse constructs dlDataResRsp response per BSSCI §5.13
	//
	// Returns map with:
	//   - command: "dlDataResRsp"
	//   - opId: Operation ID from original dlDataRes message
	//   - queId: Queue ID that identifies the downlink
	//   - success: Transmission success flag
	BuildDLDataResultResponse(opId int64, result *mioty.DLDataResult) map[string]interface{}

	// BuildDLDataResultComplete constructs dlDataResCmp response per BSSCI §5.14
	//
	// Returns map with:
	//   - command: "dlDataResCmp"
	//   - opId: Operation ID from original dlDataRes message
	BuildDLDataResultComplete(opId int64) map[string]interface{}

	// BuildDLDataRevokeComplete constructs dlDataRevCmp response per BSSCI §5.13
	//
	// Returns map with:
	//   - command: "dlDataRevCmp"
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

	// RecordDLResult logs downlink transmission result (§5.14)
	//
	// Creates system event when downlink transmission completes (success or failure).
	// Event captures transmission outcome and timing for reporting.
	//
	// Parameters:
	//   - ctx: Request context
	//   - tenant: Tenant ID string
	//   - session: Active BSSCI session
	//   - result: DL Data Result message from base station
	//
	// Returns error only on critical event store failures
	RecordDLResult(ctx context.Context, tenant string, session *Session,
		result *mioty.DLDataResult) error
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

// RoamingDetector resolves endpoint ownership and validates roaming agreements
//
// This interface supports multi-tenant roaming scenarios where an endpoint's owner tenant
// (home network) differs from the serving tenant (visited network). In production roaming
// deployments, all base stations on a server form a shared RF mesh, and any base station
// can accept traffic from visiting endpoints.
//
// The stub implementation (internal/services/bssci/roaming_detector.go) returns
// non-roaming defaults until full roaming behavior is implemented.
//
// Use Cases (Future):
//   - BSSCI uplink handling: Resolve endpoint owner to route data/events correctly
//   - SCACI broadcast: Send uplinks to owner tenant's clients (not session tenant)
//   - Downlink queue: Track ownership via owner tenant (not serving tenant)
//   - Billing/metrics: Attribute usage to owner tenant
//
// Spec References:
//   - Multi-tenant isolation: see roaming architecture docs
//   - Roaming design: ../docs/architecture/roaming/
type RoamingDetector interface {
	// IsRoaming checks if an endpoint is currently roaming.
	// Returns true when endpoint's owner tenant differs from session tenant.
	IsRoaming(ctx context.Context, endpointEUI uint64, sessionTenantID int64) (bool, error)

	// ResolveOwnerTenant determines the home network tenant for an endpoint.
	// Returns owner tenant ID regardless of which tenant's base station is serving.
	// Community edition: returns 0 (callers fall back to session tenant).
	ResolveOwnerTenant(ctx context.Context, endpointEUI uint64) (int64, error)

	// ValidateAgreement checks if a roaming agreement exists between tenants.
	// Community edition: always returns true (permit all).
	ValidateAgreement(ctx context.Context, homeTenant, visitingTenant int64) (bool, error)
}

// AttachPropagateSender abstracts propagation message sending to avoid circular dependencies.
//
// This interface is implemented by Server and consumed by PropagationReconciler.
// It provides a thin abstraction layer that allows the reconciler service to
// send attach propagate messages without depending on the entire Server struct,
// avoiding import cycles between reconciler and server packages.
//
// Spec References:
//   - BSSCI §5.8: Attach Propagate three-way handshake
//
// Implementation Notes:
//   - reconcileCtx contains reconcileJobID in Value("reconcileJobID") for metadata tracking
//   - ownerCtx contains tenant/org for database operations (resolved by reconciler)
//   - Dual context pattern avoids mixing concerns (job tracking vs tenant isolation)
type AttachPropagateSender interface {
	// SendAttachPropagateToSession sends attach propagate for a single endpoint to a session.
	//
	// Parameters:
	//   - ownerCtx: Contains tenant/org context (first parameter per Go convention)
	//   - session: Target base station session for propagation
	//   - endpoint: Endpoint to propagate (includes owner tenant ID)
	//
	// The method handles:
	//   - Operation ID generation (negative, monotonic per session)
	//   - Message encoding (MessagePack per BSSCI spec)
	//   - Pending operation persistence
	//   - Wire transmission to base station
	//
	// Returns error if encoding, persistence, or transmission fails.
	SendAttachPropagateToSession(
		ownerCtx context.Context,
		session *Session,
		endpoint *models.EndPoint,
	) error

	// SendAttachPropagateBySessionID sends attach propagate using session ID lookup.
	// Used by propagation service which only has BaseStationSession snapshots (no *Session).
	//
	// Parameters:
	//   - ctx: Context with tenant/org values for database operations
	//   - sessionID: Session identifier (string) from BaseStationSession.ID
	//   - endpoint: Endpoint to propagate
	//
	// Returns error if session not found or propagation fails.
	SendAttachPropagateBySessionID(
		ctx context.Context,
		sessionID string,
		endpoint *models.EndPoint,
	) error
}

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
	// FederationCEID is the CE instance identifier; set when Source == UplinkSourceFederation.
	FederationCEID string
	// FederationRelayID is the CE-generated relay UUID for idempotency; set when Source == UplinkSourceFederation.
	FederationRelayID string
	// ServingTenantID is the tenant resolved for the serving connection (BSSCI session or federation source).
	// Zero falls back to the ingest service's instance tenant (single-tenant CE default).
	ServingTenantID int64
}

// UplinkPayload is the canonical input to UplinkIngestService.
// Fields are the parsed, validated values extracted from the BSSCI ulData frame
// or the equivalent fields from a federation RelayedUplink.
type UplinkPayload struct {
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
	// OwnerTenantID is the resolved tenant that owns this endpoint.
	OwnerTenantID int64
	// OwnerOrgUUID is the organization UUID of the owning tenant (uuid.Nil if unresolved).
	OwnerOrgUUID uuid.UUID
	// IsDuplicate reports whether this reception was merged into an existing message row.
	IsDuplicate bool
	// MessageID is the UUID assigned to the persisted message row (empty for duplicates).
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

// DetachSignatureValidator validates detach signatures for unknown endpoints
//
// This interface abstracts the detach signature validation via direct database lookup,
// enabling proper layering and testability. The validator is called when an endpoint
// is not found in the server's session cache during detach operations.
//
// KC-Core is self-contained - validation uses direct repository calls.
//
// Spec References:
//   - BSSCI §5.7: Detach operation signature validation
//
// Implementation: KC-Core/internal/services/bssci/detach_validator_adapter.go
type DetachSignatureValidator interface {
	// ValidateDetachSignature validates an unknown endpoint's detach signature
	//
	// This method is called when:
	//   - Detach message received from base station
	//   - Endpoint not found in active sessions (unknown endpoint)
	//   - Need to validate cryptographic signature before rejecting
	//
	// The validator (direct repository mode):
	//   1. Queries endpoint repository for crypto material (Sign, NwkSnKey, PresharedKey)
	//   2. Validates signature using ValidateDetachSignatureCMAC
	//   3. Returns validation result with tenant metadata
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
	ValidationStatus string // One of: bssci.ValidationStatusValidated | ValidationStatusInvalidSignature | ValidationStatusUnknownEndpoint
}

// DownlinkDispatcher handles auto-dispatch on dlOpen=true (BSSCI §5.10.2)
//
// This interface abstracts downlink auto-dispatch when an endpoint signals it's ready
// to receive data (dlOpen=true in uplink). The dispatcher performs transactional
// reserve→send→mark-queued operations using the interfaces.Transaction pattern.
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
	//   - ownerOrgUUID: Owner organization UUID for audit logging
	//   - session: Active BSSCI session to send downlink via
	//   - epEUI: Endpoint EUI that signaled dlOpen=true
	//   - responseExp: Response expected flag from uplink
	//   - dlAck: Downlink acknowledgment flag from uplink
	//
	// Returns:
	//   - dispatched: true if downlink was successfully dispatched, false otherwise
	//   - err: error if dispatch operation failed (caller should log but not fail uplink)
	//
	// Thread Safety:
	//   Uses FOR UPDATE SKIP LOCKED to prevent concurrent dispatchers from reserving
	//   the same downlink. Multiple base stations seeing the same dlOpen will have
	//   only ONE successfully dispatch due to transaction isolation.
	DispatchIfAvailable(
		ownerCtx context.Context,
		ownerTenantID int64,
		ownerOrgUUID uuid.UUID,
		session *Session,
		epEUI uint64,
		responseExp bool,
		dlAck bool,
	) (dispatched bool, err error)
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
