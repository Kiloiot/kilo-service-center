package bssci

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// ConnectState tracks the connect operation handshake per BSSCI §3.3/§5.17:
// con initiates the operation, conCmp completes it, and an error replaces the
// normal sequence with error followed by errorAck.
type ConnectState int

// Connect handshake states. A session is provisional until ConnectStateComplete;
// provisional sessions never enter the live-session maps or the resumable index.
const (
	// ConnectStateAwaitingConnect: no con received yet
	ConnectStateAwaitingConnect ConnectState = iota
	// ConnectStateAwaitingConnectComplete: conRsp sent, waiting for conCmp
	ConnectStateAwaitingConnectComplete
	// ConnectStateAwaitingConnectErrorAck: error sent, waiting for errorAck
	ConnectStateAwaitingConnectErrorAck
	// ConnectStateComplete: handshake finished, session active
	ConnectStateComplete
	// ConnectStateTerminal: handshake failed or connection closing
	ConnectStateTerminal
)

// ProtocolSessionState is the transport-free domain state of a Base Station
// session: identity, negotiated protocol parameters, resume/handshake state,
// and operation counters. Application services (connect, lifecycle, resume)
// operate on this state and never on the transport-bearing Session.
//
//revive:disable:var-naming BsOpId/ScOpId/LastBsOpId/LastScOpId use lowercase 'd' per MIOTY BSSCI §3.2
type ProtocolSessionState struct {
	ID                string
	BaseStationEUI    uint64
	ClientVersion     string // BS-provided version (raw client claim for audit)
	NegotiatedVersion string // SC canonical version (BSSCI §4-4.5)
	SessionUUID       []byte
	DbSessionID       int64 // Database session ID (BIGINT) for persistence
	// Session resume fields (BSSCI-3.3)
	BsUUID    []byte // Base Station UUID for session resume
	IsResumed bool   // True if this is a resumed session
	// DisconnectedAt is when the connection a resumed session continues was lost.
	DisconnectedAt *time.Time
	CanResume      bool            // True if session can be resumed (from DB)
	ConnectInfo    json.RawMessage // BSSCI §5.3 connect message data (from DB)
	// Handshake tracking (BSSCI-3.3-03)
	HandshakeComplete bool  // True when connect operation is fully completed
	LastBsOpId        int64 // Last seen BS operation ID for validation
	LastScOpId        int64 // Last issued SC operation ID
	// Message encoding (BSSCI Section 1)
	Encoding string // Message encoding: "json" or "msgpack" (negotiated on first message)
	// Organization resolution
	OrganizationID   uuid.UUID // Kilo Cloud org UUID (from TLS cert or community fallback)
	ResolvedTenantID int64     // Tenant ID resolved from cert/org (vs server default s.tenantID)
	// Connect handshake state machine (BSSCI §3.3/§5.17)
	ConnectState ConnectState
	// openBsOps holds the base station operations started and not yet
	// completed on this connection.
	openBsOps map[int64]struct{}
	// mu is the domain concurrency guard for counter/state mutation.
	mu sync.Mutex
}

// OperationCounters returns the newest base-station and service-center
// operation IDs of the session.
func (p *ProtocolSessionState) OperationCounters() (lastBsOpID, lastScOpID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.LastBsOpId, p.LastScOpId
}

// startBaseStationOperation admits a base-station operation initiation: a new
// operation needs an ID above every earlier one, an operation already open
// may be initiated again (rev1 §5.2 / classic §3.2).
func (p *ProtocolSessionState) startBaseStationOperation(opID int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, open := p.openBsOps[opID]; open {
		return true
	}
	if opID <= p.LastBsOpId {
		return false
	}
	p.LastBsOpId = opID
	p.markBaseStationOperationOpen(opID)
	return true
}

// completeBaseStationOperation closes an open base-station operation and
// reports whether it was open.
func (p *ProtocolSessionState) completeBaseStationOperation(opID int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, open := p.openBsOps[opID]; !open {
		return false
	}
	delete(p.openBsOps, opID)
	return true
}

// endBaseStationOperation closes the operation an error exchange replaced.
func (p *ProtocolSessionState) endBaseStationOperation(opID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.openBsOps, opID)
}

// restoreOpenBaseStationOperations reopens the base-station operations a
// resumed session may still complete or reissue with their original IDs.
func (p *ProtocolSessionState) restoreOpenBaseStationOperations(opIDs ...int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, opID := range opIDs {
		p.markBaseStationOperationOpen(opID)
	}
}

// markBaseStationOperationOpen records an open operation; callers hold mu.
func (p *ProtocolSessionState) markBaseStationOperationOpen(opID int64) {
	if p.openBsOps == nil {
		p.openBsOps = make(map[int64]struct{})
	}
	p.openBsOps[opID] = struct{}{}
}

// NextScOpID allocates the next Service Center operation ID for this session
// (negative, strictly decrementing per BSSCI rev1 §5.2 / classic §3.2). The
// consumed ID is never rolled back on a later failure: a rollback would race
// concurrent allocations and reissue an ID already held by an in-flight
// operation, so a failed operation simply leaves a harmless gap.
func (p *ProtocolSessionState) NextScOpID() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastScOpId--
	return p.LastScOpId
}

