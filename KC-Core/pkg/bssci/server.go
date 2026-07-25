package bssci

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	// HeaderSize is the BSSCI message header size (8 bytes identifier + 4 bytes size)
	HeaderSize = 12
	// Note: Protocol identifier and version moved to KC-DB/storage/mioty/types.go
	// Use mioty.MIOTYFrameIdentifier and mioty.MIOTYProtocolVersion

	// Note: Protocol command constants moved to KC-DB/storage/mioty/types.go
	// Use mioty.CmdAttachPropagate, mioty.CmdDetachPropagate, etc.

	// Field names for message validation
	fieldNameNonce = "nonce"
)

// POSIX Error Codes for BSSCI protocol messages per MIOTY BSSCI v1.0.0 §4
//
// These error codes are sent in protocol error responses and match standard
// POSIX errno values for cross-platform compatibility.
//
//revive:disable:var-naming
const (
	// POSIX_OK indicates successful operation (no error)
	POSIX_OK = 0

	// POSIX_EPERM indicates operation not permitted
	POSIX_EPERM = 1

	// POSIX_ENOENT indicates no such file or directory
	POSIX_ENOENT = 2

	// POSIX_EIO indicates input/output error
	POSIX_EIO = 5

	// POSIX_EAGAIN indicates resource temporarily unavailable
	POSIX_EAGAIN = 11

	// POSIX_EACCES indicates permission denied
	POSIX_EACCES = 13

	// POSIX_EINVAL indicates invalid argument
	POSIX_EINVAL = 22

	// POSIX_ERANGE indicates result too large / numerical result out of range
	POSIX_ERANGE = 34

	// POSIX_ENOSYS indicates function not implemented
	POSIX_ENOSYS = 38

	// POSIX_EPROTO indicates protocol error
	POSIX_EPROTO = 71

	// POSIX_ENOTSUP indicates operation not supported
	POSIX_ENOTSUP = 95
)

//revive:enable:var-naming

// Server represents a BSSCI server
type Server struct {
	// Core infrastructure (keep)
	config   *Config
	logger   logger.Logger
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	handlers map[string]HandlerFunc
	tenantID int64
	// nextOpID removed - now per-session via session.LastScOpId (BSSCI §5.2)

	// Active sessions (keep for protocol handling)
	sessions map[string]*Session
	mu       sync.RWMutex

	// Injected services
	sessionSvc         SessionService
	versionNegotiator  VersionNegotiator
	downlinkSvc        DownlinkService
	statusSvc          StatusService
	connectionRegistry BaseStationConnectionRegistry
	queueSerializer    QueueSerializer // Downlink response frame builder
	auditLogger        AuditLogger     // Downlink audit event recorder
	tenantResolver     TenantResolver  // Queue-to-tenant mapping (replaces queueTenants map)

	// Organization resolution
	orgResolver OrganizationDirectory // Organization UUID → tenant ID resolution
	// Certificate identity enforcement (strict mode): resolver maps the TLS
	// client certificate to a tenant/org identity at accept; the directory
	// reads registered station identity for connect-time enforcement
	certIdentityResolver CertificateIdentityResolver
	bsDirectory          RegisteredBaseStationDirectory
	defaultTenantID      int64 // Community mode fallback tenant ID

	// Storage-boundary contracts (narrow, consumer-owned; satisfied
	// structurally by the KC-DB repositories)
	eventStore         EventStore
	basestationRepo    BaseStationStore
	endpointRepo       EndpointDirectory
	keyEncryptor       NetworkKeyProtector
	protocolMessages   ProtocolMessageStore
	dlrxStore          DLRXStatusStore
	bsStatusStore      BaseStationStatusStore
	downlinkQueueStore DownlinkQueueStore

	// Automatic propagation (BSSCI §5.8.3)
	propagationSvc propagation.Service

	// Multi-tenant roaming support
	roamingSvc RoamingService

	// Uplink disposition routing: Local, Relay (CE→ECE), or Drop for unknown endpoints
	dispositionResolver IngressDispositionResolver

	// Shared uplink ingest pipeline (dedup, tenant resolution, persistence, SCACI, MQTT)
	uplinkIngestSvc UplinkIngestService

	// Transactional attach / attach-propagate endpoint-session persistence
	attachPersistence EndpointAttachmentPersistence

	// runtimeConfigured records that ConfigureRuntime supplied the circular
	// dependencies; Start refuses to run without them.
	runtimeConfigured bool

	// started records that Start committed; late runtime reconfiguration and
	// double starts are rejected.
	started bool

	// stopped records that Stop ran; a stopped server cannot be restarted
	// and repeated Stop calls are no-ops.
	stopped bool

	// Relay outbox writer for CE mode: enqueues unknown-endpoint uplinks for federation relay
	relayOutbox RelayOutboxWriter

	// Unknown endpoint detach signature validation
	detachValidator DetachSignatureValidator

	// Downlink auto-dispatch on dlOpen=true (BSSCI §5.10.2)
	downlinkDispatcher DownlinkDispatcher

	// SCACI EPStatus forwarding on attach/detach completion (SCACI §3.13)
	scaciEPStatusBroadcaster SCACIEPStatusBroadcaster

	// MQTT event publishing for device lifecycle events
	mqttPublisher MQTTEventPublisher

	// Blueprint payload decoding (MIOTY Application Layer Specification)
	// Resolves blueprints and decodes payloads for uplink messages
	blueprintDecoder  BlueprintDecoder
	blueprintResolver BlueprintResolver
}

// Compile-time interface assertions
var (
	_ DownlinkCommander = (*Server)(nil)
	_ SessionDirectory  = (*Server)(nil)
	_ ULTransmitter     = (*Server)(nil)
	_ StatusRequester   = (*Server)(nil)
)

// Config holds server configuration
type Config struct {
	ListenAddr                       string
	TLSCert                          string
	TLSKey                           string
	TLSCACert                        string
	TLSMinVersion                    string // "TLS1.2" or "TLS1.3" - defaults to TLS1.3 per MIOTY spec
	ServiceCenterEUI                 uint64
	Vendor                           string
	Model                            string
	Name                             string
	SoftwareVersion                  string
	OrgEnforcementEnabled            bool   // Require valid X-Organization-ID in gRPC metadata
	MessageEncoding                  string // BSSCI Section 1: default message encoding (json/msgpack)
	DetachSignatureValidationEnabled bool   // BSSCI §5.7.1: Enable detach signature validation
	// OperationAckTimeout bounds how long the service center waits for the
	// next handshake message (conCmp or errorAck) after sending conRsp or a
	// connect-stage error (states AwaitingConnectComplete/AwaitingConnectErrorAck).
	OperationAckTimeout time.Duration
	// ConnectionEstablishmentTimeout bounds a freshly accepted connection
	// before the base station sends its con (state AwaitingConnect), so an
	// idle socket cannot hold resources indefinitely.
	ConnectionEstablishmentTimeout time.Duration
	// DuplicateWindow is the uplink deduplication window
	DuplicateWindow time.Duration
	// CertificatePollInterval is the certificate change poll interval
	CertificatePollInterval time.Duration
	// StatusRequestInterval is how often the SC polls a base station for status
	StatusRequestInterval time.Duration
	// StatusRequestInitialDelay delays the first status poll after connect
	StatusRequestInitialDelay time.Duration
	// DLRXQueryTimeout expires an unanswered dlRxStatQry after this duration
	DLRXQueryTimeout time.Duration
	// DLRXCleanupInterval is the dlRxStatQry expiry sweep cadence
	DLRXCleanupInterval time.Duration
	// DisableAttachPersistence is TEST-ONLY: skips DB persistence in attach handler.
	// MUST remain false in production. Used by tests to exercise replay protection without transaction stubs.
	DisableAttachPersistence bool
}

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
	BsUUID      []byte          // Base Station UUID for session resume
	BsOpId      int64           // Last known Base Station operation ID
	ScOpId      int64           // Last Service Center operation ID
	IsResumed   bool            // True if this is a resumed session
	CanResume   bool            // True if session can be resumed (from DB)
	ConnectInfo json.RawMessage // BSSCI §5.3 connect message data (from DB)
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
	// mu is the domain concurrency guard for counter/state mutation.
	mu sync.Mutex
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
	UserProvidedName string             // User-provided base station name
	ActiveVMTypes    map[uint64][]uint8 // Track active Variable MAC types per endpoint
	stopStatus       chan struct{}      // Channel to stop status mechanism
	ClientCert       *x509.Certificate  // TLS client certificate for org resolution
	// pendingBaseStation caches the registration looked up during the connect
	// request so connect-complete does not repeat the lookup
	pendingBaseStation *basestation.BaseStation
	// pendingErrorAcks tracks the nonzero operation IDs for which this service
	// center has sent an error frame and awaits the base station's errorAck
	// (BSSCI rev1 §5.17 / classic §3.17). The exchange is connection-scoped
	// and never survives resume; connect handshake errors (opId 0) are tracked
	// by ConnectState instead. Guarded by the ProtocolSessionState mutex.
	pendingErrorAcks map[int64]errorAckDisposition
	// certSubjectEUI is the base station EUI encoded in the TLS client
	// certificate CN (CE issuance scheme), enforced against the connect
	// bsEui in strict mode; nil for org-<UUID> certificates.
	certSubjectEUI *uint64
	// resumePendingOps is the strictly decoded pending-operation snapshot
	// loaded during a compatible resume, held on the provisional connection
	// until conCmp activation restores the cache and reissues the eligible
	// operations (BSSCI rev1 §5.3.1 / classic §3.3.1).
	resumePendingOps []*PendingOperation
}

// registerPendingErrorAck records that an error frame was sent for opId and a
// matching errorAck is now expected. opId 0 (connect) is never registered.
func (s *Session) registerPendingErrorAck(opId int64, disposition errorAckDisposition) {
	if opId == 0 {
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

// Message represents a BSSCI protocol message
type Message struct {
	Command    string      `json:"command" msgpack:"command"`
	OpId       int64       `json:"opId" msgpack:"opId"` //nolint:revive // BSSCI §2.5 requires lowercase 'd' (opId not opId)
	Data       interface{} `json:"-" msgpack:"-"`
	RawPayload []byte      `json:"-" msgpack:"-"` // Original wire bytes (MessagePack/JSON) for forensic analysis
}

// tryExtractFromMap attempts to extract command and opId from a map.
// Returns (command, opId, ok). If ok=false, caller should fall back to JSON round-trip.
func tryExtractFromMap(v map[string]interface{}) (string, int64, bool) {
	// Try "command" first, then "commandType"
	var command string
	if cmdVal, exists := v["command"]; exists {
		if cmd, ok := cmdVal.(string); ok {
			command = cmd
		} else {
			return "", 0, false // Wrong type, needs JSON fallback
		}
	} else if typeVal, exists := v["commandType"]; exists {
		if cmd, ok := typeVal.(string); ok {
			command = cmd
		} else {
			return "", 0, false
		}
	} else {
		return "", 0, false // Neither field present
	}

	// Extract opId via the canonical operation ID parsing (BSSCI §5.2)
	opIdVal, hasOpId := v["opId"]
	if !hasOpId {
		return "", 0, false
	}

	opId, ok := parseOpID(opIdVal)
	if !ok {
		return "", 0, false // Wrong type, needs JSON fallback
	}

	return command, opId, true
}

// outboundEnvelope exposes the wire envelope (command, opId) of typed BSSCI
// messages embedding mioty.BaseMessage without serialization round-trips.
type outboundEnvelope interface {
	EnvelopeCommand() string
	EnvelopeOpID() int64
}

// wrapOutboundMessage converts interface{} to *Message for consistent RawPayload capture.
// Map payloads keep their original values; typed structs (ConnectResponse, etc.)
// are retained as the original typed payload so uint64 fields such as scEui are
// encoded exactly (no JSON float64 projection).
func (s *Server) wrapOutboundMessage(msg interface{}) (*Message, error) {
	switch v := msg.(type) {
	case *Message:
		return v, nil
	case map[string]interface{}:
		command, opId, ok := tryExtractFromMap(v)
		if !ok {
			return nil, fmt.Errorf("map missing command/opId envelope")
		}
		// Ensure canonical envelope keys on the original map
		v["command"] = command
		v["opId"] = opId

		return &Message{
			Command: command,
			OpId:    opId,
			Data:    v,
		}, nil
	default:
		env, ok := msg.(outboundEnvelope)
		if !ok {
			return nil, fmt.Errorf("message type %T does not expose a BSSCI envelope", msg)
		}
		command := env.EnvelopeCommand()
		if command == "" {
			return nil, fmt.Errorf("message missing command/commandType field (type: %T)", msg)
		}

		return &Message{
			Command: command,
			OpId:    env.EnvelopeOpID(),
			Data:    msg,
		}, nil
	}
}

// outboundValidationProjection builds the map used for outbound field-catalog
// validation. Typed payloads are projected through MessagePack (uint64-exact);
// map payloads are validated directly. The projection is only inspected -
// encoding always uses the original payload.
func outboundValidationProjection(payload interface{}) (map[string]interface{}, error) {
	if m, ok := payload.(map[string]interface{}); ok {
		return m, nil
	}
	raw, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, &CatalogError{Token: errOutboundMarshalFailed, Posix: POSIX_EPROTO}
	}
	var projection map[string]interface{}
	if err := msgpack.Unmarshal(raw, &projection); err != nil {
		return nil, &CatalogError{Token: errOutboundMarshalFailed, Posix: POSIX_EPROTO}
	}
	return projection, nil
}

// HandlerFunc handles a specific command
type HandlerFunc func(s *Server, session *Session, msg *Message, data map[string]interface{}) error

// PendingOperation represents an operation that needs to be tracked for MIOTY session resume
type PendingOperation struct {
	SessionSlug   string                 // Session.ID for crash-safe resume (Issue #3: prevents cross-session operation ID collision)
	OperationID   int64                  // MIOTY operation ID
	OperationType string                 // Operation type (attPrp, detPrp, dlDataQueue, etc.)
	Message       map[string]interface{} // Complete operation message for reissue
	Endpoint      []byte                 // Endpoint EUI (if applicable)
	MACType       int                    // MAC type for VM operations
	Data          []byte                 // Data payload for operations
	Timestamp     time.Time              // Timestamp for operations
	Metadata      map[string]interface{} // Additional metadata
	CreatedAt     time.Time              // When operation was created
}

// SessionOpKey is a composite key for pending operations to prevent operation ID collision
// across concurrent base station sessions (BSSCI §§5.7-5.8.3).
// Example: BS_A and BS_B can both send detach opId=100 in parallel without conflict.
// Exported for use by StatusService to construct keys directly.
type SessionOpKey struct {
	SessionID   string // Session.ID for in-memory map lookups (ephemeral, not persisted)
	OperationID int64  // MIOTY operation ID (BS-issued positive or SC-issued negative)
}

// makeSessionOpKey creates a composite key for pending operations map access.
// Use this helper instead of raw opID to prevent cross-session collisions (Issue #3).
func makeSessionOpKey(session *Session, opID int64) SessionOpKey {
	return SessionOpKey{
		SessionID:   session.ID,
		OperationID: opID,
	}
}

// detachMetadata holds typed detach operation metadata for crash-safe persistence.
// Per BSSCI §5.7, prevents JSON round-trip type drift (float64/base64 issues).
// Use detachMetadataToMap() before persistPendingOperation, mapToDetachMetadata() on resume.
type detachMetadata struct {
	EpEui            uint64    // End Point EUI as uint64 (not float64 after JSON round-trip)
	EndpointID       int64     // Database endpoint.ID to avoid refetch in handleDetachComplete
	PacketCnt        uint32    // Packet counter as uint32
	Signature        []byte    // 4-byte signature (not base64-encoded string)
	RxTime           int64     // Reception timestamp (nanoseconds)
	SNR              float64   // Signal-to-noise ratio (dB)
	RSSI             float64   // Signal strength (dBm)
	EqSnr            *float64  // Optional AWGN equivalent SNR (dB)
	Profile          *string   // Optional MIOTY profile (e.g., "eu1")
	RxDuration       *int64    // Optional reception duration (ns)
	TenantID         int64     // Endpoint owner tenant ID for crash-safe tenant propagation
	OrgUUID          uuid.UUID // Endpoint owner organization UUID for roaming support
	ValidationStatus string    // Signature validation status: "validated", "unknown_endpoint", or "disabled" (Issue #4)
}

// safeCtx returns the server context or context.Background() if nil (for test compatibility)
func (s *Server) safeCtx() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
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
	ctx := context.Background()

	// Use session's resolved tenant (from cert or fallback to server default)
	ctx = pkgcontext.WithTenantID(ctx, resolvedTenant(session, s.tenantID))

	// Add organization ID if available (may be uuid.Nil in community mode)
	if session != nil && session.OrganizationID != uuid.Nil {
		ctx = pkgcontext.WithOrganizationID(ctx, session.OrganizationID)
	}

	// Note: UserID not available for BSSCI (machine-to-machine protocol)

	return ctx
}

// validateFiniteFloat validates a float64 value is finite and within range.
// Returns empty string on success, error token on validation failure.
func validateFiniteFloat(value, minVal, maxVal float64) string {
	if math.IsNaN(value) {
		return errInvalidFloatNaN
	}
	if math.IsInf(value, 0) {
		return errInvalidFloatInf
	}
	if value < minVal || value > maxVal {
		return errFloatOutOfRange
	}
	return ""
}

// resolveEndpointTenantID looks up the endpoint's owning tenant ID.
// Supports roaming by falling back to global lookup when tenant-scoped lookup fails.
func (s *Server) resolveEndpointTenantID(ctx context.Context, session *Session, epEui uint64) (int64, error) {
	sessionTenant := resolvedTenant(session, s.tenantID)

	// If endpointRepo not initialized (e.g., in tests without database), fall back to session tenant
	if s.endpointRepo == nil {
		return sessionTenant, nil
	}

	// Convert EUI to byte slice
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Convert to models.EUI type
	var eui models.EUI
	copy(eui[:], euiBytes)

	// Try tenant-scoped lookup first (session's resolved tenant)
	endpoint, err := s.endpointRepo.GetByEUI(ctx, sessionTenant, euiBytes)

	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// Tenant-scoped lookup failed - try global lookup for roaming endpoint
			endpoint, err = s.endpointRepo.Get(ctx, eui)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					s.logger.WarnContext(ctx, LogBSSCIEndpointNotFound,
						"ep_eui", pkgmioty.FormatEUI64(epEui),
						"session_tenant", sessionTenant)
					return 0, storage.ErrNotFound
				}
				// Database error on global lookup
				s.logger.ErrorContext(ctx, LogBSSCIFailedToResolveEndpointTenant,
					"ep_eui", pkgmioty.FormatEUI64(epEui),
					"error", err)
				return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToResolveEndpointTenant), err)
			}
			// Global lookup succeeded - endpoint is roaming
			s.logger.InfoContext(ctx, LogBSSCIResolvedRoamingEndpointTenant,
				"ep_eui", pkgmioty.FormatEUI64(epEui),
				"owner_tenant", endpoint.TenantID,
				"session_tenant", sessionTenant)
			return endpoint.TenantID, nil
		}
		// Database error on tenant-scoped lookup
		s.logger.ErrorContext(ctx, LogBSSCIFailedToResolveEndpointTenant,
			"ep_eui", pkgmioty.FormatEUI64(epEui),
			"error", err)
		return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToResolveEndpointTenant), err)
	}

	// Tenant-scoped lookup succeeded
	return endpoint.TenantID, nil
}

// validateOutboundMessage validates an outbound message complies with BSSCI field catalog.
// The map is a validation projection of the payload (see outboundValidationProjection);
// it is inspected only and never replaces the encoded payload.
// Returns nil on success, CatalogError on validation failure.
func (s *Server) validateOutboundMessage(session *Session, msgMap map[string]interface{}) error {
	// Check command field exists
	cmdVal, hasCommand := msgMap["command"]
	if !hasCommand {
		return &CatalogError{Token: errOutboundMissingCommand, Posix: POSIX_EPROTO}
	}

	command, ok := cmdVal.(string)
	if !ok {
		return &CatalogError{Token: errOutboundMissingCommand, Posix: POSIX_EPROTO}
	}

	// Validate opId presence and type (MIOTY requires opId on all outbound frames per §2.5.1)
	opIdVal, hasOpId := msgMap["opId"]
	if !hasOpId {
		return &CatalogError{Token: errOutboundMissingField, Posix: POSIX_EPROTO}
	}

	// Type check - accept numeric types (int64, float64, int, etc.) from JSON/msgpack unmarshaling
	// Accept any value including 0 (valid for conCmp), negative (SC-initiated), or positive (BS-initiated)
	switch opIdVal.(type) {
	case int64, float64, int, int32, uint32, uint64, int8, uint8:
		// Valid numeric types - accept any value
	default:
		return &CatalogError{Token: errOutboundInvalidFieldType, Posix: POSIX_EINVAL}
	}

	// Get allowed fields for this command
	allowedFields, known := mioty.AllowedOutboundFields(command)
	if !known {
		return &CatalogError{Token: errOutboundUnknownCommand, Posix: POSIX_EPROTO}
	}

	// Get mandatory fields for this command
	mandatoryFields, _ := mioty.MandatoryOutboundFields(command)

	// Build allowed field set - INCLUDE BASE FIELDS (command, opId)
	allowedSet := make(map[string]bool)
	allowedSet["command"] = true // Base field - always allowed
	allowedSet["opId"] = true    // Base field - always allowed
	for _, field := range allowedFields {
		allowedSet[field] = true
	}

	// Validate all fields in message are allowed
	ctx := s.sessionContext(session)
	for field := range msgMap {
		if !allowedSet[field] {
			s.logger.WarnContext(ctx,
				LogBSSCIOutboundDisallowedField,
				"command", command, "field", field)
			return &CatalogError{Token: errOutboundExtraField, Posix: POSIX_EPROTO}
		}
	}

	// Validate all mandatory fields are present
	for _, field := range mandatoryFields {
		if _, present := msgMap[field]; !present {
			s.logger.WarnContext(ctx,
				LogBSSCIOutboundMissingMandatoryFieldText,
				"command", command, "field", field)
			return &CatalogError{Token: errOutboundMissingMandatoryField, Posix: POSIX_EPROTO}
		}
	}

	return nil
}

// Dependencies carries every non-circular Server dependency for NewServer.
// StatusSvc is mandatory; feature-controlled collaborators (MQTT, blueprint
// decoding, detach validation, federation outbox, SCACI bridges) stay nil
// when their feature is off. The circular dependencies (propagation, downlink
// dispatcher) are supplied via ConfigureRuntime before Start.
type Dependencies struct {
	// Core protocol services
	SessionSvc         SessionService
	VersionNegotiator  VersionNegotiator
	DownlinkSvc        DownlinkService
	StatusSvc          StatusService
	ConnectionRegistry BaseStationConnectionRegistry
	QueueSerializer    QueueSerializer
	AuditLogger        AuditLogger
	TenantResolver     TenantResolver

	// Storage-boundary contracts
	EventStore        EventStore
	BaseStations      BaseStationStore
	Endpoints         EndpointDirectory
	AttachPersistence EndpointAttachmentPersistence
	OrgDirectory      OrganizationDirectory
	KeyProtector      NetworkKeyProtector

	// Certificate identity enforcement
	CertIdentityResolver CertificateIdentityResolver
	BaseStationDirectory RegisteredBaseStationDirectory

	// Ingest pipeline and routing
	UplinkIngest        UplinkIngestService
	RoamingSvc          RoamingService
	DispositionResolver IngressDispositionResolver
	RelayOutbox         RelayOutboxWriter

	// Feature-controlled collaborators
	DetachValidator          DetachSignatureValidator
	MQTTPublisher            MQTTEventPublisher
	BlueprintDecoder         BlueprintDecoder
	BlueprintResolver        BlueprintResolver
	SCACIEPStatusBroadcaster SCACIEPStatusBroadcaster

	// ProtocolMessages/DLRXStatus/BaseStationStatus/DownlinkQueue are the
	// narrow storage views; each is satisfied directly by the matching
	// KC-DB repository.
	ProtocolMessages  ProtocolMessageStore
	DLRXStatus        DLRXStatusStore
	BaseStationStatus BaseStationStatusStore
	DownlinkQueue     DownlinkQueueStore

	TenantID        int64
	DefaultTenantID int64
}

// RuntimeDependencies carries the collaborators that are constructed against
// the live *Server (circular) and therefore cannot be constructor arguments.
type RuntimeDependencies struct {
	Propagation        propagation.Service
	DownlinkDispatcher DownlinkDispatcher
}

// NewServer creates a BSSCI server from its dependency set; call
// ConfigureRuntime before Start to supply the circular collaborators.
func NewServer(cfg *Config, log logger.Logger, deps Dependencies) (*Server, error) {
	// StatusService is mandatory (single-writer architecture)
	if deps.StatusSvc == nil {
		return nil, fmt.Errorf("statusSvc is required for pending operation tracking")
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		config:   cfg,
		logger:   log,
		sessions: make(map[string]*Session),
		ctx:      ctx,
		cancel:   cancel,
		handlers: make(map[string]HandlerFunc),
		tenantID: deps.TenantID,

		eventStore:         deps.EventStore,
		basestationRepo:    deps.BaseStations,
		endpointRepo:       deps.Endpoints,
		keyEncryptor:       deps.KeyProtector,
		protocolMessages:   deps.ProtocolMessages,
		dlrxStore:          deps.DLRXStatus,
		bsStatusStore:      deps.BaseStationStatus,
		downlinkQueueStore: deps.DownlinkQueue,
		attachPersistence:  deps.AttachPersistence,

		sessionSvc:         deps.SessionSvc,
		versionNegotiator:  deps.VersionNegotiator,
		downlinkSvc:        deps.DownlinkSvc,
		statusSvc:          deps.StatusSvc,
		connectionRegistry: deps.ConnectionRegistry,
		queueSerializer:    deps.QueueSerializer,
		auditLogger:        deps.AuditLogger,
		tenantResolver:     deps.TenantResolver,

		orgResolver:     deps.OrgDirectory,
		defaultTenantID: deps.DefaultTenantID,

		certIdentityResolver:     deps.CertIdentityResolver,
		bsDirectory:              deps.BaseStationDirectory,
		uplinkIngestSvc:          deps.UplinkIngest,
		roamingSvc:               deps.RoamingSvc,
		dispositionResolver:      deps.DispositionResolver,
		relayOutbox:              deps.RelayOutbox,
		detachValidator:          deps.DetachValidator,
		mqttPublisher:            deps.MQTTPublisher,
		blueprintDecoder:         deps.BlueprintDecoder,
		blueprintResolver:        deps.BlueprintResolver,
		scaciEPStatusBroadcaster: deps.SCACIEPStatusBroadcaster,
	}

	// Register command handlers
	s.registerHandlers()

	return s, nil
}

// ConfigureRuntime supplies the circular dependencies once, before Start:
// the propagation service and the downlink dispatcher are constructed
// against the live *Server and injected back here.
func (s *Server) ConfigureRuntime(deps RuntimeDependencies) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return fmt.Errorf("runtime cannot be reconfigured after Start")
	}
	if s.runtimeConfigured {
		return fmt.Errorf("runtime already configured")
	}
	if deps.Propagation == nil {
		return fmt.Errorf("propagation service is required")
	}
	if deps.DownlinkDispatcher == nil {
		return fmt.Errorf("downlink dispatcher is required")
	}
	if s.config != nil && s.config.DetachSignatureValidationEnabled && s.detachValidator == nil {
		return fmt.Errorf("detach signature validation is enabled but no validator is wired")
	}
	s.propagationSvc = deps.Propagation
	s.downlinkDispatcher = deps.DownlinkDispatcher
	s.runtimeConfigured = true
	return nil
}

// validateRuntimeWiring rejects Start on an incompletely wired server: every
// dependency the composition root wires unconditionally must be present, so a
// wiring regression fails at startup instead of as a nil dereference under
// traffic. Feature-controlled collaborators (MQTT, key protection, detach
// validation, federation outbox) stay optional.
func (s *Server) validateRuntimeWiring() error {
	required := []struct {
		name    string
		missing bool
	}{
		{"session service", s.sessionSvc == nil},
		{"version negotiator", s.versionNegotiator == nil},
		{"downlink service", s.downlinkSvc == nil},
		{"status service", s.statusSvc == nil},
		{"connection registry", s.connectionRegistry == nil},
		{"queue serializer", s.queueSerializer == nil},
		{"audit logger", s.auditLogger == nil},
		{"tenant resolver", s.tenantResolver == nil},
		{"event store", s.eventStore == nil},
		{"base station store", s.basestationRepo == nil},
		{"endpoint directory", s.endpointRepo == nil},
		{"attach persistence", s.attachPersistence == nil},
		{"organization directory", s.orgResolver == nil},
		{"certificate identity resolver", s.certIdentityResolver == nil},
		{"base station directory", s.bsDirectory == nil},
		{"uplink ingest service", s.uplinkIngestSvc == nil},
		{"roaming service", s.roamingSvc == nil},
		{"disposition resolver", s.dispositionResolver == nil},
		{"blueprint decoder", s.blueprintDecoder == nil},
		{"blueprint resolver", s.blueprintResolver == nil},
		{"SCACI endpoint status broadcaster", s.scaciEPStatusBroadcaster == nil},
		{"protocol message store", s.protocolMessages == nil},
		{"DL RX status store", s.dlrxStore == nil},
		{"base station status store", s.bsStatusStore == nil},
		{"downlink queue store", s.downlinkQueueStore == nil},
	}
	for _, dep := range required {
		if dep.missing {
			return fmt.Errorf("%s is required", dep.name)
		}
	}
	return nil
}

// defaultOrgForSessionTenant resolves the default organization for the
// session's current tenant (community fallback path); uuid.Nil when
// unresolvable.
func (s *Server) defaultOrgForSessionTenant(ctx context.Context, session *Session) uuid.UUID {
	if s.orgResolver == nil {
		return uuid.Nil
	}
	orgID, err := s.orgResolver.GetDefaultOrgForTenant(ctx, session.ResolvedTenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToResolveDefaultOrgForBSSCISession,
			"error", err,
			"tenantID", session.ResolvedTenantID)
		return uuid.Nil
	}
	return orgID
}

// verifyCertificateFingerprint enforces the stored-certificate binding in
// strict mode: the presented client certificate's SHA-256 fingerprint must
// equal the registered station's stored fingerprint. Rows issued before
// fingerprints were stored are backfilled from the stored PEM after the
// presented certificate matched it (upgrade path); blank fingerprint with no
// stored certificate is rejected.
func (s *Server) verifyCertificateFingerprint(ctx context.Context, session *Session) error {
	registered, err := s.bsDirectory.GetGlobal(ctx, session.BaseStationEUI)
	if err != nil {
		return fmt.Errorf("registered station lookup: %w", err)
	}

	presented := crypto.CertFingerprintSHA256(session.ClientCert.Raw)

	stored := registered.TLSCertFingerprint
	if stored == "" {
		if registered.TLSCertificate == "" {
			return fmt.Errorf("station %016X has no stored certificate identity", session.BaseStationEUI)
		}
		derived, deriveErr := crypto.CertFingerprintFromPEM([]byte(registered.TLSCertificate))
		if deriveErr != nil {
			return fmt.Errorf("station %016X stored certificate unparsable: %w", session.BaseStationEUI, deriveErr)
		}
		if derived != presented {
			s.logger.WarnContext(ctx, LogBSSCICertFingerprintMismatch,
				"bsEui", session.BaseStationEUI)
			return fmt.Errorf("station %016X presented certificate does not match stored certificate", session.BaseStationEUI)
		}
		updated, backfillErr := s.bsDirectory.BackfillFingerprintIfBlank(ctx, registered.TenantID, registered.ID, derived)
		if backfillErr != nil {
			return fmt.Errorf("station %016X fingerprint backfill: %w", session.BaseStationEUI, backfillErr)
		}
		if !updated {
			// A concurrent writer set the fingerprint first: reload and compare
			reloaded, reloadErr := s.bsDirectory.GetGlobal(ctx, session.BaseStationEUI)
			if reloadErr != nil {
				return fmt.Errorf("registered station reload: %w", reloadErr)
			}
			if reloaded.TLSCertFingerprint != presented {
				s.logger.WarnContext(ctx, LogBSSCICertFingerprintMismatch,
					"bsEui", session.BaseStationEUI)
				return fmt.Errorf("station %016X presented certificate does not match registered fingerprint", session.BaseStationEUI)
			}
			return nil
		}
		s.logger.InfoContext(ctx, LogBSSCICertFingerprintBackfilled,
			"bsEui", session.BaseStationEUI)
		return nil
	}

	if stored != presented {
		s.logger.WarnContext(ctx, LogBSSCICertFingerprintMismatch,
			"bsEui", session.BaseStationEUI)
		return fmt.Errorf("station %016X presented certificate does not match registered fingerprint", session.BaseStationEUI)
	}
	return nil
}

// formatTenantID formats the server's tenant ID as a string for database operations
//
// This helper centralizes tenant ID formatting to ensure consistency across
// all event logging and database persistence operations.
//
// Returns:
//   - Tenant ID formatted as decimal string (e.g., "42")
func (s *Server) formatTenantID() string {
	return fmt.Sprintf("%d", s.tenantID)
}

