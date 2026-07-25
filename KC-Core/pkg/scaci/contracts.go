// Package scaci defines service contracts for SCACI v1.0.0 protocol operations
//
// This package defines narrow interfaces that decouple SCACI handlers from concrete
// implementations while reusing existing centralized infrastructure. These interfaces
// enable dependency inversion without duplicating logic or creating new constant catalogs.
//
// Design:
//
//   - Tenant Resolution: Service layer owns tenant resolution via injected org.Resolver.
//     HandshakeService receives certificate and resolves tenant internally (NOT pre-resolved).
//     Enables tenant re-validation during session resumption (fail-closed security).
//
//   - Session Persistence: Delegates to existing interfaces.SCACISessionRepository (14 methods).
//     NO new SessionRepository interface is created - we reuse the existing scoped interface.
//
//   - Error Handling: Services return tokens from pkg/scaci/errors_catalog.go (79+ tokens).
//     Transport layer resolves tokens via GetErrorDefinition() before sendError().
//
//   - Logging: Services use constants from pkg/scaci/log_messages.go (148+ constants).
//     NO inline log strings permitted.
//
//   - BSSCI Delegation: UL/DL services wrap pkg/scheduler interfaces (ULTransmitScheduler, DownlinkScheduler).
//     These are adapters that translate scheduler errors to SCACI error tokens.
//
//   - Message Persistence: All writes funnel through KC-DB/storage/postgres/message_repository.go.
//     Per BSSCI requirement - NO direct INSERT/UPDATE statements in services.
//
// Reused Infrastructure (no duplication):
//
//   - errors_catalog.go               - 79+ error tokens with spec sections
//   - log_messages.go                 - 148+ log message constants
//   - constants.go                    - Session timeouts, operation limits
//   - pkg/org/resolver.go             - org.Resolver interface (injected into handshake service)
//   - scheduler/errors.go             - ULTransmitScheduler, DownlinkScheduler interfaces
//   - interfaces.SCACISessionRepository - Session persistence (14 scoped methods)
//   - message_repository.go           - Canonical write paths for uplink/downlink
//
// Call Flow:
//
//  1. TLS handshake completes, transport layer extracts client certificate
//  2. Route to handleConnect(conn, &session, cert, opId, payload)
//  3. Handler threads certificate to HandshakeService.ValidateConnect(ctx, req, cert)
//  4. Service resolves tenant from certificate via injected org.Resolver
//  5. Service validates tenant on fresh connect AND session resumption
//
// This moves tenant resolution into the service layer, enabling proper
// tenant validation during session resumption and fail-closed security (reject cross-tenant
// resume attempts). Transport layer handles certificate extraction only.
package scaci