// errorAckDisposition classifies what the errorAck answering a sent error
// frame is allowed to do (BSSCI rev1 §5.17 / classic §3.17).
type errorAckDisposition int

const (
	// errorAckAckOnly: the errorAck merely closes the error exchange; it must
	// not touch any pending operation.
	errorAckAckOnly errorAckDisposition = iota
	// errorAckFinalizePendingOperation: the sent error replaced the normal
	// response/completion of a known pending SC operation, so the errorAck
	// completes that operation and its pending row is finalized.
	errorAckFinalizePendingOperation
)

// Session represents a connected Base Station session: the transport-free
// ProtocolSessionState plus the live transport resources (socket, certificate,
// background channels) that never cross an application-service boundary.
type Session struct {
	ProtocolSessionState
	Conn            net.Conn
	Connected       time.Time
	LastSeen        time.Time
	Vendor          string
	Model           string
	Name            string
	SoftwareVersion string
	Bidirectional   bool
	GeoLocation     []float64
	// MIOTY session fields
	ActiveVMTypes map[uint64][]uint8 // Track active Variable MAC types per endpoint
	stopStatus    chan struct{}      // Channel to stop status mechanism
	ClientCert    *x509.Certificate  // TLS client certificate for org resolution
	// pendingBaseStation caches the registration looked up during the connect
	// request so connect-complete does not repeat the lookup
	pendingBaseStation *basestation.BaseStation
	// pendingErrorAcks tracks the operation IDs for which this service center
	// has sent an error frame and awaits the base station's errorAck (BSSCI
	// rev1 §5.17 / classic §3.17). The exchange is connection-scoped and never
	// survives resume; connect handshake errors (opId 0 before the handshake
	// completed) are tracked by ConnectState instead. Guarded by the
	// ProtocolSessionState mutex.
	pendingErrorAcks map[int64]errorAckDisposition
	// certSubjectEUI is the base station EUI encoded in the TLS client
	// certificate CN (CE issuance scheme), enforced against the connect
	// bsEui in strict mode; nil for org-<UUID> certificates.
	certSubjectEUI *uint64
	// resume is the resume the conRsp offered, held on the provisional
	// connection until conCmp activates it or the base station refuses it.
	resume *resumeOffer
	// counterFlush coalesces the counter writes the session's frames request.
	counterFlush counterFlush
}

// resumeOffer is the previous session a resuming conRsp offered and its
// strictly decoded pending-operation snapshot, which conCmp activation
// restores and reissues (BSSCI rev1 §5.3.1 / classic §3.3.1).
type resumeOffer struct {
	previous   *Session
	pendingOps []*PendingOperation
}

// operations returns the snapshot to reissue; a new session has none.
func (o *resumeOffer) operations() []*PendingOperation {
	if o == nil {
		return nil
	}
	return o.pendingOps
}

// registerPendingErrorAck records that an error frame was sent for opId and a
// matching errorAck is now expected. A connect-stage error is left to
// ConnectState; after the handshake an error for opId 0 refuses an unexpected
// connect message and awaits its errorAck like any other.
func (s *Session) registerPendingErrorAck(opId int64, disposition errorAckDisposition) {
	if opId == 0 && !s.HandshakeComplete {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingErrorAcks == nil {
		s.pendingErrorAcks = make(map[int64]errorAckDisposition)
	}
	s.pendingErrorAcks[opId] = disposition
}

// consumePendingErrorAck removes and returns the awaited-errorAck entry for
// opId. ok is false when no error frame was sent for that operation on this
// connection, in which case the errorAck is unsolicited.
func (s *Session) consumePendingErrorAck(opId int64) (errorAckDisposition, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	disposition, ok := s.pendingErrorAcks[opId]
	if ok {
		delete(s.pendingErrorAcks, opId)
	}
	return disposition, ok
}

// BaseStationEUIBytes converts the BaseStationEUI uint64 to a byte slice.
// This is needed for roaming service calls that expect []byte parameters.
func (s *Session) BaseStationEUIBytes() []byte {
	euiBytes := mioty.EUI64(s.BaseStationEUI).ToBytes()
	return euiBytes[:]
}

// safeCtx returns the server context or context.Background() if nil (for test compatibility)
func (s *Server) safeCtx() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background() // context-root: lifecycle-fallback
}

// resolvedTenant returns the session's resolved tenant ID with defensive fallback.
// Use this helper to ensure consistent tenant resolution across all session operations.
// Ensures session.ResolvedTenantID is used instead of server default.
func resolvedTenant(session *Session, fallback int64) int64 {
	if session != nil && session.ResolvedTenantID > 0 {
		return session.ResolvedTenantID
	}
	return fallback
}

// sessionContext creates a context enriched with BSSCI session metadata for structured logging.
// The logger's extractContextFields will automatically inject tenant_id and organization_id.
func (s *Server) sessionContext(session *Session) context.Context {
	ctx := s.safeCtx()

	// Use session's resolved tenant (from cert or fallback to server default)
	ctx = pkgcontext.WithTenantID(ctx, resolvedTenant(session, s.tenantID))

	// Add organization ID if available (may be uuid.Nil in community mode)
	if session != nil && session.OrganizationID != uuid.Nil {
		ctx = pkgcontext.WithOrganizationID(ctx, session.OrganizationID)
	}

	// Note: UserID not available for BSSCI (machine-to-machine protocol)

	return ctx
}