// registerHandlers registers all command handlers
func (s *Server) registerHandlers() {
	s.handlers[mioty.CmdConnect] = s.handleConnect
	s.handlers[mioty.CmdConnectComplete] = s.handleConnectComplete
	s.handlers[mioty.CmdPing] = s.handlePing
	s.handlers[mioty.CmdPingResponse] = s.handlePingResponse
	s.handlers[mioty.CmdPingComplete] = s.handlePingComplete
	// Status response handlers (SC-initiated, so no status handler for BS-initiated)
	s.handlers[mioty.CmdStatusResponse] = s.handleStatusResponse
	s.handlers[mioty.CmdAttach] = s.handleAttach
	s.handlers[mioty.CmdAttachComplete] = s.handleAttachComplete
	s.handlers[mioty.CmdDetach] = s.handleDetach
	s.handlers[mioty.CmdDetachComplete] = s.handleDetachComplete
	s.handlers[mioty.CmdULData] = s.handleULData
	s.handlers[mioty.CmdULDataComplete] = s.handleULDataComplete
	// UL Data Transmit response handlers (SC-initiated, so no ulDataTx handler)
	s.handlers[mioty.CmdULDataTransmitResponse] = s.handleULDataTxResponse
	s.handlers[mioty.CmdError] = s.handleError
	s.handlers[mioty.CmdErrorAck] = s.handleErrorAck
	// Attach/Detach Propagate handlers
	s.handlers[mioty.CmdAttachPropagateResponse] = s.handleAttachPropagateResponse
	s.handlers[mioty.CmdDetachPropagateResponse] = s.handleDetachPropagateResponse
	// DL Data Result handlers (BSSCI §3.14)
	s.handlers[mioty.CmdDLDataResult] = s.handleDLDataResult
	s.handlers[mioty.CmdDLDataResultResponse] = s.handleDLDataResultResponse
	s.handlers[mioty.CmdDLDataResultComplete] = s.handleDLDataResultComplete
	// DL RX Status handlers (BSSCI §3.15)
	s.handlers[mioty.CmdDLRxStatus] = s.handleDLRXStatus
	s.handlers[mioty.CmdDLRxStatusResponse] = s.handleDLRXStatusResponse
	s.handlers[mioty.CmdDLRxStatusComplete] = s.handleDLRXStatusComplete
	// DL RX Status Query response handlers (BSSCI §3.16 - SC-initiated, so no dlRxStatQry handler)
	s.handlers[mioty.CmdDLRxStatusQueryResponse] = s.handleDLRXStatusQueryResponse
	// DL Data Revoke response handlers (BSSCI §3.13 - SC-initiated)
	s.handlers[mioty.CmdDLDataRevokeResponse] = s.handleDLDataRevokeResponse
	// DL Data Queue response handlers (BSSCI §3.12 - SC-initiated)
	s.handlers[mioty.CmdDLDataQueueResponse] = s.handleDLDataQueueResponse
	// VM handlers (BSSCI §4.1-4.3)
	s.handlers[mioty.CmdVMActivate] = s.handleVMActivate
	s.handlers[mioty.CmdVMActivateResponse] = s.handleVMActivateResponse
	s.handlers[mioty.CmdVMActivateComplete] = s.handleVMActivateComplete
	s.handlers[mioty.CmdVMDeactivate] = s.handleVMDeactivate
	s.handlers[mioty.CmdVMDeactivateResponse] = s.handleVMDeactivateResponse
	s.handlers[mioty.CmdVMDeactivateComplete] = s.handleVMDeactivateComplete
	s.handlers[mioty.CmdVMStatus] = s.handleVMStatus
	s.handlers[mioty.CmdVMStatusResponse] = s.handleVMStatusResponse
	s.handlers[mioty.CmdVMStatusComplete] = s.handleVMStatusComplete
	s.handlers[mioty.CmdVMDLData] = s.handleVMDLData
	s.handlers[mioty.CmdVMDLDataResponse] = s.handleVMDLDataResponse
	s.handlers[mioty.CmdVMDLDataComplete] = s.handleVMDLDataComplete
}

// certsExist checks whether all required TLS certificate files are present on disk.
func certsExist(certFile, keyFile, caFile string) bool {
	for _, f := range []string{certFile, keyFile, caFile} {
		if _, err := os.Stat(f); err != nil {
			return false
		}
	}
	return true
}

// Start starts the BSSCI server. If TLS certificates are not yet available
// (e.g., fresh deployment before certs are generated via UI), the listener
// is deferred and a background goroutine polls until the certificates appear.
// A validation failure leaves the server in its pre-Start state so the
// composition root can complete the wiring and try again; a successful Start
// is committed exactly once and cannot follow Stop.
func (s *Server) Start() error {
	// The composition root must supply the circular dependencies before the
	// server accepts traffic; an incompletely wired server refuses to start.
	s.mu.Lock()
	switch {
	case s.stopped:
		s.mu.Unlock()
		return fmt.Errorf("server already stopped: a stopped server cannot be restarted")
	case s.started:
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	case !s.runtimeConfigured:
		s.mu.Unlock()
		return fmt.Errorf("server runtime not configured: call ConfigureRuntime before Start")
	}
	if err := s.validateRuntimeWiring(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("server wiring incomplete: %w", err)
	}
	s.started = true
	s.mu.Unlock()

	if certsExist(s.config.TLSCert, s.config.TLSKey, s.config.TLSCACert) {
		if err := s.startTLSListener(); err != nil {
			return err
		}
		// Background work starts only once the listener is committed, so an
		// immediate TLS failure leaves nothing running.
		s.startDLRXQueryExpiryWorker()
		return nil
	}

	s.logger.WarnContext(s.safeCtx(), LogBSSCICertsNotFound,
		"cert", s.config.TLSCert,
		"key", s.config.TLSKey,
		"ca", s.config.TLSCACert)

	s.wg.Add(1)
	go s.waitForCertsAndStart()
	return nil
}

// waitForCertsAndStart polls for certificate files and starts the TLS listener
// once they become available (e.g., after the user generates them via the UI).
func (s *Server) waitForCertsAndStart() {
	defer s.wg.Done()

	ticker := time.NewTicker(certificatePollInterval(s.config))
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.InfoContext(s.safeCtx(), LogBSSCIDeferredListenerCancelled)
			return
		case <-ticker.C:
			if !certsExist(s.config.TLSCert, s.config.TLSKey, s.config.TLSCACert) {
				continue
			}
			s.logger.InfoContext(s.safeCtx(), LogBSSCICertsDetected)
			if err := s.startTLSListener(); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIDeferredListenerFailed, "error", err)
				continue
			}
			s.startDLRXQueryExpiryWorker()
			return
		}
	}
}

// startTLSListener loads certificates and starts the BSSCI TLS listener.
func (s *Server) startTLSListener() error {
	cert, err := tls.LoadX509KeyPair(s.config.TLSCert, s.config.TLSKey)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToLoadTLS), err)
	}

	caCert, err := os.ReadFile(s.config.TLSCACert)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToLoadCA), err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("%s", ResolveErrorMessage(errFailedToParseCA))
	}

	// TLS 1.2 minimum required for Fraunhofer AVA base station firmware compatibility.
	// TLS 1.3 preferred when client supports it.
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}

	listener, err := tls.Listen("tcp", s.config.ListenAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToStartTLS), err)
	}

	s.listener = listener
	s.logger.InfoContext(s.safeCtx(), LogBSSCIServerStarted, "address", s.config.ListenAddr)

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop stops the BSSCI server
func (s *Server) Stop() error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	s.mu.Unlock()

	s.cancel()
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToCloseListener, "error", err)
		}
	}
	s.wg.Wait()
	return nil
}

// acceptConnections accepts incoming connections
func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToAcceptConnection, "error", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToCloseConnection, "error", err)
		}
	}()

	s.logger.InfoContext(s.safeCtx(), LogBSSCINewConnection, "remote", conn.RemoteAddr().String())

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               uuid.New().String(),
			ResolvedTenantID: s.defaultTenantID, // Initialize to server default
			// Encoding left empty - detected on first frame per BSSCI Section 1
		},
		Conn:      conn,
		Connected: time.Now(),
		LastSeen:  time.Now(),
	}

	// Extract TLS client certificate and resolve organization
	if tlsConn, ok := conn.(*tls.Conn); ok {
		// Trigger TLS handshake if not already done
		if err := tlsConn.Handshake(); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCITLSHandshakeFailed, "error", err)
			return
		}

		state := tlsConn.ConnectionState()
		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			session.ClientCert = cert

			// Try to resolve organization from certificate
			ctx := context.Background()
			var orgID uuid.UUID
			var tenantID int64
			var err error

			switch {
			case s.certIdentityResolver != nil:
				identity, resolveErr := s.certIdentityResolver.ResolveCertificateIdentity(ctx, cert)
				if resolveErr != nil {
					// Strict mode: an unresolvable certificate closes the
					// connection before any con is read - no default-tenant
					// fallback (the fallback would let an unknown certificate
					// operate under the server's default tenant)
					if s.config != nil && s.config.OrgEnforcementEnabled {
						s.logger.ErrorContext(ctx, LogBSSCICertIdentityRejectedStrictMode,
							"error", resolveErr,
							"certCN", cert.Subject.CommonName)
						return
					}
					// Community fallback: default tenant + its default org
					s.logger.WarnContext(ctx, LogBSSCICertOrgResolutionFailedUsingCommunityFallback,
						"error", resolveErr,
						"certCN", cert.Subject.CommonName,
						"tenantID", session.ResolvedTenantID)
					orgID = s.defaultOrgForSessionTenant(ctx, session)
				} else {
					session.ResolvedTenantID = identity.TenantID
					session.certSubjectEUI = identity.SubjectEUI
					orgID = identity.OrganizationID
					s.logger.InfoContext(ctx, LogBSSCICertOrgResolutionSucceeded,
						"orgID", orgID.String(),
						"tenantID", identity.TenantID,
						"certCN", cert.Subject.CommonName)
				}
			case s.orgResolver != nil:
				orgID, tenantID, err = s.orgResolver.ResolveCert(ctx, cert)
				if err != nil {
					// Certificate-based resolution failed
					if s.config != nil && s.config.OrgEnforcementEnabled {
						s.logger.ErrorContext(ctx, LogBSSCICertIdentityRejectedStrictMode,
							"error", err,
							"certCN", cert.Subject.CommonName)
						return
					}
					// Community fallback: default tenant + its default org
					s.logger.WarnContext(ctx, LogBSSCICertOrgResolutionFailedUsingCommunityFallback,
						"error", err,
						"certCN", cert.Subject.CommonName,
						"tenantID", session.ResolvedTenantID)
					orgID = s.defaultOrgForSessionTenant(ctx, session)
				} else {
					// SUCCESS: cert resolution worked, use resolved tenant
					session.ResolvedTenantID = tenantID
					s.logger.InfoContext(ctx, LogBSSCICertOrgResolutionSucceeded,
						"orgID", orgID.String(),
						"tenantID", tenantID,
						"certCN", cert.Subject.CommonName)
				}
			default:
				// No resolver - community edition
				orgID = uuid.Nil
			}

			session.OrganizationID = orgID
			s.logger.DebugContext(ctx, LogBSSCISessionOrgTenantResolved,
				"orgID", orgID.String(),
				"tenantID", session.ResolvedTenantID,
				"certCN", cert.Subject.CommonName)
		} else {
			// No peer certificate provided
			// ResolvedTenantID already set to s.defaultTenantID in session init
			// Try to get default org for the server's default tenant
			ctx := context.Background()
			var orgID uuid.UUID
			if s.orgResolver != nil {
				var err error
				orgID, err = s.orgResolver.GetDefaultOrgForTenant(ctx, session.ResolvedTenantID)
				if err != nil {
					s.logger.WarnContext(ctx, LogBSSCINoPeerCertAndFailedToResolveDefaultOrgForBSSCISession,
						"error", err,
						"tenantID", session.ResolvedTenantID)
					orgID = uuid.Nil
				}
			}
			session.OrganizationID = orgID
			s.logger.DebugContext(ctx, LogBSSCISessionNoPeerCertUsingDefaults,
				"orgID", orgID.String(),
				"tenantID", session.ResolvedTenantID)
		}
	}

	// Ensure we update status to offline when connection ends
	defer func() {
		wasActive := session.ConnectState == ConnectStateComplete
		session.ConnectState = ConnectStateTerminal

		// Stop status mechanism safely
		session.mu.Lock()
		if session.stopStatus != nil {
			close(session.stopStatus)
			session.stopStatus = nil // Prevent double-close
		}
		session.mu.Unlock()

		ctx := s.sessionContext(session)

		// Session-map cleanup runs unconditionally so rejected or provisional
		// connections never linger in the live maps
		s.mu.Lock()
		delete(s.sessions, session.ID)
		s.mu.Unlock()

		// Also remove from SessionService's sessionsByUUID map to prevent stale resume
		if s.sessionSvc != nil {
			s.sessionSvc.RemoveSession(session)
		}

		// Offline status transition is guarded by connection identity: a
		// reconnect that already replaced this connection keeps the base
		// station online
		if session.BaseStationEUI != 0 && s.connectionRegistry != nil {
			euiBytes := mioty.EUI64(session.BaseStationEUI).ToBytes()

			if err := s.connectionRegistry.DisconnectBaseStationIfCurrent(ctx, euiBytes, session.ID); err != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateOfflineStatus,
					"eui", session.BaseStationEUI,
					"error", err)
			} else {
				s.logger.InfoContext(ctx, LogBSSCIBaseStationDisconnectedStatusOffline,
					"eui", session.BaseStationEUI,
					"name", session.Name)
			}
		}

		// A completed session lost unexpectedly stays resumable; anything
		// else that was persisted is terminated (BSSCI §3.3 lifecycle)
		if session.DbSessionID != 0 && s.sessionSvc != nil {
			if wasActive {
				if err := s.sessionSvc.MarkDisconnected(ctx, session); err != nil {
					s.logger.ErrorContext(ctx, LogBSSCIFailedToTerminateSession,
						"error", err,
						"sessionID", session.DbSessionID,
						"eui", session.BaseStationEUI)
				}
			} else if err := s.sessionSvc.TerminateSession(ctx, session); err != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToTerminateSession,
					"error", err,
					"sessionID", session.DbSessionID,
					"eui", session.BaseStationEUI)
			} else {
				s.logger.InfoContext(ctx, LogBSSCISessionTerminated,
					"sessionID", session.DbSessionID,
					"eui", session.BaseStationEUI)
			}
		}

		// Pending operations survive an unexpected loss of an active session:
		// they are reissued with their original opIds on resume (BSSCI §4/§5.3).
		// Only a terminal session (rejected, provisional, or terminated above)
		// has its rows removed. The cache is swept in every case - the runtime
		// session ID dies with this connection, so its entries are unreachable
		// and a resume re-hydrates from the persisted rows.
		if s.statusSvc != nil {
			if session.DbSessionID != 0 && !wasActive {
				count, err := s.statusSvc.DeletePendingOperations(ctx, session)
				if err != nil {
					s.logger.ErrorContext(ctx, LogBSSCIFailedToDeletePendingOperations,
						"error", err,
						"sessionID", session.DbSessionID)
				} else if count > 0 {
					s.logger.InfoContext(ctx, LogBSSCIDeletedPendingOperations,
						"sessionID", session.DbSessionID,
						"count", count)
				}
			}
			s.statusSvc.EvictCachedOperations(session)
		}
	}()

	// Read messages
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Read deadline follows the handshake state: a fresh connection is
		// bounded by the establishment timeout until con arrives; after conRsp
		// or a connect-stage error the ack timeout bounds the wait for conCmp
		// or errorAck; an active session reads without a deadline - liveness is
		// the ping operation's job (BSSCI §5.4)
		var deadline time.Time
		switch session.ConnectState {
		case ConnectStateAwaitingConnect:
			deadline = time.Now().Add(s.connectionEstablishmentTimeout())
		case ConnectStateAwaitingConnectComplete, ConnectStateAwaitingConnectErrorAck:
			deadline = time.Now().Add(s.operationAckTimeout())
		default:
			// Complete/Terminal: no read deadline
		}
		if err := conn.SetReadDeadline(deadline); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSetReadDeadline, "error", err)
			return
		}

		// Read header
		header := make([]byte, HeaderSize)
		if _, err := io.ReadFull(conn, header); err != nil {
			if err != io.EOF {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToReadHeader, "error", err)
			}
			return
		}

		// Verify protocol identifier
		if !bytes.Equal(header[:8], mioty.MIOTYFrameIdentifier[:]) {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIInvalidProtocolIdentifier, "got", string(header[:8]))
			return
		}

		// Get payload size
		payloadSize := binary.LittleEndian.Uint32(header[8:])
		if payloadSize > 1024*1024 { // 1MB max
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIPayloadTooLarge, "size", payloadSize)
			return
		}

		// Read payload
		payload := make([]byte, payloadSize)
		if _, err := io.ReadFull(conn, payload); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToReadPayload, "error", err)
			return
		}

		// Detect and persist encoding on first message (BSSCI Section 1)
		if session.Encoding == "" {
			encoding := detectEncoding(payload, s.config.MessageEncoding)
			session.Encoding = encoding

			// Persist encoding to database if session has been persisted
			if session.DbSessionID > 0 {
				if err := s.sessionSvc.UpdateEncoding(s.safeCtx(), resolvedTenant(session, s.tenantID), session.DbSessionID, encoding); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistEncoding,
						"encoding", encoding,
						"sessionID", session.DbSessionID,
						"error", err)
					// Non-fatal: continue with in-memory encoding
				}
			}

			s.logger.InfoContext(s.safeCtx(), LogBSSCIDetectedMessageEncoding,
				"encoding", encoding,
				"sessionID", session.DbSessionID)
		}

		// Decode message using detected/persisted encoding
		rawMsg, err := decodeMessage(payload, session.Encoding)
		if err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToDecodeMessagePack,
				"encoding", session.Encoding,
				"error", err)
			return
		}

		// Extract core fields
		command, ok := rawMsg["command"].(string)
		if !ok {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIMissingCommandField)
			return
		}

		// Debug: Log all message fields to understand what basestation is sending
		s.logger.DebugContext(s.safeCtx(), LogBSSCIReceivedMessageDebug,
			"rawMessage", rawMsg,
			"command", command)

		// Extract opId - MIOTY spec requires connect operation to use ID 0
		var opId int64
		if opIdValue, exists := rawMsg["opId"]; exists {
			if val, ok := parseOpID(opIdValue); ok {
				opId = val
			} else {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIInvalidOpIDType, "value", opIdValue)
				catalogErr := NewCatalogError(errInvalidOperationID, POSIX_EPROTO)
				_ = s.sendCatalogError(session, 0, catalogErr)
				return
			}
		} else {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIOpIDFieldNotFound, "message", rawMsg)
			return
		}

		msg := &Message{
			Command:    command,
			OpId:       opId,
			Data:       rawMsg,
			RawPayload: payload, // Capture original wire bytes for forensic analysis
		}

		// BSSCI-3.2: Validate operation ID. Connect is special (always opId
		// 0), and error/errorAck are exempt: they carry the opId of the
		// operation whose sequence they replace (rev1 §5.17 / classic §3.17),
		// in either direction, so they never start a new sequence position.
		if command != mioty.CmdConnect && command != mioty.CmdError && command != mioty.CmdErrorAck {
			// Determine if this is a base station initiated operation
			// Base station operations: positive IDs (and opId=0 for connect handshake per BSSCI §5.3)
			// Service center operations: negative IDs
			isBaseStationInitiated := opId >= 0

			if errToken := s.validateOperationID(session, opId, isBaseStationInitiated); errToken != "" {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIOperationIDValidationFailed,
					"command", command,
					"opId", opId,
					"error_token", errToken)
				// During the connect handshake the rejection enters the
				// unified error/errorAck sequence (rev1 §5.17 / classic
				// §3.17): the acknowledgement completes the failed exchange
				// and closes. An active session with a broken operation-ID
				// sequence closes immediately so resume restores counter sync.
				if !session.HandshakeComplete {
					if err := s.rejectConnect(session, opId, POSIX_EPROTO, errToken); err != nil {
						return
					}
					continue
				}
				if err := s.sendError(session, opId, POSIX_EPROTO, ResolveErrorMessage(errToken)); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, "error", err)
				}
				return
			}
		}

		// Update last seen
		session.LastSeen = time.Now()

		// Update last seen in database for connected basestations
		if session.BaseStationEUI != 0 && s.connectionRegistry != nil {
			euiBytes := mioty.EUI64(session.BaseStationEUI).ToBytes()

			// Update last seen time in database
			go func() {
				ctx := s.sessionContext(session)
				if err := s.connectionRegistry.UpdateLastSeen(ctx, euiBytes); err != nil {
					s.logger.DebugContext(ctx, LogBSSCIFailedToUpdateLastSeen,
						"eui", session.BaseStationEUI,
						"error", err)
				}
			}()
		}

		// Update session counters in database if we have a database session
		if session.DbSessionID > 0 {
			go s.updateSessionCounters(session)
		}

		// Handle message
		if err := s.handleMessage(session, msg, rawMsg); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToHandleMessage,
				"command", command,
				"error", err)
			return
		}
	}
}

// shouldNormalizeCommand reports whether an incoming command's payload must be
// normalized: only BS->SC and bidirectional inbound commands are, so the SC
// never validates its own outbound responses, and unknown commands are skipped
// as a safe default for forward compatibility (BSSCI §2.4).
func shouldNormalizeCommand(command string) bool {
	direction, exists := CommandDirectionMap[command]
	return exists && (direction == DirectionBStoSC || direction == DirectionBidirectional)
}

// handleMessage routes messages to appropriate handlers
func (s *Server) handleMessage(session *Session, msg *Message, data map[string]interface{}) error {
	// BSSCI §2.4: Normalize incoming payload to validate fields and detect unknown fields
	// Forward compatibility per §2.4-01
	// Issue #3-4 Fix: Only normalize BS→SC commands (inbound) to avoid validating our own SC→BS responses
	ctx := s.sessionContext(session)

	// Only normalize inbound BS→SC and bidirectional commands (skip SC→BS responses and unknown commands for safety)
	if shouldNormalizeCommand(msg.Command) {
		normalizedData, err := normalizePayload(ctx, s.logger, msg.Command, data)
		if err != nil {
			// Normalization failed (mandatory field missing, invalid type, etc.)
			// Send error response per BSSCI protocol
			s.logger.ErrorContext(ctx, LogBSSCIPayloadNormalizationFailed,
				"command", msg.Command,
				"opId", msg.OpId,
				"error", err)

			// BSSCI §2.4: All normalization errors are protocol violations (EPROTO)
			// Per spec: "Missing mandatory fields or present optional fields with invalid values
			// must be considered a protocol error"
			errToken := errInvalidMessageFormat
			switch {
			case errors.Is(err, ErrMandatoryFieldMissing):
				errToken = errMandatoryFieldMissing
			case errors.Is(err, ErrInvalidFieldType):
				errToken = errInvalidFieldType
			case errors.Is(err, ErrResponseExpRequiresDlOpen):
				errToken = errResponseExpRequiresDlOpen
			case errors.Is(err, ErrConditionalRuleFailed):
				errToken = errConditionalRuleFailed
			}

			// During the connect handshake a normalization failure enters the
			// unified error sequence: the error replaces conRsp/conCmp and
			// awaits errorAck (§5.17), and a failed error write terminates the
			// connection. After activation it is a per-operation error.
			if !session.HandshakeComplete {
				return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errToken)
			}
			if sendErr := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken)); sendErr != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToSendErrorResponse, "error", sendErr)
			}
			return nil // Error sent via protocol, don't close connection
		}
		// Use normalized data for handler (unknown fields removed, types coerced)
		data = normalizedData
	}

	// BSSCI-3.3-03: No operations allowed until connect handshake is complete
	// Only con, conRsp, conCmp, error, and errorAck are permitted during handshake
	if !session.HandshakeComplete {
		// conRsp is SC-initiated (SCtoBS): the base station must never send it.
		// Only con, conCmp, error, and errorAck are legal inbound during the
		// handshake; anything else - including an inbound conRsp - is a
		// protocol-ordering violation.
		allowedCommands := map[string]bool{
			mioty.CmdConnect:         true,
			mioty.CmdConnectComplete: true,
			mioty.CmdError:           true,
			mioty.CmdErrorAck:        true,
		}

		if !allowedCommands[msg.Command] {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingCommandBeforeHandshake,
				"command", msg.Command,
				"connectState", int(session.ConnectState),
				"opId", msg.OpId,
				"bsEui", session.BaseStationEUI)
			// Route through the unified reject so the error is followed by an
			// errorAck exchange (§5.17) and a failed write terminates cleanly
			return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errCommandBeforeHandshake)
		}
	}

	// Check for sublayer prefixes (BSSCI-4-02)
	// Allow both rc.* (remote control) and vm.* (virtual machine) sublayers
	if strings.Contains(msg.Command, ".") {
		// Only allow known sublayer prefixes
		if !strings.HasPrefix(msg.Command, "rc.") && !strings.HasPrefix(msg.Command, "vm.") {
			// Send error for unsupported sublayer
			if err := s.sendError(session, msg.OpId, POSIX_ENOTSUP, ResolveErrorMessage(errUnsupportedSublayerPrefix)); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, "error", err)
			}
			return nil // Don't close connection
		}
	}

	// Direction enforcement (BSSCI §5.5/§5.11-5.16): a command the service
	// center itself sends (DirectionSCtoBS) must never be processed inbound.
	// A base station sending e.g. statusCmp or dlDataQueCmp is a protocol
	// violation. The vm.* sublayer is exempt here - its enforcement lands with
	// the ECE-reserved VM work. Connect-stage SCtoBS handling (conRsp) is
	// already covered by the pre-handshake gate above.
	if !strings.HasPrefix(msg.Command, "vm.") {
		if cmdDirection, known := CommandDirectionMap[msg.Command]; known && cmdDirection == DirectionSCtoBS {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingInboundServiceCenterCommand,
				"command", msg.Command,
				"opId", msg.OpId,
				"bsEui", session.BaseStationEUI)
			if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInboundServiceCenterCommand)); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, "error", err)
			}
			return nil // Don't close connection
		}
	}

	handler, ok := s.handlers[msg.Command]
	if !ok {
		// Send BSSCI error message instead of closing socket (BSSCI-4-01)
		if err := s.sendError(session, msg.OpId, POSIX_ENOTSUP, ResolveErrorMessage(errUnsupportedCommand)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, "error", err)
		}
		return nil // Don't close connection
	}

	return handler(s, session, msg, data)
}

// sendMessage sends a message to the Base Station
func (s *Server) sendMessage(session *Session, msg interface{}) error {
	// Wrap to *Message for consistent RawPayload capture
	outMsg, err := s.wrapOutboundMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to wrap outbound message: %w", err)
	}

	// BSSCI §2.5.1: Validate outbound message fields before encoding.
	// Validation inspects a projection only; failures are returned to the
	// caller - an invalid outbound frame must never trigger sending a protocol
	// error in its place.
	projection, err := outboundValidationProjection(outMsg.Data)
	if err != nil {
		return fmt.Errorf("outbound validation failed: %w", err)
	}
	if err := s.validateOutboundMessage(session, projection); err != nil {
		var catalogErr *CatalogError
		if errors.As(err, &catalogErr) {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIOutboundValidationFailed,
				"error_token", catalogErr.Token,
				"command", outMsg.Command,
				"bs_eui", session.BaseStationEUI)
		}
		return fmt.Errorf("outbound validation failed: %w", err)
	}

	// Encode the original payload (typed or map) using the negotiated encoding
	// (BSSCI Section 1); uint64 fields survive exactly
	payload, err := encodeMessage(outMsg.Data, session.Encoding)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToEncode), err)
	}

	// Capture encoded payload for forensic analysis
	outMsg.RawPayload = payload

	// Create header
	header := make([]byte, HeaderSize)
	copy(header[:8], mioty.MIOTYFrameIdentifier[:])
	// Range guard for int -> uint32 conversion
	if len(payload) > math.MaxUint32 {
		return fmt.Errorf("%s: payload size %d exceeds maximum", ResolveErrorMessage(errPayloadTooLarge), len(payload))
	}
	//nolint:gosec // G115: int->uint32 safe, guarded by range check at line 664
	binary.LittleEndian.PutUint32(header[8:], uint32(len(payload)))

	// Send header and payload. Any failure past this point is an ambiguous
	// write: part of the frame may already be on the wire, so the connection's
	// framing can no longer be trusted and callers must close it and rely on
	// session resume for recovery (BSSCI rev1 §5.3.1 / classic §3.3.1).
	session.mu.Lock()
	defer session.mu.Unlock()

	n, err := session.Conn.Write(header)
	if err != nil {
		return fmt.Errorf("%s: %w: %w", ResolveErrorMessage(errFailedToWriteHeader), ErrAmbiguousWrite, err)
	}
	if n < len(header) {
		return fmt.Errorf("%s: %w: short write %d/%d", ResolveErrorMessage(errFailedToWriteHeader), ErrAmbiguousWrite, n, len(header))
	}

	n, err = session.Conn.Write(payload)
	if err != nil {
		return fmt.Errorf("%s: %w: %w", ResolveErrorMessage(errFailedToWritePayload), ErrAmbiguousWrite, err)
	}
	if n < len(payload) {
		return fmt.Errorf("%s: %w: short write %d/%d", ResolveErrorMessage(errFailedToWritePayload), ErrAmbiguousWrite, n, len(payload))
	}

	return nil
}

// ErrAmbiguousWrite reports a frame write that failed after bytes may have
// reached the wire. The transport framing can no longer be trusted; callers
// close the connection and rely on session resume (with the original
// operation IDs) for recovery instead of retrying on the same connection.
var ErrAmbiguousWrite = errors.New("ambiguous frame write")

// errAttachPersistenceUnavailable reports a missing endpoint attachment
// persister at a point that requires transactional persistence.
var errAttachPersistenceUnavailable = errors.New("endpoint attachment persistence not configured")

// closeTransportAfterWriteFailure closes the session transport after an
// ambiguous frame write. Persisted pending operations are deliberately
// preserved: the disconnect path marks the session resumable and resume
// reissues the operations with their original IDs.
func (s *Server) closeTransportAfterWriteFailure(session *Session, opId int64, cause error) {
	s.logger.ErrorContext(s.sessionContext(session), LogBSSCIClosingConnectionAfterWriteFailure,
		"bsEui", session.BaseStationEUI,
		"opId", opId,
		"error", cause)
	if session.Conn != nil {
		_ = session.Conn.Close()
	}
}

// operationAckTimeout returns the configured handshake wait bound, falling
// back to the package default when unset.
func (s *Server) operationAckTimeout() time.Duration {
	if s.config != nil && s.config.OperationAckTimeout > 0 {
		return s.config.OperationAckTimeout
	}
	return defaultOperationAckTimeout
}

// connectionEstablishmentTimeout returns the configured bound for a freshly
// accepted connection to send its con, falling back to the package default.
func (s *Server) connectionEstablishmentTimeout() time.Duration {
	if s.config != nil && s.config.ConnectionEstablishmentTimeout > 0 {
		return s.config.ConnectionEstablishmentTimeout
	}
	return defaultConnectionEstablishmentTimeout
}

// EffectiveDuplicateWindow returns the configured deduplication window,
// falling back to the package default; the composition root uses it to
// construct the shared message deduplicator.
func (c *Config) EffectiveDuplicateWindow() time.Duration {
	return duplicateWindow(c)
}

// duplicateWindow returns the configured uplink deduplication window, falling
// back to the package default when unset.
func duplicateWindow(cfg *Config) time.Duration {
	if cfg != nil && cfg.DuplicateWindow > 0 {
		return cfg.DuplicateWindow
	}
	return defaultDuplicateWindow
}

// certificatePollInterval returns the configured certificate change poll
// interval, falling back to the package default when unset.
func certificatePollInterval(cfg *Config) time.Duration {
	if cfg != nil && cfg.CertificatePollInterval > 0 {
		return cfg.CertificatePollInterval
	}
	return defaultCertificatePollInterval
}

// statusRequestInterval returns the configured status poll interval, falling
// back to the package default when unset.
func (s *Server) statusRequestInterval() time.Duration {
	if s.config != nil && s.config.StatusRequestInterval > 0 {
		return s.config.StatusRequestInterval
	}
	return defaultStatusRequestInterval
}

// statusRequestInitialDelay returns the configured delay before the first
// status poll, falling back to the package default when unset.
func (s *Server) statusRequestInitialDelay() time.Duration {
	if s.config != nil && s.config.StatusRequestInitialDelay > 0 {
		return s.config.StatusRequestInitialDelay
	}
	return defaultStatusRequestInitialDelay
}

// dlrxQueryTimeout returns the configured dlRxStatQry expiry, falling back to
// the package default when unset.
func (s *Server) dlrxQueryTimeout() time.Duration {
	if s.config != nil && s.config.DLRXQueryTimeout > 0 {
		return s.config.DLRXQueryTimeout
	}
	return defaultDLRXQueryTimeout
}