import (
	"context"
	"crypto/x509"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// HandshakeService manages SCACI connect flow per MIOTY §3.3
//
// This service handles connection establishment, version negotiation, and session
// resumption logic. It OWNS tenant resolution from client certificates via injected
// org.Resolver - the transport layer extracts the certificate but does NOT resolve tenant.
//
// Design:
//   - Service receives certificate and resolves tenant internally
//   - Enables tenant re-validation during session resumption
//   - org.Resolver injected into service constructor (dependency inversion)
//   - Rejects resume attempts if certificate tenant doesn't match stored session tenant
//
// Session Persistence:
//   - Uses existing interfaces.SCACISessionRepository (14 methods)
//   - CheckSessionResumable() validates resume tokens per SCACI §3.3
//   - CreateSession() establishes new sessions with UUID generation
//   - UpdateOperationIDs() tracks AC/SC operation ID progression
//
// Error Handling:
//   - Returns error tokens from errors_catalog.go (e.g., ErrMajorVersionUnsupported)
//   - Transport layer resolves tokens via GetErrorDefinition() → sendError()
//   - All errors include spec section references for traceability
type HandshakeService interface {
	// ValidateConnect processes Connect message per SCACI §3.3
	//
	// This method handles:
	//   - Tenant resolution from client certificate (via injected org.Resolver)
	//   - Version negotiation (SCACI §2.1-2.3): Semantic version comparison
	//   - Session resumption (SCACI §3.3): UUID validation, opId consistency, tenant re-validation
	//   - Fresh session creation: Generate snScUuid, initialize opId counters
	//
	// Parameters:
	//   - ctx: Request context with timeout (typically 5s for connect operations)
	//   - req: Decoded Connect message from wire (MessagePack deserialized)
	//   - cert: Client certificate from TLS handshake (service resolves tenant from this)
	//
	// Returns:
	//   - *Session: Session object for handler to map to connection (includes resolved tenantID)
	//   - *ConnectResponse: Response to send on wire (includes negotiated version, snScUuid)
	//   - string: Error token from errors_catalog.go if validation fails, "" on success
	//
	// Spec References:
	//   - §3.3: Connect handshake requirements
	//   - §3.3-02: Connect MUST use opId == 0
	//   - §3.3.1-01: Mandatory fields (version, acEui, snAcUuid)
	//
	// Example Usage:
	//   session, resp, errToken := svc.ValidateConnect(ctx, &req, clientCert)
	//   if errToken != "" {
	//       return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errToken)
	//   }
	ValidateConnect(ctx context.Context, req *Connect, cert *x509.Certificate) (*Session, *ConnectResponse, string)

	// NegotiateVersion performs semantic version negotiation per SCACI §2.1-2.3
	//
	// Compares client version against server capabilities:
	//   - Major version MUST match (reject on mismatch)
	//   - Minor version downgraded to highest common
	//   - Patch version informational only
	//
	// Parameters:
	//   - clientVersion: Version string from Connect message (e.g., "1.0.0")
	//
	// Returns:
	//   - string: Negotiated version to include in ConnectResponse
	//   - string: Error token (errMajorVersionUnsupported, errInvalidVersionFormat) or "" on success
	//
	// Spec Reference: §2.1-2.3 version negotiation rules
	NegotiateVersion(ctx context.Context, clientVersion string) (negotiatedVersion string, errToken string)

	// ResolveResume validates session resumption per SCACI §3.3 and §§2.1-2.3
	//
	// Checks if an existing session can be resumed by validating:
	//   - snAcUuid matches stored session
	//   - Operation ID progression is valid (AC/SC opIds are consistent)
	//   - Session is in resumable state (not terminated)
	//   - Protocol version matches originally negotiated version (§2.1-2.3)
	//
	// Uses interfaces.SCACISessionRepository.CheckSessionResumable() for validation
	//
	// Parameters:
	//   - ctx: Request context
	//   - tenantID: Tenant scope for session lookup
	//   - acUUID: Application Center session UUID (snAcUuid from Connect)
	//   - scUUID: Service Center session UUID (snScUuid from Connect, optional for resume)
	//   - acOpId: Last AC operation ID (from snAcOpId field)
	//   - scOpId: Last SC operation ID (from snScOpId field)
	//   - requestVersion: Protocol version from Connect message (must match stored negotiated_version)
	//
	// Returns:
	//   - bool: True if session can be resumed
	//   - string: Error token if resume validation fails, "" on success
	//     - ErrVersionMismatchOnResume if requestVersion != stored negotiated_version
	//
	// Spec Reference: §3.3 session resumption flow, §§2.1-2.3 version consistency
	ResolveResume(ctx context.Context, tenantID int64, acUUID []byte, scUUID []byte, acOpId, scOpId int64, requestVersion string) (canResume bool, errToken string)
}

// EndpointService manages endpoint register/deregister per MIOTY §3.6-3.7
//
// This service handles Application Center-initiated endpoint lifecycle operations.
// It enforces the BSSCI requirement that ALL database writes must funnel
// through message_repository.go - no direct INSERT/UPDATE statements are permitted.
//
// Persistence Strategy:
//   - Register: Uses message_repository.CreateEndpointRecord() or UpdateEndpointRecord()
//   - Deregister: Marks endpoint inactive, triggers BSSCI detach propagation
//   - All writes are tenant-scoped to enforce isolation
//
// BSSCI Integration:
//   - Deregister calls DetachPropagator.SendDetachPropagateToAll()
//   - This ensures all connected base stations clear their local endpoint state
//
// Error Handling:
//   - Returns error tokens from errors_catalog.go (e.g., errInvalidNwkKeyLength)
//   - Transport layer resolves tokens before sending error response
type EndpointService interface {
	// Register creates or updates an endpoint per SCACI §3.6
	//
	// Validates:
	//   - EpEui is non-zero
	//   - NwkKey length == 16 bytes (MIOTY §3.6.1 requirement)
	//   - ShAddr, AttachCnt, PacketCnt fit in database types
	//
	// Persistence:
	//   - New endpoint: message_repository.CreateEndpointRecord()
	//   - Existing endpoint: message_repository.UpdateEndpointRecord()
	//   - NO direct INSERT/UPDATE - enforced by automation
	//
	// Parameters:
	//   - ctx: Request context
	//   - req: Decoded Register message from wire
	//   - tenantID: Tenant scope for endpoint ownership
	//
	// Returns:
	//   - string: Error token if registration fails, "" on success
	//
	// Spec References:
	//   - §3.6.1: Register message format
	//   - §3.6.2: Field validation requirements
	//
	// Example Usage:
	//   errToken := svc.Register(ctx, &req, tenantID)
	//   if errToken != "" {
	//       return s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errToken)
	//   }
	Register(ctx context.Context, req *Register, tenantID int64) string

	// Deregister removes an endpoint and triggers BSSCI detach propagation per SCACI §3.7
	//
	// Flow:
	//   1. Validate endpoint exists for tenant
	//   2. Mark endpoint as inactive in database
	//   3. Call DetachPropagator.SendDetachPropagateToAll(epEui)
	//   4. Base stations receive detachProp messages and clear local state
	//
	// Parameters:
	//   - ctx: Request context
	//   - epEui: Endpoint EUI to deregister
	//   - tenantID: Tenant scope for ownership validation
	//
	// Returns:
	//   - string: Error token (errEndpointNotFound, errDatabaseError) or "" on success
	//
	// Spec Reference: §3.7 deregister flow
	//
	// Example Usage:
	//   errToken := svc.Deregister(ctx, req.EpEui, tenantID)
	//   if errToken != "" {
	//       return s.sendErrorWithCatalog(conn, session, opId, POSIX_ENOENT, errToken)
	//   }
	Deregister(ctx context.Context, epEui uint64, tenantID int64) string

	// GetByEUI retrieves an endpoint by EUI for a specific tenant
	//
	// This method provides a service-layer wrapper around the repository's GetByEUI
	// for use by SCACI handlers that need to look up endpoint state.
	//
	// Parameters:
	//   - ctx: Request context
	//   - tenantID: Tenant scope for endpoint lookup
	//   - eui: Endpoint EUI (8-byte slice)
	//
	// Returns:
	//   - *models.EndPoint: Endpoint record if found, nil otherwise
	//   - string: Error token (errEndpointNotFound, errDatabaseError) or "" on success
	//
	// Example Usage:
	//   endpoint, errToken := svc.GetByEUI(ctx, tenantID, eui)
	//   if errToken != "" {
	//       return s.sendErrorWithCatalog(conn, session, opId, POSIX_ENOENT, errToken)
	//   }
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, string)

	// GetGlobal retrieves an endpoint by EUI without tenant filter (cross-tenant lookup)
	//
	// Used for roaming scenarios where endpoint's owner tenant may differ from
	// session tenant. Matches BSSCI pattern at server.go:5197-5202.
	//
	// Parameters:
	//   - ctx: Request context
	//   - eui: Endpoint EUI (8-byte slice)
	//
	// Returns:
	//   - *models.EndPoint: Endpoint record if found, nil otherwise
	//   - string: Error token (ErrEndpointNotFound, ErrDatabaseError) or "" on success
	//
	// Example Usage:
	//   endpoint, errToken := svc.GetGlobal(ctx, eui)
	//   if errToken == ErrEndpointNotFound {
	//       // Endpoint not found in any tenant
	//   }
	GetGlobal(ctx context.Context, eui []byte) (*models.EndPoint, string)

	// PropagateDetachToAll triggers BSSCI detach propagation to all base stations
	//
	// Encapsulates detachPropagator usage so handlers don't need direct access
	// to BSSCI server.
	//
	// Called by handleDeregisterComplete after AC confirms endpoint deregistration.
	// Per SCACI §3.7, detach propagation happens AFTER the three-way handshake
	// completes, not during the initial Deregister message.
	//
	// Parameters:
	//   - epEui: Endpoint EUI to detach from all base stations
	//
	// Returns:
	//   - []error: Slice of errors (one per failed base station), empty if all succeeded
	//
	// Example Usage:
	//   if errs := svc.PropagateDetachToAll(ctx, epEui); len(errs) > 0 {
	//       logger.Warn("Some base stations failed detach propagation", zap.Errors("errors", errs))
	//   }
	PropagateDetachToAll(ctx context.Context, epEui uint64) []error
}

// ULService schedules uplink transmissions per MIOTY §3.9
//
// This is a SCACI-facing adapter around scheduler.ULTransmitScheduler that:
//  1. Delegates to BSSCI ULTransmitScheduler interface (pkg/scheduler)
//  2. Translates scheduler errors → SCACI error tokens
//  3. Does NOT redefine scheduler logic (pure adapter pattern)
//
// Scheduler Integration:
//   - scheduler.ULTransmitScheduler returns Go errors (scheduler.ErrSchedulerNoResources, etc.)
//   - This service maps those errors to SCACI tokens (errSchedulerUnavailable, etc.)
//   - Transport layer then resolves tokens → POSIX codes for wire protocol
//
// Error Mapping:
//   - scheduler.ErrSchedulerNoResources  → errSchedulerUnavailable (POSIX_EAGAIN)
//   - scheduler.ErrSchedulerResourceMissing → errBaseStationNotFound (POSIX_ENOENT)
//   - Generic errors                      → errDatabaseError (POSIX_EIO)
//
// This adapter exists to maintain the error token pattern across all SCACI services
// without polluting the scheduler package (which is also used by BSSCI).
type ULService interface {
	// ScheduleULTransmit delegates to BSSCI ULTransmitScheduler per SCACI §3.9
	//
	// This method:
	//   1. Calls scheduler.ScheduleULData() on the BSSCI server
	//   2. Maps scheduler errors to SCACI error tokens
	//   3. Returns SCACI-compatible response (opId, bsEui, token)
	//
	// Parameters:
	//   - ctx: Request context
	//   - req: UL Data Transmit message (includes epEui, userData, timing constraints)
	//   - tenantID: Tenant scope for endpoint/BS lookup
	//
	// Returns:
	//   - opId: BSSCI operation ID assigned to this UL transmission
	//   - bsEui: Base station EUI selected for transmission
	//   - errToken: Error token if scheduling fails, "" on success
	//
	// Spec Reference: §3.9 UL Data Transmit operation
	//
	// Example Usage:
	//   opId, bsEui, errToken := svc.ScheduleULTransmit(ctx, &req, tenantID)
	//   if errToken != "" {
	//       return s.sendErrorWithCatalog(conn, session, scOpId, POSIX_EAGAIN, errToken)
	//   }
	ScheduleULTransmit(ctx context.Context, req *mioty.ULDataTransmit, tenantID int64) (opId int64, bsEui uint64, errToken string)
}