// dlrxCleanupInterval returns the configured dlRxStatQry expiry sweep cadence,
// falling back to the package default when unset.
func (s *Server) dlrxCleanupInterval() time.Duration {
	if s.config != nil && s.config.DLRXCleanupInterval > 0 {
		return s.config.DLRXCleanupInterval
	}
	return defaultDLRXCleanupInterval
}

// startDLRXQueryExpiryWorker runs a background sweep that expires dlRxStatQry
// queries whose unsolicited dlRxStat report never arrived (BSSCI §5.16), so
// they do not linger pending forever. The worker stops when the server context
// is cancelled.
func (s *Server) startDLRXQueryExpiryWorker() {
	if s.dlrxStore == nil {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.dlrxCleanupInterval())
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.sweepExpiredDLRXQueries(time.Now())
			}
		}
	}()
}

// sweepExpiredDLRXQueries expires every dlRxStatQry older than the configured
// timeout relative to now. It is the worker's per-tick body, separated so the
// cutoff arithmetic is testable without the ticker.
func (s *Server) sweepExpiredDLRXQueries(now time.Time) {
	cutoff := now.Add(-s.dlrxQueryTimeout())
	expired, err := s.dlrxStore.ExpireDLRXStatusQuery(s.safeCtx(), cutoff)
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIDLRXQueryExpirySweepFailed, "error", err)
		return
	}
	if expired > 0 {
		s.logger.InfoContext(s.safeCtx(), LogBSSCIDLRXQueriesExpired, "count", expired)
	}
}

// rejectConnect fails the connect operation per BSSCI §5.17: an error frame
// replaces the normal response, the session enters AwaitingConnectErrorAck,
// and the connection stays open until the base station acknowledges with
// errorAck or the handshake read deadline expires. The connection is never
// closed immediately after sending the error.
func (s *Server) rejectConnect(session *Session, opId int64, posix int, errToken string) error {
	if err := s.sendError(session, opId, posix, ResolveErrorMessage(errToken)); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, "error", err)
		// Error frame could not be written - the connection is unusable
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errToken), err)
	}
	session.ConnectState = ConnectStateAwaitingConnectErrorAck
	return nil
}

// handleConnect handles the connect operation per BSSCI specification
func (s *Server) handleConnect(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	// Connect messages are only valid while no connect operation is in flight
	if session.ConnectState != ConnectStateAwaitingConnect {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingCommandBeforeHandshake,
			"command", msg.Command,
			"connectState", int(session.ConnectState),
			"opId", msg.OpId)
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidHandshakeState)
	}

	// BSSCI-3.2-03: Connect operation MUST use opId=0
	if msg.OpId != 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIInvalidConnectOperationID,
			"op_id", msg.OpId)
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidConnectOpId)
	}

	// version is mandatory in the connect message (BSSCI rev1 §5.3.1;
	// message metadata declares it Required)
	version, hasVersion := data["version"].(string)
	if !hasVersion {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errMandatoryFieldMissing)
	}
	bsEUIRaw, hasBsEUI := data["bsEui"]
	if !hasBsEUI {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errMissingBsEui)
	}

	// BSSCI-2.1-01, BSSCI-2.2-02: Version arbitration (rev1 §4.2, §5.3.2) -
	// the negotiator selects the version the service center will speak; the
	// conRsp carries it and the base station agrees via conCmp or rejects
	selectedVersion, negErr := s.versionNegotiator.Negotiate(s.safeCtx(), version)
	if negErr != nil {
		var catErr *CatalogError
		if errors.As(negErr, &catErr) {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIVersionIncompatible,
				"client_version", version,
				"error", catErr.Token)
			return s.rejectConnect(session, msg.OpId, catErr.Posix, catErr.Token)
		}
		// Fallback for unexpected errors
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errVersionIncompatible)
	}

	// bsEui is a full-range unsigned EUI-64 (BSSCI §5.3.1); values above
	// INT64_MAX are valid
	bsEUI, euiErr := coerceUint64(bsEUIRaw)
	if euiErr != nil {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidBsEui)
	}

	// Simple session field assignments
	session.BaseStationEUI = bsEUI
	session.ClientVersion = version             // Raw BS-provided version for audit
	session.NegotiatedVersion = selectedVersion // Negotiated version carried in conRsp (BSSCI §5.3.2)
	session.Vendor = getStringField(data, "vendor", "")
	session.Model = getStringField(data, "model", "")
	session.Name = getStringField(data, "name", "")
	session.SoftwareVersion = getStringField(data, "swVersion", "")

	// BSSCI §5.3: Capture connect info object for session persistence
	if infoValue, hasInfo := data["info"]; hasInfo {
		infoJSON, err := json.Marshal(infoValue)
		if err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToMarshalConnectInfo, "error", err)
		} else {
			session.ConnectInfo = infoJSON
		}
	}

	// BSSCI §5.3: bidi flag must be explicitly declared
	bidiValue, hasBidi := data["bidi"]
	if !hasBidi {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errMissingBidiFlag)
	}

	bidi, ok := bidiValue.(bool)
	if !ok {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errMissingBidiFlag)
	}
	session.Bidirectional = bidi

	// Registration and tenant authorization precede the connect response
	// (§5.3.2): an unregistered or unauthorized base station is rejected via
	// the error/errorAck sequence before any conRsp is offered
	ctx := s.sessionContext(session)
	euiBytes := mioty.EUI64(session.BaseStationEUI).ToBytes()
	baseStation, bsErr := s.connectionRegistry.GetBaseStationGlobal(ctx, euiBytes)
	if bsErr != nil || baseStation == nil {
		s.logger.ErrorContext(ctx, LogBSSCIBaseStationNotFoundInDatabase,
			"eui", session.BaseStationEUI,
			"euiHex", fmt.Sprintf("%X", euiBytes),
			"error", bsErr)
		return s.rejectConnect(session, msg.OpId, POSIX_EPERM, errBaseStationNotRegistered)
	}

	if s.config.OrgEnforcementEnabled {
		// Certificate-resolved tenant and registered tenant must agree; the
		// rejection never reveals that the EUI exists under another tenant
		if session.ResolvedTenantID != baseStation.TenantID {
			s.logger.WarnContext(ctx, LogBSSCIBaseStationNotFoundInDatabase,
				"eui", session.BaseStationEUI,
				"certTenant", session.ResolvedTenantID)
			return s.rejectConnect(session, msg.OpId, POSIX_EPERM, errBaseStationNotRegistered)
		}

		// EUI-CN certificates bind the certificate to one station: the
		// asserted EUI must equal the connect bsEui (defense-in-depth beyond
		// the spec's mutual-TLS requirement; org-<UUID> certificates carry no
		// station identity and skip this check)
		if session.certSubjectEUI != nil && *session.certSubjectEUI != session.BaseStationEUI {
			s.logger.WarnContext(ctx, LogBSSCICertSubjectEUIMismatch,
				"certEui", *session.certSubjectEUI,
				"bsEui", session.BaseStationEUI)
			return s.rejectConnect(session, msg.OpId, POSIX_EPERM, errBaseStationNotRegistered)
		}

		// The presented certificate must match the registered station's
		// stored fingerprint (both CN forms); rejection stays
		// indistinguishable from an unregistered station
		if session.ClientCert != nil && s.bsDirectory != nil {
			if fpErr := s.verifyCertificateFingerprint(ctx, session); fpErr != nil {
				s.logger.WarnContext(ctx, LogBSSCICertFingerprintMismatch,
					"eui", session.BaseStationEUI,
					"error", fpErr)
				return s.rejectConnect(session, msg.OpId, POSIX_EPERM, errBaseStationNotRegistered)
			}
		}
	} else if baseStation.TenantID > 0 {
		// Community fallback: adopt the registered base station tenant and
		// its default organization
		session.ResolvedTenantID = baseStation.TenantID
		if s.orgResolver != nil {
			if orgID, orgErr := s.orgResolver.GetDefaultOrgForTenant(ctx, baseStation.TenantID); orgErr == nil {
				session.OrganizationID = orgID
			}
		}
		ctx = s.sessionContext(session)
	}
	session.pendingBaseStation = baseStation

	// Geolocation parsing: exactly three finite numeric values with lat/lon in range
	if geoLoc, ok := data["geoLocation"].([]interface{}); ok && len(geoLoc) == 3 {
		if coords, valid := extractFloatSlice(geoLoc); valid {
			lat, lon, alt := coords[0], coords[1], coords[2]
			if lat >= models.LatitudeMin && lat <= models.LatitudeMax &&
				lon >= models.LongitudeMin && lon <= models.LongitudeMax {
				session.GeoLocation = []float64{lat, lon, alt}
			}
		}
	}

	// BSSCI-3.3: Session resume handling
	var snResume bool
	var previousSession *Session

	// Resume identity is snBsUuid scoped by tenant and base station EUI
	// (rev1 §5.3.1: the con message carries snBsUuid, snBsOpId, snScOpId -
	// there is no snScUuid field in the connect request)
	if snBsUUIDData, hasBsUUID := data["snBsUuid"]; hasBsUUID {
		bsUUID, catErr := extractSessionUUID(snBsUUIDData)
		if catErr != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToExtractSnBsUUID, "error", catErr.Token)
			return s.rejectConnect(session, msg.OpId, catErr.Posix, catErr.Token)
		}
		// Valid bsUUID extracted - continue with resume logic
		session.BsUUID = bsUUID

		// Extract optional operation counter constraints (absent means the
		// constraint is not asserted per BSSCI §5.3.1)
		var bsOpId, scOpId *int64
		if bsOp, ok := getNumericField(data, "snBsOpId"); ok {
			bsOpId = &bsOp
		}
		if scOp, ok := getNumericField(data, "snScOpId"); ok {
			scOpId = &scOp
		}

		// Delegate resume validation to SessionService, which returns a typed
		// outcome so infrastructure failures and inconsistent state never
		// silently degrade into a fresh session
		outcome := s.sessionSvc.HandleResume(ctx, session, bsUUID, bsOpId, scOpId, session.BaseStationEUI)
		switch outcome.Disposition {
		case ResumeInfrastructureFailure:
			// A lookup failure must reject the connect before conRsp; the old
			// resumable state stays intact for a later successful resume
			s.logger.ErrorContext(ctx, LogBSSCIFailedToCheckSessionResume,
				"error", outcome.Err,
				"eui", session.BaseStationEUI)
			return s.rejectConnect(session, msg.OpId, POSIX_EAGAIN, errSessionResumeUnavailable)
		case ResumeInconsistent:
			// Incompatible counters or version: atomically terminate the old
			// session (can_resume=false, pending ops removed) then start fresh
			s.logger.WarnContext(ctx, LogBSSCIFailedToCheckSessionResume,
				"error", outcome.Err,
				"eui", session.BaseStationEUI)
			if outcome.Previous != nil {
				if err := s.terminateInconsistentResume(ctx, outcome.Previous); err != nil {
					s.logger.ErrorContext(ctx, LogBSSCIFailedToTerminateSession,
						"error", err,
						"eui", session.BaseStationEUI)
					return s.rejectConnect(session, msg.OpId, POSIX_EAGAIN, errSessionResumeUnavailable)
				}
			}
			previousSession = nil
		case ResumeCompatible:
			previousSession = outcome.Previous
		default:
			previousSession = nil
		}

		if previousSession != nil {
			// Valid resume - restore the authoritative persisted counters,
			// not the constraint values reported in the connect message
			snResume = true
			session.IsResumed = true
			session.SessionUUID = previousSession.SessionUUID
			session.BsOpId = previousSession.LastBsOpId
			session.ScOpId = previousSession.LastScOpId
			session.LastBsOpId = previousSession.LastBsOpId
			session.LastScOpId = previousSession.LastScOpId
			session.DbSessionID = previousSession.DbSessionID
			session.OrganizationID = previousSession.OrganizationID     // Restore org from previous session
			session.ResolvedTenantID = previousSession.ResolvedTenantID // Restore tenant from previous session

			// Load and strictly decode the persisted pending operations BEFORE
			// conRsp: an infrastructure or decode failure rejects the resume
			// instead of silently losing protocol state. The validated snapshot
			// stays on the provisional connection until conCmp activates it.
			pendingOps, loadErr := s.loadPendingOperations(session)
			if loadErr != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToLoadPendingOperationsForSessionResume,
					"error", loadErr,
					"sessionID", session.DbSessionID)
				return s.rejectConnect(session, msg.OpId, POSIX_EAGAIN, errSessionResumeUnavailable)
			}

			// Semantic reconstruction also happens BEFORE conRsp: a resumable
			// operation that cannot be rebuilt rejects the resume in the same
			// way, preserving every row and its queue state instead of
			// silently dropping recoverable work after activation.
			if recErr := s.reconstituteResumeOperations(ctx, pendingOps); recErr != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToReconstitutePendingOperation,
					"error", recErr,
					"sessionID", session.DbSessionID)
				return s.rejectConnect(session, msg.OpId, POSIX_EAGAIN, errSessionResumeUnavailable)
			}
			session.resumePendingOps = pendingOps

			// Counter floor: the SC counter must be at or below every persisted
			// negative operation ID, so reissued IDs are never re-allocated
			for _, op := range pendingOps {
				if op.OperationID < 0 && op.OperationID < session.LastScOpId {
					session.LastScOpId = op.OperationID
				}
			}
			session.ScOpId = session.LastScOpId

			s.logger.InfoContext(s.safeCtx(), LogBSSCIResumingPreviousSession,
				"eui", session.BaseStationEUI,
				"bsOpId", session.BsOpId,
				"scOpId", session.LastScOpId,
				"dbSessionId", session.DbSessionID,
				"orgID", session.OrganizationID.String(),
				"tenantID", session.ResolvedTenantID)
		}
	}

	// If not resuming, generate new session UUID
	if !snResume {
		sessionUUID := uuid.New()
		session.SessionUUID = sessionUUID[:]
		session.LastBsOpId = 0
		session.LastScOpId = 0

		s.logger.InfoContext(s.safeCtx(), LogBSSCIStartingNewSession,
			"eui", session.BaseStationEUI)
	}

	// The session stays provisional until conCmp: live-session map and
	// resumable-index registration happen in handleConnectComplete

	s.logger.InfoContext(s.safeCtx(), LogBSSCIBaseStationConnected,
		"eui", session.BaseStationEUI,
		"name", session.Name,
		"clientVersion", session.ClientVersion,
		"negotiatedVersion", session.NegotiatedVersion,
		"resumed", snResume)

	// Convert session UUID to SessionUUID type for proper marshaling
	var sessionUUID mioty.SessionUUID
	copy(sessionUUID[:], session.SessionUUID)

	// Build connect response using canonical struct (BSSCI §4-4.5)
	// Validate software version (non-fatal per BSSCI §5.3.2 - swVersion is optional)
	switch s.config.SoftwareVersion {
	case "":
		s.logger.WarnContext(s.safeCtx(), LogBSSCISoftwareVersionNotConfiguredConnectResponseWillOmitSwVersionField,
			"bsEui", session.BaseStationEUI,
			"sessionUuid", sessionUUID)
	case "dev", "dev-local":
		s.logger.WarnContext(s.safeCtx(), LogBSSCIUsingDevelopmentSoftwareVersionInConnectResponse,
			"swVersion", s.config.SoftwareVersion,
			"bsEui", session.BaseStationEUI)
	}

	response := mioty.ConnectResponse{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdConnectResponse,
			OpId:        msg.OpId,
		},
		Version:   session.NegotiatedVersion, // SC canonical version, not client echo
		ScEui:     s.config.ServiceCenterEUI,
		Vendor:    &s.config.Vendor,
		Model:     &s.config.Model,
		Name:      &s.config.Name,
		SwVersion: &s.config.SoftwareVersion,
		SnResume:  snResume,
		SnScUuid:  sessionUUID,
	}

	// If resuming, include operation counters
	if snResume {
		response.SnBsOpId = &session.BsOpId
		response.SnScOpId = &session.ScOpId
	}

	if err := s.sendMessage(session, response); err != nil {
		session.ConnectState = ConnectStateTerminal
		return err
	}
	session.ConnectState = ConnectStateAwaitingConnectComplete
	return nil
}

// terminateInconsistentResume atomically retires a resumable session whose
// reported counters or negotiated version are incompatible with the current
// connect: it terminates the session (status=terminated, can_resume=false)
// and removes its pending operations so the base station starts genuinely
// fresh. The previous session is a DB-hydrated snapshot, never a live one.
func (s *Server) terminateInconsistentResume(ctx context.Context, previous *Session) error {
	if err := s.sessionSvc.TerminateSession(ctx, previous); err != nil {
		return fmt.Errorf("terminate inconsistent resume session: %w", err)
	}
	if previous.DbSessionID != 0 && s.statusSvc != nil {
		if _, err := s.statusSvc.DeletePendingOperations(ctx, previous); err != nil {
			return fmt.Errorf("remove pending operations of inconsistent resume session: %w", err)
		}
	}
	return nil
}

// reconstituteResumeOperations semantically rebuilds every payload-bearing
// pending operation in place before the resume is offered. Any failure
// rejects the whole resume so no row or downlink queue state is ever
// mutated for work the service center can no longer represent on the wire.
func (s *Server) reconstituteResumeOperations(ctx context.Context, pendingOps []*PendingOperation) error {
	for _, op := range pendingOps {
		if op.Metadata == nil {
			continue
		}
		var (
			reconstituted map[string]interface{}
			err           error
		)
		switch op.OperationType {
		case mioty.CmdULDataTransmit:
			reconstituted, err = s.reconstitueULDataTxMessage(op.Message, op.Metadata, op)
		case mioty.CmdDLDataQueue:
			reconstituted, err = s.reconstitueDLDataQueMessage(op.Message, op.Metadata, op)
		case mioty.CmdDLDataRevoke:
			reconstituted, err = s.reconstitueDLDataRevMessage(op.Message, op.Metadata)
		default:
			continue
		}
		if err != nil {
			s.logger.ErrorContext(ctx, LogBSSCIFailedToReconstitutePendingOperation,
				"error", err,
				"opId", op.OperationID,
				"type", op.OperationType)
			return fmt.Errorf("reconstitute pending operation %d (%s): %w", op.OperationID, op.OperationType, err)
		}
		op.Message = reconstituted
	}
	return nil
}

// handleConnectComplete handles the connect complete operation. The connect
// operation is already complete at the protocol level when conCmp arrives, so
// failures past this point never send a BSSCI error - partial persistence is
// compensated and the connection closed for the base station to reconnect.
func (s *Server) handleConnectComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	// conCmp is only valid after conRsp was sent
	if session.ConnectState != ConnectStateAwaitingConnectComplete {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingCommandBeforeHandshake,
			"command", msg.Command,
			"connectState", int(session.ConnectState),
			"opId", msg.OpId)
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidHandshakeState)
	}

	// Validate that connect complete has opId == 0 (BSSCI-3.3)
	if msg.OpId != 0 {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidConnectCompleteOpId)
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCIBaseStationConnectionCompletedSuccessfully,
		"eui", session.BaseStationEUI,
		"name", session.Name,
		"sessionID", session.ID)

	ctx := s.sessionContext(session)

	// Registration and tenant authorization already ran before conRsp
	baseStation := session.pendingBaseStation
	if baseStation == nil {
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf("%s: EUI %016X", ResolveErrorMessage(errBaseStationNotRegistered), session.BaseStationEUI)
	}

	// Activation is one application operation: persist the session and register
	// the live connection/status BEFORE publishing to the in-memory registries.
	// The operation is protocol-complete, so any failure closes the connection
	// without a BSSCI error - partial state is compensated and the base station
	// reconnects.

	// Step 1: persist the session row (terminates a stale active session first,
	// aborting on failure rather than leaving two active sessions)
	if err := s.sessionSvc.PersistSession(ctx, session, baseStation, session.IsResumed, session.ConnectInfo); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToPersistSession,
			"error", err,
			"eui", session.BaseStationEUI)
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf("session persistence failed after conCmp: %w", err)
	}

	// Step 2: register the live connection and base-station online status. A
	// failure here compensates the just-persisted session and closes without
	// publishing anything to the live registries.
	if err := s.connectionRegistry.RegisterConnection(ctx, session, baseStation); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateConnectionStatus,
			"eui", session.BaseStationEUI,
			"error", err)
		if termErr := s.sessionSvc.TerminateSession(ctx, session); termErr != nil {
			s.logger.ErrorContext(ctx, LogBSSCIFailedToTerminateSession,
				"error", termErr,
				"eui", session.BaseStationEUI)
		}
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf("connection registration failed after conCmp: %w", err)
	}

	// Step 3: both durable steps succeeded - publish to the live-session map
	// and the resumable index
	s.publishLiveSession(ctx, session)
	s.sessionSvc.StoreSessionByUUID(session)

	// Update the base station's bidi capability and GPS location in the
	// database (non-critical enrichment; failure is logged, not fatal)
	if s.basestationRepo != nil {
		updates := map[string]interface{}{
			"bidi": session.Bidirectional,
		}
		// Persist GPS coordinates from connect handshake (BSSCI §3.3.1)
		if len(session.GeoLocation) == 3 {
			updates["latitude"] = session.GeoLocation[0]
			updates["longitude"] = session.GeoLocation[1]
			updates["altitude"] = session.GeoLocation[2]
			updates["location_source"] = models.LocationSourceGPS
			updates["location_updated_at"] = time.Now()
		}
		if err := s.basestationRepo.Update(ctx, resolvedTenant(session, s.tenantID), baseStation.ID, updates); err != nil {
			s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateBaseStationBidiCapability,
				"eui", session.BaseStationEUI,
				"bidi", session.Bidirectional,
				"error", err)
		} else {
			s.logger.InfoContext(ctx, LogBSSCIUpdatedBaseStationBidiCapability,
				"eui", session.BaseStationEUI,
				"bidi", session.Bidirectional)
		}
	}

	// BSSCI-3.3-03: Delegate handshake completion to SessionService
	s.sessionSvc.MarkHandshakeComplete(session)
	session.ConnectState = ConnectStateComplete

	// Active sessions read without a deadline; liveness is the ping
	// operation's job (BSSCI §5.4)
	if session.Conn != nil {
		if err := session.Conn.SetReadDeadline(time.Time{}); err != nil {
			s.logger.WarnContext(ctx, LogBSSCIFailedToSetReadDeadline, "error", err)
		}
	}

	// MIOTY session resume: restore the validated pending-operation snapshot
	// loaded before conRsp (never re-read or re-written here - the DB rows are
	// already authoritative), reissue the eligible operations in deterministic
	// order with their original IDs, and only then start status polling so the
	// first status request cannot interleave with the reissue sequence.
	if pendingOps := session.resumePendingOps; len(pendingOps) > 0 {
		session.resumePendingOps = nil
		s.logger.InfoContext(ctx, LogBSSCIResumingSessionWithPendingOps,
			"bsEui", session.BaseStationEUI,
			"sessionID", session.DbSessionID,
			"pendingCount", len(pendingOps))

		for _, op := range pendingOps {
			s.logger.DebugContext(ctx, LogBSSCIProcessingPendingOperationForResume,
				"opId", op.OperationID,
				"type", op.OperationType)

			// Cache-only hydration: the loaded DB rows are authoritative and
			// must not be UPSERTed back. BS-initiated (positive) operations
			// are hydrated for response correlation but never transmitted.
			// Semantic reconstruction already happened before conRsp.
			if s.statusSvc != nil {
				s.statusSvc.RestorePendingOperation(session, op.OperationID, op)
			}

			if !isResumableScOperation(op.OperationID, op.OperationType) {
				continue
			}

			// Normalize message types to fix JSON float64 conversion before reissuing
			normalizedMsg := s.normalizeMessageTypes(op.Message, op.OperationType)
			if err := s.sendMessage(session, normalizedMsg); err != nil {
				// A send failure means the connection is broken: abort
				// activation so status polling never starts on a dead
				// transport. The rows stay persisted for the next resume.
				s.logger.ErrorContext(ctx, LogBSSCIFailedToReissuePendingOperation,
					"error", err,
					"opId", op.OperationID,
					"type", op.OperationType)
				if errors.Is(err, ErrAmbiguousWrite) {
					s.closeTransportAfterWriteFailure(session, op.OperationID, err)
				}
				return fmt.Errorf("reissue pending operation %d after resume: %w", op.OperationID, err)
			}
		}
	} else if session.DbSessionID > 0 {
		s.logger.DebugContext(ctx, LogBSSCINoPendingOperationsToResume,
			"bsEui", session.BaseStationEUI,
			"sessionID", session.DbSessionID)
	}

	// Start status mechanism only after the reissue sequence completed
	s.startStatusMechanism(session)
	s.logger.InfoContext(ctx, LogBSSCIStartedStatusMechanismForBaseStation,
		"bsEui", session.BaseStationEUI)

	// BSSCI §5.8.3: Trigger automatic attach propagate reconciliation for newly connected BS
	// Replay all bidirectional endpoints to the newly connected base station
	// Run asynchronously to avoid blocking the complete message response
	if s.propagationSvc != nil {
		go func() {
			// Create lightweight snapshot for reconciliation
			snapshot := propagation.BaseStationSession{
				ID:                session.ID,
				BaseStationEUI:    session.BaseStationEUI,
				TenantID:          resolvedTenant(session, s.tenantID),
				OrganizationID:    nil, // Will be enriched in reconciler
				HandshakeComplete: true,
			}
			// Note: bs parameter unused by ReconcileBaseStation - it only needs session.TenantID
			if err := s.propagationSvc.ReconcileBaseStation(ctx, snapshot, nil); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIBaseStationReconciliationFailed,
					"bs_eui", session.BaseStationEUI,
					"error", err)
			}
		}()
	}

	// Connection is now fully established and operational
	// No response needed for Connect Complete messages
	return nil
}

// handlePing handles ping operations
func (s *Server) handlePing(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	// Send ping response
	response := map[string]interface{}{
		"command": mioty.CmdPingResponse,
		"opId":    msg.OpId,
	}

	return s.sendMessage(session, response)
}

// handlePingResponse handles ping response from base station (for SC-initiated pings)
func (s *Server) handlePingResponse(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	// Extract result if present
	var result int
	if r, ok := data["result"].(float64); ok {
		result = int(r)
	}

	s.logger.DebugContext(s.safeCtx(), LogBSSCIReceivedPingResponse,
		"sessionID", session.ID,
		"opId", msg.OpId,
		"result", result)

	// Send ping complete to finish three-way handshake
	response := map[string]interface{}{
		"command": mioty.CmdPingComplete,
		"opId":    msg.OpId,
	}

	return s.sendMessage(session, response)
}

// handlePingComplete handles ping complete operations
func (s *Server) handlePingComplete(_ *Server, session *Session, _ *Message, _ map[string]interface{}) error {
	// Ping completed successfully - persist timestamp for monitoring (BSSCI §5.4)
	ctx := s.sessionContext(session)
	if err := s.sessionSvc.UpdatePingTimestamp(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", session.DbSessionID,
			"baseStationEui", session.BaseStationEUI)
		// Non-fatal: log error but don't fail the handshake
	}

	return nil
}

// SendPing sends a ping request to a base station (SC-initiated ping per BSSCI 3.4)
func (s *Server) SendPing(sessionID string) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Ensure handshake is complete before sending ping (BSSCI §5.4)
	if !session.HandshakeComplete {
		return fmt.Errorf("%s for session %s", ResolveErrorMessage(errCannotSendPing), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID and
	// persist the counter before the frame is written; never roll back. Ping
	// is idempotent liveness traffic, so no pending record is persisted.
	opId, err := s.beginScOperation(session)
	if err != nil {
		return err
	}

	msg := map[string]interface{}{
		"command": mioty.CmdPing,
		"opId":    opId,
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingPingToBaseStation,
		"sessionID", sessionID,
		"bsEui", session.BaseStationEUI,
		"opId", opId)

	if err := s.sendMessage(session, msg); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			s.closeTransportAfterWriteFailure(session, opId, err)
		}
		return err
	}

	return nil
}

// InitiatePing sends a ping request to a base station by EUI (BSSCI §5.4)
// Implements PingCommander interface for API layer access
func (s *Server) InitiatePing(ctx context.Context, baseStationEUI uint64, tenantID int64) (int64, error) {
	// Lookup session via SessionDirectory
	sessionInterface := s.GetSessionByEUI(baseStationEUI)
	if sessionInterface == nil {
		return 0, NewCatalogError(errSessionNotFound, POSIX_ENOENT)
	}

	session, ok := sessionInterface.(*Session)
	if !ok {
		return 0, fmt.Errorf("invalid session type for EUI %016X", baseStationEUI)
	}

	// Ensure handshake is complete before sending ping (BSSCI §5.4)
	if !session.HandshakeComplete {
		return 0, NewCatalogError(errCannotSendPing, POSIX_EPROTO)
	}

	// Validate tenant ownership (defense-in-depth)
	sessionTenantID := resolvedTenant(session, s.tenantID)
	if sessionTenantID != tenantID {
		return 0, fmt.Errorf("base station %016X belongs to different tenant (session: %d, caller: %d)",
			baseStationEUI, sessionTenantID, tenantID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID and
	// persist the counter before the frame is written; never roll back. Ping
	// is idempotent liveness traffic, so no pending record is persisted.
	opId, err := s.beginScOperation(session)
	if err != nil {
		return 0, err
	}

	msg := map[string]interface{}{
		"command": mioty.CmdPing,
		"opId":    opId,
	}

	s.logger.InfoContext(ctx, LogBSSCISendingPingToBaseStation,
		"sessionID", session.ID,
		"bsEui", session.BaseStationEUI,
		"opId", opId)

	if err := s.sendMessage(session, msg); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			s.closeTransportAfterWriteFailure(session, opId, err)
		}
		return 0, err
	}

	return opId, nil
}

// handleStatus handles status operations