// DLService manages downlink queue per MIOTY §3.10-3.11
//
// This is a SCACI-facing adapter around scheduler.DownlinkScheduler that:
//  1. Delegates to BSSCI DownlinkScheduler interface
//  2. Translates scheduler errors → SCACI error tokens
//  3. Maintains queue ID consistency (SC-issued negative IDs)
//
// Queue ID Semantics:
//   - SC issues negative queue IDs (queId < 0)
//   - BS issues positive operation IDs (opId > 0)
//   - These are correlated but distinct per BSSCI §3.12
//
// This service ensures SCACI handlers don't need to understand BSSCI scheduling
// internals while maintaining proper error token propagation.
type DLService interface {
	// QueueDownlink delegates to BSSCI DownlinkScheduler per SCACI §3.10
	//
	// Flow:
	//   1. Call scheduler.EnqueueDownlink() on BSSCI server
	//   2. Scheduler selects bidirectional base station
	//   3. Returns SC-issued queue ID (negative) + BS EUI
	//
	// Parameters:
	//   - ctx: Request context
	//   - req: DL Data Queue message (includes epEui, userData, cntDepend, etc.)
	//   - tenantID: Tenant scope for endpoint/BS lookup
	//
	// Returns:
	//   - queId: SC-issued queue ID (negative per SCACI semantics)
	//   - bsEui: Base station EUI selected for downlink
	//   - errToken: Error token if queueing fails, "" on success
	//
	// Spec Reference: §3.10 DL Data Queue operation
	//
	// Example Usage:
	//   queId, bsEui, errToken := svc.QueueDownlink(ctx, &req, tenantID)
	//   if errToken != "" {
	//       return s.sendErrorWithCatalog(conn, session, scOpId, POSIX_EAGAIN, errToken)
	//   }
	QueueDownlink(ctx context.Context, req *mioty.DLDataQueue, tenantID int64) (queId uint64, bsEui uint64, errToken string)

	// RevokeDownlink removes queued downlink per SCACI §3.11
	//
	// Flow:
	//   1. Call scheduler.RevokeDownlink() with queId
	//   2. Scheduler removes from queue if still pending
	//   3. Returns BS EUI where downlink was queued
	//
	// Error Mapping:
	//   - scheduler.ErrSchedulerQueueNotFound → errDownlinkNotFound (POSIX_ENOENT)
	//   - scheduler.ErrSchedulerNoResources   → errSchedulerUnavailable (POSIX_EAGAIN)
	//
	// Parameters:
	//   - ctx: Request context
	//   - queId: SC-issued queue ID (from prior QueueDownlink call)
	//   - tenantID: Tenant scope for authorization
	//
	// Returns:
	//   - bsEui: Base station EUI where downlink was queued
	//   - errToken: Error token if revoke fails, "" on success
	//
	// Spec Reference: §3.11 DL Data Revoke operation
	//
	// Example Usage:
	//   bsEui, errToken := svc.RevokeDownlink(ctx, req.QueId, tenantID)
	//   if errToken != "" {
	//       return s.sendErrorWithCatalog(conn, session, scOpId, POSIX_ENOENT, errToken)
	//   }
	RevokeDownlink(ctx context.Context, queId uint64, tenantID int64) (bsEui uint64, errToken string)

	// Storage persistence methods for downlink queue database operations

	// GetDownlinkByQueueID retrieves a downlink by queue ID
	GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error)

	// EnqueueDownlink persists a downlink message to the queue
	EnqueueDownlink(ctx context.Context, dlMsg *storage.DownlinkMessage) (*storage.DownlinkMessage, error)

	// UpdateDownlinkStatus updates the status field of a downlink message.
	// orgID parameter scopes updates to a specific organization per §3.10.
	UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error

	// GetDownlinkByPacketCnt retrieves a downlink by endpoint and packet count
	GetDownlinkByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) (*storage.DownlinkMessage, error)

	// GetDownlinkQueue retrieves all pending/scheduled downlinks for an endpoint
	GetDownlinkQueue(ctx context.Context, deviceEUI string, tenantID string) ([]*storage.DownlinkMessage, error)

	// RevokeDownlinkByID revokes a downlink by database ID
	RevokeDownlinkByID(ctx context.Context, queId int64, tenantID string) error
}

// StatusService provides server status and monitoring per MIOTY §3.5
//
// This service encapsulates:
//   - Server uptime calculation (serviceStart timestamp)
//   - Base station statistics (active BS count)
//   - Resource availability reporting
//
// Architectural Role:
//   - Moves baseStationRepo and serviceStart from Server struct into service layer
//   - Allows handlers to query status without direct repository access
//   - Maintains single responsibility: status/monitoring only
//
// Usage:
//   - handleStatus calls GetUptime() and GetActiveBaseStationCount()
//   - No error tokens needed (status queries don't fail critically)
type StatusService interface {
	// GetUptime returns server uptime in seconds
	//
	// Calculates time elapsed since Service Center started.
	// Used by Status handler to populate uptimeSec field per SCACI §3.5.2
	//
	// Returns:
	//   - int64: Uptime in seconds
	GetUptime() int64

	// GetBaseStations returns all base stations for tenant
	//
	// Queries baseStationRepo.List() for all BSs in tenant scope.
	// Used by Status handler to populate baseStations array per SCACI §3.5.2
	//
	// Parameters:
	//   - ctx: Request context
	//   - tenantID: Tenant scope for BS filtering
	//
	// Returns:
	//   - []*models.BaseStation: Slice of base stations (may be empty)
	//   - error: Database error (nil on success)
	GetBaseStations(ctx context.Context, tenantID int64) ([]*models.BaseStation, error)

	// GetBaseStation retrieves a single base station by EUI
	//
	// Encapsulates baseStationRepo.GetByEUI so handlers don't need direct
	// access to the repository.
	//
	// Used by UL Data Transmit handler to validate BS ownership before scheduling.
	//
	// Parameters:
	//   - ctx: Request context
	//   - tenantID: Tenant scope for BS filtering
	//   - eui: Base station EUI (8-byte slice)
	//
	// Returns:
	//   - *models.BaseStation: Base station record if found, nil otherwise
	//   - error: Database error or storage.ErrNotFound
	GetBaseStation(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)

	// GetPreferredBaseStation returns the last-attached base station EUI for an endpoint
	//
	// SCACI §3.9.1 Production Readiness: Used by UL Data Transmit handler for
	// "Service Center preferred BS" selection when client does not specify bsEui.
	//
	// Parameters:
	//   - ctx: Request context
	//   - tenantID: Tenant scope for endpoint filtering
	//   - epEui: Endpoint EUI (8-byte slice)
	//
	// Returns:
	//   - *uint64: Preferred BS EUI if endpoint has last_attached_bs_eui set, nil otherwise
	//   - bool: true if preference found (non-NULL), false if endpoint not found or NULL
	//   - error: Database error (nil on success or not found)
	GetPreferredBaseStation(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error)
}

// SessionValidator validates SCACI Connect message fields (SCACI §3.3)
//
// This service provides field-level validation for Connect messages, ensuring
// SCACI protocol requirements before business logic proceeds.
//
// Spec References:
//   - §3.3-02: Connect MUST use opId == 0
//   - §3.3.1-01: Mandatory field validation (version, acEui, snAcUuid)
type SessionValidator interface {
	// ValidateConnectFields validates Connect message field requirements
	//
	// Checks:
	//   - opId must be 0 per SCACI §3.3-02
	//   - Required fields present (version, acEui, snAcUuid)
	//
	// Parameters:
	//   - req: Decoded Connect message from wire
	//
	// Returns:
	//   - string: Error token from errors_catalog.go if validation fails, "" on success
	//
	// Example Usage:
	//   if errToken := s.sessionValidator.ValidateConnectFields(&req); errToken != "" {
	//       return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errToken)
	//   }
	// Pure and stateless: performs no I/O and no logging, so it takes no context.
	ValidateConnectFields(req *Connect) string
}