// handleAttach handles attach operations per BSSCI 3.6.1 and 3.6.2
func (s *Server) handleAttach(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// Extract and validate mandatory epEui field (full-range unsigned EUI-64)
	epEUI, hasEpEUI := getUint64Field(data, "epEui")
	if !hasEpEUI {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingEpEui))
	}

	// Extract and validate ALL mandatory fields per BSSCI 3.6.1
	rxTime, hasRxTime := getNumericField(data, "rxTime")          // Reception time (Unix UTC ns)
	attachCnt, hasAttachCnt := getNumericField(data, "attachCnt") // Attach counter

	// Validate mandatory field presence
	if !hasRxTime {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingRxTime))
	}
	if !hasAttachCnt {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingAttachCnt))
	}
	// Validate attachCnt is 24-bit unsigned (0 to 0xFFFFFF per BSSCI §5.6.1)
	if attachCnt < 0 || attachCnt > 0xFFFFFF {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidAttachCntRange))
	}

	// Extract and validate mandatory SNR/RSSI fields per BSSCI 3.6.1
	snr, hasValidSnr := getFloatFieldValidated(data, "snr")    // Signal-to-noise ratio in dB
	rssi, hasValidRssi := getFloatFieldValidated(data, "rssi") // Signal strength in dBm

	if !hasValidSnr {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSnrValue))
	}
	if !hasValidRssi {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRssiValue))
	}

	// Extract and validate nonce (mandatory 4-byte array per BSSCI §5.6.1)
	nonceData, ok := data["nonce"]
	if !ok {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingNonce))
	}
	nonce, errToken := validateByteArray(nonceData, "nonce", 4)
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	// Extract and validate signature (mandatory 4-byte array per BSSCI §5.6.1)
	signData, ok := data["sign"]
	if !ok {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingSign))
	}
	sign, errToken := validateByteArray(signData, "sign", 4)
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	// Extract optional radio capability flags per BSSCI 3.6.1
	var dualChan bool
	_, hasDualChan := data["dualChan"]
	if hasDualChan {
		dualChan = getBoolField(data, "dualChan", false)
	}

	var repetition bool
	_, hasRepetition := data["repetition"]
	if hasRepetition {
		repetition = getBoolField(data, "repetition", false)
	}

	var wideCarrOff bool
	_, hasWideCarrOff := data["wideCarrOff"]
	if hasWideCarrOff {
		wideCarrOff = getBoolField(data, "wideCarrOff", false)
	}

	var longBlkDist bool
	_, hasLongBlkDist := data["longBlkDist"]
	if hasLongBlkDist {
		longBlkDist = getBoolField(data, "longBlkDist", false)
	}

	// Extract optional fields with validation
	eqSnr := snr // Default to snr value
	var rxDuration int64
	if rawEqSnr, exists := data["eqSnr"]; exists && rawEqSnr != nil {
		if eqSnrFloat, valid := getFloatFieldValidated(data, "eqSnr"); valid {
			eqSnr = eqSnrFloat
		} else {
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidEqSnrValue))
		}
	}
	rxDuration, hasRxDuration := getNumericField(data, "rxDuration")

	var profile string
	var hasProfile bool
	if profileData, exists := data["profile"]; exists {
		if profileStr, ok := profileData.(string); ok {
			profile = profileStr
			hasProfile = true
		}
	}

	// Extract subpackets if provided
	var subpackets []interface{}
	if sp, ok := data["subpackets"]; ok {
		if spArray, ok := sp.([]interface{}); ok {
			subpackets = spArray
		}
	}

	// Optional short address if assigned by Base Station
	shAddr, hasShAddr := getNumericField(data, "shAddr")
	var scAssignedShAddr bool

	s.logger.InfoContext(s.safeCtx(), LogBSSCIEndPointAttachRequestWithFullTelemetry,
		"baseStation", session.BaseStationEUI,
		"endPoint", epEUI,
		"attachCnt", attachCnt,
		"rxTime", rxTime,
		"nonceLen", len(nonce),
		"signLen", len(sign),
		"dualChan", dualChan,
		"repetition", repetition,
		"wideCarrOff", wideCarrOff,
		"longBlkDist", longBlkDist)

	// Store pending operation for completion tracking
	// Make a copy of the data to avoid mutations
	pendingData := map[string]interface{}{
		"epEui":       pkgmioty.FormatEUI64(epEUI),
		"attachCnt":   attachCnt,
		"rxTime":      rxTime,
		"nonce":       nonce,
		"sign":        sign,
		"dualChan":    dualChan,
		"repetition":  repetition,
		"wideCarrOff": wideCarrOff,
		"longBlkDist": longBlkDist,
		"rssi":        rssi,
		"snr":         snr,
		"eqSnr":       eqSnr,
		"subpackets":  subpackets,
	}

	// Optional fields: only store when present
	if hasRxDuration {
		pendingData["rxDuration"] = rxDuration
	}
	// BSSCI-ATTACH-023: Empty profile preserves existing DB value (consistent with detach at line 2600)
	// Rationale: "absent or empty" should not overwrite; only non-empty strings update the profile field
	if hasProfile && profile != "" {
		pendingData["profile"] = profile
	}

	// Track pending operation for the three-way handshake
	// StatusService is the single path for pending operation persistence
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   int64(msg.OpId),
		OperationType: mioty.CmdAttach,
		Message:       data,
		Metadata:      pendingData,
		CreatedAt:     time.Now(),
	}
	err := s.statusSvc.RecordPendingOperation(ctx, session, int64(msg.OpId), pendingOp, session.DbSessionID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToRecordPendingAttachOperation, "opId", msg.OpId, "error", err)
		return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errOperationFailed))
	}

	// Convert endpoint EUI to bytes for database lookup
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes[:], epEUI)

	// Look up endpoint in database to get network keys
	var nwkSnKey []byte
	var responseShAddr uint16

	if s.endpointRepo != nil {
		tenantID := resolvedTenant(session, s.tenantID)
		endpoint, err := s.endpointRepo.GetByEUI(ctx, resolvedTenant(session, s.tenantID), epEUIBytes)
		if err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIEndpointNotProvisionedForAttach,
				"epEui", epEUI,
				"error", err)
			// Clean up pending operation before returning error
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			// Return error per BSSCI - endpoint must be provisioned
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errEndpointNotProvisioned))
		}

		// Store endpoint metadata for attach propagation (BSSCI §5.8.2)
		// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
		if s.statusSvc != nil {
			if pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId)); err == nil {
				pendingOp.Metadata["endpointID"] = endpoint.ID
				pendingOp.Metadata["endpointTenantID"] = endpoint.TenantID
				// Note: Metadata changes are in-memory only; StatusService already persisted the operation
				// If DB persistence of metadata updates is required, call RecordPendingOperation here
			}
		}

		// Detect roaming and validate agreements
		var isRoaming bool
		var ownerTenantID int64
		if s.roamingSvc != nil {
			isRoaming, ownerTenantID, err = s.roamingSvc.DetectAndValidateRoaming(ctx, epEUIBytes, tenantID)
			if err != nil {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIRoamingValidationFailed,
					"epEui", epEUI,
					"servingTenant", tenantID,
					"error", err)
				// Clean up pending operation before returning error
				if err := s.removePendingOperation(session, msg.OpId); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
				}
				return s.sendError(session, msg.OpId, POSIX_EPERM, ResolveErrorMessage(errRoamingNotAllowed))
			}

			// If roaming, record the attach event
			if isRoaming {
				s.logger.InfoContext(s.safeCtx(), LogBSSCIRoamingEndpointAttaching,
					"epEui", epEUI,
					"ownerTenant", ownerTenantID,
					"servingTenant", tenantID)

				// Record roaming attach event
				if err := s.roamingSvc.RecordAttach(ctx, epEUIBytes, session.BaseStationEUIBytes(), tenantID); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRecordRoamingAttach,
						"error", err)
					// Non-fatal: continue with attach
				}

				// Update session with roaming endpoint
				if session.DbSessionID > 0 {
					if err := s.roamingSvc.UpdateSessionRoaming(ctx, session.DbSessionID, epEUIBytes, true, tenantID); err != nil {
						s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateSessionRoaming,
							"error", err)
						// Non-fatal: continue with attach
					}
				}

				// Store owner tenant ID in pending operation metadata
				if s.statusSvc != nil {
					if pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId)); err == nil {
						pendingOp.Metadata["ownerTenantID"] = ownerTenantID
						pendingOp.Metadata["isRoaming"] = true
					}
				}
			}
		} else {
			// If roaming service is not configured, owner equals serving tenant
			_ = tenantID // Implicit: ownerTenantID := tenantID (not needed due to earlier declaration)
		}

		// Get network session key from endpoint
		nwkSnKey = endpoint.NwkSnKey

		if len(nwkSnKey) != 16 {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIInvalidNetworkKeyLengthForAttach,
				"epEui", epEUI,
				"keyLength", len(nwkSnKey))
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errNwkSnKeyInvalidLength))
		}

		// Replay protection: enforce monotonic attach counter (BSSCI §5.6.1)
		if endpoint.AttachCnt != nil {
			storedCnt := int64(*endpoint.AttachCnt)

			// Handle 24-bit rollover: allow reset to 0 only when stored is near max
			isRollover := storedCnt > 0xFFFF00 && attachCnt < 0x100

			if !isRollover && attachCnt <= storedCnt {
				s.logger.WarnContext(
					s.safeCtx(),
					LogBSSCIAttachCounterReplay,
					"tenantId", resolvedTenant(session, s.tenantID),
					"epEui", epEUI,
					"storedAttachCnt", storedCnt,
					"incomingAttachCnt", attachCnt,
				)
				if err := s.removePendingOperation(session, msg.OpId); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
				}
				return s.sendError(session, msg.OpId, POSIX_EPROTO,
					ResolveErrorMessage(errAttachCounterNotMonotonic))
			}
		}

		//nolint:gosec // G115: attachCnt validated to be <= 0xFFFFFF on line 1431, safe for uint32
		if err := ValidateAttachSignature(epEUI, uint32(attachCnt), sign, nwkSnKey); err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIAttachSignatureValidationFailed,
				"epEui", epEUI,
				"error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
		}

		sessionKey, err := DeriveSessionKey(epEUI, nonce, sign, nwkSnKey)
		if err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToDeriveSessionKey,
				"epEui", epEUI,
				"error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
		}

		// Encrypt session key using raw GCM (nonce + ciphertext + tag ~44 bytes)
		// This satisfies BSSCI §5.6.2 storage requirements while maintaining authenticated encryption
		encryptedSessionKey := sessionKey
		if s.keyEncryptor != nil {
			var err error
			encryptedSessionKey, err = s.keyEncryptor.EncryptKeyRaw(sessionKey)
			if err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToEncryptSessionKey, "error", err)
				if err := s.removePendingOperation(session, msg.OpId); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
				}
				return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errFailedToEncryptKey))
			}
		}

		// Use short address from database or BS-provided
		if hasShAddr {
			// Short address must fit in uint16 range (0-65535 per MIOTY spec)
			if shAddr < 0 || shAddr > 65535 {
				return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errInvalidFieldValue))
			}
			responseShAddr = uint16(shAddr)
			// Update database if BS assigned a different short address
			if endpoint.ShAddr == nil || *endpoint.ShAddr != responseShAddr { // Model now stores uint16, no cast needed
				scAssignedShAddr = false
			}
		} else if endpoint.ShAddr != nil {
			// Model now stores uint16, no conversion needed
			responseShAddr = *endpoint.ShAddr
			scAssignedShAddr = true
		} else {
			// Generate short address from lower bits of EUI
			responseShAddr = uint16(epEUI & 0xFFFF) //nolint:gosec // G115: bitwise AND guarantees value fits uint16
			scAssignedShAddr = true
		}

		attachUpdates := map[string]interface{}{
			"attach_cnt":          attachCnt,
			"last_attach_rx_time": rxTime,
			"nonce":               nonce,
			"sign":                sign,
		}

		if hasRxDuration {
			attachUpdates["last_attach_rx_duration"] = rxDuration
		}
		if hasDualChan {
			attachUpdates["dual_chan"] = dualChan
		}
		if hasRepetition {
			attachUpdates["repetition"] = repetition
		}
		if hasWideCarrOff {
			attachUpdates["wide_carr_off"] = wideCarrOff
		}
		if hasLongBlkDist {
			attachUpdates["long_blk_dist"] = longBlkDist
		}
		if hasShAddr || scAssignedShAddr {
			attachUpdates["sh_addr"] = responseShAddr
		}

		// Persist subpackets if present (BSSCI §5.6.1 optional field)
		if raw, ok := data["subpackets"].(map[string]interface{}); ok {
			if sp, err := NormalizeSubpackets(raw); err != nil {
				s.logger.WarnContext(ctx, LogBSSCIFailedToNormalizeAttachSubpackets, "error", err)
			} else {
				if encoded, err := json.Marshal(sp); err == nil {
					attachUpdates["last_attach_subpackets"] = string(encoded)
				}
			}
		}

		if s.config != nil && s.config.DisableAttachPersistence {
			// Skip DB persistence for test scenarios while still exercising replay protection and validation.
			// However, still call UpdateRadioMetricsSelective to test profile handling logic.
			var eui models.EUI
			if len(epEUIBytes) == 8 {
				copy(eui[:], epEUIBytes)

				update := interfaces.RadioMetricsUpdate{
					SNR:        snr,
					RSSI:       rssi,
					EqSNR:      eqSnr,
					RxTime:     rxTime,
					RxDuration: nil, // preserve if not present
					Profile:    nil, // preserve if not present
				}

				if hasRxDuration {
					update.RxDuration = &rxDuration
				}

				// BSSCI-ATTACH-023: Only update profile when non-empty to preserve existing DB value
				if hasProfile && profile != "" {
					update.Profile = &profile
				}

				if err := s.endpointRepo.UpdateRadioMetricsSelective(ctx, resolvedTenant(session, s.tenantID), eui, update); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateRadioMetrics, "error", err)
				}
			}
			return nil
		}

		// The attachment persister owns the transactional endpoint + session
		// upsert; any persistence failure keeps the exact response behavior:
		// remove the pending operation and answer with the database error.
		rec := AttachSessionRecord{
			TenantID:         tenantID,
			BSLookupTenantID: resolvedTenant(session, s.tenantID),
			EndpointID:       endpoint.ID,
			EndpointUpdates:  attachUpdates,
			EncryptedKey:     encryptedSessionKey,
			//nolint:gosec // G115: attachCnt validated to be within 0..0xFFFFFF above
			AttachCnt:      uint32(attachCnt),
			ShAddr:         responseShAddr,
			BaseStationEUI: session.BaseStationEUIBytes(),
		}
		if s.attachPersistence == nil {
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
		}
		if err := s.attachPersistence.PersistAttachSession(ctx, rec); err != nil {
			if rmErr := s.removePendingOperation(session, msg.OpId); rmErr != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", rmErr)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
		}

		// Update radio metrics using selective update (preserves optional fields)

		// Update radio metrics using selective update (preserves optional fields)
		var eui models.EUI
		if len(epEUIBytes) == 8 {
			copy(eui[:], epEUIBytes)

			update := interfaces.RadioMetricsUpdate{
				SNR:        snr,
				RSSI:       rssi,
				EqSNR:      eqSnr,
				RxTime:     rxTime,
				RxDuration: nil, // preserve if not present
				Profile:    nil, // preserve if not present
			}

			if hasRxDuration {
				update.RxDuration = &rxDuration
			}

			// BSSCI-ATTACH-023: Only update profile when non-empty to preserve existing DB value
			if hasProfile && profile != "" {
				update.Profile = &profile
			}

			if err := s.endpointRepo.UpdateRadioMetricsSelective(ctx, resolvedTenant(session, s.tenantID), eui, update); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateRadioMetrics, "error", err)
			}
		}

		// Build attach response per BSSCI 3.6.2
		response := map[string]interface{}{
			"command": mioty.CmdAttachResponse,
			"opId":    msg.OpId,
		}

		numericKey := make([]interface{}, 16)
		for i := 0; i < 16; i++ {
			if i < len(sessionKey) {
				numericKey[i] = int(sessionKey[i])
			} else {
				numericKey[i] = 0
			}
		}

		response["nwkSnKey"] = numericKey
		if scAssignedShAddr {
			response["shAddr"] = responseShAddr
		}

		return s.sendMessage(session, response)
	}
	return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errEndpointNotProvisioned))
}

// handleDetach handles detach operations per BSSCI 3.7
func (s *Server) handleDetach(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	// Extract and validate mandatory epEui field (full-range unsigned EUI-64)
	epEUI, hasEpEUI := getUint64Field(data, "epEui")
	if !hasEpEUI {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingEpEui))
	}

	// Extract and validate ALL mandatory fields per BSSCI 3.7.1
	rxTime, hasRxTime := getNumericField(data, "rxTime")          // Unix UTC center of last subpacket (ns)
	packetCnt, hasPacketCnt := getNumericField(data, "packetCnt") // EP packet counter

	// Extract and validate mandatory SNR/RSSI fields per BSSCI 3.7.1
	snr, hasValidSnr := getFloatFieldValidated(data, "snr")    // Signal-to-noise ratio in dB
	rssi, hasValidRssi := getFloatFieldValidated(data, "rssi") // Signal strength in dBm

	// Validate mandatory field presence per BSSCI 3.7.1 and clause 2.4
	if !hasRxTime {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingRxTime))
	}
	if !hasPacketCnt {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingPacketCnt))
	}
	if !hasValidSnr {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSnrValue))
	}
	if !hasValidRssi {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRssiValue))
	}

	// Extract optional fields with validation
	eqSnr := snr // Default to snr value
	if _, exists := data["eqSnr"]; exists {
		if eqSnrFloat, valid := getFloatFieldValidated(data, "eqSnr"); valid {
			eqSnr = eqSnrFloat
		} else {
			// Present optional field with invalid value (BSSCI 2.4)
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidEqSnrValue))
		}
	}
	rxDuration, _ := getNumericField(data, "rxDuration") // First to last subpacket center (ns)
	profile := getStringField(data, "profile", "")       // MIOTY profile (e.g., "eu1")

	// Extract and validate signature format using validateByteArray helper (mandatory per BSSCI 3.7.1)
	// Cryptographic validation happens later after endpoint lookup (see below)
	// Radio spec §3.7.1 says signature is "analogous to attach" but MIOTY spec TBD: SC signature derivation method unclear
	sign, errToken := validateByteArray(data["sign"], "sign", 4)
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	// Extract subpackets if provided (optional per BSSCI §5.7.1)
	// Note: Subpackets structure is map[string]interface{} (not []interface{})
	var subpacketsMap map[string]interface{}
	if sp, ok := data["subpackets"]; ok {
		if spMap, ok := sp.(map[string]interface{}); ok {
			subpacketsMap = spMap
		}
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCIEndPointDetachRequestWithTelemetry,
		"baseStation", session.BaseStationEUI,
		"endPoint", epEUI,
		"packetCnt", packetCnt,
		"rxTime", rxTime,
		"snr", snr,
		"rssi", rssi,
		"signLen", len(sign))

	// Build typed detachMetadata struct for crash-safe persistence (BSSCI §5.7.1)
	packetCntUint, errToken := safeUint32(packetCnt, "packetCnt")
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	typedMetadata := &detachMetadata{
		EpEui:     epEUI,
		PacketCnt: packetCntUint,
		Signature: sign,
		RxTime:    rxTime,
		SNR:       snr,
		RSSI:      rssi,
	}
	// Add optional fields
	if eqSnr != snr {
		typedMetadata.EqSnr = &eqSnr
	}
	if profile != "" {
		typedMetadata.Profile = &profile
	}
	if rxDuration > 0 {
		typedMetadata.RxDuration = &rxDuration
	}

	// Fetch endpoint early to resolve owner tenant/org for roaming support (DET-02)
	ctx := s.sessionContext(session)
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEUI)

	// Try same-tenant lookup first, then cross-tenant fallback for roaming
	endpoint, err := s.endpointRepo.GetByEUI(ctx, resolvedTenant(session, s.tenantID), euiBytes)
	if err != nil {
		// Cross-tenant fallback - construct models.EUI for Get method
		eui := models.EUI{}
		copy(eui[:], euiBytes)
		endpoint, err = s.endpointRepo.Get(ctx, eui)
	}

	// Resolve endpoint owner tenant/org (or fall back to session tenant)
	var ownerTenantID int64
	var ownerOrgUUID uuid.UUID
	var ownerCtx context.Context
	var validationStatus string // Detach signature validation status from internal validator

	if err == nil && endpoint != nil {
		ownerTenantID = endpoint.TenantID
		// FIX-4: Handle org resolver errors explicitly
		if s.orgResolver != nil {
			var orgErr error
			ownerOrgUUID, orgErr = s.orgResolver.GetDefaultOrgForTenant(context.Background(), ownerTenantID)
			if orgErr != nil {
				s.logger.WarnContext(ctx, LogBSSCIFailedToResolveOrganizationForEndpointOwner,
					"tenantID", ownerTenantID,
					"error", orgErr)
				ownerOrgUUID = uuid.Nil // Explicit fallback
			}
		}
		// FIX-1: Use Background() to avoid session context pollution
		ownerCtx = pkgcontext.WithTenantID(context.Background(), ownerTenantID)
		if ownerOrgUUID != uuid.Nil {
			ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
		}
	} else {
		// Unknown endpoint - validate via internal validator and use returned tenant metadata (BSSCI §5.7)
		s.logger.WarnContext(ctx, LogBSSCIDetachFromUnknownEndpoint,
			"error", err,
			"epEui", epEUI)

		// Validate unknown endpoint signature via internal detach validator service
		if s.detachValidator != nil {
			result, validationErr := s.detachValidator.ValidateDetachSignature(ctx, epEUI, sign)
			if validationErr != nil {
				// Check for typed sentinel: endpoint not found
				if errors.Is(validationErr, ErrDetachValidationEndpointNotFound) {
					s.logger.WarnContext(ctx, LogBSSCIUnknownEndpointNotFoundDuringDetachValidation,
						"epEui", epEUI)
					return s.sendError(session, msg.OpId, POSIX_ENOENT, ResolveErrorMessage(errEndpointNotFound))
				}

				// Check for signature validation failure with metadata
				if errors.Is(validationErr, ErrDetachSignatureInvalid) && result != nil {
					s.logger.WarnContext(ctx, LogBSSCIUnknownEndpointDetachSignatureInvalid,
						"epEui", epEUI,
						"tenant_id", result.TenantID,
						"owner_tenant_id", result.OwnerTenantID,
						"validation_status", result.ValidationStatus)
					return s.sendError(session, msg.OpId, POSIX_EACCES, ResolveErrorMessage(errDetachSignatureInvalid))
				}

				// All other errors treated as signature validation failure
				s.logger.WarnContext(ctx, LogBSSCIUnknownEndpointDetachSignatureValidationFailed,
					"epEui", epEUI,
					"error", validationErr)
				return s.sendError(session, msg.OpId, POSIX_EACCES, ResolveErrorMessage(errDetachSignatureInvalid))
			}

			// Double-check result validity
			if !result.Valid {
				return s.sendError(session, msg.OpId, POSIX_EACCES, ResolveErrorMessage(errDetachSignatureInvalid))
			}

			// SUCCESS: Use validator-returned tenant metadata
			ownerTenantID = result.TenantID
			validationStatus = result.ValidationStatus

			// Resolve org UUID using orgResolver (matches known endpoint pattern at lines 2622-2630)
			if s.orgResolver != nil {
				var orgErr error
				ownerOrgUUID, orgErr = s.orgResolver.GetDefaultOrgForTenant(context.Background(), ownerTenantID)
				if orgErr != nil {
					s.logger.WarnContext(ctx, LogBSSCIFailedToResolveOrganizationForUnknownEndpointOwner,
						"tenant_id", ownerTenantID,
						"error", orgErr)
					ownerOrgUUID = uuid.Nil
				}
			}

			// Build owner context with tenant + org
			ownerCtx = pkgcontext.WithTenantID(context.Background(), ownerTenantID)
			if ownerOrgUUID != uuid.Nil {
				ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
			}

			s.logger.InfoContext(ctx, LogBSSCIUnknownEndpointDetachSignatureValidatedSuccessfully,
				"epEui", epEUI,
				"tenantId", ownerTenantID,
				"ownerTenantId", result.OwnerTenantID,
				"validationStatus", validationStatus)
		} else {
			// Validator not configured - fall back to session tenant
			ownerTenantID = resolvedTenant(session, s.tenantID)
			ownerCtx = pkgcontext.WithTenantID(context.Background(), ownerTenantID)
			s.logger.WarnContext(ctx, LogBSSCIDetachValidatorNotConfiguredUsingSessionTenantForUnknownEndpoint,
				"epEui", epEUI,
				"session_tenant", ownerTenantID)
		}
	}

	// Update typed metadata with owner context for crash-safe resume
	typedMetadata.TenantID = ownerTenantID
	typedMetadata.OrgUUID = ownerOrgUUID

	// Detect roaming for detach operation
	var isRoaming bool
	servingTenantID := resolvedTenant(session, s.tenantID)
	if s.roamingSvc != nil && endpoint != nil {
		isRoaming, ownerTenantID, err = s.roamingSvc.DetectAndValidateRoaming(ctx, euiBytes, servingTenantID)
		if err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIRoamingValidationFailedDuringDetach,
				"epEui", epEUI,
				"servingTenant", servingTenantID,
				"error", err)
			// For detach, we allow it even if roaming validation fails
			// The endpoint is leaving anyway
			isRoaming = false
			ownerTenantID = servingTenantID
		}

		// If roaming, record the detach event
		if isRoaming {
			s.logger.InfoContext(s.safeCtx(), LogBSSCIRoamingEndpointDetaching,
				"epEui", epEUI,
				"ownerTenant", ownerTenantID,
				"servingTenant", servingTenantID)

			// Record roaming detach event
			if err := s.roamingSvc.RecordDetach(ctx, euiBytes, session.BaseStationEUIBytes(), servingTenantID); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRecordRoamingDetach,
					"error", err)
				// Non-fatal: continue with detach
			}

			// Update session to remove roaming endpoint
			if session.DbSessionID > 0 {
				if err := s.roamingSvc.UpdateSessionRoaming(ctx, session.DbSessionID, euiBytes, false, servingTenantID); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateSessionRoamingForDetach,
						"error", err)
					// Non-fatal: continue with detach
				}
			}

			// Update metadata with actual owner tenant ID
			typedMetadata.TenantID = ownerTenantID
		}
	}

	// Set validation status for signature tracking (BSSCI §5.7)
	if endpoint != nil {
		// Known endpoint - signature can be validated against endpoint.Sign
		typedMetadata.ValidationStatus = ValidationStatusValidated
	} else {
		// Unknown endpoint - signature cannot be validated (preshared-key resolver still needed)
		typedMetadata.ValidationStatus = ValidationStatusUnknownEndpoint
	}

	// Convert typed metadata to map for JSON persistence
	metadataMap := detachMetadataToMap(typedMetadata)
	// Add subpackets to metadata map for crash-resume (not part of typed struct)
	if subpacketsMap != nil {
		metadataMap["subpackets"] = subpacketsMap
	}

	// Store endpoint.ID if known (avoid double-fetch in handleDetachComplete)
	if endpoint != nil {
		metadataMap["endpointID"] = endpoint.ID
		typedMetadata.EndpointID = endpoint.ID
	}

	// Persist pending operation via StatusService (handles both DB + map with SessionOpKey).
	// This row is a crash-resume aid for a BS-initiated operation (positive
	// opId): the abort-before-send rule protects SC-initiated recovery only,
	// and aborting here would drop a live detach on a transient DB failure, so
	// persistence stays best-effort (BSSCI §5.7).
	if session.DbSessionID != 0 {
		if err := s.persistPendingOperation(session, int64(msg.OpId), mioty.CmdDetach, data, euiBytes, metadataMap); err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToPersistPendingOperationMigrationNeeded,
				"error", err,
				"sessionID", session.DbSessionID,
				"opId", msg.OpId)
		}
	}

	// Persist detach message to mioty_messages table under endpoint owner (BSSCI §5.7.1 Finding 6)
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)
	detachMsg := &mioty.DetachMessage{
		CommandType:    mioty.CmdDetach,
		OpId:           int64(msg.OpId),
		EpEui:          euiBytes,
		BasestationEui: bsEuiBytes,
		RxTime:         rxTime,
		PacketCnt:      packetCntUint,
		SNR:            snr,
		RSSI:           rssi,
		Signature:      sign,
		MessageType:    mioty.MessageTypeDetach,
		Direction:      mioty.DirectionUplink,
		InterfaceType:  mioty.InterfaceBSSCI,
		TenantID:       ownerTenantID,
	}
	// Set OrgUUID only when valid
	if ownerOrgUUID != uuid.Nil {
		ownerOrgUUIDStr := ownerOrgUUID.String()
		detachMsg.OrgUUID = &ownerOrgUUIDStr
	}
	// Add optional fields
	if typedMetadata.EqSnr != nil {
		detachMsg.EqSnr = typedMetadata.EqSnr
	}
	if typedMetadata.Profile != nil {
		detachMsg.Profile = typedMetadata.Profile
	}
	if typedMetadata.RxDuration != nil {
		detachMsg.RxDuration = typedMetadata.RxDuration
	}
	// Persist subpackets if present (BSSCI §5.7.1 optional field)
	if subpacketsMap != nil {
		if normalizedSp, err := NormalizeSubpackets(subpacketsMap); err != nil {
			s.logger.WarnContext(ownerCtx, LogBSSCIFailedToNormalizeDetachSubpackets, "error", err)
		} else {
			detachMsg.Subpackets = normalizedSp
		}
	}

	// Normalize payload for audit completeness (DET-03: convert numeric arrays to concrete types)
	normalizedPayload := normalizeDetachPayload(data)

	if s.protocolMessages != nil {
		if err := s.protocolMessages.CreateDetachMessage(ownerCtx, detachMsg, normalizedPayload); err != nil {
			s.logger.WarnContext(ownerCtx, LogBSSCIFailedToPersistDetachMessage,
				"error", err,
				"epEui", epEUI,
				"bsEui", session.BaseStationEUI,
				"opId", msg.OpId)
		}
	}

	// Process endpoint telemetry and signature validation if endpoint was found
	if endpoint != nil {
		// Cryptographic signature validation
		if s.config.DetachSignatureValidationEnabled {
			if err := ValidateDetachSignature(sign, endpoint.Sign); err != nil {
				s.logger.WarnContext(ownerCtx, LogBSSCIDetachSignatureValidationFailed,
					"epEui", pkgmioty.FormatEUI64(epEUI),
					"error", err)
				return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
			}
		}

		// Update endpoints table with detach telemetry (BSSCI §5.7.1 Finding 2)
		// Use UpdateFields to persist last_detach_* columns (Update() doesn't touch them)
		packetCntInt := int64(packetCntUint)
		propagateStatus := PropagateStatusDetachReceived
		updates := map[string]interface{}{
			"last_attached_bs_eui":   session.BaseStationEUIBytes(),
			"last_propagate_time":    rxTime,
			"last_detach_time":       rxTime,
			"last_detach_sign":       sign,
			"last_detach_packet_cnt": packetCntInt,
			"propagate_status":       propagateStatus,
		}
		if err := s.endpointRepo.UpdateFields(ownerCtx, ownerTenantID, endpoint.ID, updates); err != nil {
			s.logger.WarnContext(ownerCtx, LogBSSCIFailedToUpdateEndpointDetachTelemetry,
				"error", err,
				"epEui", epEUI)
		}
	}

	// Build detach response per BSSCI 3.7.2
	// Convert signature to Numeric[4] array format (sign is guaranteed to be 4 bytes here)
	numericSign := make([]interface{}, 4)
	for i := 0; i < 4; i++ {
		numericSign[i] = int(sign[i])
	}

	response := map[string]interface{}{
		"command": mioty.CmdDetachResponse,
		"opId":    msg.OpId,
		"sign":    numericSign, // Numeric[4] format per BSSCI 3.7.2
	}

	return s.sendMessage(session, response)
}

// handleAttachComplete handles attach complete operations per BSSCI 3.6.3
func (s *Server) handleAttachComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// Retrieve the pending attach operation
	// StatusService is the single path for pending operation persistence
	pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId))

	if err != nil || pendingOp == nil || pendingOp.OperationType != mioty.CmdAttach {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIReceivedAttCmpWithoutPendingAttach,
			"baseStation", session.BaseStationEUI,
			"opId", int64(msg.OpId))
		return fmt.Errorf("%s for opId %d", ResolveErrorMessage(errNoPendingAttachOperation), msg.OpId)
	}

	// Extract the endpoint EUI from pending operation
	epEUI, _ := parseMetadataEUI(pendingOp.Metadata["epEui"])

	// Log successful completion
	s.logger.InfoContext(s.safeCtx(), LogBSSCIAttachOperationCompletedSuccessfully,
		"baseStation", session.BaseStationEUI,
		"endPoint", epEUI,
		"opId", int64(msg.OpId),
		"duration", time.Since(pendingOp.CreatedAt))

	// Record event for successful attach
	if s.eventStore != nil {
		// Extract attach details from pending operation
		attachCnt, _ := pendingOp.Metadata["attachCnt"].(int64)
		rxTime, _ := pendingOp.Metadata["rxTime"].(int64)
		rssi, _ := pendingOp.Metadata["rssi"].(float64)
		snr, _ := pendingOp.Metadata["snr"].(float64)

		// Create simple event like other handlers do
		details := map[string]interface{}{
			"epEui":     pkgmioty.FormatEUI64(epEUI),
			"bsEui":     pkgmioty.FormatEUI64(session.BaseStationEUI),
			"attachCnt": attachCnt,
			"rxTime":    rxTime,
			"rssi":      rssi,
			"snr":       snr,
			"operation": "attach_complete",
		}

		detailsJSON, _ := json.Marshal(details)

		if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
			EventType:   models.EventTypeEndpointAttached,
			Category:    mioty.CategoryEndpoint,
			Severity:    SeverityInfo,
			Title:       fmt.Sprintf(models.EventTitleEndpointAttachedViaBS, pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI)),
			Description: "Endpoint successfully attached to network",
			Details:     detailsJSON,
			Status:      EventStatusNew,
			CreatedAt:   time.Now(),
		}); err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateAttachEvent, "error", err)
		}
	}

	// Publish attach event to MQTT (roaming-safe: uses endpoint owner org from pending metadata)
	if s.mqttPublisher != nil {
		var epOwnerOrg string
		if rawTenant, ok := pendingOp.Metadata["endpointTenantID"]; ok {
			var epTID int64
			switch v := rawTenant.(type) {
			case int64:
				epTID = v
			case float64:
				if v >= 0 {
					epTID = int64(v)
				}
			case int:
				epTID = int64(v)
			}
			if epTID > 0 && s.orgResolver != nil {
				if orgUUID, orgErr := s.orgResolver.GetDefaultOrgForTenant(s.safeCtx(), epTID); orgErr == nil && orgUUID != uuid.Nil {
					epOwnerOrg = orgUUID.String()
				}
			}
		}
		if epOwnerOrg != "" {
			go func() {
				if pubErr := s.mqttPublisher.PublishAttach(ctx, epOwnerOrg,
					epEUI, session.BaseStationEUI); pubErr != nil {
					s.logger.WarnContext(ctx, LogBSSCIFailedToPublishAttachEventToMQTT, "error", pubErr)
				}
			}()
		} else {
			s.logger.WarnContext(ctx, LogBSSCIMQTTPublishSkippedOrgUnresolved,
				"epEui", epEUI, "event", MQTTEventKeyAttach)
		}
	}

	// Clear the pending operation
	// BSSCI §§5.11-5.12.3 Gap 1: Use removePendingOperation helper (has dual-path logic)
	if err := s.removePendingOperation(session, int64(msg.OpId)); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
			"error", err,
			"opId", msg.OpId)
	}

	// BSSCI §5.8.2: Trigger automatic attach propagate to other base stations
	// Run asynchronously to avoid blocking the complete message response
	if s.propagationSvc != nil && epEUI != 0 {
		endpointIDForPropagate := pendingOp.Metadata["endpointID"]
		if endpointIDForPropagate != nil {
			// Extract endpoint ID with type conversion (handles JSON persistence)
			var epID int64
			switch v := endpointIDForPropagate.(type) {
			case int64:
				epID = v
			case float64:
				if v >= 0 {
					epID = int64(v)
				}
			case int:
				epID = int64(v)
			}

			if epID > 0 {
				// 3-tier endpoint lookup for roaming safety
				var endpoint *models.EndPoint
				var err error

				// Tier 1: Use stored endpoint tenant (new ops after deployment)
				var epTenantID int64
				if rawTenant, ok := pendingOp.Metadata["endpointTenantID"]; ok {
					switch v := rawTenant.(type) {
					case int64:
						epTenantID = v
					case float64:
						if v >= 0 {
							epTenantID = int64(v)
						}
					case int:
						epTenantID = int64(v)
					}
				}

				if epTenantID > 0 {
					endpoint, err = s.endpointRepo.GetByID(s.safeCtx(), epID, epTenantID)
				}

				// Tier 2: EUI lookup with session tenant (legacy ops)
				if endpoint == nil && epEUI != 0 {
					euiBytes := make([]byte, 8)
					binary.BigEndian.PutUint64(euiBytes, epEUI)
					sessionTenant := resolvedTenant(session, s.tenantID)
					endpoint, _ = s.endpointRepo.GetByEUI(s.safeCtx(), sessionTenant, euiBytes)
				}

				// Tier 3: Cross-tenant EUI lookup (roaming fallback)
				if endpoint == nil && epEUI != 0 {
					euiBytes := make([]byte, 8)
					binary.BigEndian.PutUint64(euiBytes, epEUI)
					var eui models.EUI
					copy(eui[:], euiBytes)
					endpoint, _ = s.endpointRepo.Get(s.safeCtx(), eui)
				}

				if err != nil || endpoint == nil {
					// Build basic context for error logging (fallback when endpoint not found)
					errorCtx := pkgcontext.WithTenantID(context.Background(), resolvedTenant(session, s.tenantID))
					s.logger.ErrorContext(errorCtx, LogBSSCIEndpointNotFoundForAttachPropagation,
						"endpoint_id", epID, "epEui", epEUI, "error", err)
				} else {
					go func(ep *models.EndPoint) {
						// Build owner context from endpoint's tenant (not BS session tenant)
						ownerCtx := pkgcontext.WithTenantID(context.Background(), ep.TenantID)

						// Resolve organization (pass ownerCtx for context propagation)
						if s.orgResolver != nil {
							orgUUID, err := s.orgResolver.GetDefaultOrgForTenant(ownerCtx, ep.TenantID)
							if err == nil && orgUUID != uuid.Nil {
								ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, orgUUID)
							}
						}

						// Trigger propagation with enriched owner context
						activeSessions := s.ConnectedSessionsSnapshot()
						if err := s.propagationSvc.TriggerEndpointPropagate(ownerCtx, epID, activeSessions); err != nil {
							s.logger.ErrorContext(ownerCtx, LogBSSCIAutomaticPropagationFailedAfterOTAAttach,
								"endpoint_id", epID, "error", err)
						}
					}(endpoint)
				}
			}
		}
	}

	// SCACI §3.13: Forward EPStatus to SCACI ACs after attach completion
	// Run asynchronously to not block BSSCI flow
	if s.scaciEPStatusBroadcaster != nil && epEUI != 0 {
		tenantID := resolvedTenant(session, s.tenantID)
		// Build owner context with tenant/org for proper context propagation
		ownerCtx := pkgcontext.WithTenantID(context.Background(), tenantID)
		if session.OrganizationID != uuid.Nil {
			ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, session.OrganizationID)
		}
		go func(ctx context.Context, eui uint64, tid int64) {
			epStatusData := &EPStatusData{
				EpEui:    eui,
				EpStatus: pkgmioty.EPStatusAttached, // SCACI §3.13.1
			}
			// Extract OTA fields from pending operation metadata if available
			if attachCnt, ok := pendingOp.Metadata["attachCnt"].(int64); ok && attachCnt >= 0 && attachCnt <= int64(^uint32(0)) {
				v := uint32(attachCnt) // #nosec G115 -- bounds checked above
				epStatusData.AttachCnt = &v
			}
			if snr, ok := pendingOp.Metadata["snr"].(float64); ok {
				epStatusData.Snr = &snr
			}
			if rssi, ok := pendingOp.Metadata["rssi"].(float64); ok {
				epStatusData.Rssi = &rssi
			}
			// SCACI §3.13.1: Extract nonce (4-byte array → mioty.Numeric4)
			if nonceBytes, ok := pendingOp.Metadata["nonce"].([]byte); ok && len(nonceBytes) == 4 {
				var nonce mioty.Numeric4
				copy(nonce[:], nonceBytes)
				epStatusData.Nonce = &nonce
			}
			// SCACI §3.13.1: Extract sign (4-byte array → mioty.Numeric4)
			if signBytes, ok := pendingOp.Metadata["sign"].([]byte); ok && len(signBytes) == 4 {
				var sign mioty.Numeric4
				copy(sign[:], signBytes)
				epStatusData.Sign = &sign
			}
			// SCACI §3.13.1: Extract eqSnr
			if eqSnr, ok := pendingOp.Metadata["eqSnr"].(float64); ok {
				epStatusData.EqSnr = &eqSnr
			}
			// SCACI §3.13.1: Extract subpackets ([]interface{} of maps → *mioty.Subpackets)
			if subpacketsRaw, ok := pendingOp.Metadata["subpackets"].([]interface{}); ok && len(subpacketsRaw) > 0 {
				subpackets := &mioty.Subpackets{}
				for _, sp := range subpacketsRaw {
					if spMap, ok := sp.(map[string]interface{}); ok {
						if snrVal, ok := spMap["snr"].(float64); ok {
							subpackets.SNR = append(subpackets.SNR, snrVal)
						}
						if rssiVal, ok := spMap["rssi"].(float64); ok {
							subpackets.RSSI = append(subpackets.RSSI, rssiVal)
						}
						if freqVal, ok := spMap["frequency"].(float64); ok {
							subpackets.Frequency = append(subpackets.Frequency, int64(freqVal))
						}
						if phaseVal, ok := spMap["phase"].(float64); ok {
							subpackets.Phase = append(subpackets.Phase, phaseVal)
						}
					}
				}
				if len(subpackets.SNR) > 0 || len(subpackets.RSSI) > 0 || len(subpackets.Frequency) > 0 {
					epStatusData.Subpackets = subpackets
				}
			}
			if err := s.scaciEPStatusBroadcaster.BroadcastEPStatus(ctx, tid, epStatusData); err != nil {
				s.logger.WarnContext(ctx, LogBSSCIEPStatusForwardFailed, "epEui", pkgmioty.FormatEUI64(eui), "error", err)
			}
		}(ownerCtx, epEUI, tenantID)
	}

	// No response needed for complete messages per BSSCI spec
	return nil
}

// handleDetachComplete handles detach complete operations per BSSCI 3.7.3
func (s *Server) handleDetachComplete(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// Retrieve the pending detach operation
	// StatusService is the single path for pending operation persistence
	pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId))

	if err != nil || pendingOp == nil || pendingOp.OperationType != mioty.CmdDetach {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIReceivedDetCmpWithoutPendingDetach,
			"baseStation", session.BaseStationEUI,
			"opId", int64(msg.OpId))
		return fmt.Errorf("%s for opId %d", ResolveErrorMessage(errNoPendingDetachOperation), msg.OpId)
	}

	// Reconstruct typed metadata for type safety (BSSCI §5.7.3)
	var typedMeta *detachMetadata
	if pendingOp.Metadata != nil {
		typedMeta = mapToDetachMetadata(pendingOp.Metadata)
	}

	// Extract the endpoint EUI from typed metadata or fallback to raw map
	var epEUI uint64
	if typedMeta != nil {
		epEUI = typedMeta.EpEui
	} else if pendingOp.Metadata != nil {
		// Fallback to raw map (handles legacy data or normalization failure)
		epEUI, _ = parseMetadataEUI(pendingOp.Metadata["epEui"])
	}
	if epEUI == 0 {
		// Final fallback to current message
		if epEuiRaw, ok := getUint64Field(data, "epEui"); ok {
			epEUI = epEuiRaw
		}
	}

	// Log successful completion
	s.logger.InfoContext(s.safeCtx(), LogBSSCIDetachOperationCompletedSuccessfully,
		"baseStation", session.BaseStationEUI,
		"endPoint", epEUI,
		"opId", int64(msg.OpId),
		"duration", time.Since(pendingOp.CreatedAt))

	// Update endpoint to clear attachment state on successful detach
	// Use stored endpoint.ID from metadata (avoids DB refetch)
	if s.endpointRepo != nil && epEUI != 0 {
		// Try to get endpoint ID from typed metadata
		var endpointID int64
		if typedMeta != nil {
			endpointID = typedMeta.EndpointID
		}

		if endpointID > 0 {
			// Extract detach telemetry from typed metadata (with fallbacks)
			var rxTime int64
			var packetCnt uint32
			var snr, rssi float64
			var sign []byte

			if typedMeta != nil {
				rxTime = typedMeta.RxTime
				packetCnt = typedMeta.PacketCnt
				snr = typedMeta.SNR
				rssi = typedMeta.RSSI
				sign = typedMeta.Signature
			} else {
				// Fallback to raw map extraction
				rxTime, _ = pendingOp.Metadata["rxTime"].(int64)
				if pc, ok := pendingOp.Metadata["packetCnt"].(float64); ok {
					packetCnt = uint32(pc)
				} else if pc, ok := pendingOp.Metadata["packetCnt"].(int64); ok {
					packetCnt = uint32(pc) //nolint:gosec // G115: Packet count from metadata, range validated by protocol
				}
				snr, _ = pendingOp.Metadata["snr"].(float64)
				rssi, _ = pendingOp.Metadata["rssi"].(float64)
				sign, _ = pendingOp.Metadata["sign"].([]byte)
			}

			// Use shared endpoint detach helper (reused by SCACI)
			telemetry := map[string]interface{}{
				"packetCnt": packetCnt,
			}
			if len(sign) == 4 {
				telemetry["sign"] = sign
			}

			if err := endpoint.DetachEndpoint(ctx, s.endpointRepo, resolvedTenant(session, s.tenantID), endpointID, telemetry); err != nil {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointDetachState, "error", err)
			}

			// Update radio metrics from detach using selective update
			var eui models.EUI
			epEUIBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(epEUIBytes, epEUI)
			copy(eui[:], epEUIBytes)

			// Extract optional fields from typed metadata with fallbacks
			var eqSnr float64
			var rxDuration *int64
			var profile *string

			if typedMeta != nil {
				eqSnr = snr // default to snr if not present
				if typedMeta.EqSnr != nil {
					eqSnr = *typedMeta.EqSnr
				}
				rxDuration = typedMeta.RxDuration
				profile = typedMeta.Profile
			} else {
				// Fallback to raw map extraction
				eqSnr, _ = pendingOp.Metadata["eqSnr"].(float64)
				if rxDur, ok := pendingOp.Metadata["rxDuration"].(int64); ok {
					rxDuration = &rxDur
				}
				if prof, ok := pendingOp.Metadata["profile"].(string); ok {
					profile = &prof
				}
			}

			update := interfaces.RadioMetricsUpdate{
				SNR:        snr,
				RSSI:       rssi,
				EqSNR:      eqSnr,
				RxTime:     rxTime,
				RxDuration: rxDuration,
				Profile:    profile,
			}

			if err := s.endpointRepo.UpdateRadioMetricsSelective(ctx, resolvedTenant(session, s.tenantID), eui, update); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateRadioMetrics, "error", err)
			}
		}
	}

	// Record event for successful detach
	if s.eventStore != nil && epEUI != 0 {
		details := map[string]interface{}{
			"epEui":     pkgmioty.FormatEUI64(epEUI), // Use hex string to avoid JSON precision loss
			"bsEui":     pkgmioty.FormatEUI64(session.BaseStationEUI),
			"operation": "detach_complete",
		}

		detailsJSON, _ := json.Marshal(details)

		if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
			EventType:   models.EventTypeEndpointDetached,
			Category:    mioty.CategoryEndpoint,
			Severity:    SeverityInfo,
			Title:       fmt.Sprintf(models.EventTitleEndpointDetachedViaBS, pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI)),
			Description: "Endpoint successfully detached from network",
			Details:     detailsJSON,
			Status:      EventStatusNew,
			CreatedAt:   time.Now(),
		}); err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateDetachEvent, "error", err)
		}
	}

	// Publish detach event to MQTT (roaming-safe: uses endpoint owner org from metadata)
	if s.mqttPublisher != nil {
		var ownerOrg string
		var epEuiPublish uint64

		if typedMeta != nil {
			if typedMeta.OrgUUID != uuid.Nil {
				ownerOrg = typedMeta.OrgUUID.String()
			}
			epEuiPublish = typedMeta.EpEui
		} else if pendingOp.Metadata != nil {
			epEuiPublish, _ = parseMetadataEUI(pendingOp.Metadata["epEui"])
			if orgStr, ok := pendingOp.Metadata["orgUuid"].(string); ok {
				if parsed, parseErr := uuid.Parse(orgStr); parseErr == nil && parsed != uuid.Nil {
					ownerOrg = orgStr
				}
			}
		}

		if epEuiPublish > 0 && ownerOrg != "" {
			go func() {
				if pubErr := s.mqttPublisher.PublishDetach(ctx, ownerOrg,
					epEuiPublish, session.BaseStationEUI); pubErr != nil {
					s.logger.WarnContext(ctx, LogBSSCIFailedToPublishDetachEventToMQTT, "error", pubErr)
				}
			}()
		} else if epEuiPublish > 0 {
			s.logger.WarnContext(ctx, LogBSSCIMQTTPublishSkippedOrgUnresolved,
				"epEui", epEuiPublish, "event", MQTTEventKeyDetach)
		}
	}

	// Clear the pending operation
	// BSSCI §§5.11-5.12.3 Gap 1: Use removePendingOperation helper (has dual-path logic)
	if err := s.removePendingOperation(session, int64(msg.OpId)); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
			"error", err,
			"opId", msg.OpId)
	}

	// SCACI §3.13: Forward EPStatus to SCACI ACs after detach completion
	// Run asynchronously to not block BSSCI flow
	if s.scaciEPStatusBroadcaster != nil && epEUI != 0 {
		tenantID := resolvedTenant(session, s.tenantID)
		// Build owner context with tenant/org for proper context propagation
		ownerCtx := pkgcontext.WithTenantID(context.Background(), tenantID)
		if session.OrganizationID != uuid.Nil {
			ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, session.OrganizationID)
		}
		go func(ctx context.Context, eui uint64, tid int64) {
			epStatusData := &EPStatusData{
				EpEui:    eui,
				EpStatus: pkgmioty.EPStatusDetached, // SCACI §3.13.1
			}
			// Extract OTA fields from pending operation metadata if available
			if typedMeta != nil {
				if typedMeta.SNR != 0 {
					snr := typedMeta.SNR
					epStatusData.Snr = &snr
				}
				if typedMeta.RSSI != 0 {
					rssi := typedMeta.RSSI
					epStatusData.Rssi = &rssi
				}
				// SCACI §3.13.1: Extract signature (4-byte array → mioty.Numeric4)
				if len(typedMeta.Signature) == 4 {
					var sign mioty.Numeric4
					copy(sign[:], typedMeta.Signature)
					epStatusData.Sign = &sign
				}
				// SCACI §3.13.1: Extract eqSnr
				if typedMeta.EqSnr != nil {
					epStatusData.EqSnr = typedMeta.EqSnr
				}
			}
			// Fallback: Extract from raw metadata if typedMeta doesn't have sign/eqSnr
			if pendingOp != nil && pendingOp.Metadata != nil {
				// Fallback for sign (uses "sign" key for raw-map format)
				if epStatusData.Sign == nil {
					if signBytes, ok := pendingOp.Metadata["sign"].([]byte); ok && len(signBytes) == 4 {
						var sign mioty.Numeric4
						copy(sign[:], signBytes)
						epStatusData.Sign = &sign
					}
				}
				// Fallback for eqSnr
				if epStatusData.EqSnr == nil {
					if eqSnr, ok := pendingOp.Metadata["eqSnr"].(float64); ok {
						epStatusData.EqSnr = &eqSnr
					}
				}
			}
			if err := s.scaciEPStatusBroadcaster.BroadcastEPStatus(ctx, tid, epStatusData); err != nil {
				s.logger.WarnContext(ctx, LogBSSCIEPStatusForwardFailed, "epEui", pkgmioty.FormatEUI64(eui), "error", err)
			}
		}(ownerCtx, epEUI, tenantID)
	}

	// No response needed for complete messages per BSSCI spec
	return nil
}

// handleULData handles uplink data operations
func (s *Server) handleULData(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// epEui is a full-range unsigned EUI-64 (BSSCI §5.10.1)
	epEUI, ok := getUint64Field(data, "epEui")
	if !ok {
		return fmt.Errorf("%s", ResolveErrorMessage(errMissingEpEui))
	}

	// Route by endpoint ownership before committing to ingest
	if s.dispositionResolver != nil {
		disposition, dispErr := s.dispositionResolver.Resolve(ctx, epEUI)
		if dispErr != nil {
			s.logger.ErrorContext(ctx, LogBSSCIDispositionResolutionFailedRejectingUplink,
				"ep_eui", epEUI, "error", dispErr)
			return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errFailedToPersistULData))
		}
		switch disposition {
		case DispositionDrop:
			// Endpoint unknown, no relay configured; accept silently to prevent BS retry storms
			s.logger.DebugContext(ctx, LogBSSCIEndpointNotFound,
				"ep_eui", epEUI)
			return s.sendMessage(session, map[string]interface{}{
				"command": mioty.CmdULDataResponse,
				"opId":    msg.OpId,
			})
		case DispositionRelay:
			if s.relayOutbox != nil {
				// Enqueue the raw BSSCI frame for CE→ECE relay.
				// Send ulDataRsp only if the insert succeeds; on failure send a BSSCI error
				// so the base station retries rather than silently losing the packet.
				rawFrame, _ := json.Marshal(data) // best-effort; frame already validated above
				_, enqErr := s.relayOutbox.Enqueue(ctx, epEUI, session.BaseStationEUI, rawFrame, time.Now().UnixNano())
				if enqErr != nil {
					s.logger.ErrorContext(ctx, LogBSSCIFailedToEnqueueRelayUplink,
						"ep_eui", epEUI, "error", enqErr)
					return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errFailedToPersistULData))
				}
			} else {
				// No relay configured yet; treat as drop
				s.logger.DebugContext(ctx, LogBSSCIEndpointNotFound, "ep_eui", epEUI)
			}
			return s.sendMessage(session, map[string]interface{}{
				"command": mioty.CmdULDataResponse,
				"opId":    msg.OpId,
			})
		case DispositionLocal:
			// Fall through to full ingest below
		}
	}

	// Get packet counter
	packetCnt, ok := getNumericField(data, "packetCnt")
	if !ok {
		return fmt.Errorf("%s", ResolveErrorMessage(errMissingPacketCnt))
	}

	// Handle userData using the normalized helper
	var userData []byte
	if userDataRaw, ok := data["userData"]; ok {
		userData = s.normalizeUserDataField(userDataRaw)
	} else {
		userData = []byte{} // Empty payload is valid per spec
	}

	// Validate mandatory signal quality metrics per BSSCI 3.10.1
	snr, snrValid := getFloatFieldValidated(data, "snr")
	if !snrValid {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSnrValue))
	}

	rssi, rssiValid := getFloatFieldValidated(data, "rssi")
	if !rssiValid {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRssiValue))
	}

	// Extract eqSnr if present and valid (optional field per BSSCI spec)
	// Use pointer to distinguish nil (not present) from 0.0 (valid value)
	var eqSnrPtr *float64
	if eqSnrVal, hasEqSnr := getNumericField(data, "eqSnr"); hasEqSnr {
		eqSnrFloat := float64(eqSnrVal)
		eqSnrPtr = &eqSnrFloat
	}

	// Validate rxTime as mandatory field before dedup (BSSCI 3.10.1)
	// Must validate before CheckAndRecord to prevent invalid packets from poisoning dedup cache
	rxTimeValue, hasRxTime := data["rxTime"]
	if !hasRxTime {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingRxTime))
	}

	var rxTime int64
	switch v := rxTimeValue.(type) {
	case int64:
		rxTime = v
	case uint64:
		if v > math.MaxInt64 {
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errRxTimeExceedsMaximum))
		}
		rxTime = int64(v)
	case float64:
		rxTime = int64(v)
	case int:
		rxTime = int64(v)
	case uint:
		rxTime = int64(v) //nolint:gosec // G115: BSSCI rxTime from protocol fits int64
	default:
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRxTimeFormat))
	}

	if rxTime <= 0 {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRxTimeValue))
	}

	// Extract optional fields for deduplicator (SCACI §3.8.1)
	var rxDurationPtr *int64
	if rxDuration, ok := getNumericField(data, "rxDuration"); ok {
		rxDurationPtr = &rxDuration
	}
	var profilePtr *string
	if profile, ok := data["profile"].(string); ok {
		profilePtr = &profile
	}
	var modePtr *string
	if mode, ok := data["mode"].(string); ok {
		modePtr = &mode
	}
	var subpacketsPtr *mioty.Subpackets
	if raw, ok := data["subpackets"].(map[string]interface{}); ok {
		if sp, err := NormalizeSubpackets(raw); err == nil {
			subpacketsPtr = sp
		}
	}

	// Extract additional fields for processing
	dlOpen := getBoolField(data, "dlOpen", false)
	responseExp := getBoolField(data, "responseExp", false)
	dlAck := getBoolField(data, "dlAck", false)

	// Delegate full pipeline to ingest service (dedup → persist → SCACI → MQTT).
	// Downlink dispatch is handled here since it requires the active BSSCI session object.
	if s.uplinkIngestSvc == nil {
		return fmt.Errorf("uplinkIngestSvc is required for handleULData")
	}
	{
		epEuiVal := epEUI
		packetCntVal, errToken := safeUint32(packetCnt, "packetCnt")
		if errToken != "" {
			return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errToken))
		}
		var formatPtr *uint8
		if fmtRaw, ok := getNumericField(data, "format"); ok {
			fmtVal, fmtErrToken := safeUint8(fmtRaw, "format")
			if fmtErrToken != "" {
				return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(fmtErrToken))
			}
			formatPtr = &fmtVal
		}
		payload := &UplinkPayload{
			EpEUI:       epEuiVal,
			BsEUI:       session.BaseStationEUI,
			PacketCnt:   packetCntVal,
			UserData:    userData,
			SNR:         snr,
			RSSI:        rssi,
			EqSNR:       eqSnrPtr,
			RxTime:      rxTime,
			RxDuration:  rxDurationPtr,
			Profile:     profilePtr,
			Mode:        modePtr,
			Format:      formatPtr,
			Subpackets:  subpacketsPtr,
			DLOpen:      dlOpen,
			ResponseExp: responseExp,
			DlAck:       dlAck,
		}
		result, ingestErr := s.uplinkIngestSvc.Ingest(ctx, payload, UplinkIngestOptions{
			Source:          UplinkSourceBSSCI,
			ServingTenantID: resolvedTenant(session, s.tenantID),
		})
		if ingestErr != nil {
			s.logger.ErrorContext(ctx, LogBSSCIUplinkIngestFailed,
				"ep_eui", epEuiVal, "packet_cnt", packetCntVal, "error", ingestErr)
			return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errFailedToPersistULData))
		}
		if dlOpen && s.downlinkDispatcher != nil {
			ownerCtx := pkgcontext.WithTenantID(ctx, result.OwnerTenantID)
			if result.OwnerOrgUUID != uuid.Nil {
				ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, result.OwnerOrgUUID)
			}
			dispatched, dispErr := s.downlinkDispatcher.DispatchIfAvailable(
				ownerCtx, result.OwnerTenantID, result.OwnerOrgUUID,
				session, epEuiVal, responseExp, dlAck,
			)
			if dispErr != nil {
				s.logger.WarnContext(ownerCtx, LogBSSCIDownlinkDispatchError,
					"epEui", epEuiVal, "error", dispErr)
			} else if dispatched {
				s.logger.InfoContext(ownerCtx, LogBSSCIDownlinkDispatched,
					"epEui", epEuiVal, "sessionID", session.ID, "ownerTenantId", result.OwnerTenantID)
			} else {
				s.logger.DebugContext(ownerCtx, LogBSSCIDownlinkWindowOpen,
					"epEui", epEuiVal, "responseExp", responseExp, "dlAck", dlAck)
			}
		}
		return s.sendMessage(session, map[string]interface{}{
			"command": mioty.CmdULDataResponse,
			"opId":    msg.OpId,
		})
	}
}

// handleULDataComplete handles UL data completion (BSSCI-3.10.3)
func (s *Server) handleULDataComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	s.logger.DebugContext(s.safeCtx(), LogBSSCIReceivedULDataCompletion,
		"sessionID", session.ID,
		"opId", msg.OpId)

	// No response required for completion message (BSSCI spec)
	return nil
}

// handleError handles error messages from Base Station
func (s *Server) handleError(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	code := getNumericFieldInt(data, "code", -1)
	message := getStringField(data, "message", "unknown error")

	s.logger.ErrorContext(s.safeCtx(), LogBSSCIBaseStationReportedError,
		"code", code,
		"message", message,
		"baseStation", session.BaseStationEUI)

	// Handshake error routing (BSSCI §5.17): an error with opId 0 while
	// waiting for conCmp means the base station rejected the offered version
	// or connect response. The service center acknowledges with errorAck,
	// which completes the failed connect operation, and closes.
	if msg.OpId == 0 && session.ConnectState == ConnectStateAwaitingConnectComplete {
		errorAckMsg := map[string]interface{}{
			"command": mioty.CmdErrorAck,
			"opId":    msg.OpId,
		}
		if sendErr := s.sendMessage(session, errorAckMsg); sendErr != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorAck, "error", sendErr)
		}
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf("base station rejected connect response: code=%d %s", code, message)
	}

	// BSSCI §5.17: an inbound error is answered ONLY with errorAck - never with
	// an operation-specific *Cmp, and the operation type is never guessed. The
	// error and errorAck replace the normal response/completion sequence.
	errorAckMsg := map[string]interface{}{
		"command": mioty.CmdErrorAck,
		"opId":    msg.OpId,
	}
	if sendErr := s.sendMessage(session, errorAckMsg); sendErr != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorAck, "error", sendErr)
		return sendErr
	}

	// Typed compensation only for an operation this service center is actually
	// tracking (an errored operation is finalized without updating domain
	// state). An unmatched error is acknowledged and audited but touches no
	// unrelated pending state.
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		if op, lookupErr := s.statusSvc.GetPendingOperation(session, int64(msg.OpId)); lookupErr == nil {
			pendingOp = op
		}
	}
	if pendingOp != nil {
		s.logger.InfoContext(s.safeCtx(), LogBSSCIErrorOperationHandshakeCompletedDatabaseNotUpdated,
			"opId", msg.OpId,
			"operationType", pendingOp.OperationType)
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperationFromDatabase,
				"error", err,
				"opId", msg.OpId)
		}
	}

	return nil
}

// handleErrorAck handles the errorAck message (BSSCI §3.17.2)
// This acknowledges error reception from the base station and allows operation recovery
func (s *Server) handleErrorAck(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	s.logger.InfoContext(s.safeCtx(), LogBSSCIReceivedErrorAckFromBaseStation,
		"opId", msg.OpId,
		"baseStationEUI", session.BaseStationEUI)

	// Handshake errorAck routing (BSSCI §5.17): the acknowledgement completes
	// the failed connect operation; the connection closes
	if msg.OpId == 0 && session.ConnectState == ConnectStateAwaitingConnectErrorAck {
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf("connect operation failed and was acknowledged by the base station")
	}

	// An errorAck for the connect operation outside the awaiting state is a
	// protocol-ordering violation
	if msg.OpId == 0 && session.ConnectState != ConnectStateComplete {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidHandshakeState)
	}

	// An errorAck is only meaningful when this service center actually sent an
	// error frame for that operation on this connection (BSSCI rev1 §5.17 /
	// classic §3.17). Consuming the recorded expectation prevents a spurious
	// or forged errorAck from finalizing an unrelated in-flight operation.
	disposition, awaited := session.consumePendingErrorAck(msg.OpId)
	if !awaited {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIUnsolicitedErrorAck,
			"opId", msg.OpId,
			"baseStationEUI", session.BaseStationEUI)
		return nil
	}

	// Only an error that replaced a pending SC operation's normal sequence may
	// finalize that operation; ack-only errors touch no pending state.
	if disposition == errorAckFinalizePendingOperation && msg.OpId < 0 {
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperationAfterErrorAck,
				"error", err,
				"opId", msg.OpId)
			// Note: No manual fallback - trust StatusService single-writer pattern
		}
	}

	// An errorAck that completes an error sent during the connect handshake
	// finishes the failed exchange: the state machine goes Terminal and the
	// connection closes (BSSCI rev1 §5.17 / classic §3.17)
	if session.ConnectState == ConnectStateAwaitingConnectErrorAck {
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf("connect-stage error acknowledged by the base station; closing")
	}

	return nil
}

// Helper functions

// trimJSONWhitespace strips leading ASCII whitespace and UTF-8 BOM from a byte slice.
// Per RFC 8259, JSON allows insignificant whitespace before or after any token.
// Returns the trimmed slice without modifying the original.
func trimJSONWhitespace(data []byte) []byte {
	// Strip UTF-8 BOM (0xEF 0xBB 0xBF) if present
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	// Strip leading ASCII whitespace (space, tab, newline, carriage return)
	start := 0
	for start < len(data) {
		b := data[start]
		if b == 0x20 || b == 0x09 || b == 0x0A || b == 0x0D {
			start++
		} else {
			break
		}
	}

	return data[start:]
}

// detectEncoding performs a non-mutating syntax check to detect message encoding
// Per BSSCI Section 1, messages can be either JSON or MessagePack
// Returns "json" or "msgpack" based on first byte inspection
// Falls back to configDefault for empty or ambiguous payloads
func detectEncoding(rawFrame []byte, configDefault string) string {
	if len(rawFrame) == 0 {
		return configDefault // Use configured default for empty frames
	}

	// Trim leading whitespace and BOM for JSON detection per RFC 8259
	trimmed := trimJSONWhitespace(rawFrame)
	if len(trimmed) == 0 {
		return configDefault
	}

	firstByte := trimmed[0]

	// JSON detection: objects start with '{', arrays with '['
	if firstByte == '{' || firstByte == '[' {
		return EncodingJSON
	}

	// MessagePack detection (map markers):
	// - fixmap: 0x80-0x8f (10000000 to 10001111)
	// - map 16: 0xde
	// - map 32: 0xdf
	// BSSCI messages are typically objects, so we check map markers
	if (firstByte >= 0x80 && firstByte <= 0x8f) || firstByte == 0xde || firstByte == 0xdf {
		return EncodingMessagePack
	}

	// Use configured default for ambiguous payloads
	return configDefault
}

// encodeMessage encodes a message using the specified encoding
// Per BSSCI Section 1, supports both JSON and MessagePack encoding
func encodeMessage(msg interface{}, encoding string) ([]byte, error) {
	switch encoding {
	case EncodingJSON:
		return json.Marshal(msg)
	case EncodingMessagePack:
		return msgpack.Marshal(msg)
	default:
		// Default to msgpack per BSSCI spec
		return msgpack.Marshal(msg)
	}
}

// decodeMessage decodes a raw frame using the specified encoding
// Per BSSCI Section 1, supports both JSON and MessagePack encoding
func decodeMessage(rawFrame []byte, encoding string) (map[string]interface{}, error) {
	var data map[string]interface{}

	switch encoding {
	case EncodingJSON:
		return decodeJSONFrame(rawFrame)
	case EncodingMessagePack:
		if err := msgpack.Unmarshal(rawFrame, &data); err != nil {
			return nil, err
		}
		return data, nil
	default:
		// Default to msgpack per BSSCI spec
		if err := msgpack.Unmarshal(rawFrame, &data); err != nil {
			return nil, err
		}
		return data, nil
	}
}

// decodeJSONFrame decodes a JSON-encoded BSSCI frame strictly: numbers are
// preserved as json.Number (the full uint64 EUI range survives decoding), the
// frame must contain exactly one JSON object, and trailing content is rejected.
func decodeJSONFrame(rawFrame []byte) (map[string]interface{}, error) {
	// Trim BOM and leading whitespace per RFC 8259
	trimmed := trimJSONWhitespace(rawFrame)

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()

	var data map[string]interface{}
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: trailing content after JSON frame", ResolveErrorMessage(errInvalidMessageFormat))
	}
	return data, nil
}

// normalizeStrictDecodedMap converts a strict-decoded (UseNumber) map into the
// value shapes the legacy decoder produced: integral numbers within the exact
// float64 range become float64 (so existing consumers keep working), while
// integers beyond that range stay exact as int64/uint64 - this is what
// preserves full-range EUI-64 and counter values across resume.
func normalizeStrictDecodedMap(m map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		m[k] = normalizeStrictDecodedValue(v)
	}
	return m
}