// CertificateVerifier validates TLS client certificates for SCACI connections
//
// This service performs certificate validation beyond basic TLS handshake verification,
// ensuring certificates meet KiloCenter security requirements for multi-tenant isolation.
//
// Validations performed:
//   - Certificate expiry (NotBefore/NotAfter against current time)
//   - Extended key usage (must include ClientAuth for mutual TLS)
//   - Subject presence (basic validation that cert has valid subject)
//
// This service is called by HandshakeService.ValidateConnect() after TLS handshake
// completes but before tenant resolution, providing defense-in-depth security.
type CertificateVerifier interface {
	// VerifyCertificate validates client certificate security properties
	//
	// Checks performed:
	//   1. Certificate expiry: NotBefore <= now <= NotAfter
	//   2. Key usage: ExtKeyUsageClientAuth present
	//   3. Subject: Has valid subject (not empty)
	//
	// Parameters:
	//   - cert: Client certificate from TLS handshake (peer certificate)
	//
	// Returns:
	//   - string: Error token from errors_catalog.go if validation fails, "" on success
	//
	// Example Usage:
	//   if errToken := s.certVerifier.VerifyCertificate(ctx, cert); errToken != "" {
	//       return nil, nil, errToken
	//   }
	VerifyCertificate(ctx context.Context, cert *x509.Certificate) string
}

// OperationRecorder persists SCACI operations for resume safety (SCACI §3.4)
//
// This service wraps operation repository calls with a simplified API, hiding
// model construction details from handlers. All persistence goes through
// interfaces.SCACIOperationRepository to maintain dependency inversion.
//
// Direction Constants:
//   - Use models.OperationDirectionInbound for AC → SC requests
//   - Use models.OperationDirectionOutbound for SC → AC responses
//
// Spec Reference: §3.4 operation state tracking requirements
type OperationRecorder interface {
	// Record persists a SCACI operation (request or response)
	//
	// Handlers build the data map explicitly - no reflection or auto-conversion.
	// This keeps the recorder focused on persistence without data transformation.
	//
	// Parameters:
	//   - ctx: Request context (typically 5s timeout for operation recording)
	//   - session: Active session (provides SessionID, TenantID for scoping)
	//   - opId: Operation ID from message
	//   - command: Command string (e.g., "register", "deregister", "ulDataTx")
	//   - direction: models.OperationDirectionInbound or models.OperationDirectionOutbound
	//   - data: Pre-built map for RequestData/ResponseData field (handler constructs this)
	//
	// Returns:
	//   - error: Only on critical DB failures (operation tracking is best-effort)
	//
	// Example Usage (handleRegister):
	//   reqData := map[string]interface{}{"epEui": req.EpEui, "bidi": req.Bidi}
	//   err := s.operationRecorder.Record(ctx, session, opId, CmdRegister,
	//       models.OperationDirectionInbound, reqData)
	//   if err != nil {
	//       s.logger.Warn(LogSCACIRecordRegisterOpFailed, zap.Error(err))
	//   }
	Record(ctx context.Context, session *Session, opId int64,
		command string, direction models.OperationDirection,
		data map[string]interface{}) error
}