func normalizeStrictDecodedValue(v interface{}) interface{} {
	switch t := v.(type) {
	case json.Number:
		if i, err := jsonNumberToInt64(t); err == nil {
			if i >= -int64(maxExactFloat64Integer) && i <= int64(maxExactFloat64Integer) {
				return float64(i)
			}
			return i
		}
		if u, err := jsonNumberToUint64(t); err == nil {
			return u
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t
	case map[string]interface{}:
		return normalizeStrictDecodedMap(t)
	case []interface{}:
		for i, e := range t {
			t[i] = normalizeStrictDecodedValue(e)
		}
		return t
	default:
		return v
	}
}

func getStringField(data map[string]interface{}, key string, defaultValue string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return defaultValue
}

// getNumericField extracts a signed 64-bit protocol field. Unsigned overflow,
// non-integral floats, and float magnitudes beyond the exact integer range are
// rejected via the canonical numeric coercion (coerceInt64).
func getNumericField(data map[string]interface{}, key string) (int64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}
	v, err := coerceInt64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// getUint64Field extracts an unsigned 64-bit protocol field (e.g. an EUI-64),
// preserving the full uint64 range including values above INT64_MAX. Negative
// values, non-integral floats, and float magnitudes beyond the exact integer
// range are rejected via the canonical numeric coercion (coerceUint64).
func getUint64Field(data map[string]interface{}, key string) (uint64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}
	v, err := coerceUint64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// parseOpID validates and extracts operation ID from interface{} value.
// Per BSSCI §5.2, operation IDs must be precise 64-bit integers; the canonical
// numeric coercion rejects non-integral floats and uint64 overflow.
// Returns (value, true) on success or (0, false) on validation failure.
func parseOpID(value interface{}) (int64, bool) {
	v, err := coerceInt64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

func getBoolField(data map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return defaultValue
}

func getNumericFieldInt(data map[string]interface{}, key string, defaultValue int) int {
	if val, ok := getNumericField(data, key); ok {
		return int(val)
	}
	return defaultValue
}

// getFloatFieldValidated extracts a float field and validates it's actually
// numeric via the canonical numeric coercion (coerceFloat64).
// Returns the value and true if field exists and is numeric, or 0 and false otherwise
func getFloatFieldValidated(data map[string]interface{}, key string) (float64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}
	v, err := coerceFloat64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// validateByteArray validates a MessagePack byte array (e.g., 4-byte nonce/sign).
// Returns (validBytes, "") on success or (nil, errToken) on validation failure.
// Accepts []byte directly or []interface{} (MessagePack numeric array format).
// All array elements must be integers in range 0-255.
func validateByteArray(data interface{}, fieldName string, expectedLen int) ([]byte, string) {
	switch v := data.(type) {
	case []byte:
		if len(v) != expectedLen {
			// Return appropriate error token based on field name
			if fieldName == fieldNameNonce {
				return nil, errInvalidNonceElement
			}
			return nil, errInvalidSignElement
		}
		return v, ""
	case []interface{}:
		if len(v) != expectedLen {
			if fieldName == fieldNameNonce {
				return nil, errInvalidNonceElement
			}
			return nil, errInvalidSignElement
		}
		result := make([]byte, expectedLen)
		for i, elem := range v {
			// Canonical numeric coercion enforces integer 0-255 range
			b, err := numericToByte(elem)
			if err != nil {
				if fieldName == fieldNameNonce {
					return nil, errInvalidNonceElement
				}
				return nil, errInvalidSignElement
			}
			result[i] = b
		}
		return result, ""
	default:
		// Invalid type
		if fieldName == fieldNameNonce {
			return nil, errInvalidNonceElement
		}
		return nil, errInvalidSignElement
	}
}

// detachMetadataToMap converts typed detachMetadata to map for JSON persistence.
// Prevents type drift by explicitly converting uint64/uint32/[]byte before marshal.
func detachMetadataToMap(m *detachMetadata) map[string]interface{} {
	result := map[string]interface{}{
		"epEui":            pkgmioty.FormatEUI64(m.EpEui),
		"endpointID":       m.EndpointID,
		"packetCnt":        m.PacketCnt,
		"signature":        m.Signature,
		"rxTime":           m.RxTime,
		"snr":              m.SNR,
		"rssi":             m.RSSI,
		"tenantId":         m.TenantID,
		"validationStatus": m.ValidationStatus, // Issue #4: signature validation tracking for crash-safe resume
	}
	// Store orgUuid only when valid (not nil UUID)
	if m.OrgUUID != uuid.Nil {
		result["orgUuid"] = m.OrgUUID.String()
	}
	if m.EqSnr != nil {
		result["eqSnr"] = *m.EqSnr
	}
	if m.Profile != nil {
		result["profile"] = *m.Profile
	}
	if m.RxDuration != nil {
		result["rxDuration"] = *m.RxDuration
	}
	return result
}

// mapToDetachMetadata reconstructs typed detachMetadata from JSON-decoded map.
// Normalizes float64 → int64 → uint32/uint64 and validates 4-byte signature.
// Returns nil if required fields missing or signature invalid.
func mapToDetachMetadata(m map[string]interface{}) *detachMetadata {
	if m == nil {
		return nil
	}

	// Extract required fields with type safety
	epEui, ok := parseMetadataEUI(m["epEui"])
	if !ok {
		return nil
	}

	// Extract endpointID (optional for backward compatibility with old pending operations)
	// Canonical numeric coercion accepts both legacy float64-decoded values and
	// the strict json.Number decode used on resume (exact uint64/int64 range).
	var endpointID int64
	if id, err := coerceInt64(m["endpointID"]); err == nil {
		endpointID = id
	}

	packetCntInt, err := coerceInt64(m["packetCnt"])
	if err != nil {
		return nil
	}
	rxTime, err := coerceInt64(m["rxTime"])
	if err != nil {
		return nil
	}
	snr, err := coerceFloat64(m["snr"])
	if err != nil {
		return nil
	}
	rssi, err := coerceFloat64(m["rssi"])
	if err != nil {
		return nil
	}

	// Validate and extract signature (must be exactly 4 bytes)
	var signature []byte
	switch sig := m["signature"].(type) {
	case []byte:
		if len(sig) != 4 {
			return nil // Invalid signature length
		}
		signature = sig
	case string:
		// Base64-encoded signature (JSON round-trip artifact)
		decoded, err := base64.StdEncoding.DecodeString(sig)
		if err != nil || len(decoded) != 4 {
			return nil
		}
		signature = decoded
	default:
		return nil
	}

	// Build typed metadata with safe conversions (using conversions.go helpers)
	packetCnt, errToken := safeUint32(packetCntInt, "packetCnt")
	if errToken != "" {
		return nil // Failed conversion
	}

	result := &detachMetadata{
		EpEui:      epEui,
		EndpointID: endpointID,
		PacketCnt:  packetCnt,
		Signature:  signature,
		RxTime:     rxTime,
		SNR:        snr,
		RSSI:       rssi,
	}

	// Extract tenantId (optional for backward compatibility with old pending operations)
	if tenantID, err := coerceInt64(m["tenantId"]); err == nil {
		result.TenantID = tenantID
	}

	// Extract orgUuid (optional for backward compatibility)
	if orgUuidStr, ok := m["orgUuid"].(string); ok {
		if parsedUUID, err := uuid.Parse(orgUuidStr); err == nil {
			result.OrgUUID = parsedUUID
		}
	}

	// Optional fields
	if eqSnr, err := coerceFloat64(m["eqSnr"]); err == nil && m["eqSnr"] != nil {
		result.EqSnr = &eqSnr
	}
	if profileStr, ok := m["profile"].(string); ok {
		result.Profile = &profileStr
	}
	if rxDur, err := coerceInt64(m["rxDuration"]); err == nil && m["rxDuration"] != nil {
		result.RxDuration = &rxDur
	}

	// Extract validationStatus (Issue #4: optional for backward compatibility with old pending operations)
	if validationStatus, ok := m["validationStatus"].(string); ok {
		result.ValidationStatus = validationStatus
	} else {
		// Default to "validated" for backward compatibility with operations persisted before Issue #4
		result.ValidationStatus = ValidationStatusValidated
	}

	return result
}

func parseMetadataEUI(value interface{}) (uint64, bool) {
	if s, ok := value.(string); ok {
		parsed, err := validation.ParseEUI(s)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	v, err := coerceUint64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// normalizeDetachPayload converts raw detach message payload to concrete types for JSONB storage.
// Ensures epEui/bsEui/packetCnt/rxTime are uint64/uint32/int64 (not float64), and sign is []byte (not []interface{}).
// Required for DET-03: spec-compliant audit payload preservation (BSSCI §5.7.1).
func normalizeDetachPayload(data map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{})

	for k, v := range data {
		switch k {
		case "sign":
			// Convert []interface{} to []byte for JSONB storage
			if arr, ok := v.([]interface{}); ok {
				bytes := make([]byte, len(arr))
				for i, val := range arr {
					if num, ok := val.(float64); ok {
						bytes[i] = byte(num)
					}
				}
				normalized[k] = bytes
			} else if bytes, ok := v.([]byte); ok {
				normalized[k] = bytes
			} else {
				normalized[k] = v
			}

		case "epEui", "bsEui":
			// Ensure uint64 not float64
			if num, ok := v.(float64); ok {
				normalized[k] = uint64(num)
			} else {
				normalized[k] = v
			}

		case "packetCnt":
			// Ensure uint32 not float64
			if num, ok := v.(float64); ok {
				normalized[k] = uint32(num)
			} else {
				normalized[k] = v
			}

		case "rxTime":
			// Ensure int64 not float64
			if num, ok := v.(float64); ok {
				normalized[k] = int64(num)
			} else {
				normalized[k] = v
			}

		case "rssi", "snr", "eqSnr":
			// Keep as float64 but ensure concrete type
			if num, ok := v.(float64); ok {
				normalized[k] = num
			} else {
				normalized[k] = v
			}

		default:
			normalized[k] = v
		}
	}

	return normalized
}

// NormalizeSubpackets converts raw subpackets map to typed Subpackets struct.
// Reusable helper for attach, detach, UL data handlers, and federation frame parsing (BSSCI §5.6.1, §5.7.1, §5.10.1).
// Returns (subpackets, nil) on success or (nil, error) if no valid arrays found.
func NormalizeSubpackets(raw map[string]interface{}) (*mioty.Subpackets, error) {
	sp := &mioty.Subpackets{}

	// Extract SNR array (signal to noise ratio in dB)
	if snrVals, ok := raw["snr"].([]interface{}); ok {
		if extracted, valid := extractFloatSlice(snrVals); valid {
			sp.SNR = extracted
		}
	}

	// Extract RSSI array (signal strength in dBm)
	if rssiVals, ok := raw["rssi"].([]interface{}); ok {
		if extracted, valid := extractFloatSlice(rssiVals); valid {
			sp.RSSI = extracted
		}
	}

	// Extract frequency array (Hz)
	if freqVals, ok := raw["frequency"].([]interface{}); ok {
		if extracted, valid := extractInt64Slice(freqVals); valid {
			sp.Frequency = extracted
		}
	}

	// Extract phase array (degrees ±180)
	if phaseVals, ok := raw["phase"].([]interface{}); ok {
		if extracted, valid := extractFloatSlice(phaseVals); valid {
			sp.Phase = extracted
		}
	}

	// Validate at least one measurement array is present
	if len(sp.SNR)+len(sp.RSSI)+len(sp.Frequency)+len(sp.Phase) == 0 {
		return nil, errors.New("no valid subpacket arrays found")
	}

	return sp, nil
}

// extractFloatSlice validates and extracts float64 slice from a wire array
// via the canonical numeric coercion (coerceFloat64).
// Rejects non-numeric values. Returns (slice, true) on success or (nil, false) on failure.
func extractFloatSlice(values []interface{}) ([]float64, bool) {
	result := make([]float64, len(values))
	for i, v := range values {
		f, err := coerceFloat64(v)
		if err != nil {
			// Non-numeric value - reject entire array
			return nil, false
		}
		result[i] = f
	}
	return result, true
}

// extractInt64Slice validates and extracts int64 slice from a wire array via
// the canonical numeric coercion (coerceInt64).
// Rejects non-integers and non-numeric values. Returns (slice, true) on success or (nil, false) on failure.
func extractInt64Slice(values []interface{}) ([]int64, bool) {
	result := make([]int64, len(values))
	for i, v := range values {
		n, err := coerceInt64(v)
		if err != nil {
			// Non-numeric or non-integer value - reject entire array
			return nil, false
		}
		result[i] = n
	}
	return result, true
}

// normalizeAttPrpMessage reconstructs an attPrp message with correct BSSCI types
func normalizeAttPrpMessage(msg map[string]interface{}) map[string]interface{} {
	// Start with original map to preserve unknown/optional fields
	normalized := make(map[string]interface{})
	for k, v := range msg {
		normalized[k] = v
	}

	// Command stays as string
	if cmd, ok := msg["command"].(string); ok {
		normalized["command"] = cmd
	}

	// opId must be int64 per BSSCI 3.2 (supports full 64-bit range)
	if opId, ok := msg["opId"].(float64); ok {
		normalized["opId"] = int64(opId)
	}

	// epEui must be uint64 per BSSCI 3.8.1
	if epEui, ok := msg["epEui"].(float64); ok {
		normalized["epEui"] = uint64(epEui)
	}

	// Helper to coerce boolean fields (handles bool, float64 0/1)
	getBool := func(key string) bool {
		if val, ok := msg[key].(bool); ok {
			return val
		}
		// Fallback for older data that might store booleans as numbers
		if val, ok := msg[key].(float64); ok {
			return val != 0
		}
		return false
	}

	// Boolean fields with coercion support
	normalized["bidi"] = getBool("bidi")
	normalized["dualChan"] = getBool("dualChan")
	normalized["repetition"] = getBool("repetition")
	normalized["wideCarrOff"] = getBool("wideCarrOff")
	normalized["longBlkDist"] = getBool("longBlkDist")

	// nwkSnKey must be Numeric[16] array - ALWAYS 16 elements
	normalizedKey := make([]interface{}, 16)
	if keyArray, ok := msg["nwkSnKey"].([]interface{}); ok {
		for i := 0; i < 16; i++ {
			if i < len(keyArray) {
				if val, ok := keyArray[i].(float64); ok {
					normalizedKey[i] = uint8(val)
				} else {
					normalizedKey[i] = uint8(0)
				}
			} else {
				normalizedKey[i] = uint8(0) // Pad with zeros
			}
		}
	} else {
		// Fill with zeros if missing
		for i := 0; i < 16; i++ {
			normalizedKey[i] = uint8(0)
		}
	}
	normalized["nwkSnKey"] = normalizedKey

	// shAddr must be uint16 per BSSCI spec
	if shAddr, ok := msg["shAddr"].(float64); ok {
		normalized["shAddr"] = uint16(shAddr)
	}

	// lastPacketCnt must be uint32
	if packetCnt, ok := msg["lastPacketCnt"].(float64); ok {
		normalized["lastPacketCnt"] = uint32(packetCnt)
	}

	return normalized
}

// normalizeDetPrpMessage reconstructs a detPrp message with correct types
func normalizeDetPrpMessage(msg map[string]interface{}) map[string]interface{} {
	// Start with original to preserve optional fields
	normalized := make(map[string]interface{})
	for k, v := range msg {
		normalized[k] = v
	}

	// Fix required numeric types
	if opId, ok := msg["opId"].(float64); ok {
		normalized["opId"] = int64(opId)
	}
	if epEui, ok := msg["epEui"].(float64); ok {
		normalized["epEui"] = uint64(epEui)
	}

	return normalized
}

// normalizeMessageTypes fixes numeric types after JSON unmarshaling
func (s *Server) normalizeMessageTypes(msg map[string]interface{}, opType string) map[string]interface{} {
	switch opType {
	case mioty.CmdAttachPropagate:
		return normalizeAttPrpMessage(msg)
	case mioty.CmdDetachPropagate:
		return normalizeDetPrpMessage(msg)
	case mioty.CmdDLDataRevoke:
		return normalizeDlDataRevMessage(msg)
	default:
		// For unknown types, at minimum fix opId to int64
		normalized := make(map[string]interface{})
		for k, v := range msg {
			normalized[k] = v
		}
		if opId, ok := msg["opId"].(float64); ok {
			normalized["opId"] = int64(opId)
		}
		return normalized
	}
}

// normalizeDlDataRevMessage fixes numeric types for dlDataRev messages
func normalizeDlDataRevMessage(msg map[string]interface{}) map[string]interface{} {
	// Start with all existing fields to preserve any future optional data
	normalized := make(map[string]interface{})
	for k, v := range msg {
		normalized[k] = v
	}

	// Ensure command is present
	normalized["command"] = "dlDataRev"

	// Coerce numeric types to ensure proper MessagePack encoding
	if opId, ok := msg["opId"].(float64); ok {
		normalized["opId"] = int64(opId)
	}

	if epEui, ok := msg["epEui"].(float64); ok {
		normalized["epEui"] = uint64(epEui)
	}

	// queId should be uint64 per canonical MIOTY type
	if queId, ok := msg["queId"].(float64); ok {
		normalized["queId"] = uint64(queId)
	} else if queId, ok := msg["queId"].(int64); ok {
		// Handle int64 input and convert to uint64
		queIdSafe, errToken := safeUint64(queId, "queId")
		if errToken != "" {
			// NOTE: Normalization function has no logger access, overflow handled with 0 default
			// Downstream protocol validation will catch invalid queId=0
			queIdSafe = 0
		}
		normalized["queId"] = queIdSafe
	}

	return normalized
}

// normalizeUserDataField converts various MessagePack Numeric[n] formats to []byte
func (s *Server) normalizeUserDataField(userDataRaw interface{}) []byte {
	switch v := userDataRaw.(type) {
	case []byte:
		return v
	case []interface{}:
		// Convert Numeric[n] to []byte
		userData := make([]byte, len(v))
		for i, elem := range v {
			// Canonical numeric coercion enforces integer 0-255 range
			b, err := numericToByte(elem)
			if err != nil {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIUnsupportedUserDataElementType,
					"index", i,
					"type", fmt.Sprintf("%T", elem))
				userData[i] = 0
				continue
			}
			userData[i] = b
		}
		return userData
	case string:
		// Strings might be base64 encoded - log and skip
		s.logger.WarnContext(s.safeCtx(), LogBSSCIReceivedStringUserData,
			"value", v)
		return []byte{}
	default:
		// Log unknown type
		s.logger.WarnContext(s.safeCtx(), LogBSSCIUnknownUserDataType,
			"type", fmt.Sprintf("%T", v))
		return []byte{}
	}
}

// SendAttachPropagateToSession implements AttachPropagateSender interface
// Called by propagation service/reconciler with full session and endpoint context
// Normalizes nullable endpoint fields and delegates to existing SendAttachPropagate
//
// BSSCI §5.8-5.8.3: Automatic endpoint propagation
func (s *Server) SendAttachPropagateToSession(
	_ context.Context, // ownerCtx kept on the interface for tenant/org propagation; this impl delegates to a context-less SendAttachPropagate
	session *Session,
	endpoint *models.EndPoint,
) error {
	// Normalize nullable short address (default 0 when nil)
	// Model now stores uint16, no conversion needed
	var shAddr uint16
	if endpoint.ShAddr != nil {
		shAddr = *endpoint.ShAddr
	}

	// LastPacketCnt is uint32 (not nullable) - direct access
	lastPkt := endpoint.LastPacketCnt

	// Validate network key length
	if len(endpoint.NwkSnKey) != 16 {
		return fmt.Errorf("endpoint %d nwkSnKey length %d != 16", endpoint.ID, len(endpoint.NwkSnKey))
	}

	// Convert boolean to uint8 (inline pattern from existing code)
	repetition := uint8(0)
	if endpoint.Repetition {
		repetition = uint8(1)
	}

	// Delegate to existing SendAttachPropagate with all validated fields
	return s.SendAttachPropagate(
		session.ID,
		endpoint.EUI.ToUint64(), // Use EUI.ToUint64() method
		endpoint.NwkSnKey,       // []byte slice
		shAddr,
		endpoint.Bidi,
		lastPkt,
		endpoint.DualChan,
		repetition,
		endpoint.WideCarrOff,
		endpoint.LongBlkDist,
	)
}

// SendAttachPropagateBySessionID implements interface method for propagation service
// Looks up session by ID and delegates to SendAttachPropagateToSession
//
// BSSCI §5.8-5.8.3: Automatic endpoint propagation
func (s *Server) SendAttachPropagateBySessionID(
	ctx context.Context,
	sessionID string,
	endpoint *models.EndPoint,
) error {
	// Session-specific propagation (BSSCI §5.8.3)
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Delegate to SendAttachPropagateToSession with context as first parameter
	return s.SendAttachPropagateToSession(ctx, session, endpoint)
}

// SendAttachPropagate sends an attach propagate command to a specific session
func (s *Server) SendAttachPropagate(sessionID string, endpointEUI uint64, nwkSnKey []byte,
	shortAddr uint16, bidirectional bool, lastPacketCnt uint32, dualChannel bool,
	repetition uint8, wideCarrOff bool, longBlkDist bool) error {

	// BSSCI-3.8.1-01: Validate nwkSnKey is exactly 16 bytes
	if len(nwkSnKey) != 16 {
		return fmt.Errorf("%s, got %d bytes", ResolveErrorMessage(errInvalidNwkSnKeyLength), len(nwkSnKey))
	}

	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	ctx := s.sessionContext(session)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// BSSCI-3.3-03: Don't send operations to sessions that haven't completed handshake
	if !session.HandshakeComplete {
		return fmt.Errorf("%s for session %s", ResolveErrorMessage(errHandshakeNotComplete), sessionID)
	}

	// Check if we should propagate to this base station
	// For downlink to work, BOTH the base station AND endpoint must support bidirectional
	// If the endpoint doesn't support bidirectional, no point propagating to any base station
	// If the base station doesn't support bidirectional, surface the issue to operators
	if bidirectional && !session.Bidirectional {
		s.logger.WarnContext(s.safeCtx(), LogBSSCICannotAttachPropagateToNonBidiBaseStation,
			"sessionID", sessionID,
			"bsEui", session.BaseStationEUI,
			"epEui", endpointEUI)

		if s.eventStore != nil {
			details := map[string]interface{}{
				"epEui":  pkgmioty.FormatEUI64(endpointEUI), // Use hex string for event details
				"bsEui":  pkgmioty.FormatEUI64(session.BaseStationEUI),
				"reason": "Base station does not support bidirectional operation",
			}
			detailsJSON, _ := json.Marshal(details)

			if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
				TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
				EventType:   EventTypeAttachOperationFailed,
				Category:    mioty.CategoryEndpoint,
				Severity:    SeverityError,
				Title:       fmt.Sprintf("Attach propagate failed for endpoint %s", pkgmioty.FormatEUI64(endpointEUI)),
				Description: "Target base station is not bidirectional",
				Details:     detailsJSON,
				Status:      EventStatusNew,
				SourceType:  mioty.SourceTypeEndpoint,
				SourceName:  pkgmioty.FormatEUI64(endpointEUI),
				CreatedAt:   time.Now(),
			}); err != nil {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateNonBidiAttachFailureEvent, "error", err)
			}
		}

		return fmt.Errorf("%s: %016X", ResolveErrorMessage(errBaseStationNotBidirectional), session.BaseStationEUI)
	}

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.beginScOperation(session)
	if err != nil {
		return err
	}

	// Convert repetition uint8 to boolean (non-zero means repetition enabled)
	repetitionBool := repetition > 0

	// Convert nwkSnKey from []byte to []interface{} array per MIOTY spec (Numeric[16])
	// MIOTY spec requires this as Numeric[16], not Binary
	// Use interface{} slice to ensure proper MessagePack encoding as array
	nwkSnKeyArray := make([]interface{}, 16)
	for i := 0; i < 16 && i < len(nwkSnKey); i++ {
		nwkSnKeyArray[i] = uint8(nwkSnKey[i])
	}

	// Build attach propagate message per BSSCI v1.0.0 spec
	message := map[string]interface{}{
		"command":       mioty.CmdAttachPropagate,
		"opId":          opId,
		"epEui":         endpointEUI, // MUST be Numeric[8] per BSSCI spec, NOT string!
		"bidi":          bidirectional,
		"nwkSnKey":      nwkSnKeyArray, // Numeric[16] array per MIOTY spec
		"shAddr":        shortAddr,     // Keep as uint16 per MIOTY spec
		"lastPacketCnt": lastPacketCnt,
		"dualChan":      dualChannel,
		"repetition":    repetitionBool,
		"wideCarrOff":   wideCarrOff,
		"longBlkDist":   longBlkDist,
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingAttachPropagate,
		"sessionID", sessionID,
		"endpointEui", endpointEUI,
		"shortAddr", shortAddr,
		"bidirectional", bidirectional)

	// Debug log the complete message
	s.logger.InfoContext(s.safeCtx(), LogBSSCIDebugFullAttachPropagateMessage,
		"message", message)

	// Resolve owner tenant + ctx before metadata (BSSCI §5.8.3 multi-tenant roaming).
	// Returns euiBytes for downstream persistence + repository updates.
	euiBytes, endpointTenantID, ownerOrgUUID, ownerCtx := s.resolveEndpointOwnerContext(ctx, session, endpointEUI)

	// Create metadata with owner tenant/org info (BSSCI §5.8.3)
	metadata := map[string]interface{}{
		"epEui":         pkgmioty.FormatEUI64(endpointEUI),
		"tenantId":      endpointTenantID, // Stored as int64, unmarshals as float64
		"shortAddr":     shortAddr,
		"bidirectional": bidirectional,
		"lastPacketCnt": lastPacketCnt,
		"dualChannel":   dualChannel,
		"repetition":    repetitionBool, // Store as bool per BSSCI §5.8.1
		"wideCarrOff":   wideCarrOff,
		"longBlkDist":   longBlkDist,
	}
	if ownerOrgUUID != uuid.Nil {
		metadata["organizationId"] = ownerOrgUUID.String()
	}

	// The recovery record must be durable before the frame is written; a
	// persistence failure aborts the send, leaving only a consumed-ID gap.
	if err := s.persistPendingOperation(session, opId, mioty.CmdAttachPropagate, message, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistPendingOperation, "error", err)
		return err
	}

	// Update endpoint in database with attach propagate information
	if s.endpointRepo != nil && endpointTenantID > 0 {
		// Re-fetch endpoint to ensure we have latest state
		endpoint, err := s.endpointRepo.GetByEUI(ownerCtx, endpointTenantID, euiBytes)
		if err != nil {
			// Try fallback one more time
			var eui models.EUI
			binary.BigEndian.PutUint64(eui[:], endpointEUI)
			endpoint, err = s.endpointRepo.Get(ownerCtx, eui)
		}

		if err == nil && endpoint != nil {
			// Encrypt session key using raw GCM for attach propagate
			// This matches the attach handler encryption strategy (BSSCI §5.6.2)
			encryptedKey := nwkSnKey
			if s.keyEncryptor != nil {
				var encErr error
				encryptedKey, encErr = s.keyEncryptor.EncryptKeyRaw(nwkSnKey)
				if encErr != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToEncryptNetworkKeyStoringUnencrypted,
						"error", encErr,
						"epEui", endpointEUI)
					// Fall back to unencrypted (will be caught by DB constraint)
					encryptedKey = nwkSnKey
				}
			}

			updates := map[string]interface{}{
				"propagated_at":   time.Now(),
				"sh_addr":         shortAddr,
				"bidi":            bidirectional,
				"last_packet_cnt": lastPacketCnt,
				"dual_chan":       dualChannel,
				"repetition":      repetitionBool, // Use bool per BSSCI §5.8.1
				"wide_carr_off":   wideCarrOff,
				"long_blk_dist":   longBlkDist,
			}

			// The attachment persister owns the owner-tenant transaction;
			// error strings propagate to the caller unchanged.
			if s.attachPersistence == nil {
				return fmt.Errorf("failed to begin attach propagate transaction: %w", errAttachPersistenceUnavailable)
			}
			if err := s.attachPersistence.PersistAttachPropagateSession(ownerCtx, AttachPropagateSessionRecord{
				TenantID:        endpointTenantID,
				EndpointID:      endpoint.ID,
				EndpointUpdates: updates,
				EncryptedKey:    encryptedKey,
				ShAddr:          shortAddr,
				BaseStationEUI:  session.BaseStationEUIBytes(),
			}); err != nil {
				return err
			}

			s.logger.DebugContext(s.safeCtx(), LogBSSCIUpdatedEndpointWithAttachPropagateInfo,
				"epEui", endpointEUI,
				"shortAddr", shortAddr)
		} else {
			s.logger.DebugContext(s.safeCtx(), LogBSSCIEndpointNotFoundInDatabaseForAttachPropagate,
				"epEui", endpointEUI)
		}
	}

	// NOTE: Event creation moved to RecordAttachPropagate (called on successful message persistence)
	// which creates a single "attPrp" event. Removed duplicate "attach_propagate_initiated" events
	// that were creating 2 extra records per operation.

	if err := s.sendMessage(session, message); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}

		return err
	}

	// BSSCI §5.8.3: Persist attach propagate message to mioty_messages for audit trail
	if s.protocolMessages != nil {
		bsEuiBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

		var orgUUIDStr *string
		if ownerOrgUUID != uuid.Nil {
			orgStr := ownerOrgUUID.String()
			orgUUIDStr = &orgStr
		}

		// Encrypt network key before persistence (production security requirement - BSSCI §5.6.2)
		// Matches the attach handler encryption strategy
		encryptedKey := nwkSnKey
		if s.keyEncryptor != nil {
			var encErr error
			encryptedKey, encErr = s.keyEncryptor.EncryptKeyRaw(nwkSnKey)
			if encErr != nil {
				s.logger.WarnContext(ownerCtx, LogBSSCIFailedToEncryptNetworkKeyStoringUnencrypted,
					"error", encErr,
					"epEui", endpointEUI)
				// Continue with unencrypted key (fallback for dev/testing)
			}
		}

		propagateMsg := &mioty.AttachPropagateMessage{
			CommandType:   mioty.CmdAttachPropagate,
			OpId:          opId,
			EpEui:         endpointEUI,
			Bidi:          bidirectional,
			NwkSnKey:      encryptedKey, // Use encrypted key for storage
			ShAddr:        shortAddr,
			LastPacketCnt: lastPacketCnt,
			DualChan:      dualChannel,
			Repetition:    repetitionBool,
			WideCarrOff:   wideCarrOff,
			LongBlkDist:   longBlkDist,
			// Metadata fields
			BasestationEui: bsEuiBytes,
			TenantID:       endpointTenantID, // Endpoint owner tenant (BSSCI §5.8.3 roaming)
			OrgUUID:        orgUUIDStr,       // Endpoint owner org (roaming)
			MessageType:    mioty.MessageTypeAttachPropagate,
			Direction:      mioty.DirectionDownlink,
			InterfaceType:  mioty.InterfaceBSSCI,
		}

		if s.protocolMessages != nil {
			if err := s.protocolMessages.CreateAttachPropagateMessage(ownerCtx, propagateMsg); err != nil {
				s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistAttachPropagateMessage,
					"error", err,
					"epEui", endpointEUI,
					"bsEui", session.BaseStationEUI,
					"opId", opId)
				// Continue - persistence failure shouldn't block operation
			}
		}
	}

	return nil
}

// SendAttachPropagateToAll sends attach propagate to all connected sessions
func (s *Server) SendAttachPropagateToAll(endpointEUI uint64, nwkSnKey []byte,
	shortAddr uint16, bidirectional bool, lastPacketCnt uint32, dualChannel bool,
	repetition uint8, wideCarrOff bool, longBlkDist bool) []error {

	s.mu.RLock()
	sessionIDs := make([]string, 0, len(s.sessions))
	s.logger.WarnContext(s.safeCtx(), LogBSSCIDebuggingCheckingSessionsForAttachPropagate,
		"endpointEUI", endpointEUI,
		"totalSessions", len(s.sessions))

	for id, session := range s.sessions {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIDebuggingSessionStatus,
			"sessionID", id,
			"baseStationEUI", session.BaseStationEUI,
			"handshakeComplete", session.HandshakeComplete)

		// BSSCI-3.3-03: Only send to sessions that have completed their handshake
		if session.HandshakeComplete {
			sessionIDs = append(sessionIDs, id)
		}
	}
	s.mu.RUnlock()

	// If no sessions available, emit failure event and return error
	if len(sessionIDs) == 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCINoConnectedBaseStationsForAttachPropagate,
			"endpointEUI", endpointEUI)

		// Create failure event
		details := map[string]interface{}{
			"epEui":     pkgmioty.FormatEUI64(endpointEUI), // Use hex string for event details
			"shortAddr": shortAddr,
			"reason":    "No connected base stations available",
		}
		detailsJSON, _ := json.Marshal(details)

		// NOTE: No session exists for this server-level error (no base stations connected).
		// Using s.defaultTenantID (not s.tenantID) is intentional for consistency with
		// other no-session error paths.
		if err := s.eventStore.CreateEvent(s.safeCtx(), &models.SystemEvent{
			TenantID:    fmt.Sprintf("%d", s.defaultTenantID),
			EventType:   EventTypeAttachOperationFailed,
			Category:    mioty.CategoryEndpoint,
			Severity:    SeverityError,
			Title:       fmt.Sprintf("Attach propagate failed for endpoint %s", pkgmioty.FormatEUI64(endpointEUI)),
			Description: "No connected base stations available to propagate attach",
			Details:     detailsJSON,
			Status:      EventStatusNew,
			SourceType:  mioty.SourceTypeEndpoint,
			SourceName:  pkgmioty.FormatEUI64(endpointEUI),
			CreatedAt:   time.Now(),
		}); err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateAttachFailedEvent, "error", err)
		}

		return []error{fmt.Errorf("%s", ResolveErrorMessage(errNoConnectedBaseStations))}
	}

	var errors []error
	for _, sessionID := range sessionIDs {
		if err := s.SendAttachPropagate(sessionID, endpointEUI, nwkSnKey, shortAddr,
			bidirectional, lastPacketCnt, dualChannel, repetition, wideCarrOff, longBlkDist); err != nil {
			errors = append(errors, fmt.Errorf("session %s: %w", sessionID, err))
		}
	}

	return errors
}

// GetConnectedSessionEUIs returns the EUIs of all connected sessions as hex strings.
// Used by EndpointAttachmentService for event logging.
func (s *Server) GetConnectedSessionEUIs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	euis := make([]string, 0, len(s.sessions))
	for _, session := range s.sessions {
		// Connected is a time.Time - check if not zero (session is active)
		if !session.Connected.IsZero() && session.HandshakeComplete {
			euis = append(euis, fmt.Sprintf("%016x", session.BaseStationEUI))
		}
	}
	return euis
}

// GetConnectedSessions returns information about all connected sessions
func (s *Server) GetConnectedSessions() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]map[string]interface{}, 0, len(s.sessions))
	for id, session := range s.sessions {
		sessions = append(sessions, map[string]interface{}{
			"id":                id,
			"baseStationEui":    session.BaseStationEUI,
			"connected":         session.Connected,
			"lastSeen":          session.LastSeen,
			"vendor":            session.Vendor,
			"model":             session.Model,
			"name":              session.Name,
			"clientVersion":     session.ClientVersion,
			"negotiatedVersion": session.NegotiatedVersion,
			"bidirectional":     session.Bidirectional,
			"handshakeComplete": session.HandshakeComplete,
			// ATT-02: Tenant/org fields for roaming-aware propagation
			"resolvedTenantID": session.ResolvedTenantID,
			"organizationID":   session.OrganizationID.String(),
		})
	}

	return sessions
}

// GetSessionByEUI returns a session for a given base station EUI
// Returns nil if no connected session exists for the given EUI
// Thread-safe: holds lock during lookup to prevent race with disconnection
func (s *Server) GetSessionByEUI(bsEui uint64) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.sessions {
		if session.BaseStationEUI == bsEui && session.HandshakeComplete {
			// Return session while still holding lock
			// Caller must handle nil and be aware session could disconnect
			return session
		}
	}
	return nil
}

// publishLiveSession makes session the only live session for its base station. Eviction and
// publication share one critical section on s.mu so a by-EUI lookup can never resolve to the
// connection being replaced.
func (s *Server) publishLiveSession(ctx context.Context, session *Session) {
	s.mu.Lock()
	var displaced []*Session
	for id, live := range s.sessions {
		if id != session.ID && live.BaseStationEUI == session.BaseStationEUI {
			displaced = append(displaced, live)
			delete(s.sessions, id)
		}
	}
	s.sessions[session.ID] = session
	s.mu.Unlock()

	for _, stale := range displaced {
		s.logger.WarnContext(ctx, LogBSSCIDisplacedLiveSessionForBaseStation,
			"eui", session.BaseStationEUI,
			"displacedSessionID", stale.ID,
			"sessionID", session.ID)
		s.sessionSvc.RemoveSession(stale)
		if stale.Conn != nil {
			if err := stale.Conn.Close(); err != nil {
				s.logger.WarnContext(ctx, LogBSSCIFailedToCloseDisplacedSessionConnection,
					"error", err,
					"displacedSessionID", stale.ID)
			}
		}
	}
}

// CloseSessionByEUI finds and closes any active session for the given EUI.
// Sessions are keyed by session.ID, so we iterate to find matching EUI.
// This method terminates the DB session, removes from cache, and closes connection.
// Used by gRPC layer when EUI is changed to ensure the old session doesn't persist.
func (s *Server) CloseSessionByEUI(ctx context.Context, eui uint64) error {
	// Find session by EUI (sessions are keyed by ID, not EUI)
	s.mu.RLock()
	var targetSession *Session
	for _, session := range s.sessions {
		if session.BaseStationEUI == eui {
			targetSession = session
			break
		}
	}
	s.mu.RUnlock()

	if targetSession == nil {
		return nil // No active session for this EUI
	}

	s.logger.InfoContext(ctx, LogBSSCIClosingBSSCISessionDueToEUIChange,
		"eui", eui,
		"sessionID", targetSession.ID)

	// Terminate DB session record
	if targetSession.DbSessionID != 0 && s.sessionSvc != nil {
		if err := s.sessionSvc.TerminateSession(ctx, targetSession); err != nil {
			s.logger.WarnContext(ctx, LogBSSCIFailedToTerminateDBSessionDuringEUIChange,
				"error", err,
				"sessionID", targetSession.DbSessionID)
		}
	}

	// Remove from SessionService's sessionsByUUID map to prevent stale resume
	if s.sessionSvc != nil {
		s.sessionSvc.RemoveSession(targetSession)
	}

	// Remove from in-memory session map
	s.mu.Lock()
	delete(s.sessions, targetSession.ID)
	s.mu.Unlock()

	// Close connection to trigger cleanup
	if targetSession.Conn != nil {
		if err := targetSession.Conn.Close(); err != nil {
			s.logger.WarnContext(ctx, LogBSSCIFailedToCloseConnectionDuringEUIChangeCleanup,
				"error", err,
				"sessionID", targetSession.ID)
		}
	}

	return nil
}

// handleAttachPropagateResponse handles attach propagate response from base station
func (s *Server) handleAttachPropagateResponse(srv *Server, session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// BSSCI 3.8.2: attPrpRsp only carries command and opId. No result field per spec.
	// Default to 0 (success) if result field is missing - spec-compliant behavior.
	result := getNumericFieldInt(data, "result", 0)

	s.logger.DebugContext(s.safeCtx(), LogBSSCIAttachPropagateResponseReceived,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId,
		"result", result)

	// Result codes: 0 = success, non-zero = error
	if result != 0 {
		// Attach propagate failed - DO NOT send completion or update endpoint
		return s.handlePropagateResponseFailure(ctx, session, msg, result, propagateResponseConfig{
			rejectedLog:     LogBSSCIAttachPropagateRejectedByBaseStation,
			failureErrToken: errAttachPropagateFailed,
			operationType:   EventTypeAttachPropagateFailed,
			eventType:       EventTypeEndpointAttachFailed,
			titleFormat:     TitleAttachPropagateFailedForEndpointOnBS,
		})
	}

	// Success case - proceed with three-way handshake completion
	s.logger.InfoContext(s.safeCtx(), LogBSSCIAttachPropagateAcceptedByBaseStation,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId)

	// BSSCI three-way handshake: Service Center must send attPrpCmp after successful attPrpRsp
	completionMsg := map[string]interface{}{
		"command": mioty.CmdAttachPropagateComplete,
		"opId":    msg.OpId, // Use same operation ID
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingAttachPropagateComplete,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId)

	// Send the completion message
	if err := s.sendMessage(session, completionMsg); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendAttachPropagateComplete,
			"baseStation", session.BaseStationEUI,
			"opId", msg.OpId,
			"error", err)
		// Keep op in pending for retry
		return err
	}

	// Now perform the cleanup that would happen in handleAttachPropagateComplete
	// Since WE send the Cmp (not receive it), we need to do the cleanup here
	return s.handleAttachPropagateComplete(srv, session, msg, data)
}

// handleAttachPropagateComplete performs cleanup after attach propagate completion
// This is called after we send attPrpCmp (Service Center sends it, not base station)
func (s *Server) handleAttachPropagateComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	ctx := s.sessionContext(session)

	s.logger.InfoContext(s.safeCtx(), LogBSSCIAttachPropagateCompleted,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId)

	// BSSCI-3.8.3-01: Get pending operation BEFORE removing it to access metadata
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		pendingOp, _ = s.statusSvc.GetPendingOperation(session, int64(msg.OpId))
	}

	// Remove completed operation from pending operations for MIOTY session resume
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation,
			"error", err,
			"opId", msg.OpId,
			"sessionID", session.DbSessionID)
	}

	// BSSCI §3.8.3: Record completion event UNCONDITIONALLY for audit trail
	// This is distinct from RecordAttachPropagate (EventTypeAttachPropagateInitiated)
	// which is called in the pendingOp block for endpoint-specific tracking
	// NOTE: Endpoint EUI extracted from pendingOp below; this block may lack epEUI
	if s.eventStore != nil {
		completionEvent := &models.SystemEvent{
			TenantID:    strconv.FormatInt(resolvedTenant(session, s.tenantID), 10),
			EventType:   EventTypeAttachPropagateCompleted,
			Title:       TitleAttachPropagateCompleted,
			Description: fmt.Sprintf("Attach propagate completed for opId %d, BS %016X", msg.OpId, session.BaseStationEUI),
			Severity:    SeverityInfo,
			Category:    mioty.CategoryEndpoint, // Use endpoint category for UI visibility
			SourceType:  mioty.SourceTypeEndpoint,
			SourceName:  fmt.Sprintf("%016X", session.BaseStationEUI),
			Status:      EventStatusNew,
			CreatedAt:   time.Now(),
		}

		if err := s.eventStore.CreateEvent(ctx, completionEvent); err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateCompletionEvent,
				"error", err,
				"opId", msg.OpId)
		}
	}

	// Update endpoint to track which base station accepted the attachment
	if s.endpointRepo != nil && pendingOp != nil && pendingOp.Metadata != nil {
		// Extract owner tenant/org from metadata for roaming support
		var ownerTenantID int64
		if tenantIDFloat, ok := pendingOp.Metadata["tenantId"].(float64); ok {
			ownerTenantID = int64(tenantIDFloat)
		} else {
			ownerTenantID = resolvedTenant(session, s.tenantID)
			s.logger.WarnContext(ctx, LogBSSCIMissingTenantInMetadata, "opId", msg.OpId)
		}

		var ownerOrg uuid.UUID
		if orgIDStr, ok := pendingOp.Metadata["organizationId"].(string); ok {
			ownerOrg, _ = uuid.Parse(orgIDStr)
		}

		ownerCtx := pkgcontext.WithTenantID(ctx, ownerTenantID)
		if ownerOrg != uuid.Nil {
			ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrg)
		}

		// Extract endpoint EUI from pending operation metadata
		// Handle both uint64 (direct) and float64 (from JSON deserialization)
		var epEUI uint64
		var hasEUI bool

		epEUI, hasEUI = parseMetadataEUI(pendingOp.Metadata["epEui"])

		if hasEUI {
			// Convert endpoint EUI to bytes
			epEUIBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(epEUIBytes[:], epEUI)

			// Get endpoint to update attachment state
			endpoint, err := s.endpointRepo.GetByEUI(ownerCtx, ownerTenantID, epEUIBytes)
			if err == storage.ErrNotFound {
				var eui models.EUI
				binary.BigEndian.PutUint64(eui[:], epEUI)
				endpoint, err = s.endpointRepo.Get(ownerCtx, eui)
			}
			if err != nil {
				s.logger.WarnContext(ownerCtx, LogBSSCIEndpointNotFoundForPropagate, "epEui", epEUI)
				return nil
			}
			// Re-assign to actual endpoint tenant (metadata may be stale)
			ownerTenantID = endpoint.TenantID

			// Re-resolve organization for the actual tenant (metadata org may be wrong)
			// Keep metadata org as fallback in case resolver is unavailable
			if s.orgResolver != nil {
				newOrg, orgErr := s.orgResolver.GetDefaultOrgForTenant(ctx, ownerTenantID)
				if orgErr != nil {
					s.logger.WarnContext(ctx, LogBSSCIOrgLookupFailed,
						"tenantID", ownerTenantID,
						"error", orgErr,
						"context", "attach_propagate_complete_fallback",
						"fallbackOrg", ownerOrg.String())
				} else {
					// Only overwrite on successful lookup
					ownerOrg = newOrg
				}
			}
			// If resolver is nil or errored, ownerOrg retains metadata value

			// Rebuild context with actual tenant and re-resolved org
			// Use Background() to avoid session context pollution
			ownerCtx = pkgcontext.WithTenantID(context.Background(), ownerTenantID)
			if ownerOrg != uuid.Nil {
				ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrg)
			}

			// Update endpoint with attachment to this base station
			attachUpdates := map[string]interface{}{
				"last_attached_bs_eui": session.BaseStationEUIBytes(),
				"last_propagate_time":  time.Now().UnixNano(),
				"propagate_status":     PropagateStatusAttached,
				"propagated":           true,
				"propagated_at":        time.Now(),
				"ep_status":            EndpointStatusAttached,
			}

			if err := s.endpointRepo.UpdateFields(ownerCtx, ownerTenantID, endpoint.ID, attachUpdates); err != nil {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointAttachmentState,
					"error", err,
					"epEui", epEUI,
					"bsEui", session.BaseStationEUI)
			} else {
				s.logger.InfoContext(s.safeCtx(), LogBSSCIEndpointAttachedToBaseStation,
					"epEui", epEUI,
					"bsEui", session.BaseStationEUI)
			}

			// Create event for successful attachment
			if s.eventStore != nil {
				epEUIHex := pkgmioty.FormatEUI64(epEUI)
				bsEUIHex := pkgmioty.FormatEUI64(session.BaseStationEUI)

				// Extract short address from metadata (uint16)
				var shAddr uint16
				if shAddrFloat, ok := pendingOp.Metadata["shortAddr"].(float64); ok {
					shAddr = uint16(shAddrFloat)
				} else if shAddrInt, ok := pendingOp.Metadata["shortAddr"].(int64); ok {
					shAddr = uint16(shAddrInt) //nolint:gosec // G115: shAddr validated by BSSCI
				}

				details := map[string]interface{}{
					"epEui":   epEUIHex,
					"bsEui":   bsEUIHex,
					"shAddr":  shAddr,
					"success": true,
					"time":    time.Now().UnixNano(),
				}
				detailsJSON, _ := json.Marshal(details)

				event := &models.SystemEvent{
					TenantID:    strconv.FormatInt(ownerTenantID, 10),
					EventType:   "attPrp",
					Category:    models.EventCategoryEndpoint,
					Severity:    models.EventSeverityInfo,
					SourceType:  models.SourceTypeServiceCenter,
					SourceName:  epEUIHex,
					Title:       fmt.Sprintf("Attach Propagate: EP %s to BS %s", epEUIHex, bsEUIHex),
					Description: fmt.Sprintf("Endpoint %s keys propagated to base station %s with short address %d", epEUIHex, bsEUIHex, shAddr),
					Details:     json.RawMessage(detailsJSON),
				}

				if err := s.eventStore.CreateEvent(ownerCtx, event); err != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateAttachmentEvent, "error", err)
				}
			}

			// Propagate re-fires on every BS reconnect; publish only on the real attach transition.
			if s.mqttPublisher != nil && endpoint.EpStatus != EndpointStatusAttached {
				if ownerOrg != uuid.Nil {
					go func() {
						if pubErr := s.mqttPublisher.PublishAttach(ownerCtx, ownerOrg.String(), epEUI, session.BaseStationEUI); pubErr != nil {
							s.logger.WarnContext(ownerCtx, LogBSSCIFailedToPublishAttachEventToMQTT, "error", pubErr)
						}
					}()
				} else {
					s.logger.WarnContext(ownerCtx, LogBSSCIMQTTPublishSkippedOrgUnresolved,
						"epEui", epEUI, "event", MQTTEventKeyAttach)
				}
			}

			// BSSCI §3.8.3: Persist attPrpCmp to messages table
			// Gated by hasEUI to ensure usable audit rows with valid ep_eui
			// NOTE: This is separate from the attPrp row persisted at send time
			// NwkSnKey intentionally omitted: already in attPrp row, avoid key duplication
			if s.protocolMessages != nil {
				// Re-extract shAddr at this scope level for message persistence
				var msgShAddr uint16
				if shAddrFloat, ok := pendingOp.Metadata["shortAddr"].(float64); ok {
					msgShAddr = uint16(shAddrFloat)
				} else if shAddrInt, ok := pendingOp.Metadata["shortAddr"].(int64); ok {
					msgShAddr = uint16(shAddrInt) //nolint:gosec // G115: shAddr validated by BSSCI
				}

				// Extract Bidi from metadata (stored as "bidirectional")
				var msgBidi bool
				if bidiBool, ok := pendingOp.Metadata["bidirectional"].(bool); ok {
					msgBidi = bidiBool
				}

				// Extract LastPacketCnt from metadata
				var msgLastPacketCnt uint32
				if lpcFloat, ok := pendingOp.Metadata["lastPacketCnt"].(float64); ok {
					msgLastPacketCnt = uint32(lpcFloat)
				} else if lpcInt, ok := pendingOp.Metadata["lastPacketCnt"].(int64); ok {
					msgLastPacketCnt = uint32(lpcInt) //nolint:gosec // G115: lastPacketCnt validated by BSSCI
				}

				// Extract optional fields for audit completeness
				var msgDualChan, msgRepetition, msgWideCarrOff, msgLongBlkDist bool
				if dualChan, ok := pendingOp.Metadata["dualChannel"].(bool); ok {
					msgDualChan = dualChan
				}
				if repetition, ok := pendingOp.Metadata["repetition"].(bool); ok {
					msgRepetition = repetition
				}
				if wideCarrOff, ok := pendingOp.Metadata["wideCarrOff"].(bool); ok {
					msgWideCarrOff = wideCarrOff
				}
				if longBlkDist, ok := pendingOp.Metadata["longBlkDist"].(bool); ok {
					msgLongBlkDist = longBlkDist
				}

				// Build bsEUIBytes at this scope for message persistence
				var msgBsEUIBytes [8]byte
				binary.BigEndian.PutUint64(msgBsEUIBytes[:], session.BaseStationEUI)

				completionMsg := &mioty.AttachPropagateMessage{
					CommandType:    mioty.CmdAttachPropagateComplete, // "attPrpCmp"
					OpId:           int64(msg.OpId),
					TenantID:       ownerTenantID,
					BasestationEui: msgBsEUIBytes[:],
					EpEui:          epEUI,
					ShAddr:         msgShAddr,
					Bidi:           msgBidi,
					LastPacketCnt:  msgLastPacketCnt,
					DualChan:       msgDualChan,
					Repetition:     msgRepetition,
					WideCarrOff:    msgWideCarrOff,
					LongBlkDist:    msgLongBlkDist,
					// NwkSnKey omitted: security concern, already in attPrp row
					MessageType:   mioty.MessageTypeAttachPropagate,
					Direction:     mioty.DirectionDownlink,
					InterfaceType: mioty.InterfaceBSSCI,
				}

				if err := s.protocolMessages.CreateAttachPropagateMessage(ownerCtx, completionMsg); err != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToPersistAttachPropagateComplete,
						"error", err,
						"opId", msg.OpId,
						"epEui", epEUI,
						"bsEui", session.BaseStationEUI)
				}
			}
		}
	}

	// No response needed for complete messages
	return nil
}

// SendDetachPropagate sends a detach propagate command to a specific session
func (s *Server) SendDetachPropagate(sessionID string, endpointEUI uint64) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	ctx := s.sessionContext(session)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// BSSCI-3.3-03: Don't send operations to sessions that haven't completed handshake
	if !session.HandshakeComplete {
		return fmt.Errorf("%s for session %s", ResolveErrorMessage(errHandshakeNotComplete), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.beginScOperation(session)
	if err != nil {
		return err
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingDetachPropagate,
		"sessionID", sessionID,
		"endpointEui", endpointEUI)

	// Resolve owner tenant + ctx before metadata (BSSCI §3.9 multi-tenant roaming).
	// shAddr defaults to 0 if the endpoint is unknown — handled downstream.
	euiBytes, endpointTenantID, ownerOrgUUID, ownerCtx := s.resolveEndpointOwnerContext(ctx, session, endpointEUI)

	// Build detach propagate message per BSSCI v1.0.0 spec
	// Per BSSCI §3.9.1: detPrp requires only command, opId, epEui (shAddr is NOT in spec)
	message := map[string]interface{}{
		"command": mioty.CmdDetachPropagate,
		"opId":    opId,
		"epEui":   endpointEUI, // MUST be Numeric[8] per BSSCI spec, NOT string!
	}

	// Create metadata with owner tenant/org info (BSSCI §5.9)
	metadata := map[string]interface{}{
		"endpointEUI": pkgmioty.FormatEUI64(endpointEUI), // Backward compatibility
		"epEui":       endpointEUI,                       // MUST be Numeric[8] per BSSCI spec, NOT string!
		"tenantId":    endpointTenantID,                  // Owner tenant for roaming support
	}
	if ownerOrgUUID != uuid.Nil {
		metadata["organizationId"] = ownerOrgUUID.String()
	}

	// The recovery record must be durable before the frame is written; a
	// persistence failure aborts the send, leaving only a consumed-ID gap.
	if err := s.persistPendingOperation(session, opId, mioty.CmdDetachPropagate, message, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistPendingOperation, "error", err)
		return err
	}

	// Update endpoint in database to mark detach propagate initiated
	if s.endpointRepo != nil && endpointTenantID > 0 {
		// Get the endpoint using owner tenant context
		endpoint, err := s.endpointRepo.GetByEUI(ownerCtx, endpointTenantID, euiBytes)
		if err == nil && endpoint != nil {
			updates := map[string]interface{}{
				"last_detach_time":     time.Now().UnixNano(),
				"propagate_status":     PropagateStatusDetaching, // In-flight status
				"propagated":           false,
				"last_attached_bs_eui": session.BaseStationEUIBytes(),
			}

			// Update the endpoint with detach propagate info
			if err := s.endpointRepo.UpdateFields(ownerCtx, endpointTenantID, endpoint.ID, updates); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointWithDetachInfo,
					"epEui", endpointEUI,
					"endpointId", endpoint.ID,
					"error", err)
			} else {
				s.logger.DebugContext(s.safeCtx(), LogBSSCIUpdatedEndpointWithDetachInfo,
					"epEui", endpointEUI)
			}
		} else {
			// Endpoint not found in database - this is okay for detach
			s.logger.DebugContext(s.safeCtx(), LogBSSCIEndpointNotFoundInDatabaseForDetachPropagate,
				"epEui", endpointEUI)
		}
	}

	// NOTE: Duplicate event creation removed. The message persistence below creates
	// the detPrp record in the messages table, and completion creates the detPrpCmp record.
	// This avoids creating 2 extra events per operation.

	// Persist detach propagate message to messages audit trail (BSSCI §5.9)
	// Encode BS EUI to bytes for storage
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

	// Convert org UUID to *string for nullable DB column
	var orgUUIDStr *string
	if ownerOrgUUID != uuid.Nil {
		orgStr := ownerOrgUUID.String()
		orgUUIDStr = &orgStr
	}

	// Build DetachPropagateMessage (mirrors attach propagate at 4846-4865)
	propagateMsg := &mioty.DetachPropagateMessage{
		// Protocol fields from BSSCI §5.9.1
		CommandType: mioty.CmdDetachPropagate,
		OpId:        opId,
		EpEui:       endpointEUI,

		// Metadata for messages table storage
		BasestationEui: bsEuiBytes,
		TenantID:       endpointTenantID,
		OrgUUID:        orgUUIDStr,
		MessageType:    mioty.MessageTypeDetachPropagate,
		Direction:      mioty.DirectionDownlink,
		InterfaceType:  mioty.InterfaceBSSCI,
		ReceivedAt:     time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Persist to messages (non-blocking - log errors only)
	if s.protocolMessages != nil {
		if err := s.protocolMessages.CreateDetachPropagateMessage(ownerCtx, propagateMsg); err != nil {
			s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToPersistDetachPropagateMessage,
				"ep_eui", endpointEUI,
				"bs_eui", session.BaseStationEUI,
				"tenant_id", endpointTenantID,
				"error", err)
			// Don't return error - persistence failure shouldn't abort protocol handshake
		}
	}

	if err := s.sendMessage(session, message); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}

		return err
	}

	return nil
}

// SendDetachPropagateToAll sends detach propagate to all connected sessions
func (s *Server) SendDetachPropagateToAll(endpointEUI uint64) []error {
	s.mu.RLock()
	sessionIDs := make([]string, 0, len(s.sessions))
	for id, session := range s.sessions {
		// BSSCI-3.3-03: Only send to sessions that have completed their handshake
		if session.HandshakeComplete {
			sessionIDs = append(sessionIDs, id)
		}
	}
	s.mu.RUnlock()

	// If no sessions available, emit failure event and return error
	if len(sessionIDs) == 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCINoConnectedBaseStationsForDetachPropagate,
			"endpointEUI", endpointEUI)

		// Create failure event
		details := map[string]interface{}{
			"epEui":  pkgmioty.FormatEUI64(endpointEUI), // Use hex string for event details
			"reason": "No connected base stations available",
		}
		detailsJSON, _ := json.Marshal(details)

		// NOTE: No session exists for this server-level error (no base stations connected).
		// Using s.defaultTenantID (not s.tenantID) is intentional for consistency with
		// other no-session error paths.
		if err := s.eventStore.CreateEvent(s.safeCtx(), &models.SystemEvent{
			TenantID:    fmt.Sprintf("%d", s.defaultTenantID),
			EventType:   EventTypeDetachOperationFailed,
			Category:    mioty.CategoryEndpoint,
			Severity:    SeverityError,
			Title:       fmt.Sprintf("Detach propagate failed for endpoint %s", pkgmioty.FormatEUI64(endpointEUI)),
			Description: "No connected base stations available to propagate detach",
			Details:     detailsJSON,
			Status:      EventStatusNew,
			SourceType:  mioty.SourceTypeEndpoint,
			SourceName:  pkgmioty.FormatEUI64(endpointEUI),
			CreatedAt:   time.Now(),
		}); err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateDetachFailedEvent, "error", err)
		}

		return []error{fmt.Errorf("%s", ResolveErrorMessage(errNoConnectedBaseStations))}
	}

	var errors []error
	for _, sessionID := range sessionIDs {
		if err := s.SendDetachPropagate(sessionID, endpointEUI); err != nil {
			errors = append(errors, fmt.Errorf("session %s: %w", sessionID, err))
		}
	}

	return errors
}

// handleDetachPropagateResponse handles detach propagate response from base station
func (s *Server) handleDetachPropagateResponse(srv *Server, session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// BSSCI 3.9.2: detPrpRsp only carries command and opId. No result field per spec.
	// Default to 0 (success) if result field is missing - spec-compliant behavior.
	result := getNumericFieldInt(data, "result", 0)

	s.logger.DebugContext(s.safeCtx(), LogBSSCIDetachPropagateResponseReceived,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId,
		"result", result)

	// Result codes: 0 = success, non-zero = error
	if result != 0 {
		// Detach propagate failed - DO NOT send completion or update endpoint
		return s.handlePropagateResponseFailure(ctx, session, msg, result, propagateResponseConfig{
			rejectedLog:     LogBSSCIDetachPropagateRejectedByBaseStation,
			failureErrToken: errDetachPropagateFailed,
			operationType:   EventTypeDetachPropagateFailed,
			eventType:       EventTypeEndpointDetachFailed,
			titleFormat:     TitleDetachPropagateFailedForEndpointOnBS,
		})
	}

	// Success case - proceed with three-way handshake completion
	s.logger.InfoContext(s.safeCtx(), LogBSSCIDetachPropagateAcceptedByBaseStation,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId)

	// BSSCI three-way handshake: Service Center must send detPrpCmp after successful detPrpRsp
	completionMsg := map[string]interface{}{
		"command": mioty.CmdDetachPropagateComplete,
		"opId":    msg.OpId, // Use same operation ID
	}

	s.logger.DebugContext(s.safeCtx(), LogBSSCISendingDetachPropagateComplete,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId)

	// Send the completion message
	if err := s.sendMessage(session, completionMsg); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToSendDetachPropagateComplete,
			"baseStation", session.BaseStationEUI,
			"opId", msg.OpId,
			"error", err)
		return err
	}

	// Now perform the cleanup that would happen in handleDetachPropagateComplete
	// Since WE send the Cmp (not receive it), we need to do the cleanup here
	return s.handleDetachPropagateComplete(srv, session, msg, data)
}

// handleDetachPropagateComplete performs cleanup after detach propagate completion
// This is called after we send detPrpCmp (Service Center sends it, not base station)
func (s *Server) handleDetachPropagateComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	s.logger.InfoContext(s.safeCtx(), LogBSSCIDetachPropagateCompleted,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId)

	// BSSCI-3.9.3-01: Get pending operation BEFORE removing it to access metadata
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		pendingOp, _ = s.statusSvc.GetPendingOperation(session, int64(msg.OpId))
	}

	// Remove completed operation from pending operations for MIOTY session resume
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation,
			"error", err,
			"opId", msg.OpId,
			"sessionID", session.DbSessionID)
	}

	// BSSCI-3.9.3-01: Update endpoint to mark as detached from this base station
	if s.endpointRepo != nil && pendingOp != nil && pendingOp.Metadata != nil {
		// Extract endpoint EUI from pending operation metadata
		var epEUI uint64
		var hasEUI bool

		if v, ok := pendingOp.Metadata["endpointEUI"]; ok {
			epEUI, hasEUI = parseMetadataEUI(v)
		}
		if !hasEUI {
			epEUI, hasEUI = parseMetadataEUI(pendingOp.Metadata["epEui"])
		}

		if hasEUI {
			// Convert endpoint EUI to bytes
			epEUIBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(epEUIBytes[:], epEUI)

			// Extract owner tenant/org from metadata (BSSCI §5.9 multi-tenant roaming)
			var endpointTenantID int64
			var ownerOrgUUID uuid.UUID
			var ownerCtx context.Context

			// Extract tenantId from metadata
			if v, ok := pendingOp.Metadata["tenantId"]; ok {
				switch val := v.(type) {
				case int64:
					endpointTenantID = val
				case float64:
					endpointTenantID = int64(val)
				}
			}

			// Extract organizationId from metadata if present
			if v, ok := pendingOp.Metadata["organizationId"]; ok {
				if orgIDStr, ok := v.(string); ok {
					ownerOrgUUID, _ = uuid.Parse(orgIDStr)
				}
			}

			// Build owner context
			if endpointTenantID > 0 {
				// Use Background() to avoid session context pollution
				ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
				if ownerOrgUUID != uuid.Nil {
					ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
				}
			} else {
				// Fallback to session tenant if metadata missing
				endpointTenantID = resolvedTenant(session, s.tenantID)
				// Use Background() to avoid session context pollution
				ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
			}

			// Get endpoint to update detachment state using owner tenant
			epModel, err := s.endpointRepo.GetByEUI(ownerCtx, endpointTenantID, epEUIBytes)
			if err == nil && epModel != nil {
				// Use shared endpoint detach helper (no telemetry for detach propagate path)
				if err := endpoint.DetachEndpoint(ownerCtx, s.endpointRepo, endpointTenantID, epModel.ID, nil); err != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointDetachmentState,
						"error", err,
						"epEui", epEUI,
						"bsEui", session.BaseStationEUI)
				} else {
					s.logger.InfoContext(s.safeCtx(), LogBSSCIEndpointDetachedFromBaseStation,
						"epEui", epEUI,
						"bsEui", session.BaseStationEUI)
				}

				// Create single event for successful detachment (consolidated from 2 redundant events)
				if s.eventStore != nil {
					// Extract operation ID safely (avoid nil dereference)
					opID := msg.OpId
					if pendingOp != nil {
						opID = pendingOp.OperationID
					}

					details := map[string]interface{}{
						"epEui":        pkgmioty.FormatEUI64(epEUI),
						"bsEui":        pkgmioty.FormatEUI64(session.BaseStationEUI),
						"operation":    "detach_propagate_complete",
						"operation_id": fmt.Sprintf("%d", opID), // For operation filtering
					}
					detailsJSON, _ := json.Marshal(details)
					// Create single endpoint event with proper source fields and owner context
					if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
						TenantID:    fmt.Sprintf("%d", endpointTenantID),
						EventType:   models.EventTypeDetachPropagateCompleted,
						Category:    mioty.CategoryEndpoint,
						Severity:    SeverityInfo,
						Title:       fmt.Sprintf(models.EventTitleEndpointDetachedFromBS, pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI)),
						Description: "Detach propagate completed successfully",
						Details:     detailsJSON,
						Status:      EventStatusNew,
						SourceType:  mioty.SourceTypeEndpoint,
						SourceName:  pkgmioty.FormatEUI64(epEUI), // Critical for UI filtering
						CreatedAt:   time.Now(),
					}); err != nil {
						s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateDetachmentEvent, "error", err)
					}
				}

				// Propagate re-fires on every BS reconnect; publish only on the real detach transition.
				if s.mqttPublisher != nil && epModel.EpStatus != endpoint.EndpointStatusDetached {
					if ownerOrgUUID != uuid.Nil {
						go func() {
							if pubErr := s.mqttPublisher.PublishDetach(ownerCtx, ownerOrgUUID.String(), epEUI, session.BaseStationEUI); pubErr != nil {
								s.logger.WarnContext(ownerCtx, LogBSSCIFailedToPublishDetachEventToMQTT, "error", pubErr)
							}
						}()
					} else {
						s.logger.WarnContext(ownerCtx, LogBSSCIMQTTPublishSkippedOrgUnresolved,
							"epEui", epEUI, "event", MQTTEventKeyDetach)
					}
				}

				// BSSCI §5.9.3: Persist detPrpCmp to messages table
				// Gated by hasEUI to ensure usable audit rows with valid ep_eui
				if s.protocolMessages != nil {
					var msgBsEUIBytes [8]byte
					binary.BigEndian.PutUint64(msgBsEUIBytes[:], session.BaseStationEUI)

					completionMsg := &mioty.DetachPropagateMessage{
						CommandType:    mioty.CmdDetachPropagateComplete, // "detPrpCmp"
						OpId:           msg.OpId,                         // Already int64
						TenantID:       endpointTenantID,
						BasestationEui: msgBsEUIBytes[:],
						EpEui:          epEUI,
						MessageType:    mioty.MessageTypeDetachPropagate,
						Direction:      mioty.DirectionDownlink,
						InterfaceType:  mioty.InterfaceBSSCI,
					}
					if ownerOrgUUID != uuid.Nil {
						orgStr := ownerOrgUUID.String()
						completionMsg.OrgUUID = &orgStr
					}

					if err := s.protocolMessages.CreateDetachPropagateMessage(ownerCtx, completionMsg); err != nil {
						s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToPersistDetachPropagateComplete,
							"error", err,
							"opId", msg.OpId,
							"epEui", epEUI,
							"bsEui", session.BaseStationEUI)
					}
				}
			} else {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIEndpointNotFoundForDetachCompletion,
					"epEui", epEUI,
					"error", err)
			}
		}
	}

	// No response needed for complete messages
	return nil
}