// ErrorRecorder handles SCACI error persistence and event creation per §3.14
//
// This service separates error handling concerns from the wire protocol layer,
// coordinating persistence, system events, and logging when errors occur.
// Implements dependency inversion to enable unit testing with mocks.
//
// Design:
//   - Wire protocol (sendError) stays in server.go
//   - Persistence logic moves to service layer via this interface
//   - System events emitted for monitoring/alerting
//
// Error Flow:
//  1. SC sends error → RecordOutboundError (persist + event) → sendError (wire)
//  2. AC sends error → RecordInboundError (persist + event) → sendErrorAck (wire)
//  3. AC sends errorAck → CompleteErrorHandshake (mark operation complete)
//
// Spec Reference: SCACI v1.0.0 §3.14 (Error Messages)
type ErrorRecorder interface {
	// RecordOutboundError records SC-originated error before wire transmission
	//
	// Called by sendErrorWithCatalog to persist error state and emit events
	// before the error message is sent on the wire.
	//
	// Flow:
	//   1. Resolve error token → POSIX code + message via catalog
	//   2. Update scaci_operation_log with state=failed, error columns
	//   3. Emit system event via SystemEventStore.RecordSCACIError
	//   4. Return resolved code/message for wire transmission
	//
	// Parameters:
	//   - ctx: Request context
	//   - session: Active SCACI session (provides ID, TenantID)
	//   - opId: Operation ID being failed
	//   - command: SCACI command that triggered the error
	//   - errorToken: Catalog token (e.g., "scaci.error.missing_ep_eui")
	//   - defaultCode: Fallback POSIX code if catalog entry has POSIXCode=0
	//   - contextDetail: Additional context for logging/persistence
	//
	// Returns:
	//   - posixCode: Resolved POSIX code (catalog if non-zero, else defaultCode)
	//   - message: Resolved message from catalog (for wire)
	//   - error: Only on critical failures (best-effort persistence)
	RecordOutboundError(ctx context.Context, session *Session, opId int64,
		command string, errorToken string, defaultCode int, contextDetail string) (posixCode int, message string, err error)

	// RecordInboundError records AC-originated error (CmdError received)
	//
	// Called by handleInboundError when AC sends an error message to SC.
	// Per §3.14, SC must acknowledge with errorAck after recording.
	//
	// Parameters:
	//   - ctx: Request context
	//   - session: Active SCACI session
	//   - opId: Operation ID from error message
	//   - posixCode: POSIX error code from error message
	//   - message: Error message text from error message
	//
	// Returns:
	//   - error: Only on critical failures
	RecordInboundError(ctx context.Context, session *Session, opId int64,
		posixCode int, message string) error

	// CompleteErrorHandshake marks operation complete when errorAck received
	//
	// Called by handleErrorAck when AC acknowledges SC-sent error.
	// Per §3.14.2, errorAck completes the error sequence.
	//
	// Parameters:
	//   - ctx: Request context
	//   - session: Active SCACI session
	//   - opId: Operation ID from errorAck message
	//
	// Returns:
	//   - error: Only on critical failures
	CompleteErrorHandshake(ctx context.Context, session *Session, opId int64) error
}

// SessionPersistence wraps existing interfaces.SCACISessionRepository for async session persistence
//
// This interface abstracts session persistence so handlers don't directly call
// sessionRepo.UpdateSession/CreateSession. It wraps the existing repository with async handling
// and proper error logging, following the established pattern from handler_connect.go.
//
// Implementation Notes:
//   - Uses already-injected interfaces.SCACISessionRepository (no new KC-DB exports)
//   - Performs async persistence in goroutine (matches current behavior)
//   - Logs errors but doesn't block handler response (best-effort persistence)
//
// Example Usage (handler_connect.go):
//
//	s.sessionPersistence.PersistResumeAsync(ctx, session, tlsVersion, cipherSuite)
//	// Handler continues without waiting for DB write
type SessionPersistence interface {
	// PersistResumeAsync updates the persisted row of a resumed session after
	// the Connect handshake: lastHeartbeat, status, TLS evidence, opId
	// counters, metadata. Fresh sessions are persisted synchronously via
	// PersistConnectSync - this path never creates rows and never mutates the
	// live session (the goroutine reads an immutable snapshot only).
	//
	// Parameters:
	//   - session: Resumed session (Resumed == true, ID > 0)
	//   - tlsVersion: TLS version negotiated (e.g., "TLS 1.2", "TLS 1.3") per SCACI §1
	//   - cipherSuite: TLS cipher suite name per SCACI §1
	//
	// Goroutine Behavior:
	//   - Spawns goroutine bounded by ConnectPersistTimeout, detached from the
	//     caller's cancellation
	//   - Logs errors but doesn't propagate to handler (best-effort persistence)
	PersistResumeAsync(ctx context.Context, session *Session, tlsVersion, cipherSuite string)

	// PersistConnectSync creates session synchronously, returning DB ID for operation logging.
	//
	// Used for fresh connects only - ensures session.ID is assigned BEFORE operation logging
	// so that Connect audit rows have real session IDs (SCACI §3.3-04 audit trail).
	//
	// Resumed sessions use PersistResumeAsync (they already have session.ID > 0).
	//
	// Parameters:
	//   - ctx: Request context with timeout (typically ConnectPersistTimeout)
	//   - session: Session object with all metadata (must have ID == 0 for fresh connect)
	//   - certFingerprint: SHA256 fingerprint of client certificate
	//   - certSubject: Certificate subject DN
	//   - remoteAddr: Client IP address from connection
	//   - tlsVersion: TLS version negotiated (e.g., "TLS 1.2", "TLS 1.3") per SCACI §1
	//   - cipherSuite: TLS cipher suite name per SCACI §1
	//   - negotiatedVersion: Protocol version from successful negotiation per SCACI §§2.1-2.3
	//
	// Returns:
	//   - int64: Database-assigned session ID (> 0 on success)
	//   - error: Database error if persistence fails
	//
	// Example Usage (handler_connect.go):
	//   if newSession.ID == 0 {
	//       id, err := s.sessionPersistence.PersistConnectSync(ctx, newSession, ...)
	//       if err != nil { return sendError(POSIX_EINVAL, ErrInternalError) }
	//       newSession.ID = id
	//   }
	//   // Now operationRecorder.Record works with real session ID
	PersistConnectSync(ctx context.Context, session *Session, certFingerprint, certSubject, remoteAddr, tlsVersion, cipherSuite, negotiatedVersion string) (int64, error)

	// PersistHeartbeatAsync updates session heartbeat timestamp in database
	//
	// Called by ping handlers (handlePing, handlePingResponse, handlePingComplete)
	// and SC-initiated ping (initiatePing) to persist keepalive activity.
	//
	// Behavior:
	//   - Spawns goroutine with ConnectPersistTimeout (non-blocking)
	//   - Uses context-aware logging for tenant/org tracing
	//   - Logs errors but doesn't fail the operation (best-effort)
	//   - Skips persistence if session == nil or session.ID == 0 (not yet persisted)
	//
	// Parameters:
	//   - ctx: Request context with session metadata (from s.sessionContext(session))
	//   - session: Active SCACI session with ID, TenantID populated
	//
	// Spec Reference: SCACI §3.4 ping operation keepalive
	PersistHeartbeatAsync(ctx context.Context, session *Session)
}