// Database persistence methods for MIOTY session resume

// beginScOperation allocates the next SC operation ID and durably persists the
// session counters before any frame is written (BSSCI rev1 §5.2 / classic
// §3.2). The durable order for every SC-issued operation is: allocate the ID,
// persist the counter, persist the pending record (recoverable operations
// only), then write the frame. A failure after allocation leaves a harmless
// consumed-ID gap; the counter is never rolled back because a rollback races
// concurrent allocations.
func (s *Server) beginScOperation(session *Session) (int64, error) {
	opId := session.NextScOpID()
	if err := s.sessionSvc.UpdateSessionCounters(s.sessionContext(session), session); err != nil {
		return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistSessionCounters), err)
	}
	return opId, nil
}

// persistPendingOperation stores a pending operation via StatusService (BSSCI §5.11-5.12.3 single writer)
// StatusService handles both DB persistence and in-memory map update using SessionOpKey composite key
func (s *Server) persistPendingOperation(session *Session, opId int64, opType string, message map[string]interface{}, euiBytes []byte, metadata map[string]interface{}) error {
	ctx := s.safeCtx()

	// An SC operation without a persisted session has no recovery identity;
	// letting it on the wire would make it unrecoverable after a crash. This
	// is an inconsistent-session error, never a silent no-persistence mode.
	if session.DbSessionID == 0 {
		s.logger.ErrorContext(ctx, LogBSSCIDatabaseNotAvailableForPendingOpPersistence,
			"opId", opId,
			"opType", opType)
		return NewCatalogError(errPendingOpSessionNotPersisted, POSIX_EPROTO)
	}

	pendingOp := s.buildPendingOperation(session, opId, opType, message, euiBytes, metadata)

	// StatusService is the single path for pending operation persistence. A
	// failure is surfaced to the caller: an SC operation whose recovery record
	// was never durably written must not go on the wire.
	if err := s.statusSvc.RecordPendingOperation(ctx, session, opId, pendingOp, session.DbSessionID); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToPersistPendingOperationMigrationNeeded,
			"error", err,
			"sessionID", session.DbSessionID,
			"opId", opId)
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistPendingOperation), err)
	}

	s.logger.DebugContext(ctx, LogBSSCIPersistedPendingOperation,
		"sessionID", session.DbSessionID,
		"opId", opId,
		"opType", opType)

	return nil
}

// buildPendingOperation assembles the recovery record for an SC-initiated
// operation, extracting the VM MACType and payload data from metadata when
// present.
func (s *Server) buildPendingOperation(session *Session, opId int64, opType string, message map[string]interface{}, euiBytes []byte, metadata map[string]interface{}) *PendingOperation {
	var macType int
	var data []byte
	if metadata != nil {
		// Extract MACType
		if mt, ok := metadata["macType"].(int); ok {
			macType = mt
		} else if mt, ok := metadata["macType"].(uint8); ok {
			macType = int(mt)
		} else if mt, ok := metadata["macType"].(float64); ok {
			macType = int(mt)
		}

		// Extract Data (VM payload)
		if d, ok := metadata["data"].([]byte); ok {
			data = d
		} else if dataStr, ok := metadata["data"].(string); ok {
			// Handle base64-encoded data
			if decoded, err := base64.StdEncoding.DecodeString(dataStr); err == nil {
				data = decoded
			}
		} else if userDataStr, ok := metadata["userData"].(string); ok && opType == mioty.CmdULDataTransmit {
			// For mioty.CmdULDataTransmit operations, also check userData field
			if decoded, err := base64.StdEncoding.DecodeString(userDataStr); err == nil {
				data = decoded
			}
		}
	}

	return &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opId,
		OperationType: opType,
		Message:       message,
		Endpoint:      euiBytes,
		MACType:       macType,
		Data:          data,
		Metadata:      metadata,
		CreatedAt:     time.Now(),
	}
}

// persistPendingOperationBatch durably records several recovery records in one
// repository transaction (all-or-nothing) so a multi-frame sequence such as
// the dlRxStatQry/dlDataQue pair never has a partially persisted recovery
// state. The same inconsistent-session rule as persistPendingOperation
// applies.
func (s *Server) persistPendingOperationBatch(session *Session, ops []*PendingOperation) error {
	ctx := s.safeCtx()

	if session.DbSessionID == 0 {
		s.logger.ErrorContext(ctx, LogBSSCIDatabaseNotAvailableForPendingOpPersistence,
			"opCount", len(ops))
		return NewCatalogError(errPendingOpSessionNotPersisted, POSIX_EPROTO)
	}

	if err := s.statusSvc.RecordPendingOperations(ctx, session, ops, session.DbSessionID); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToPersistPendingOperationMigrationNeeded,
			"error", err,
			"sessionID", session.DbSessionID,
			"opCount", len(ops))
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistPendingOperation), err)
	}

	return nil
}

// updatePendingOperationMetadata updates the metadata of an existing pending operation (Issue #3: accepts *Session for composite key)
func (s *Server) updatePendingOperationMetadata(session *Session, opId int64, metadata map[string]interface{}) error {
	sessionID := session.DbSessionID
	if sessionID == 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIDatabaseNotAvailableForPendingOpUpdate)
		return nil
	}

	// Convert metadata to JSON for storage
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToMarshalMeta), err)
	}

	// StatusService owns pending-operation persistence and its cache: the DB
	// write happens first, the cache mirror only on success.
	if err := s.statusSvc.UpdatePendingOperationMetadata(s.safeCtx(), session, opId, metadata, json.RawMessage(metadataJSON)); err != nil {
		return err
	}

	s.logger.DebugContext(s.safeCtx(), LogBSSCIUpdatedPendingOperationMetadata,
		"sessionID", sessionID,
		"opId", opId,
		"metadata", metadata)

	return nil
}

// removePendingOperation removes a completed operation from database and memory (Issue #3: accepts *Session for composite key)
// BSSCI §§5.11-5.12.3 Gap 1: Dual-path for test compatibility
func (s *Server) removePendingOperation(session *Session, opId int64) error {
	ctx := s.sessionContext(session)
	// StatusService is the single path for pending operation persistence
	return s.statusSvc.RemovePendingOperation(ctx, session, opId)
}

// isResumableScOperation reports whether a persisted pending operation may be
// reissued on session resume: only SC-initiated (negative ID) non-VM commands
// qualify (BSSCI rev1 §5.3.1 / classic §3.3.1). BS-initiated operations are
// hydrated for response correlation but never transmitted by the SC, and VM
// operations require re-established endpoint VM state before reissue.
func isResumableScOperation(opID int64, opType string) bool {
	if opID >= 0 {
		return false
	}
	switch opType {
	case mioty.CmdStatus, mioty.CmdAttachPropagate, mioty.CmdDetachPropagate,
		mioty.CmdULDataTransmit, mioty.CmdDLDataQueue, mioty.CmdDLDataRevoke,
		mioty.CmdDLRxStatusQuery:
		return true
	default:
		return false
	}
}

// loadPendingOperations loads pending operations from database for session resume (Issue #3: accepts *Session for SessionSlug field)
func (s *Server) loadPendingOperations(session *Session) ([]*PendingOperation, error) {
	sessionID := session.DbSessionID
	if sessionID == 0 || s.statusSvc == nil {
		s.logger.DebugContext(s.safeCtx(), LogBSSCIDatabaseNotAvailableForPendingOpsLoad)
		return nil, nil
	}

	// Retrieve the raw persisted rows through the pending-operation owner
	repoOps, err := s.statusSvc.PersistedOperations(s.safeCtx(), sessionID)
	if err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToQueryPendingOperationsFromDatabase,
			"error", err,
			"sessionID", sessionID)
		return nil, err
	}

	var pendingOps []*PendingOperation
	for _, repoOp := range repoOps {
		opId := repoOp.OperationID
		opType := repoOp.OperationType
		endpointEui := repoOp.EndpointEUI
		operationDataJSON := repoOp.OperationData
		metadataJSON := repoOp.Metadata
		createdAt := repoOp.CreatedAt

		// Deserialize operation data with the strict frame decoder (UseNumber,
		// single object, trailing content rejected) so uint64 EUI values
		// survive resume exactly. A malformed persisted operation is an
		// infrastructure inconsistency: the caller rejects the resume rather
		// than silently losing protocol state.
		operationData, err := decodeJSONFrame(operationDataJSON)
		if err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUnmarshalOperationData,
				"error", err,
				"opId", opId)
			return nil, fmt.Errorf("%s: opId %d: %w", ResolveErrorMessage(errFailedToDecode), opId, err)
		}
		operationData = normalizeStrictDecodedMap(operationData)

		// Deserialize metadata under the same strict rules
		var metadata map[string]interface{}
		if len(metadataJSON) > 0 {
			metadata, err = decodeJSONFrame(metadataJSON)
			if err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUnmarshalMetadata,
					"error", err,
					"opId", opId)
				return nil, fmt.Errorf("%s: opId %d metadata: %w", ResolveErrorMessage(errFailedToDecode), opId, err)
			}
			metadata = normalizeStrictDecodedMap(metadata)
		} else {
			metadata = make(map[string]interface{})
		}

		// Normalize detach metadata on resume to prevent JSON round-trip type drift (BSSCI §5.7.1)
		if opType == mioty.CmdDetach && len(metadata) > 0 {
			if typedMeta := mapToDetachMetadata(metadata); typedMeta != nil {
				// Successfully reconstructed typed metadata, convert back to map with correct types
				normalizedMeta := detachMetadataToMap(typedMeta)
				// Preserve subpackets if present (not part of typed struct)
				if subpackets, ok := metadata["subpackets"]; ok {
					normalizedMeta["subpackets"] = subpackets
				}
				metadata = normalizedMeta
				s.logger.DebugContext(s.safeCtx(), LogBSSCINormalizedDetachMetadataOnResume,
					"opId", opId,
					"epEui", typedMeta.EpEui)
			} else {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToNormalizeDetachMetadataOnResumeUsingRawData,
					"opId", opId)
			}
		}

		// Extract MAC type and data from metadata if available
		var macType int
		var data []byte
		if m, ok := metadata["macType"].(float64); ok {
			macType = int(m)
		}
		if d, ok := metadata["data"].([]byte); ok {
			data = d
		} else if dataStr, ok := metadata["data"].(string); ok {
			// Handle base64-encoded data
			if decoded, err := base64.StdEncoding.DecodeString(dataStr); err == nil {
				data = decoded
			}
		} else if userDataStr, ok := metadata["userData"].(string); ok && opType == mioty.CmdULDataTransmit {
			// For mioty.CmdULDataTransmit operations, also check userData field (legacy compatibility)
			if decoded, err := base64.StdEncoding.DecodeString(userDataStr); err == nil {
				data = decoded
				s.logger.DebugContext(s.safeCtx(), LogBSSCILoadedUserDataFromMetadataForULDataTx,
					"opId", opId,
					"dataLen", len(data))
			}
		}

		pendingOp := &PendingOperation{
			SessionSlug:   session.ID,
			OperationID:   opId,
			OperationType: opType,
			Message:       operationData,
			Endpoint:      endpointEui,
			MACType:       macType,
			Data:          data,
			Metadata:      metadata,
			CreatedAt:     createdAt,
		}

		pendingOps = append(pendingOps, pendingOp)
	}

	s.logger.DebugContext(s.safeCtx(), LogBSSCILoadedPendingOperationsFromDatabase,
		"sessionID", sessionID,
		"count", len(pendingOps))

	return pendingOps, nil
}

// resolveEndpointOwnerContext determines the owning tenant + organization for
// an endpoint by EUI (BSSCI §5.8.3 / §3.9 multi-tenant roaming), converts the
// EUI to its 8-byte big-endian representation for database use, then builds an
// owner-scoped context using Background() to avoid session-context pollution.
// Falls back to the session tenant when the endpoint repository is nil or the
// endpoint is not found in either the session tenant or by global EUI lookup.
func (s *Server) resolveEndpointOwnerContext(ctx context.Context, session *Session, endpointEUI uint64) (
	euiBytes []byte,
	endpointTenantID int64,
	ownerOrgUUID uuid.UUID,
	ownerCtx context.Context,
) {
	euiBytes = make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, endpointEUI)

	if s.endpointRepo == nil {
		// No repository - use session tenant
		endpointTenantID = resolvedTenant(session, s.tenantID)
		ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
		return
	}

	// Try session tenant first (happy path)
	sessionTenantID := resolvedTenant(session, s.tenantID)
	endpoint, err := s.endpointRepo.GetByEUI(ctx, sessionTenantID, euiBytes)

	// Fallback: endpoint roamed from a different tenant — look up by global EUI
	if err == storage.ErrNotFound {
		var eui models.EUI
		binary.BigEndian.PutUint64(eui[:], endpointEUI)
		endpoint, err = s.endpointRepo.Get(ctx, eui)
	}

	if err != nil || endpoint == nil {
		// Endpoint not found in either lookup - use session tenant as fallback
		endpointTenantID = sessionTenantID
		ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
		return
	}

	// Use endpoint's tenant (not session tenant) for all downstream operations
	endpointTenantID = endpoint.TenantID

	// Resolve organization for owner tenant
	if s.orgResolver != nil {
		ownerOrgUUID, err = s.orgResolver.GetDefaultOrgForTenant(ctx, endpointTenantID)
		if err != nil {
			s.logger.WarnContext(ctx, LogBSSCIOrgLookupFailed, "tenantID", endpointTenantID, "error", err)
			// Continue without org - ownerOrgUUID remains Nil
			ownerOrgUUID = uuid.Nil
		}
	}

	// Build owner-scoped context for all downstream operations
	ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
	if ownerOrgUUID != uuid.Nil {
		ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
	}
	return
}

// propagateResponseConfig carries the per-operation catalog references the
// shared propagate-response failure handler needs.
type propagateResponseConfig struct {
	// rejectedLog is the outer "rejected by base station" log token emitted
	// before the failure-handling block runs.
	rejectedLog string
	// failureErrToken is the error-catalog token used both for the error
	// response message (via ResolveErrorMessage) and as a stable identifier
	// in failure metadata.
	failureErrToken string
	// operationType is the snake_case token written to details["operation"]
	// (matches SQL filters in operation_status_queries.go). MUST be one of
	// EventTypeAttachPropagateFailed or EventTypeDetachPropagateFailed.
	operationType string
	// eventType is the snake_case token written to SystemEvent.EventType.
	// MUST be one of EventTypeEndpointAttachFailed or EventTypeEndpointDetachFailed.
	eventType string
	// titleFormat is a `fmt.Sprintf` format string accepting (epEUI, bsEUI)
	// from pkg/bssci/constants.go — TitleAttachPropagateFailedForEndpointOnBS
	// or TitleDetachPropagateFailedForEndpointOnBS.
	titleFormat string
}

// markPendingOpFailed marks the pending operation as failed, merges failure
// metadata into pendingOp.Metadata, and persists the result. Returns the
// resolved owner tenant/org context (roaming-safe), the merged metadata, the
// canonical operation ID, and ok=true iff the session carries a persisted
// operation to update. Callers should gate event creation on ok and
// s.eventStore != nil.
func (s *Server) markPendingOpFailed(
	ctx context.Context,
	session *Session,
	msg *Message,
	result int,
) (ownerTenantID int64, ownerOrg uuid.UUID, metadata map[string]interface{}, opID int64, ok bool) {
	if session.DbSessionID <= 0 {
		return 0, uuid.Nil, nil, 0, false
	}

	var pendingOp *PendingOperation
	if s.statusSvc != nil {
		pendingOp, _ = s.statusSvc.GetPendingOperation(session, int64(msg.OpId))
	}

	metadata = make(map[string]interface{})
	if pendingOp != nil && pendingOp.Metadata != nil {
		for k, v := range pendingOp.Metadata {
			metadata[k] = v
		}
	}
	metadata["failed"] = true
	metadata["failureReason"] = fmt.Sprintf(PropagateFailureReasonFormat, result)
	metadata["failedAt"] = time.Now().UnixNano()
	metadata["failureCode"] = result

	if err := s.updatePendingOperationMetadata(session, msg.OpId, metadata); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistFailureMetadata,
			"error", err,
			"opId", msg.OpId)
	}

	ownerTenantID = resolvedTenant(session, s.tenantID)
	switch v := metadata["tenantId"].(type) {
	case int64:
		ownerTenantID = v
	case float64:
		ownerTenantID = int64(v)
	default:
		s.logger.WarnContext(ctx, LogBSSCIMissingTenantInMetadata, "opId", msg.OpId)
	}

	if orgIDStr, hasOrg := metadata["organizationId"].(string); hasOrg {
		ownerOrg, _ = uuid.Parse(orgIDStr) // Nil UUID is a valid fallback
	}

	opID = int64(msg.OpId)
	if pendingOp != nil {
		opID = pendingOp.OperationID
	}

	return ownerTenantID, ownerOrg, metadata, opID, true
}

// extractEndpointEUIFromMetadata reads the endpoint EUI persisted in pending
// operation metadata, falling back from "epEui" (attach) to "endpointEUI"
// (detach). Returns 0 when neither key carries a parseable EUI.
func extractEndpointEUIFromMetadata(metadata map[string]interface{}) uint64 {
	if v, ok := metadata["epEui"]; ok {
		if eui, parsed := parseMetadataEUI(v); parsed && eui != 0 {
			return eui
		}
	}
	if v, ok := metadata["endpointEUI"]; ok {
		if eui, parsed := parseMetadataEUI(v); parsed {
			return eui
		}
	}
	return 0
}

// recordEndpointFailureEvent emits the endpoint-side SystemEvent for a
// propagate failure with source fields populated for UI filtering.
func (s *Server) recordEndpointFailureEvent(
	ownerCtx context.Context,
	ownerTenantID int64,
	cfg propagateResponseConfig,
	epEUI, bsEUI uint64,
	result int,
	details json.RawMessage,
) {
	epEUIStr := pkgmioty.FormatEUI64(epEUI)
	bsEUIStr := pkgmioty.FormatEUI64(bsEUI)
	if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", ownerTenantID),
		EventType:   cfg.eventType,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityError,
		Title:       fmt.Sprintf(cfg.titleFormat, epEUIStr, bsEUIStr),
		Description: fmt.Sprintf(PropagateFailureDescriptionFormat, result),
		Details:     details,
		Status:      EventStatusNew,
		SourceType:  mioty.SourceTypeEndpoint,
		SourceName:  epEUIStr,
		CreatedAt:   time.Now(),
	}); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateFailureEvent, "error", err)
	}
}

// recordBaseStationFailureEvent emits the symmetric base-station-side
// SystemEvent for a propagate failure (BSSCI §5.8.3 / §3.9 roaming).
func (s *Server) recordBaseStationFailureEvent(
	ownerCtx context.Context,
	ownerTenantID int64,
	cfg propagateResponseConfig,
	epEUI, bsEUI uint64,
	result int,
	details json.RawMessage,
) {
	epEUIStr := pkgmioty.FormatEUI64(epEUI)
	bsEUIStr := pkgmioty.FormatEUI64(bsEUI)
	if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", ownerTenantID),
		EventType:   cfg.eventType,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityError,
		Title:       fmt.Sprintf(TitleRejectedOperationForEndpoint, cfg.operationType, epEUIStr),
		Description: fmt.Sprintf(PropagateFailureDescriptionFormat, result),
		Details:     details,
		Status:      EventStatusNew,
		SourceType:  mioty.SourceTypeBaseStation,
		SourceName:  bsEUIStr,
		CreatedAt:   time.Now(),
	}); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateBaseStationFailureEvent, "error", err)
	}
}

// handlePropagateResponseFailure runs the shared failure path for attach- and
// detach-propagate response handlers when the base station reports a non-zero
// result. It logs the rejection, marks the pending operation as failed with
// metadata, creates symmetric endpoint + base-station system events with
// owner-tenant context (BSSCI §5.8.3 / §3.9 roaming), and sends a
// catalog-derived error response to the base station per BSSCI-4-01. Returns
// nil to keep the connection open — an individual operation failure should
// not close the session.
func (s *Server) handlePropagateResponseFailure(
	ctx context.Context,
	session *Session,
	msg *Message,
	result int,
	cfg propagateResponseConfig,
) error {
	s.logger.ErrorContext(s.safeCtx(), cfg.rejectedLog,
		"baseStation", session.BaseStationEUI,
		"opId", msg.OpId,
		"result", result)

	ownerTenantID, ownerOrg, metadata, opID, ok := s.markPendingOpFailed(ctx, session, msg, result)
	if ok && s.eventStore != nil {
		ownerCtx := pkgcontext.WithTenantID(ctx, ownerTenantID)
		if ownerOrg != uuid.Nil {
			ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrg)
		}

		epEUI := extractEndpointEUIFromMetadata(metadata)
		bsEUI := session.BaseStationEUI

		details := map[string]interface{}{
			"epEui":        pkgmioty.FormatEUI64(epEUI),
			"bsEui":        pkgmioty.FormatEUI64(bsEUI),
			"operation":    cfg.operationType,
			"operation_id": fmt.Sprintf("%d", opID),
			"failureCode":  result,
			"reason":       fmt.Sprintf(PropagateFailureReasonFormat, result),
		}
		detailsJSON, _ := json.Marshal(details)

		s.recordEndpointFailureEvent(ownerCtx, ownerTenantID, cfg, epEUI, bsEUI, result, detailsJSON)
		s.recordBaseStationFailureEvent(ownerCtx, ownerTenantID, cfg, epEUI, bsEUI, result, detailsJSON)
	}

	// Send error response to base station per BSSCI-4-01. POSIX error code is
	// mandatory; the response message is sourced from the catalog so attach and
	// detach paths produce identical wire format.
	errorMsg := map[string]interface{}{
		"command": mioty.CmdError,
		"opId":    msg.OpId,
		"code":    POSIX_EPROTO,
		"message": ResolveErrorMessage(cfg.failureErrToken),
	}

	if err := s.sendMessage(session, errorMsg); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorMessageToBaseStation,
			"error", err)
	}

	// Keep connection open - individual operation failure shouldn't close session
	return nil
}

// toFloat64Value coerces a numeric interface{} value to float64, returning
// ok=false when the value is nil or not a recognized numeric type.
func toFloat64Value(value interface{}) (float64, bool) {
	if value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

// getFloat64Field extracts a float64 field from a map[string]interface{} payload.
// Returns ok=false when the key is missing or the value is nil/non-numeric.
func getFloat64Field(data map[string]interface{}, key string) (float64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}
	return toFloat64Value(value)
}

// toFloat64 converts an interface{} value to float64, handling common numeric types.
// Used for parsing array elements in geoLocation and similar fields. Delegates
// to toFloat64Value; preserved as the public-within-package API used by
// status_handlers.go.
func toFloat64(value interface{}) (float64, bool) {
	return toFloat64Value(value)
}

// ============================================================================
// BSSCI Validation Functions (BSSCI-2.x, BSSCI-3.x, BSSCI-4.x)
// ============================================================================

// sendError sends a BSSCI error message (BSSCI-4-01, BSSCI-4-02)
func (s *Server) sendError(session *Session, opId int64, code int, message string) error {
	// Use canonical Error struct (BSSCI §4-4.5)
	errorMsg := mioty.Error{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdError,
			OpId:        opId,
		},
		Code:    code,
		Message: message,
	}

	// Use context.Background() if s.ctx is nil (e.g., in tests)
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	s.logger.WarnContext(ctx, LogBSSCISendingBSSCIError,
		"opId", opId,
		"code", code,
		"message", message)

	if err := s.sendMessage(session, errorMsg); err != nil {
		return err
	}

	// The base station will answer with errorAck (BSSCI rev1 §5.17 / classic
	// §3.17). Record the expectation so handleErrorAck only acts on
	// acknowledgements this service center actually solicited. Plain
	// rejections are ack-only; sendErrorReplacingOperation registers the
	// finalizing disposition for errors that replace a pending SC operation.
	session.registerPendingErrorAck(opId, errorAckAckOnly)
	return nil
}

// sendErrorReplacingOperation sends an error frame that replaces the normal
// response/completion sequence of a known pending SC operation (BSSCI rev1
// §5.17 / classic §3.17). The base station's errorAck then completes that
// operation, so the errorAck is registered with the finalizing disposition.
func (s *Server) sendErrorReplacingOperation(session *Session, opId int64, code int, message string) error {
	if err := s.sendError(session, opId, code, message); err != nil {
		return err
	}
	session.registerPendingErrorAck(opId, errorAckFinalizePendingOperation)
	return nil
}

// sendCatalogError resolves catalog error token and sends error message
// Services return *CatalogError, transport resolves token via ResolveErrorMessage
func (s *Server) sendCatalogError(session *Session, opId int64, err *CatalogError) error {
	resolvedMsg := ResolveErrorMessage(err.Token)
	if sendErr := s.sendError(session, opId, err.Posix, resolvedMsg); sendErr != nil {
		// Use context.Background() if s.ctx is nil (e.g., in tests)
		ctx := s.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		s.logger.ErrorContext(ctx, LogBSSCIFailedToSendCatalogError, "error", sendErr)
	}
	return fmt.Errorf("%s", err.Token)
}

// validateOperationID validates operation IDs per BSSCI-3.2 requirements
func (s *Server) validateOperationID(session *Session, opId int64, isBaseStationInitiated bool) string {
	// Connect operation must use ID 0 (BSSCI-3.2-03)
	if session.BaseStationEUI == 0 && opId != 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIInvalidConnectOperationID,
			"op_id", opId)
		return errConnectOpIDNotZero
	}

	// IMPORTANT: All messages within the same operation use the same opId
	// Only NEW operations require strictly incrementing/decrementing IDs

	if isBaseStationInitiated {
		// Base Station operations must use positive IDs (BSSCI-3.2-01)
		if opId <= 0 && opId != 0 { // Allow 0 for connect
			s.logger.WarnContext(s.safeCtx(), LogBSSCIOperationIDNotPositive,
				"op_id", opId)
			return errOperationIDNotPositive
		}

		// Only update LastBsOpID if this is a NEW operation (greater than last)
		// Messages within same operation reuse the same opId
		if opId != 0 && opId > session.LastBsOpId {
			// This is a new operation, must be strictly incrementing
			session.LastBsOpId = opId
		} else if opId != 0 && opId < session.LastBsOpId {
			// This is an old operation ID being reused incorrectly
			s.logger.WarnContext(s.safeCtx(), LogBSSCIOperationIDBackwards,
				"op_id", opId,
				"last_bs_op_id", session.LastBsOpId)
			return errOperationIDBackwards
		}
		// If opId == session.LastBsOpId, it's part of the same operation - allowed
	} else {
		// Service Center initiated operations (BSSCI §3.2-02)
		// SC operations MUST use negative operation IDs
		if opId >= 0 {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCISCOperationIDMustNotIncrease,
				"opId", opId)
			return errSCOperationIDMustBeNegative
		}

		// With atomic send-and-persist pattern, LastScOpID is updated when we SEND
		// operations (not during validation). Validation only checks responses match what we sent.
		//
		// Special case: initial state (LastScOpID = 0) accepts any negative ID for backwards
		// compatibility and session resume scenarios.
		if session.LastScOpId == 0 {
			// Initial state - accept any negative operation ID
			// With atomic pattern, we update LastScOpID when sending, not here
			return ""
		}

		// Normal validation: check against tracked operation counter
		// For negative SC IDs: more negative = newer, less negative = older (pending)
		// Accept responses for any operation between LastScOpId and 0 (pending operations)
		if opId < session.LastScOpId {
			// opId more negative than LastScOpId = future operation we haven't sent yet
			s.logger.WarnContext(s.safeCtx(), LogBSSCISCOperationIDMustNotIncrease,
				"op_id", opId,
				"last_sc_op_id", session.LastScOpId)
			return errOperationIDIncreasing
		}
		// opId >= LastScOpId (less negative or equal) = valid pending operation response
	}

	return ""
}

// extractSessionUUID extracts UUID from various formats (BSSCI-3.3)
func extractSessionUUID(data interface{}) ([]byte, *CatalogError) {
	if data == nil {
		return nil, &CatalogError{Token: errUUIDDataNil, Posix: POSIX_EPROTO}
	}

	switch v := data.(type) {
	case []interface{}:
		if len(v) != 16 {
			return nil, &CatalogError{Token: errInvalidUUIDLength, Posix: POSIX_EPROTO}
		}
		uuid := make([]byte, 16)
		for i, val := range v {
			if b, ok := val.(int8); ok {
				// MessagePack encodes bytes above 0x7F as negative int8;
				// the two's-complement bit pattern is the intended byte
				uuid[i] = byte(b) //nolint:gosec // G115: intentional two's-complement byte extraction
				continue
			}
			b, err := numericToByte(val)
			if err != nil {
				return nil, &CatalogError{Token: errInvalidUUIDByteType, Posix: POSIX_EPROTO}
			}
			uuid[i] = b
		}
		return uuid, nil
	case []byte:
		if len(v) != 16 {
			return nil, &CatalogError{Token: errInvalidUUIDLength, Posix: POSIX_EPROTO}
		}
		return v, nil
	default:
		return nil, &CatalogError{Token: errUnsupportedUUIDType, Posix: POSIX_EPROTO}
	}
}

// updateSessionCounters updates the session operation counters in the database
func (s *Server) updateSessionCounters(session *Session) {
	if session.DbSessionID == 0 || s.sessionSvc == nil {
		return
	}

	// SessionService owns counter persistence via the atomic UpdateOperationIDs
	// statement; this fire-and-forget path keeps its debug-only error handling.
	if err := s.sessionSvc.UpdateSessionCounters(s.safeCtx(), session); err != nil {
		s.logger.DebugContext(s.safeCtx(), LogBSSCIFailedToUpdateSessionCounters,
			"error", err,
			"bsOpId", session.LastBsOpId,
			"scOpId", session.LastScOpId)
	}
}

// ConnectedSessionsSnapshot returns lightweight snapshots of all connected base station sessions
// Used by propagation service to broadcast attach propagate messages to multiple base stations
// Implements SessionSnapshotProvider interface for SCACI integration
//
// BSSCI §5.8-5.8.3: Automatic endpoint propagation across multi-BS networks
func (s *Server) ConnectedSessionsSnapshot() []propagation.BaseStationSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshots := make([]propagation.BaseStationSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		if !session.HandshakeComplete {
			continue
		}

		var orgIDStr *string
		if session.OrganizationID != uuid.Nil {
			orgStr := session.OrganizationID.String()
			orgIDStr = &orgStr
		}

		snapshots = append(snapshots, propagation.BaseStationSession{
			ID:                session.ID,
			BaseStationEUI:    session.BaseStationEUI,
			TenantID:          session.ResolvedTenantID,
			OrganizationID:    orgIDStr,
			HandshakeComplete: session.HandshakeComplete,
		})
	}

	return snapshots
}