// NOTES ON MISSING INTERFACES:
//
// SessionRepository:
//   - NOT defined here - we REUSE interfaces.SCACISessionRepository (14 methods)
//   - Existing interface is already narrow and scoped to session operations
//   - Creating a duplicate interface would violate DRY principle
//   - Server struct uses: sessionRepo interfaces.SCACISessionRepository
//
// org.Resolver:
//   - Defined in pkg/org/resolver.go
//   - Injected into HandshakeService for certificate → tenant resolution
//   - Supports strict mode (fail-closed) and community mode (fallback to default tenant)
//
// DetachPropagator:
//   - Already defined in server.go (lines 67-76)
//   - Used by EndpointService implementation for BSSCI integration
//   - No changes needed - existing interface is correct

// SessionCounterStore persists the paired AC/SC operation ID counters
// (SCACI §3.2). Satisfied structurally by the SCACI session repository.
type SessionCounterStore interface {
	UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, acOpId, scOpId int64) error
}

// OperationStore owns the SCACI operation lifecycle rows used for the
// three-way handshake audit trail and resume replay (SCACI §3.2-§3.3).
// Satisfied structurally by the SCACI operation repository.
type OperationStore interface {
	RecordOperation(ctx context.Context, req *models.SCACIOperationRequest) (*models.SCACIOperation, error)
	UpdateOperationState(ctx context.Context, sessionID int64, opId int64, state models.OperationState, responseData map[string]interface{}) error
	GetOperationByOpID(ctx context.Context, sessionID int64, opId int64) (*models.SCACIOperation, error)
	GetPendingOperations(ctx context.Context, sessionID int64) ([]*models.SCACIOperation, error)
}

// OrganizationDirectory resolves the default organization for a tenant.
// Satisfied structurally by org.Resolver implementations.
type OrganizationDirectory interface {
	GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error)
}

// SessionSnapshotSource provides lightweight snapshots of the connected base
// station sessions for propagation fan-out. Satisfied by the BSSCI server.
type SessionSnapshotSource interface {
	ConnectedSessionsSnapshot() []propagation.BaseStationSession
}

// EndpointPropagator triggers attach propagation for an endpoint across the
// given sessions. Satisfied by the propagation service.
type EndpointPropagator interface {
	TriggerEndpointPropagate(ctx context.Context, endpointID int64, activeSessions []propagation.BaseStationSession) error
}

// ErrorOperationStore covers the failed-operation persistence the error
// recorder performs (SCACI §3.14). Satisfied structurally by the SCACI
// operation repository.
type ErrorOperationStore interface {
	UpdateOperationStateWithError(ctx context.Context, sessionID int64, opId int64,
		state models.OperationState, errorCode int, errorToken string, errorMessage string,
		responseData map[string]interface{}) error
	CompleteFailedOperation(ctx context.Context, sessionID int64, opId int64, responseData map[string]interface{}) error
}
