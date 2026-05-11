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
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
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

	// Concrete dependencies KEPT (needed by services)
	connectionMgr *basestation.ConnectionManager // Real collaborator for ConnectionService
	storage       interfaces.Storage             // Uses repository interfaces

	// Injected services
	sessionSvc      SessionService
	downlinkSvc     DownlinkService
	statusSvc       StatusService
	connectionSvc   ConnectionService
	broadcaster     SCACIBroadcaster
	queueSerializer QueueSerializer // Downlink response frame builder
	auditLogger     AuditLogger     // Downlink audit event recorder
	tenantResolver  TenantResolver  // Queue-to-tenant mapping (replaces queueTenants map)

	// Organization resolution
	orgResolver     org.Resolver // Organization UUID → tenant ID resolution
	defaultTenantID int64        // Community mode fallback tenant ID

	// Keep essential infrastructure
	deduplicator    *MessageDeduplicator
	eventStore      interfaces.SystemEventStore
	basestationRepo interfaces.BaseStationRepository
	endpointRepo    interfaces.EndpointRepository
	keyEncryptor    *crypto.KeyEncryptor

	// Automatic propagation (BSSCI §5.8.3)
	propagationSvc propagation.Service

	// Multi-tenant roaming support
	roamingSvc RoamingService

	// Uplink disposition routing: Local, Relay (CE→ECE), or Drop for unknown endpoints
	dispositionResolver IngressDispositionResolver

	// Shared uplink ingest pipeline (dedup, tenant resolution, persistence, SCACI, MQTT)
	uplinkIngestSvc UplinkIngestService

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

	// broadcastFn allows tests to inject stub implementations for SendAttachPropagateToAll.
	// Production code initializes this to s.SendAttachPropagateToAll in constructors.
	broadcastFn func(endpointEUI uint64, nwkSnKey []byte, shortAddr uint16, bidirectional bool, lastPacketCnt uint32, dualChannel bool, repetition uint8, wideCarrOff bool, longBlkDist bool) []error

	// testNormalizationSpy allows tests to observe normalization decisions in handleMessage.
	// Production code leaves this nil. Tests set it to capture (command, shouldNormalize) pairs.
	testNormalizationSpy func(cmd string, normalized bool)
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
	// DisableAttachPersistence is TEST-ONLY: skips DB persistence in attach handler.
	// MUST remain false in production. Used by tests to exercise replay protection without transaction stubs.
	DisableAttachPersistence bool
}

// Session represents a connected Base Station session
//
//revive:disable:var-naming BsOpId/ScOpId/LastBsOpId/LastScOpId use lowercase 'd' per MIOTY BSSCI §3.2
type Session struct {
	ID                string
	BaseStationEUI    uint64
	Conn              net.Conn
	Connected         time.Time
	LastSeen          time.Time
	ClientVersion     string // BS-provided version (raw client claim for audit)
	NegotiatedVersion string // SC canonical version (BSSCI §4-4.5)
	Vendor            string
	Model             string
	Name              string
	SoftwareVersion   string
	Bidirectional     bool
	GeoLocation       []float64
	SessionUUID       []byte
	// MIOTY session fields
	DbSessionID      int64              // Database session ID (BIGINT) for persistence
	UserProvidedName string             // User-provided base station name
	ActiveVMTypes    map[uint64][]uint8 // Track active Variable MAC types per endpoint
	stopStatus       chan struct{}      // Channel to stop status mechanism
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
	OrganizationID   uuid.UUID         // Kilo Cloud org UUID (from TLS cert or community fallback)
	ResolvedTenantID int64             // Tenant ID resolved from cert/org (vs server default s.tenantID)
	ClientCert       *x509.Certificate // TLS client certificate for org resolution
	mu               sync.Mutex
}

// BaseStationEUIBytes converts the BaseStationEUI uint64 to a byte slice.
// This is needed for roaming service calls that expect []byte parameters.
func (s *Session) BaseStationEUIBytes() []byte {
	euiBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		euiBytes[i] = byte(s.BaseStationEUI >> (8 * (7 - i)))
	}
	return euiBytes
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

	// Extract opId - handle int64 or float64
	opIdVal, hasOpId := v["opId"]
	if !hasOpId {
		return "", 0, false
	}

	var opId int64
	switch typed := opIdVal.(type) {
	case int64:
		opId = typed
	case float64:
		opId = int64(typed)
	default:
		return "", 0, false // Wrong type, needs JSON fallback
	}

	return command, opId, true
}

// wrapOutboundMessage converts interface{} to *Message for consistent RawPayload capture
// Handles *Message, map[string]interface{}, and typed structs (ConnectResponse, etc.) via JSON round-trip
func (s *Server) wrapOutboundMessage(msg interface{}) (*Message, error) {
	switch v := msg.(type) {
	case *Message:
		return v, nil
	case map[string]interface{}:
		// Try direct extraction first
		command, opId, ok := tryExtractFromMap(v)
		if !ok {
			// Fall back to JSON round-trip normalization
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal map: %w", err)
			}
			var normalized map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
				return nil, fmt.Errorf("failed to unmarshal map: %w", err)
			}

			// Retry extraction after normalization
			command, opId, ok = tryExtractFromMap(normalized)
			if !ok {
				return nil, fmt.Errorf("map missing command/opId after normalization")
			}

			// Populate normalized map with extracted values
			normalized["command"] = command
			normalized["opId"] = opId
			v = normalized // Callers see sanitized values
		} else {
			// Direct extraction succeeded - ensure keys are normalized in original map
			v["command"] = command
			v["opId"] = opId
		}

		return &Message{
			Command: command,
			OpId:    opId,
			Data:    v,
		}, nil
	default:
		// Handle typed structs (ConnectResponse, PingResponse, etc.) via JSON marshaling
		jsonBytes, err := json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message: %w", err)
		}
		var msgMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &msgMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %w", err)
		}

		// Extract command from commandType or command field (MIOTY spec variance)
		command, ok := msgMap["commandType"].(string)
		if !ok {
			command, ok = msgMap["command"].(string)
			if !ok {
				return nil, fmt.Errorf("message missing command/commandType field (type: %T)", msg)
			}
		}

		// Extract opId (JSON unmarshals numbers as float64)
		opId, ok := msgMap["opId"].(float64)
		if !ok {
			return nil, fmt.Errorf("message missing opId field (type: %T)", msg)
		}

		// CRITICAL: Populate msgMap with normalized values for validation
		// Without this, validateOutboundMessage fails with missing command/opId
		msgMap["command"] = command
		msgMap["opId"] = int64(opId)

		return &Message{
			Command: command,
			OpId:    int64(opId),
			Data:    msgMap,
		}, nil
	}
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
// Returns nil on success, CatalogError on validation failure.
func (s *Server) validateOutboundMessage(session *Session, message interface{}) error {
	// Marshal to map for field inspection
	var msgMap map[string]interface{}
	switch v := message.(type) {
	case map[string]interface{}:
		msgMap = v
	default:
		// Convert via JSON round-trip
		jsonBytes, err := json.Marshal(message)
		if err != nil {
			return &CatalogError{Token: errOutboundMarshalFailed, Posix: POSIX_EPROTO}
		}
		if err := json.Unmarshal(jsonBytes, &msgMap); err != nil {
			return &CatalogError{Token: errOutboundMarshalFailed, Posix: POSIX_EPROTO}
		}
	}

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

// NewServer creates a new BSSCI server with injected service dependencies
func NewServer(
	cfg *Config,
	log logger.Logger,
	connectionMgr *basestation.ConnectionManager,
	stor interfaces.Storage,
	eventStore interfaces.SystemEventStore,
	basestationRepo interfaces.BaseStationRepository,
	endpointRepo interfaces.EndpointRepository,
	tenantID int64,
	sessionSvc SessionService,
	downlinkSvc DownlinkService,
	statusSvc StatusService,
	connectionSvc ConnectionService,
	broadcaster SCACIBroadcaster,
	queueSerializer QueueSerializer,
	auditLogger AuditLogger,
	tenantResolver TenantResolver,
	orgResolver org.Resolver,
	defaultTenantID int64,
) (*Server, error) {
	// StatusService is mandatory (single-writer architecture)
	if statusSvc == nil {
		return nil, fmt.Errorf("statusSvc is required for pending operation tracking")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize key encryptor for sensitive data
	keyEncryptor, err := crypto.NewKeyEncryptor()
	if err != nil {
		log.Warn(LogBSSCIFailedToInitializeKeyEncryptor, "error", err)
		// Continue without encryption rather than failing
	}

	s := &Server{
		config:          cfg,
		logger:          log,
		sessions:        make(map[string]*Session),
		ctx:             ctx,
		cancel:          cancel,
		handlers:        make(map[string]HandlerFunc),
		connectionMgr:   connectionMgr,
		deduplicator:    NewMessageDeduplicator(5 * time.Minute), // 5 minute dedup window per MIOTY spec
		storage:         stor,
		tenantID:        tenantID,
		eventStore:      eventStore,
		basestationRepo: basestationRepo,
		endpointRepo:    endpointRepo,
		keyEncryptor:    keyEncryptor,
		// Injected services
		sessionSvc:      sessionSvc,
		downlinkSvc:     downlinkSvc,
		statusSvc:       statusSvc,
		connectionSvc:   connectionSvc,
		queueSerializer: queueSerializer,
		auditLogger:     auditLogger,
		tenantResolver:  tenantResolver,
		broadcaster:     broadcaster,
		// Organization resolution
		orgResolver:     orgResolver,
		defaultTenantID: defaultTenantID,
	}

	// Initialize broadcast hook for production (tests can override)
	s.broadcastFn = s.SendAttachPropagateToAll

	// Register command handlers
	s.registerHandlers()

	return s, nil
}

// SetPropagationService injects the propagation service after server construction
// Post-construction initialization to avoid circular dependency in main.go
// The propagation service depends on Server as AttachPropagateSender, so it must be
// constructed after the server exists and then injected back.
//
// BSSCI §5.8-5.8.3: Automatic endpoint propagation to multiple base stations
func (s *Server) SetPropagationService(svc propagation.Service) {
	s.propagationSvc = svc
}

// SetRoamingService injects the roaming service after server construction
// Post-construction initialization for modularity and testability.
// The roaming service handles multi-tenant endpoint ownership resolution.
func (s *Server) SetRoamingService(svc RoamingService) {
	s.roamingSvc = svc
}

// SetDispositionResolver injects the ingress disposition resolver.
// The resolver classifies each incoming uplink as Local, Relay, or Drop before ingest.
func (s *Server) SetDispositionResolver(r IngressDispositionResolver) {
	s.dispositionResolver = r
}

// SetUplinkIngestService injects the shared uplink ingest pipeline.
// When set, handleULData delegates dedup, tenant resolution, persistence, SCACI, and MQTT to this service.
func (s *Server) SetUplinkIngestService(svc UplinkIngestService) {
	s.uplinkIngestSvc = svc
}

// SetRelayOutboxWriter injects the CE federation relay outbox writer.
// When set, handleULData enqueues DispositionRelay uplinks instead of dropping them.
func (s *Server) SetRelayOutboxWriter(w RelayOutboxWriter) {
	s.relayOutbox = w
}

// GetDeduplicator exposes the server's message deduplicator for injection into the ingest service.
// The same deduplicator instance must be shared so dedup state is consistent.
func (s *Server) GetDeduplicator() *MessageDeduplicator {
	return s.deduplicator
}

// SetDetachValidator wires the detach signature validator for unknown endpoint validation.
// Called after service construction to enable signature validation via internal validator.
func (s *Server) SetDetachValidator(validator DetachSignatureValidator) {
	s.detachValidator = validator
}

// SetDownlinkDispatcher wires the auto-dispatch service (BSSCI §5.10.2)
// Called after service construction to enable dlOpen=true automatic dispatch.
func (s *Server) SetDownlinkDispatcher(dispatcher DownlinkDispatcher) {
	s.downlinkDispatcher = dispatcher
}

// SetBroadcaster wires the SCACI broadcaster for uplink/downlink forwarding.
// Called after SCACI server construction to complete the BSSCI → SCACI bridge.
// Preserves startup order: BSSCI → SCACI → wire broadcaster.
//
// Legacy scaci parameter removed - broadcaster is now the only interface.
func (s *Server) SetBroadcaster(broadcaster SCACIBroadcaster) {
	s.broadcaster = broadcaster
}

// SetSCACIEPStatusBroadcaster wires the SCACI EPStatus broadcaster for attach/detach forwarding.
// Called after SCACI server construction to complete the BSSCI → SCACI EPStatus bridge.
// Per SCACI §3.13: SC sends EPStatus to all ACs when endpoints attach/detach.
func (s *Server) SetSCACIEPStatusBroadcaster(broadcaster SCACIEPStatusBroadcaster) {
	s.scaciEPStatusBroadcaster = broadcaster
}

// SetBlueprintDecoder wires the blueprint decoder service for payload decoding.
// Per MIOTY Application Layer Specification: decode payloads using blueprint JSON definitions.
func (s *Server) SetBlueprintDecoder(decoder BlueprintDecoder) {
	s.blueprintDecoder = decoder
}

// SetBlueprintResolver wires the blueprint resolver service for blueprint lookup.
// Per MIOTY Application Layer Specification: resolve blueprints by TypeEUI or device model.
func (s *Server) SetBlueprintResolver(resolver BlueprintResolver) {
	s.blueprintResolver = resolver
}

// SetMQTTPublisher wires the MQTT event publisher for device lifecycle events.
// Called after publisher creation to enable outbound MQTT publishing from BSSCI handlers.
func (s *Server) SetMQTTPublisher(pub MQTTEventPublisher) {
	s.mqttPublisher = pub
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
	s.handlers[mioty.CmdStatusComplete] = s.handleStatusComplete
	s.handlers[mioty.CmdAttach] = s.handleAttach
	s.handlers[mioty.CmdAttachComplete] = s.handleAttachComplete
	s.handlers[mioty.CmdDetach] = s.handleDetach
	s.handlers[mioty.CmdDetachComplete] = s.handleDetachComplete
	s.handlers[mioty.CmdULData] = s.handleULData
	s.handlers[mioty.CmdULDataComplete] = s.handleULDataComplete
	// UL Data Transmit response handlers (SC-initiated, so no ulDataTx handler)
	s.handlers[mioty.CmdULDataTransmitResponse] = s.handleULDataTxResponse
	s.handlers[mioty.CmdULDataTransmitComplete] = s.handleULDataTxComplete
	s.handlers[mioty.CmdError] = s.handleError
	s.handlers[mioty.CmdErrorAck] = s.handleErrorAck
	// Attach/Detach Propagate handlers
	s.handlers[mioty.CmdAttachPropagateResponse] = s.handleAttachPropagateResponse
	s.handlers[mioty.CmdAttachPropagateComplete] = s.handleAttachPropagateComplete
	s.handlers[mioty.CmdDetachPropagateResponse] = s.handleDetachPropagateResponse
	s.handlers[mioty.CmdDetachPropagateComplete] = s.handleDetachPropagateComplete
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
	s.handlers[mioty.CmdDLRxStatusQueryComplete] = s.handleDLRXStatusQueryComplete
	// DL Data Revoke response handlers (BSSCI §3.13 - SC-initiated)
	s.handlers[mioty.CmdDLDataRevokeResponse] = s.handleDLDataRevokeResponse
	s.handlers[mioty.CmdDLDataRevokeComplete] = s.handleDLDataRevokeComplete
	// DL Data Queue response handlers (BSSCI §3.12 - SC-initiated)
	s.handlers[mioty.CmdDLDataQueueResponse] = s.handleDLDataQueueResponse
	s.handlers[mioty.CmdDLDataQueueComplete] = s.handleDLDataQueueComplete
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
func (s *Server) Start() error {
	if certsExist(s.config.TLSCert, s.config.TLSKey, s.config.TLSCACert) {
		return s.startTLSListener()
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

	const pollInterval = 10 * time.Second
	ticker := time.NewTicker(pollInterval)
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
	s.cancel()
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to close listener", "error", err)
		}
	}
	if s.deduplicator != nil {
		s.deduplicator.Stop()
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
			s.logger.ErrorContext(s.safeCtx(), "Failed to close connection", "error", err)
		}
	}()

	s.logger.InfoContext(s.safeCtx(), LogBSSCINewConnection, "remote", conn.RemoteAddr().String())

	session := &Session{
		ID:               uuid.New().String(),
		Conn:             conn,
		Connected:        time.Now(),
		LastSeen:         time.Now(),
		ResolvedTenantID: s.defaultTenantID, // Initialize to server default
		// Encoding left empty - detected on first frame per BSSCI Section 1
	}

	// Extract TLS client certificate and resolve organization
	if tlsConn, ok := conn.(*tls.Conn); ok {
		// Trigger TLS handshake if not already done
		if err := tlsConn.Handshake(); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "TLS handshake failed", "error", err)
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

			if s.orgResolver != nil {
				orgID, tenantID, err = s.orgResolver.ResolveCert(ctx, cert)
				if err != nil {
					// Certificate-based resolution failed - use community fallback
					s.logger.WarnContext(ctx, "BSSCI cert org resolution failed, using community fallback",
						"error", err,
						"certCN", cert.Subject.CommonName,
						"tenantID", session.ResolvedTenantID)

					// Get default org for the server's default tenant
					orgID, err = s.orgResolver.GetDefaultOrgForTenant(ctx, session.ResolvedTenantID)
					if err != nil {
						s.logger.ErrorContext(ctx, "Failed to resolve default org for BSSCI session",
							"error", err,
							"tenantID", session.ResolvedTenantID)
						// Continue with nil UUID - will be NULL in database
						orgID = uuid.Nil
						// ResolvedTenantID already set to s.defaultTenantID in session init
					}
					// else: fallback succeeded, ResolvedTenantID still s.defaultTenantID
				} else {
					// SUCCESS: cert resolution worked, use resolved tenant
					session.ResolvedTenantID = tenantID
					s.logger.InfoContext(ctx, "BSSCI cert org resolution succeeded",
						"orgID", orgID.String(),
						"tenantID", tenantID,
						"certCN", cert.Subject.CommonName)
				}
			} else {
				// No org resolver - community edition
				orgID = uuid.Nil
			}

			session.OrganizationID = orgID
			s.logger.DebugContext(ctx, "BSSCI session org/tenant resolved",
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
					s.logger.WarnContext(ctx, "No peer cert and failed to resolve default org for BSSCI session",
						"error", err,
						"tenantID", session.ResolvedTenantID)
					orgID = uuid.Nil
				}
			}
			session.OrganizationID = orgID
			s.logger.DebugContext(ctx, "BSSCI session no peer cert, using defaults",
				"orgID", orgID.String(),
				"tenantID", session.ResolvedTenantID)
		}
	}

	// Ensure we update status to offline when connection ends
	defer func() {
		// Stop status mechanism safely
		session.mu.Lock()
		if session.stopStatus != nil {
			close(session.stopStatus)
			session.stopStatus = nil // Prevent double-close
		}
		session.mu.Unlock()

		if session.BaseStationEUI != 0 && s.connectionMgr != nil {
			// Convert uint64 EUI to [8]byte format
			var euiBytes [8]byte
			for i := 7; i >= 0; i-- {
				euiBytes[i] = byte(session.BaseStationEUI >> (8 * (7 - i)))
			}

			// Update connection status to offline
			status := &basestation.ConnectionStatus{
				IsOnline:       false,
				LastSeen:       time.Now(),
				ConnectionType: basestation.ConnectionTypeBSSCI,
				SessionID:      session.ID,
			}

			ctx := s.sessionContext(session)
			if err := s.connectionMgr.UpdateConnectionStatus(ctx, euiBytes, status); err != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateOfflineStatus,
					"eui", session.BaseStationEUI,
					"error", err)
			} else {
				s.logger.InfoContext(ctx, LogBSSCIBaseStationDisconnectedStatusOffline,
					"eui", session.BaseStationEUI,
					"name", session.Name)
			}

			// Remove from sessions map
			s.mu.Lock()
			delete(s.sessions, session.ID)
			s.mu.Unlock()

			// Also remove from SessionService's sessionsByUUID map to prevent stale resume
			s.sessionSvc.RemoveSession(session)

			// Terminate database session per BSSCI §3 "new session starts, discarding state"
			if session.DbSessionID != 0 && s.sessionSvc != nil {
				if err := s.sessionSvc.TerminateSession(ctx, session); err != nil {
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

			// Clean up pending operations for this session (bulk delete)
			if session.DbSessionID != 0 && s.storage != nil {
				// Guard against nil repository before calling methods
				pendingOpsRepo := s.storage.PendingOperations()
				if pendingOpsRepo != nil {
					count, err := pendingOpsRepo.DeleteBySession(ctx, session.DbSessionID)
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
			}
		}
	}()

	// Read messages
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Set read deadline
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to set read deadline", "error", err)
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
				if err := s.sessionSvc.UpdateEncoding(s.safeCtx(), session.DbSessionID, encoding); err != nil {
					s.logger.ErrorContext(s.safeCtx(), "Failed to persist encoding",
						"encoding", encoding,
						"sessionID", session.DbSessionID,
						"error", err)
					// Non-fatal: continue with in-memory encoding
				}
			}

			s.logger.InfoContext(s.safeCtx(), "Detected message encoding",
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

		// BSSCI-3.2: Validate operation ID (except for connect which is special)
		if command != mioty.CmdConnect {
			// Determine if this is a base station initiated operation
			// Base station operations: positive IDs (and opId=0 for connect handshake per BSSCI §5.3)
			// Service center operations: negative IDs
			isBaseStationInitiated := opId >= 0

			if errToken := s.validateOperationID(session, opId, isBaseStationInitiated); errToken != "" {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIOperationIDValidationFailed,
					"command", command,
					"opId", opId,
					"error_token", errToken)
				if err := s.sendError(session, opId, POSIX_EPROTO, ResolveErrorMessage(errToken)); err != nil {
					s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
				}
				return
			}
		}

		// Update last seen
		session.LastSeen = time.Now()

		// Update last seen in database for connected basestations
		if session.BaseStationEUI != 0 && s.connectionMgr != nil {
			var euiBytes [8]byte
			for i := 7; i >= 0; i-- {
				euiBytes[i] = byte(session.BaseStationEUI >> (8 * (7 - i)))
			}

			// Update last seen time in database
			go func() {
				ctx := s.sessionContext(session)
				if err := s.connectionMgr.UpdateLastSeen(ctx, euiBytes); err != nil {
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

// handleMessage routes messages to appropriate handlers
func (s *Server) handleMessage(session *Session, msg *Message, data map[string]interface{}) error {
	// BSSCI §2.4: Normalize incoming payload to validate fields and detect unknown fields
	// Forward compatibility per §2.4-01
	// Issue #3-4 Fix: Only normalize BS→SC commands (inbound) to avoid validating our own SC→BS responses
	ctx := s.sessionContext(session)

	// Check if this command should be normalized (only BS→SC and bidirectional inbound commands)
	direction, exists := CommandDirectionMap[msg.Command]
	shouldNormalize := exists && (direction == DirectionBStoSC || direction == DirectionBidirectional)

	// Notify test spy if present (for TestNormalizationOnlyForBStoSCCommands)
	if s.testNormalizationSpy != nil {
		s.testNormalizationSpy(msg.Command, shouldNormalize)
	}

	// Only normalize inbound BS→SC and bidirectional commands (skip SC→BS responses and unknown commands for safety)
	if shouldNormalize {
		normalizedData, err := normalizePayload(ctx, s.logger, msg.Command, data)
		if err != nil {
			// Normalization failed (mandatory field missing, invalid type, etc.)
			// Send error response per BSSCI protocol
			s.logger.ErrorContext(ctx, "Payload normalization failed",
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

			if sendErr := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken)); sendErr != nil {
				s.logger.ErrorContext(ctx, "Failed to send error response", "error", sendErr)
			}
			return nil // Error sent via protocol, don't close connection
		}
		// Use normalized data for handler (unknown fields removed, types coerced)
		data = normalizedData
	}

	// BSSCI-3.3-03: No operations allowed until connect handshake is complete
	// Only con, conRsp, conCmp, error, and errorAck are permitted during handshake
	if !session.HandshakeComplete {
		allowedCommands := map[string]bool{
			mioty.CmdConnect:         true,
			mioty.CmdConnectResponse: true,
			mioty.CmdConnectComplete: true,
			mioty.CmdError:           true,
			mioty.CmdErrorAck:        true,
		}

		if !allowedCommands[msg.Command] {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingCommandBeforeHandshake,
				"command", msg.Command,
				"handshake_complete", session.HandshakeComplete,
				"opId", msg.OpId,
				"bsEui", session.BaseStationEUI)
			if err := s.sendError(session, msg.OpId, POSIX_EPROTO,
				ResolveErrorMessage(errCommandBeforeHandshake)); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
			}
			return nil // Don't close connection, just reject the command
		}
	}

	// Check for sublayer prefixes (BSSCI-4-02)
	// Allow both rc.* (remote control) and vm.* (virtual machine) sublayers
	if strings.Contains(msg.Command, ".") {
		// Only allow known sublayer prefixes
		if !strings.HasPrefix(msg.Command, "rc.") && !strings.HasPrefix(msg.Command, "vm.") {
			// Send error for unsupported sublayer
			if err := s.sendError(session, msg.OpId, POSIX_ENOTSUP, ResolveErrorMessage(errUnsupportedSublayerPrefix)); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
			}
			return nil // Don't close connection
		}
	}

	handler, ok := s.handlers[msg.Command]
	if !ok {
		// Send BSSCI error message instead of closing socket (BSSCI-4-01)
		if err := s.sendError(session, msg.OpId, POSIX_ENOTSUP, ResolveErrorMessage(errUnsupportedCommand)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
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

	// BSSCI §2.5.1: Validate outbound message fields before encoding
	if err := s.validateOutboundMessage(session, outMsg.Data); err != nil {
		// Check if this is a catalog error that should be sent to the base station
		var catalogErr *CatalogError
		if errors.As(err, &catalogErr) {
			// Send catalog error to base station per BSSCI protocol
			ctx := s.sessionContext(session)
			s.logger.ErrorContext(ctx, LogBSSCIOutboundValidationFailed,
				"error_token", catalogErr.Token,
				"bs_eui", session.BaseStationEUI)
			if sendErr := s.sendCatalogError(session, outMsg.OpId, catalogErr); sendErr != nil {
				return fmt.Errorf("failed to send catalog error: %w", sendErr)
			}
			return nil // Error sent via protocol, don't propagate
		}
		// Non-catalog errors are internal failures
		return fmt.Errorf("outbound validation failed: %w", err)
	}

	// Encode message using negotiated encoding (BSSCI Section 1)
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

	// Send header and payload
	session.mu.Lock()
	defer session.mu.Unlock()

	if _, err := session.Conn.Write(header); err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToWriteHeader), err)
	}

	if _, err := session.Conn.Write(payload); err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToWritePayload), err)
	}

	return nil
}

// handleConnect handles the connect operation per BSSCI specification
func (s *Server) handleConnect(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	// BSSCI-3.2-03: Connect operation MUST use opId=0
	if msg.OpId != 0 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIInvalidConnectOperationID,
			"op_id", msg.OpId)
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidConnectOpId)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
		}
		return fmt.Errorf("%s: %d", ResolveErrorMessage(errInvalidConnectOpId), msg.OpId)
	}

	// Extract version field (optional per BSSCI §2.4-02, defaults to canonical version)
	version := getStringField(data, "version", mioty.MIOTYProtocolVersion)
	bsEUI, ok := getNumericField(data, "bsEui")
	if !ok {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingBsEui)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
		}
		return fmt.Errorf("%s", ResolveErrorMessage(errMissingBsEui))
	}

	// BSSCI-2.1-01, BSSCI-2.2-02: Version negotiation (delegate to SessionService)
	if err := s.sessionSvc.ValidateVersion(version); err != nil {
		if catErr, ok := err.(*CatalogError); ok {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIVersionIncompatible,
				"client_version", version,
				"error", catErr.Token)
			return s.sendCatalogError(session, msg.OpId, catErr)
		}
		// Fallback for unexpected errors
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errVersionIncompatible)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
		}
		return fmt.Errorf("%s", ResolveErrorMessage(errVersionIncompatible))
	}

	// Range guard for int64 -> uint64 conversion
	if bsEUI < 0 {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidBsEui)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
		}
		return fmt.Errorf("%s: bsEUI=%d", ResolveErrorMessage(errInvalidBsEui), bsEUI)
	}

	// Simple session field assignments
	session.BaseStationEUI = uint64(bsEUI)
	session.ClientVersion = version                        // Raw BS-provided version for audit
	session.NegotiatedVersion = mioty.MIOTYProtocolVersion // SC canonical version (BSSCI §4-4.5)
	session.Vendor = getStringField(data, "vendor", "")
	session.Model = getStringField(data, "model", "")
	session.Name = getStringField(data, "name", "")
	session.SoftwareVersion = getStringField(data, "swVersion", "")

	// BSSCI §5.3: Capture connect info object for session persistence
	if infoValue, hasInfo := data["info"]; hasInfo {
		infoJSON, err := json.Marshal(infoValue)
		if err != nil {
			s.logger.WarnContext(s.safeCtx(), "Failed to marshal connect info", "error", err)
		} else {
			session.ConnectInfo = infoJSON
		}
	}

	// BSSCI §5.3: bidi flag must be explicitly declared
	bidiValue, hasBidi := data["bidi"]
	if !hasBidi {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingBidiFlag)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
		}
		return fmt.Errorf("%s", ResolveErrorMessage(errMissingBidiFlag))
	}

	bidi, ok := bidiValue.(bool)
	if !ok {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingBidiFlag)); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
		}
		return fmt.Errorf("%s: not a boolean", ResolveErrorMessage(errMissingBidiFlag))
	}
	session.Bidirectional = bidi

	// Geolocation parsing: all three values must parse and lat/lon must be in range
	if geoLoc, ok := data["geoLocation"].([]interface{}); ok && len(geoLoc) == 3 {
		lat, latOk := geoLoc[0].(float64)
		lon, lonOk := geoLoc[1].(float64)
		alt, altOk := geoLoc[2].(float64)
		if latOk && lonOk && altOk &&
			lat >= models.LatitudeMin && lat <= models.LatitudeMax &&
			lon >= models.LongitudeMin && lon <= models.LongitudeMax {
			session.GeoLocation = []float64{lat, lon, alt}
		}
	}

	// BSSCI-3.3: Session resume handling
	var snResume bool
	var previousSession *Session
	var err error

	// Extract snBsUuid and scUUIDToMatch (simple map access)
	if snBsUUIDData, hasBsUUID := data["snBsUuid"]; hasBsUUID {
		bsUUID, catErr := extractSessionUUID(snBsUUIDData)
		if catErr != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToExtractSnBsUUID, "error", catErr.Token)
			return s.sendCatalogError(session, msg.OpId, catErr)
		}
		// Valid bsUUID extracted - continue with resume logic
		session.BsUUID = bsUUID

		// Extract scUUIDToMatch from extra field if present
		var scUUIDToMatch []byte
		if extra, hasExtra := data["extra"].(map[string]interface{}); hasExtra {
			if snScUUIDData, hasScUUID := extra["snScUuid"]; hasScUUID {
				if scUUID, catErr := extractSessionUUID(snScUUIDData); catErr == nil {
					scUUIDToMatch = scUUID
				}
			}
		}

		// Extract operation counters
		var bsOpId, scOpId int64
		if bsOp, ok := getNumericField(data, "snBsOpId"); ok {
			bsOpId = bsOp
		}
		if scOp, ok := getNumericField(data, "snScOpId"); ok {
			scOpId = scOp
		}

		// Delegate resume validation to SessionService
		previousSession, err = s.sessionSvc.HandleResume(session, bsUUID, scUUIDToMatch, bsOpId, scOpId, session.BaseStationEUI)
		if err != nil {
			// Log error but continue with new session
			s.logger.WarnContext(s.safeCtx(), "Failed to check session resume",
				"error", err,
				"eui", session.BaseStationEUI)
		}

		if previousSession != nil {
			// Valid resume - copy state from previous session
			snResume = true
			session.IsResumed = true
			session.SessionUUID = previousSession.SessionUUID
			session.BsOpId = bsOpId
			session.ScOpId = scOpId
			session.LastBsOpId = bsOpId
			session.LastScOpId = scOpId
			session.DbSessionID = previousSession.DbSessionID
			session.OrganizationID = previousSession.OrganizationID     // Restore org from previous session
			session.ResolvedTenantID = previousSession.ResolvedTenantID // Restore tenant from previous session

			s.logger.InfoContext(s.safeCtx(), LogBSSCIResumingPreviousSession,
				"eui", session.BaseStationEUI,
				"bsOpId", session.BsOpId,
				"scOpId", session.ScOpId,
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

	// Store session in sessions map (protocol handling)
	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()

	// Delegate UUID-based session storage to SessionService
	s.sessionSvc.StoreSessionByUUID(session)

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
		s.logger.WarnContext(s.safeCtx(), "Software version not configured - ConnectResponse will omit swVersion field",
			"bsEui", session.BaseStationEUI,
			"sessionUuid", sessionUUID)
	case "dev", "dev-local":
		s.logger.WarnContext(s.safeCtx(), "Using development software version in ConnectResponse",
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

	return s.sendMessage(session, response)
}

// handleConnectComplete handles the connect complete operation
func (s *Server) handleConnectComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	// Validate that connect complete has opId == 0 (BSSCI-3.3)
	if msg.OpId != 0 {
		errDef := GetErrorDefinition(errInvalidConnectCompleteOpId)
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, errDef.Message); err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to send error response", "error", err)
		}
		return fmt.Errorf("%s: %d", errDef.Message, msg.OpId)
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCIBaseStationConnectionCompletedSuccessfully,
		"eui", session.BaseStationEUI,
		"name", session.Name,
		"sessionID", session.ID)

	// Convert uint64 EUI to [8]byte format for lookups
	var euiBytes [8]byte
	for i := 7; i >= 0; i-- {
		euiBytes[i] = byte(session.BaseStationEUI >> (8 * (7 - i)))
	}
	ctx := s.sessionContext(session)

	// Global EUI lookup — tenant not yet known at connect time
	baseStation, err := s.connectionSvc.GetBaseStationGlobal(ctx, euiBytes, s.connectionMgr)
	if err != nil || baseStation == nil {
		s.logger.ErrorContext(ctx, LogBSSCIBaseStationNotFoundInDatabase,
			"eui", session.BaseStationEUI,
			"euiHex", fmt.Sprintf("%X", euiBytes),
			"error", err)

		// Send error response using catalog
		if sendErr := s.sendError(session, msg.OpId, POSIX_EPERM, ResolveErrorMessage(errBaseStationNotRegistered)); sendErr != nil {
			s.logger.ErrorContext(ctx, "Failed to send error response", "error", sendErr)
		}
		return fmt.Errorf("%s: EUI %016X", ResolveErrorMessage(errBaseStationNotRegistered), session.BaseStationEUI)
	}

	// Repair session tenant to match base station's registered tenant
	if baseStation.TenantID > 0 {
		session.ResolvedTenantID = baseStation.TenantID
		if s.orgResolver != nil {
			orgID, orgErr := s.orgResolver.GetDefaultOrgForTenant(ctx, baseStation.TenantID)
			if orgErr == nil {
				session.OrganizationID = orgID
			}
		}
		// Rebuild context with correct tenant for all subsequent operations
		ctx = s.sessionContext(session)
	}

	s.logger.InfoContext(ctx, LogBSSCIBaseStationFoundInDatabase,
		"eui", session.BaseStationEUI,
		"name", session.Name,
		"tenant_id", baseStation.TenantID)

	// Delegate database session persistence to SessionService
	if err := s.sessionSvc.PersistSession(ctx, session, baseStation, session.IsResumed, session.ConnectInfo); err != nil {
		s.logger.ErrorContext(ctx, "Failed to persist session",
			"error", err,
			"eui", session.BaseStationEUI)
		// Continue - session can still function without DB persistence
	}

	// Store session in local sessions map with DbSessionID populated
	if session.DbSessionID > 0 {
		s.mu.Lock()
		s.sessions[session.ID] = session
		s.mu.Unlock()
		// SessionsByUUID storage is handled by SessionService in handleConnect
	}

	// Update the base station's bidi capability and GPS location in the database
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

	// Delegate connection registration to ConnectionService
	if err := s.connectionSvc.RegisterConnection(ctx, session, baseStation, s.connectionMgr); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateConnectionStatus,
			"eui", session.BaseStationEUI,
			"error", err)
	}

	// BSSCI-3.3-03: Delegate handshake completion to SessionService
	s.sessionSvc.MarkHandshakeComplete(session)

	// Start status mechanism for this session
	s.startStatusMechanism(session)
	s.logger.InfoContext(ctx, LogBSSCIStartedStatusMechanismForBaseStation,
		"bsEui", session.BaseStationEUI)

	// MIOTY session resume: Load and reissue pending operations
	if session.DbSessionID > 0 {
		pendingOps, err := s.loadPendingOperations(session)
		if err != nil {
			s.logger.ErrorContext(ctx, LogBSSCIFailedToLoadPendingOperationsForSessionResume,
				"error", err,
				"sessionID", session.DbSessionID)
		} else if len(pendingOps) > 0 {
			s.logger.InfoContext(ctx, LogBSSCIResumingSessionWithPendingOps,
				"bsEui", session.BaseStationEUI,
				"sessionID", session.DbSessionID,
				"pendingCount", len(pendingOps))

			// Reissue all pending operations per MIOTY BSSCI v1.0.0 session resume requirements
			for _, op := range pendingOps {
				s.logger.DebugContext(ctx, LogBSSCIProcessingPendingOperationForResume,
					"opId", op.OperationID,
					"type", op.OperationType)

				// Reconstitute operations BEFORE storing or sending
				if op.OperationType == mioty.CmdULDataTransmit && op.Metadata != nil {
					reconstituted, err := s.reconstitueULDataTxMessage(op.Message, op.Metadata, op)
					if err != nil {
						s.logger.ErrorContext(ctx, LogBSSCIFailedToReconstituteULDataTxSkipping,
							"error", err,
							"opId", op.OperationID)
						continue
					}
					op.Message = reconstituted
				} else if op.OperationType == mioty.CmdDLDataQueue && op.Metadata != nil {
					// Reconstitute dlDataQue with counter-dependent payloads
					reconstituted, err := s.reconstitueDLDataQueMessage(op.Message, op.Metadata, op)
					if err != nil {
						s.logger.ErrorContext(ctx, LogBSSCIFailedToReconstituteDLDataQueSkipping,
							"error", err,
							"opId", op.OperationID)
						continue
					}
					op.Message = reconstituted
				} else if op.OperationType == mioty.CmdDLDataRevoke && op.Metadata != nil {
					// Reconstitute mioty.CmdDLDataRevoke with proper integer types
					reconstituted, err := s.reconstitueDLDataRevMessage(op.Message, op.Metadata)
					if err != nil {
						s.logger.ErrorContext(ctx, LogBSSCIFailedToReconstituteDLDataRevSkipping,
							"error", err,
							"opId", op.OperationID)
						continue
					}
					op.Message = reconstituted
				}

				// Delegate pending operation storage to StatusService
				// BSSCI §§5.11-5.12.3 Gap 1: Guard against nil statusSvc (test compatibility)
				if s.statusSvc != nil {
					if err := s.statusSvc.RecordPendingOperation(ctx, session, op.OperationID, op, session.DbSessionID); err != nil {
						s.logger.ErrorContext(ctx, "Failed to record pending operation",
							"error", err,
							"opId", op.OperationID)
						// Continue - operation can still be reissued
					}
				}

				// Normalize message types to fix JSON float64 conversion before reissuing
				normalizedMsg := s.normalizeMessageTypes(op.Message, op.OperationType)
				if err := s.sendMessage(session, normalizedMsg); err != nil {
					s.logger.ErrorContext(ctx, LogBSSCIFailedToReissuePendingOperation,
						"error", err,
						"opId", op.OperationID,
						"type", op.OperationType)
				}
			}
		} else {
			s.logger.DebugContext(ctx, LogBSSCINoPendingOperationsToResume,
				"bsEui", session.BaseStationEUI,
				"sessionID", session.DbSessionID)
		}
	}

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

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	session.mu.Lock()
	session.LastScOpId--
	opId := session.LastScOpId
	session.mu.Unlock()

	msg := map[string]interface{}{
		"command": mioty.CmdPing,
		"opId":    opId,
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingPingToBaseStation,
		"sessionID", sessionID,
		"bsEui", session.BaseStationEUI,
		"opId", opId)

	// Send with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(session, msg); err != nil {
		// CRITICAL: Rollback on send failure to maintain operation ID consistency
		session.mu.Lock()
		session.LastScOpId++
		session.mu.Unlock()
		return err
	}

	// Success - persist counter to DB for session resume
	if err := s.sessionSvc.UpdateSessionCounters(s.safeCtx(), session); err != nil {
		s.logger.Error(LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", sessionID,
			"opId", opId)
		// Don't fail the operation - message was sent successfully
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

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	session.mu.Lock()
	session.LastScOpId--
	opId := session.LastScOpId
	session.mu.Unlock()

	msg := map[string]interface{}{
		"command": mioty.CmdPing,
		"opId":    opId,
	}

	s.logger.InfoContext(ctx, LogBSSCISendingPingToBaseStation,
		"sessionID", session.ID,
		"bsEui", session.BaseStationEUI,
		"opId", opId)

	// Send with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(session, msg); err != nil {
		// CRITICAL: Rollback on send failure to maintain operation ID consistency
		session.mu.Lock()
		session.LastScOpId++
		session.mu.Unlock()
		return 0, err
	}

	// Success - persist counter to DB for session resume
	if err := s.sessionSvc.UpdateSessionCounters(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateDatabaseSession,
			"error", err,
			"sessionID", session.ID,
			"opId", opId)
		// Don't fail the operation - message was sent successfully
	}

	return opId, nil
}

// handleStatus handles status operations

// handleAttach handles attach operations per BSSCI 3.6.1 and 3.6.2
func (s *Server) handleAttach(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// Extract and validate mandatory epEui field
	epEUI, hasEpEUI := getNumericField(data, "epEui")
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
		"epEui":       pkgmioty.FormatEUI64(uint64(epEUI)),
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
		s.logger.Error("Failed to record pending attach operation", "opId", msg.OpId, "error", err)
		return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errOperationFailed))
	}

	// Convert endpoint EUI to bytes for database lookup
	epEUIBytes := make([]byte, 8)
	// EUIs are always positive in MIOTY protocol
	if epEUI < 0 {
		return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errInvalidFieldValue))
	}
	binary.BigEndian.PutUint64(epEUIBytes[:], uint64(epEUI))

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
				s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
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
				s.logger.WarnContext(s.safeCtx(), "Roaming validation failed",
					"epEui", epEUI,
					"servingTenant", tenantID,
					"error", err)
				// Clean up pending operation before returning error
				if err := s.removePendingOperation(session, msg.OpId); err != nil {
					s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
				}
				return s.sendError(session, msg.OpId, POSIX_EPERM, ResolveErrorMessage(errRoamingNotAllowed))
			}

			// If roaming, record the attach event
			if isRoaming {
				s.logger.InfoContext(s.safeCtx(), "Roaming endpoint attaching",
					"epEui", epEUI,
					"ownerTenant", ownerTenantID,
					"servingTenant", tenantID)

				// Record roaming attach event
				if err := s.roamingSvc.RecordAttach(ctx, epEUIBytes, session.BaseStationEUIBytes(), tenantID); err != nil {
					s.logger.ErrorContext(s.safeCtx(), "Failed to record roaming attach",
						"error", err)
					// Non-fatal: continue with attach
				}

				// Update session with roaming endpoint
				if session.DbSessionID > 0 {
					if err := s.roamingSvc.UpdateSessionRoaming(ctx, session.DbSessionID, epEUIBytes, true, tenantID); err != nil {
						s.logger.ErrorContext(s.safeCtx(), "Failed to update session roaming",
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
				s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
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
					s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
				}
				return s.sendError(session, msg.OpId, POSIX_EPROTO,
					ResolveErrorMessage(errAttachCounterNotMonotonic))
			}
		}

		//nolint:gosec // G115: attachCnt validated to be <= 0xFFFFFF on line 1431, safe for uint32
		if err := ValidateAttachSignature(uint64(epEUI), uint32(attachCnt), sign, nwkSnKey); err != nil {
			s.logger.WarnContext(s.safeCtx(), "Attach signature validation failed",
				"epEui", epEUI,
				"error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
		}

		sessionKey, err := DeriveSessionKey(uint64(epEUI), nonce, sign, nwkSnKey)
		if err != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to derive session key",
				"epEui", epEUI,
				"error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
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
				s.logger.ErrorContext(s.safeCtx(), "Failed to encrypt session key", "error", err)
				if err := s.removePendingOperation(session, msg.OpId); err != nil {
					s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
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
				s.logger.WarnContext(ctx, "Failed to normalize attach subpackets", "error", err)
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
					s.logger.ErrorContext(s.safeCtx(), "Failed to update radio metrics", "error", err)
				}
			}
			return nil
		}

		tx, txErr := s.storage.BeginTx(ctx)
		if txErr != nil {
			s.logger.ErrorContext(s.safeCtx(), "Failed to begin transaction", "error", txErr)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
		}

		var commitErr error
		defer func() {
			if commitErr != nil {
				_ = tx.Rollback()
			}
		}()

		if err := tx.EndPoints().UpdateFields(ctx, tenantID, endpoint.ID, attachUpdates); err != nil {
			commitErr = err
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointAttachMetadata, "error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
		}

		endpointIDStr := fmt.Sprintf("%d", endpoint.ID)
		activeSession, getErr := tx.EndPointSessions().GetActive(ctx, endpointIDStr)
		if getErr != nil {
			commitErr = getErr
			s.logger.ErrorContext(s.safeCtx(), "Failed to load endpoint session", "error", getErr)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
		}

		now := time.Now().UTC()

		// Lookup base station ID for session enrichment (tenant-scoped)
		var primaryBsID *int64
		if s.basestationRepo != nil {
			bs, bsErr := s.basestationRepo.GetByEUI(ctx, resolvedTenant(session, s.tenantID), session.BaseStationEUIBytes())
			if bsErr == nil && bs != nil {
				primaryBsID = &bs.ID
			}
		}

		// responseShAddr is uint16, model ShAddr is *int32 (INTEGER 0-65535)
		shAddrInt32 := int32(responseShAddr)

		if activeSession != nil {
			activeSession.SessionKey = encryptedSessionKey
			//nolint:gosec // G115: attachCnt validated to be <= 0xFFFFFF on line 1431, safe for int32
			activeSession.AttachCnt = int32(attachCnt)
			activeSession.LastActivityAt = now
			activeSession.ShAddr = &shAddrInt32
			activeSession.PrimaryBaseStationID = primaryBsID
			if err := tx.EndPointSessions().Update(ctx, activeSession); err != nil {
				commitErr = err
				s.logger.ErrorContext(s.safeCtx(), "Failed to update endpoint session", "error", err)
				if err := s.removePendingOperation(session, msg.OpId); err != nil {
					s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
				}
				return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
			}
		} else {
			newSession := &models.EndPointSession{
				TenantID:   tenantID,
				EndPointID: endpoint.ID,
				SessionID:  uuid.New().String(),
				SessionKey: encryptedSessionKey,
				//nolint:gosec // G115: attachCnt validated to be <= 0xFFFFFF on line 1431, safe for int32
				AttachCnt:            int32(attachCnt),
				Status:               "active",
				StartedAt:            now,
				LastActivityAt:       now,
				ShAddr:               &shAddrInt32,
				PrimaryBaseStationID: primaryBsID,
			}
			if err := tx.EndPointSessions().Create(ctx, newSession); err != nil {
				commitErr = err
				s.logger.ErrorContext(s.safeCtx(), "Failed to create endpoint session", "error", err)
				if err := s.removePendingOperation(session, msg.OpId); err != nil {
					s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
				}
				return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
			}
		}

		if err := tx.Commit(); err != nil {
			commitErr = err
			s.logger.ErrorContext(s.safeCtx(), "Failed to commit attach transaction", "error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to remove pending operation", "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
		}
		commitErr = nil

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
				s.logger.ErrorContext(s.safeCtx(), "Failed to update radio metrics", "error", err)
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
	// Extract and validate mandatory epEui field
	epEUI, hasEpEUI := getNumericField(data, "epEui")
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
	epEuiUint, errToken := safeUint64(epEUI, "epEui")
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}
	packetCntUint, errToken := safeUint32(packetCnt, "packetCnt")
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	typedMetadata := &detachMetadata{
		EpEui:     epEuiUint,
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
	binary.BigEndian.PutUint64(euiBytes, epEuiUint)

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
				s.logger.WarnContext(ctx, "Failed to resolve organization for endpoint owner",
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
		s.logger.WarnContext(ctx, "Detach from unknown endpoint",
			"error", err,
			"epEui", epEuiUint)

		// Validate unknown endpoint signature via internal detach validator service
		if s.detachValidator != nil {
			result, validationErr := s.detachValidator.ValidateDetachSignature(ctx, epEuiUint, sign)
			if validationErr != nil {
				// Check for typed sentinel: endpoint not found
				if errors.Is(validationErr, ErrDetachValidationEndpointNotFound) {
					s.logger.WarnContext(ctx, "Unknown endpoint not found during detach validation",
						"epEui", epEuiUint)
					return s.sendError(session, msg.OpId, POSIX_ENOENT, ResolveErrorMessage(errEndpointNotFound))
				}

				// Check for signature validation failure with metadata
				if errors.Is(validationErr, ErrDetachSignatureInvalid) && result != nil {
					s.logger.WarnContext(ctx, "Unknown endpoint detach signature invalid",
						"epEui", epEuiUint,
						"tenant_id", result.TenantID,
						"owner_tenant_id", result.OwnerTenantID,
						"validation_status", result.ValidationStatus)
					return s.sendError(session, msg.OpId, POSIX_EACCES, ResolveErrorMessage(errDetachSignatureInvalid))
				}

				// All other errors treated as signature validation failure
				s.logger.WarnContext(ctx, "Unknown endpoint detach signature validation failed",
					"epEui", epEuiUint,
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
					s.logger.WarnContext(ctx, "Failed to resolve organization for unknown endpoint owner",
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

			s.logger.InfoContext(ctx, "Unknown endpoint detach signature validated successfully",
				"epEui", epEuiUint,
				"tenantId", ownerTenantID,
				"ownerTenantId", result.OwnerTenantID,
				"validationStatus", validationStatus)
		} else {
			// Validator not configured - fall back to session tenant
			ownerTenantID = resolvedTenant(session, s.tenantID)
			ownerCtx = pkgcontext.WithTenantID(context.Background(), ownerTenantID)
			s.logger.WarnContext(ctx, "Detach validator not configured - using session tenant for unknown endpoint",
				"epEui", epEuiUint,
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
			s.logger.WarnContext(s.safeCtx(), "Roaming validation failed during detach",
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
			s.logger.InfoContext(s.safeCtx(), "Roaming endpoint detaching",
				"epEui", epEUI,
				"ownerTenant", ownerTenantID,
				"servingTenant", servingTenantID)

			// Record roaming detach event
			if err := s.roamingSvc.RecordDetach(ctx, euiBytes, session.BaseStationEUIBytes(), servingTenantID); err != nil {
				s.logger.ErrorContext(s.safeCtx(), "Failed to record roaming detach",
					"error", err)
				// Non-fatal: continue with detach
			}

			// Update session to remove roaming endpoint
			if session.DbSessionID > 0 {
				if err := s.roamingSvc.UpdateSessionRoaming(ctx, session.DbSessionID, euiBytes, false, servingTenantID); err != nil {
					s.logger.ErrorContext(s.safeCtx(), "Failed to update session roaming for detach",
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

	// Persist pending operation via StatusService (handles both DB + map with SessionOpKey)
	// BSSCI §5.7: StatusService is single writer for crash safety (guard against zero DbSessionID)
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
			s.logger.WarnContext(ownerCtx, "Failed to normalize detach subpackets", "error", err)
		} else {
			detachMsg.Subpackets = normalizedSp
		}
	}

	// Normalize payload for audit completeness (DET-03: convert numeric arrays to concrete types)
	normalizedPayload := normalizeDetachPayload(data)

	if s.storage != nil && s.storage.MIOTYMessages() != nil {
		if err := s.storage.MIOTYMessages().CreateDetachMessage(ownerCtx, detachMsg, normalizedPayload); err != nil {
			s.logger.WarnContext(ownerCtx, LogBSSCIFailedToPersistDetachMessage,
				"error", err,
				"epEui", epEuiUint,
				"bsEui", session.BaseStationEUI,
				"opId", msg.OpId)
		}
	}

	// Process endpoint telemetry and signature validation if endpoint was found
	if endpoint != nil {
		// Cryptographic signature validation
		if s.config.DetachSignatureValidationEnabled {
			if err := ValidateDetachSignature(sign, endpoint.Sign); err != nil {
				s.logger.WarnContext(ownerCtx, "Detach signature validation failed",
					"epEui", pkgmioty.FormatEUI64(epEuiUint),
					"error", err)
				return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
			}
		}

		// Update endpoints table with detach telemetry (BSSCI §5.7.1 Finding 2)
		// Use UpdateFields to persist last_detach_* columns (Update() doesn't touch them)
		bsEuiInt := int64(session.BaseStationEUI) //nolint:gosec // G115: EUIs are 48-bit per MIOTY spec, safe for int64
		packetCntInt := int64(packetCntUint)
		propagateStatus := PropagateStatusDetachReceived
		updates := map[string]interface{}{
			"last_attached_bs_eui":   bsEuiInt,
			"last_propagate_time":    rxTime,
			"last_detach_time":       rxTime,
			"last_detach_sign":       sign,
			"last_detach_packet_cnt": packetCntInt,
			"propagate_status":       propagateStatus,
		}
		if err := s.endpointRepo.UpdateFields(ownerCtx, ownerTenantID, endpoint.ID, updates); err != nil {
			s.logger.WarnContext(ownerCtx, "Failed to update endpoint detach telemetry",
				"error", err,
				"epEui", epEuiUint)
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
				if orgUUID, orgErr := s.orgResolver.GetDefaultOrgForTenant(s.safeCtx(), epTID); orgErr == nil {
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
					binary.BigEndian.PutUint64(euiBytes, uint64(epEUI))
					sessionTenant := resolvedTenant(session, s.tenantID)
					endpoint, _ = s.endpointRepo.GetByEUI(s.safeCtx(), sessionTenant, euiBytes)
				}

				// Tier 3: Cross-tenant EUI lookup (roaming fallback)
				if endpoint == nil && epEUI != 0 {
					euiBytes := make([]byte, 8)
					binary.BigEndian.PutUint64(euiBytes, uint64(epEUI))
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
				s.logger.Warn(LogBSSCIEPStatusForwardFailed, "epEui", pkgmioty.FormatEUI64(eui), "error", err)
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
		if epEuiRaw, ok := getNumericField(data, "epEui"); ok {
			epEUI = uint64(epEuiRaw) //nolint:gosec // G115: EUIs from message validated as positive
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
				s.logger.ErrorContext(s.safeCtx(), "Failed to update radio metrics", "error", err)
			}
		}
	}

	// Record event for successful detach
	if s.eventStore != nil && epEUI != 0 {
		details := map[string]interface{}{
			"epEui":     pkgmioty.FormatEUI64(uint64(epEUI)), // Use hex string to avoid JSON precision loss
			"bsEui":     pkgmioty.FormatEUI64(session.BaseStationEUI),
			"operation": "detach_complete",
		}

		detailsJSON, _ := json.Marshal(details)

		if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
			EventType:   models.EventTypeEndpointDetached,
			Category:    mioty.CategoryEndpoint,
			Severity:    SeverityInfo,
			Title:       fmt.Sprintf(models.EventTitleEndpointDetachedViaBS, pkgmioty.FormatEUI64(uint64(epEUI)), pkgmioty.FormatEUI64(session.BaseStationEUI)),
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
				s.logger.Warn(LogBSSCIEPStatusForwardFailed, "epEui", pkgmioty.FormatEUI64(eui), "error", err)
			}
		}(ownerCtx, epEUI, tenantID)
	}

	// No response needed for complete messages per BSSCI spec
	return nil
}

// handleULData handles uplink data operations
func (s *Server) handleULData(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	epEUI, ok := getNumericField(data, "epEui")
	if !ok {
		return fmt.Errorf("%s", ResolveErrorMessage(errMissingEpEui))
	}

	// Route by endpoint ownership before committing to ingest
	if s.dispositionResolver != nil {
		epEuiForDisp, errToken := safeUint64(epEUI, "epEui")
		if errToken != "" {
			return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errToken))
		}
		disposition, dispErr := s.dispositionResolver.Resolve(ctx, epEuiForDisp)
		if dispErr != nil {
			s.logger.ErrorContext(ctx, "Disposition resolution failed; rejecting uplink",
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
				epU, _ := safeUint64(epEUI, "epEui")
				_, enqErr := s.relayOutbox.Enqueue(ctx, epU, session.BaseStationEUI, rawFrame, time.Now().UnixNano())
				if enqErr != nil {
					s.logger.ErrorContext(ctx, "Failed to enqueue relay uplink",
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
		epEuiVal, errToken := safeUint64(epEUI, "epEui")
		if errToken != "" {
			return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errToken))
		}
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
		result, ingestErr := s.uplinkIngestSvc.Ingest(ctx, payload, UplinkIngestOptions{Source: UplinkSourceBSSCI})
		if ingestErr != nil {
			s.logger.ErrorContext(ctx, "Uplink ingest failed",
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

	// Check if this error is related to a pending operation
	// Base stations sometimes send error messages instead of proper response messages
	// We must complete the three-way handshake to prevent timeout loops
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
	var pendingOp *PendingOperation
	var err error
	if s.statusSvc != nil {
		pendingOp, err = s.statusSvc.GetPendingOperation(session, int64(msg.OpId))
	}
	hasPendingOp := err == nil

	// For Service Center initiated operations (negative opId), always try to complete
	// the handshake even if we don't have the operation in our tracking
	isServiceCenterOp := msg.OpId < 0

	if (hasPendingOp && pendingOp != nil) || isServiceCenterOp {
		// Determine operation type
		var operationType string
		if pendingOp != nil {
			operationType = pendingOp.OperationType
		} else {
			// For Service Center operations without tracking, guess based on recent state
			// Service Center uses negative opIds for attach/detach propagate operations
			// This is our best guess - most SC operations are propagate operations
			if msg.OpId < 0 {
				// Default to detach propagate as it's most common for error scenarios
				operationType = mioty.CmdDetachPropagate
			} else {
				operationType = "unknown"
			}
		}

		s.logger.InfoContext(s.safeCtx(), LogBSSCICompletingThreeWayHandshakeForError,
			"opId", msg.OpId,
			"operationType", operationType)

		// Send appropriate completion message based on operation type
		var completionCommand string
		switch operationType {
		case mioty.CmdAttachPropagate:
			completionCommand = mioty.CmdAttachPropagateComplete
		case mioty.CmdDetachPropagate:
			completionCommand = mioty.CmdDetachPropagateComplete
		default:
			// For other operations, acknowledge with errorAck per BSSCI-4-02
			errorAckMsg := map[string]interface{}{
				"command": mioty.CmdErrorAck,
				"opId":    msg.OpId,
			}
			if err := s.sendMessage(session, errorAckMsg); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorAck, "error", err)
			}
			return nil
		}

		// Send completion message
		completionMsg := map[string]interface{}{
			"command": completionCommand,
			"opId":    msg.OpId,
		}

		if err := s.sendMessage(session, completionMsg); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendCompletionMessageForErrorOperation,
				"error", err,
				"command", completionCommand)
			return err
		}

		// For error cases, we do NOT call the completion handler
		// The completion handler should only be called for successful operations
		// When the base station sends an error (e.g., "unknown endpoint"),
		// we complete the handshake but don't update our database state
		s.logger.InfoContext(s.safeCtx(), LogBSSCIErrorOperationHandshakeCompletedDatabaseNotUpdated,
			"opId", msg.OpId,
			"operationType", operationType)

		// Clean up the pending operation from database and memory
		// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both cache and DB removal
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperationFromDatabase,
				"error", err,
				"opId", msg.OpId)
		}
		// Note: No manual fallback - trust StatusService single-writer pattern

		return nil
	}

	// For non-pending operations, just acknowledge per BSSCI-4-02
	errorAckMsg := map[string]interface{}{
		"command": mioty.CmdErrorAck,
		"opId":    msg.OpId,
	}
	if err := s.sendMessage(session, errorAckMsg); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorAck, "error", err)
	}

	return nil
}

// handleErrorAck handles the errorAck message (BSSCI §3.17.2)
// This acknowledges error reception from the base station and allows operation recovery
func (s *Server) handleErrorAck(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	s.logger.InfoContext(s.safeCtx(), LogBSSCIReceivedErrorAckFromBaseStation,
		"opId", msg.OpId,
		"baseStationEUI", session.BaseStationEUI)

	// For SC-initiated operations (negative opId), clean up pending operation
	// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both cache and DB removal
	if msg.OpId < 0 {
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperationAfterErrorAck,
				"error", err,
				"opId", msg.OpId)
			// Note: No manual fallback - trust StatusService single-writer pattern
		}
	}

	// For BS-initiated operations (positive opId), just log - no cleanup needed
	// The base station is acknowledging our error message

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
		// Trim BOM and leading whitespace per RFC 8259
		trimmed := trimJSONWhitespace(rawFrame)
		if err := json.Unmarshal(trimmed, &data); err != nil {
			return nil, err
		}
		return data, nil
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

func getStringField(data map[string]interface{}, key string, defaultValue string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return defaultValue
}

func getNumericField(data map[string]interface{}, key string) (int64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(float64(v)), true
	case uint:
		return int64(v), true //nolint:gosec // G115: BSSCI protocol values fit int64 range
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true //nolint:gosec // G115: BSSCI protocol values fit int64 range
	default:
		return 0, false
	}
}

// parseOpID validates and extracts operation ID from interface{} value.
// Per BSSCI §5.2, operation IDs must be precise 64-bit integers.
// Rejects non-integral floats and uint64 values that overflow int64 range.
// Returns (value, true) on success or (0, false) on validation failure.
func parseOpID(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case float64:
		// Only accept if no fractional part and within int64 range
		if v != math.Trunc(v) || v < math.MinInt64 || v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	case float32:
		f64 := float64(v)
		// Only accept if no fractional part and within int64 range
		if f64 != math.Trunc(f64) || f64 < math.MinInt64 || f64 > math.MaxInt64 {
			return 0, false
		}
		return int64(f64), true
	case uint:
		if v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	default:
		return 0, false
	}
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

// getFloatFieldValidated extracts a float field and validates it's actually numeric
// Returns the value and true if field exists and is numeric, or 0 and false otherwise
func getFloatFieldValidated(data map[string]interface{}, key string) (float64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		// Invalid type for numeric field
		return 0, false
	}
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
			// Extract numeric value
			var numVal float64
			switch e := elem.(type) {
			case float64:
				numVal = e
			case int64:
				numVal = float64(e)
			case int:
				numVal = float64(e)
			default:
				// Non-numeric element
				if fieldName == fieldNameNonce {
					return nil, errInvalidNonceElement
				}
				return nil, errInvalidSignElement
			}
			// Validate range: must be integer 0-255
			if numVal < 0 || numVal > 255 || numVal != float64(int(numVal)) {
				if fieldName == fieldNameNonce {
					return nil, errInvalidNonceElement
				}
				return nil, errInvalidSignElement
			}
			result[i] = byte(numVal)
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
	var endpointID int64
	if idFloat, ok := m["endpointID"].(float64); ok {
		endpointID = int64(idFloat)
	}

	packetCntFloat, ok := m["packetCnt"].(float64)
	if !ok {
		return nil
	}
	rxTimeFloat, ok := m["rxTime"].(float64)
	if !ok {
		return nil
	}
	snr, ok := m["snr"].(float64)
	if !ok {
		return nil
	}
	rssi, ok := m["rssi"].(float64)
	if !ok {
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
	packetCnt, errToken := safeUint32(int64(packetCntFloat), "packetCnt")
	if errToken != "" {
		return nil // Failed conversion
	}

	result := &detachMetadata{
		EpEui:      epEui,
		EndpointID: endpointID,
		PacketCnt:  packetCnt,
		Signature:  signature,
		RxTime:     int64(rxTimeFloat),
		SNR:        snr,
		RSSI:       rssi,
	}

	// Extract tenantId (optional for backward compatibility with old pending operations)
	if tenantIdFloat, ok := m["tenantId"].(float64); ok {
		result.TenantID = int64(tenantIdFloat)
	}

	// Extract orgUuid (optional for backward compatibility)
	if orgUuidStr, ok := m["orgUuid"].(string); ok {
		if parsedUUID, err := uuid.Parse(orgUuidStr); err == nil {
			result.OrgUUID = parsedUUID
		}
	}

	// Optional fields
	if eqSnrFloat, ok := m["eqSnr"].(float64); ok {
		result.EqSnr = &eqSnrFloat
	}
	if profileStr, ok := m["profile"].(string); ok {
		result.Profile = &profileStr
	}
	if rxDurFloat, ok := m["rxDuration"].(float64); ok {
		rxDur := int64(rxDurFloat)
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
	switch v := value.(type) {
	case string:
		parsed, err := validation.ParseEUI(v)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case uint64:
		return v, true
	case int64:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case float64:
		if v < 0 {
			return 0, false
		}
		const maxSafeFloat64Int = 1 << 53
		if v > float64(maxSafeFloat64Int) {
			return 0, false
		}
		return uint64(v), true
	default:
		return 0, false
	}
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

// extractFloatSlice validates and extracts float64 slice from MessagePack array.
// Rejects non-numeric values. Returns (slice, true) on success or (nil, false) on failure.
func extractFloatSlice(values []interface{}) ([]float64, bool) {
	result := make([]float64, len(values))
	for i, v := range values {
		switch val := v.(type) {
		case float64:
			result[i] = val
		case float32:
			result[i] = float64(val)
		case int64:
			result[i] = float64(val)
		case int:
			result[i] = float64(val)
		case int8:
			result[i] = float64(val)
		case int16:
			result[i] = float64(val)
		case int32:
			result[i] = float64(val)
		case uint:
			result[i] = float64(val)
		case uint8:
			result[i] = float64(val)
		case uint16:
			result[i] = float64(val)
		case uint32:
			result[i] = float64(val)
		case uint64:
			result[i] = float64(val)
		default:
			// Non-numeric value - reject entire array
			return nil, false
		}
	}
	return result, true
}

// extractInt64Slice validates and extracts int64 slice from MessagePack array.
// Rejects non-integers and non-numeric values. Returns (slice, true) on success or (nil, false) on failure.
func extractInt64Slice(values []interface{}) ([]int64, bool) {
	result := make([]int64, len(values))
	for i, v := range values {
		switch val := v.(type) {
		case float64:
			// Only accept if no fractional part
			if val != float64(int64(val)) || val < math.MinInt64 || val > math.MaxInt64 {
				return nil, false
			}
			result[i] = int64(val)
		case int64:
			result[i] = val
		case int:
			result[i] = int64(val)
		case int8:
			result[i] = int64(val)
		case int16:
			result[i] = int64(val)
		case int32:
			result[i] = int64(val)
		case uint:
			if val > math.MaxInt64 {
				return nil, false
			}
			result[i] = int64(val)
		case uint8:
			result[i] = int64(val)
		case uint16:
			result[i] = int64(val)
		case uint32:
			result[i] = int64(val)
		case uint64:
			if val > math.MaxInt64 {
				return nil, false
			}
			result[i] = int64(val)
		default:
			// Non-numeric or non-integer value - reject entire array
			return nil, false
		}
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
			switch num := elem.(type) {
			case uint8:
				userData[i] = num
			case int8:
				userData[i] = byte(num)
			case uint16:
				userData[i] = byte(num)
			case int16:
				userData[i] = byte(num)
			case uint32:
				userData[i] = byte(num)
			case int32:
				userData[i] = byte(num)
			case uint64:
				userData[i] = byte(num)
			case int64:
				userData[i] = byte(num)
			case int:
				userData[i] = byte(num)
			case uint:
				userData[i] = byte(num)
			case float32:
				userData[i] = byte(num)
			case float64:
				userData[i] = byte(num)
			default:
				// Log warning for unsupported element type
				s.logger.WarnContext(s.safeCtx(), LogBSSCIUnsupportedUserDataElementType,
					"index", i,
					"type", fmt.Sprintf("%T", elem))
				userData[i] = 0
			}
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
	ownerCtx context.Context,
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
	session.mu.Lock()
	session.LastScOpId--
	opId := session.LastScOpId
	session.mu.Unlock()

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

	// Convert endpointEUI to bytes for database storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, endpointEUI)

	// Resolve endpoint owner tenant BEFORE creating metadata (BSSCI §5.8.3 multi-tenant roaming)
	// This ensures metadata contains correct tenant/org for resume scenarios
	var endpointTenantID int64
	var ownerOrgUUID uuid.UUID
	var ownerCtx context.Context

	if s.endpointRepo != nil {
		// Try session tenant first (happy path)
		tenantID := resolvedTenant(session, s.tenantID)
		endpoint, err := s.endpointRepo.GetByEUI(ctx, tenantID, euiBytes)

		// Fallback: endpoint roamed from different tenant
		if err == storage.ErrNotFound {
			var eui models.EUI
			binary.BigEndian.PutUint64(eui[:], endpointEUI)
			endpoint, err = s.endpointRepo.Get(ctx, eui)
		}

		if err == nil && endpoint != nil {
			// Use endpoint's tenant (not session tenant) for all operations
			endpointTenantID = endpoint.TenantID

			// Resolve organization for owner tenant
			if s.orgResolver != nil {
				ownerOrgUUID, err = s.orgResolver.GetDefaultOrgForTenant(ctx, endpointTenantID)
				if err != nil {
					s.logger.Warn(LogBSSCIOrgLookupFailed, "tenantID", endpointTenantID, "error", err)
					// Continue without org - ownerOrgUUID remains Nil
				}
			}

			// Build owner-scoped context for all downstream operations
			// Use Background() to avoid session context pollution
			ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
			if ownerOrgUUID != uuid.Nil {
				ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
			}
		} else {
			// Endpoint not found - use session tenant as fallback
			endpointTenantID = tenantID
			// Use Background() to avoid session context pollution
			ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
		}
	} else {
		// No repository - use session tenant
		endpointTenantID = resolvedTenant(session, s.tenantID)
		// Use Background() to avoid session context pollution
		ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
	}

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

	if err := s.persistPendingOperation(session, opId, mioty.CmdAttachPropagate, message, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistPendingOperation, "error", err)
		// Continue anyway - persistence failure shouldn't block operation
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

			tx, txErr := s.storage.BeginTx(ownerCtx) // Use owner context
			if txErr != nil {
				return fmt.Errorf("failed to begin attach propagate transaction: %w", txErr)
			}
			var commitErr error
			defer func() {
				if commitErr != nil {
					_ = tx.Rollback()
				}
			}()

			if err := tx.EndPoints().UpdateFields(ownerCtx, endpointTenantID, endpoint.ID, updates); err != nil {
				commitErr = err
				return fmt.Errorf("failed to update endpoint: %w", err)
			}

			endpointIDStr := fmt.Sprintf("%d", endpoint.ID)
			activeSession, getErr := tx.EndPointSessions().GetActive(ownerCtx, endpointIDStr)
			if getErr != nil {
				commitErr = getErr
				return fmt.Errorf("failed to load endpoint session: %w", getErr)
			}

			now := time.Now().UTC()

			// Lookup base station ID for session enrichment (tenant-scoped to owner)
			var primaryBsID *int64
			if s.basestationRepo != nil {
				bs, bsErr := s.basestationRepo.GetByEUI(ownerCtx, endpointTenantID, session.BaseStationEUIBytes())
				if bsErr == nil && bs != nil {
					primaryBsID = &bs.ID
				}
			}

			// shortAddr is uint16, model ShAddr is *int32 (INTEGER 0-65535)
			shAddrInt32 := int32(shortAddr)

			if activeSession != nil {
				activeSession.SessionKey = encryptedKey
				activeSession.LastActivityAt = now
				activeSession.ShAddr = &shAddrInt32
				activeSession.PrimaryBaseStationID = primaryBsID
				if err := tx.EndPointSessions().Update(ownerCtx, activeSession); err != nil {
					commitErr = err
					return fmt.Errorf("failed to update endpoint session: %w", err)
				}
			} else {
				newSession := &models.EndPointSession{
					TenantID:             endpointTenantID, // Use owner tenant
					EndPointID:           endpoint.ID,
					SessionID:            uuid.New().String(),
					SessionKey:           encryptedKey,
					Status:               string(models.SessionStatusActive),
					UplinkMode:           "standard", // Default uplink mode per BSSCI §5.2
					StartedAt:            now,
					LastActivityAt:       now,
					ShAddr:               &shAddrInt32,
					PrimaryBaseStationID: primaryBsID,
				}
				if err := tx.EndPointSessions().Create(ownerCtx, newSession); err != nil {
					commitErr = err
					return fmt.Errorf("failed to create endpoint session: %w", err)
				}
			}

			if err := tx.Commit(); err != nil {
				commitErr = err
				return fmt.Errorf("failed to commit attach propagate transaction: %w", err)
			}
			commitErr = nil

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

	// Send with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(session, message); err != nil {
		// CRITICAL: Rollback on send failure to maintain operation ID consistency
		session.mu.Lock()
		session.LastScOpId++
		session.mu.Unlock()

		// Clean up persisted operation (guard for community builds)
		if session.DbSessionID > 0 {
			if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
					"sessionID", session.DbSessionID,
					"opId", opId,
					"error", cleanupErr)
			}
		}

		return err
	}

	// BSSCI §5.8.3: Persist attach propagate message to mioty_messages for audit trail
	if s.storage != nil && s.storage.MIOTYMessages() != nil {
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

		if s.storage != nil && s.storage.MIOTYMessages() != nil {
			if err := s.storage.MIOTYMessages().CreateAttachPropagateMessage(ownerCtx, propagateMsg); err != nil {
				s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistAttachPropagateMessage,
					"error", err,
					"epEui", endpointEUI,
					"bsEui", session.BaseStationEUI,
					"opId", opId)
				// Continue - persistence failure shouldn't block operation
			}
		}
	}

	// Success - persist counter to DB for session resume
	if s.sessionSvc != nil {
		if err := s.sessionSvc.UpdateSessionCounters(s.safeCtx(), session); err != nil {
			s.logger.Error(LogBSSCIFailedToUpdateDatabaseSession,
				"error", err,
				"sessionID", session.ID,
				"opId", opId)
			// Don't fail the operation - message was sent successfully
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

	s.logger.InfoContext(ctx, "Closing BSSCI session due to EUI change",
		"eui", eui,
		"sessionID", targetSession.ID)

	// Terminate DB session record
	if targetSession.DbSessionID != 0 && s.sessionSvc != nil {
		if err := s.sessionSvc.TerminateSession(ctx, targetSession); err != nil {
			s.logger.WarnContext(ctx, "Failed to terminate DB session during EUI change",
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
			s.logger.WarnContext(ctx, "Failed to close connection during EUI change cleanup",
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
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIAttachPropagateRejectedByBaseStation,
			"baseStation", session.BaseStationEUI,
			"opId", msg.OpId,
			"result", result)

		// Mark the pending operation as failed and persist to database
		if session.DbSessionID > 0 {
			// Get existing metadata or create new
			// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
			var pendingOp *PendingOperation
			if s.statusSvc != nil {
				pendingOp, _ = s.statusSvc.GetPendingOperation(session, int64(msg.OpId))
			}

			metadata := make(map[string]interface{})
			if pendingOp != nil && pendingOp.Metadata != nil {
				// Copy existing metadata
				for k, v := range pendingOp.Metadata {
					metadata[k] = v
				}
			}

			// Add failure information
			metadata["failed"] = true
			metadata["failureReason"] = fmt.Sprintf("Base station rejected with code %d", result)
			metadata["failedAt"] = time.Now().UnixNano()
			metadata["failureCode"] = result

			// Persist to database
			if err := s.updatePendingOperationMetadata(session, msg.OpId, metadata); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistFailureMetadata,
					"error", err,
					"opId", msg.OpId)
			}

			// Create system event for failure visibility in UI
			if s.eventStore != nil {
				// Extract owner tenant/org from metadata for roaming support
				ownerTenantID := resolvedTenant(session, s.tenantID)
				switch v := metadata["tenantId"].(type) {
				case int64:
					ownerTenantID = v
				case float64:
					ownerTenantID = int64(v)
				default:
					s.logger.Warn(LogBSSCIMissingTenantInMetadata, "opId", msg.OpId)
				}

				var ownerOrg uuid.UUID
				if orgIDStr, ok := metadata["organizationId"].(string); ok {
					ownerOrg, _ = uuid.Parse(orgIDStr) // Ignore parse error, Nil UUID is valid fallback
				}

				// Build owner context for event creation
				ownerCtx := pkgcontext.WithTenantID(ctx, ownerTenantID)
				if ownerOrg != uuid.Nil {
					ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrg)
				}

				// Extract endpoint EUI if available
				var epEUI uint64
				if v, ok := metadata["epEui"]; ok {
					epEUI, _ = parseMetadataEUI(v)
				}
				// Also check endpointEUI field for detach
				if epEUI == 0 {
					if v, ok := metadata["endpointEUI"]; ok {
						epEUI, _ = parseMetadataEUI(v)
					}
				}

				// Determine operation type
				operationType := "propagate_failed"
				eventType := "endpoint_operation_failed"
				if pendingOp != nil {
					switch pendingOp.OperationType {
					case mioty.CmdAttachPropagate:
						operationType = "attach_propagate_failed"
						eventType = "endpoint_attach_failed"
					case mioty.CmdDetachPropagate:
						operationType = "detach_propagate_failed"
						eventType = "endpoint_detach_failed"
					}
				}

				// Extract operation ID safely (avoid nil dereference)
				opID := msg.OpId
				if pendingOp != nil {
					opID = pendingOp.OperationID
				}

				details := map[string]interface{}{
					"epEui":        pkgmioty.FormatEUI64(epEUI),
					"bsEui":        pkgmioty.FormatEUI64(session.BaseStationEUI),
					"operation":    operationType,
					"operation_id": fmt.Sprintf("%d", opID), // For operation filtering
					"failureCode":  result,
					"reason":       fmt.Sprintf("Base station rejected with code %d", result),
				}
				detailsJSON, _ := json.Marshal(details)

				title := fmt.Sprintf("Operation failed for endpoint %s on BS %s", pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI))
				if pendingOp != nil && pendingOp.OperationType == mioty.CmdAttachPropagate {
					title = fmt.Sprintf("Attach propagate failed for endpoint %s on BS %s", pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI))
				} else if pendingOp != nil && pendingOp.OperationType == mioty.CmdDetachPropagate {
					title = fmt.Sprintf("Detach propagate failed for endpoint %s on BS %s", pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI))
				}

				// Create endpoint event with source fields for UI visibility (use owner tenant for roaming)
				if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
					TenantID:    fmt.Sprintf("%d", ownerTenantID),
					EventType:   eventType,
					Category:    mioty.CategoryEndpoint,
					Severity:    SeverityError,
					Title:       title,
					Description: fmt.Sprintf("Base station rejected operation with result code %d", result),
					Details:     detailsJSON,
					Status:      EventStatusNew,
					SourceType:  mioty.SourceTypeEndpoint,
					SourceName:  pkgmioty.FormatEUI64(epEUI), // Critical for UI filtering
					CreatedAt:   time.Now(),
				}); err != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateFailureEvent, "error", err)
				}

				// Also create base station event for symmetric tracking (use owner tenant for roaming)
				if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
					TenantID:    fmt.Sprintf("%d", ownerTenantID),
					EventType:   eventType,
					Category:    mioty.CategoryEndpoint,
					Severity:    SeverityError,
					Title:       fmt.Sprintf("Rejected %s for endpoint %s", operationType, pkgmioty.FormatEUI64(epEUI)),
					Description: fmt.Sprintf("Base station rejected operation with result code %d", result),
					Details:     detailsJSON,
					Status:      EventStatusNew,
					SourceType:  mioty.SourceTypeBaseStation,
					SourceName:  pkgmioty.FormatEUI64(session.BaseStationEUI), // Critical for UI filtering
					CreatedAt:   time.Now(),
				}); err != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateBaseStationFailureEvent, "error", err)
				}
			}
		}

		// Send error response to base station per BSSCI-4-01
		// MUST use POSIX error code, not the base station's result code
		errorMsg := map[string]interface{}{
			"command": "error",
			"opId":    msg.OpId,
			"code":    POSIX_EPROTO, // Protocol error per BSSCI spec
			"message": fmt.Sprintf("Attach propagate rejected by base station with result code %d", result),
		}

		if err := s.sendMessage(session, errorMsg); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorMessageToBaseStation,
				"error", err)
		}

		// Return nil to keep connection open - individual operation failure shouldn't close session
		return nil
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
			s.logger.Warn(LogBSSCIMissingTenantInMetadata, "opId", msg.OpId)
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
				s.logger.Warn(LogBSSCIEndpointNotFoundForPropagate, "epEui", epEUI)
				return nil
			}
			// Re-assign to actual endpoint tenant (metadata may be stale)
			ownerTenantID = endpoint.TenantID

			// Re-resolve organization for the actual tenant (metadata org may be wrong)
			// Keep metadata org as fallback in case resolver is unavailable
			if s.orgResolver != nil {
				newOrg, orgErr := s.orgResolver.GetDefaultOrgForTenant(ctx, ownerTenantID)
				if orgErr != nil {
					s.logger.Warn(LogBSSCIOrgLookupFailed,
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
				"last_attached_bs_eui": session.BaseStationEUI,
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

			// BSSCI §3.8.3: Persist attPrpCmp to messages table
			// Gated by hasEUI to ensure usable audit rows with valid ep_eui
			// NOTE: This is separate from the attPrp row persisted at send time
			// NwkSnKey intentionally omitted: already in attPrp row, avoid key duplication
			if s.storage != nil && s.storage.MIOTYMessages() != nil {
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

				if err := s.storage.MIOTYMessages().CreateAttachPropagateMessage(ownerCtx, completionMsg); err != nil {
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

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	session.mu.Lock()
	session.LastScOpId--
	opId := session.LastScOpId
	session.mu.Unlock()

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingDetachPropagate,
		"sessionID", sessionID,
		"endpointEui", endpointEUI)

	// Convert endpointEUI to bytes for database storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, endpointEUI)

	// Resolve endpoint owner tenant BEFORE creating metadata (BSSCI §3.9 multi-tenant roaming)
	// This ensures metadata contains correct tenant/org for resume scenarios
	var endpointTenantID int64
	var ownerOrgUUID uuid.UUID
	var ownerCtx context.Context

	if s.endpointRepo != nil {
		// Try session tenant first (happy path)
		tenantID := resolvedTenant(session, s.tenantID)
		endpoint, err := s.endpointRepo.GetByEUI(ctx, tenantID, euiBytes)

		// Fallback: endpoint roamed from different tenant
		if err == storage.ErrNotFound {
			var eui models.EUI
			binary.BigEndian.PutUint64(eui[:], endpointEUI)
			endpoint, err = s.endpointRepo.Get(ctx, eui)
		}

		if err == nil && endpoint != nil {
			// Use endpoint's tenant (not session tenant) for all operations
			endpointTenantID = endpoint.TenantID

			// Resolve organization for owner tenant
			if s.orgResolver != nil {
				ownerOrgUUID, err = s.orgResolver.GetDefaultOrgForTenant(ctx, endpointTenantID)
				if err != nil {
					s.logger.Warn(LogBSSCIOrgLookupFailed, "tenantID", endpointTenantID, "error", err)
					// Continue without org - ownerOrgUUID remains Nil
				}
			}

			// Build owner-scoped context for all downstream operations
			// FIX-1: Use Background() to avoid session context pollution
			ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
			if ownerOrgUUID != uuid.Nil {
				ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
			}
		} else {
			// Endpoint not found - use session tenant as fallback, shAddr defaults to 0
			endpointTenantID = tenantID
			// FIX-1: Use Background() to avoid session context pollution
			ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
		}
	} else {
		// No repository - use session tenant, shAddr defaults to 0
		endpointTenantID = resolvedTenant(session, s.tenantID)
		// FIX-1: Use Background() to avoid session context pollution
		ownerCtx = pkgcontext.WithTenantID(context.Background(), endpointTenantID)
	}

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

	if err := s.persistPendingOperation(session, opId, mioty.CmdDetachPropagate, message, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistPendingOperation, "error", err)
		// Continue anyway - persistence failure shouldn't block operation
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
				"last_attached_bs_eui": session.BaseStationEUI,
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
	if s.storage != nil && s.storage.MIOTYMessages() != nil {
		if err := s.storage.MIOTYMessages().CreateDetachPropagateMessage(ownerCtx, propagateMsg); err != nil {
			s.logger.ErrorContext(ownerCtx, "Failed to persist detach propagate message",
				"ep_eui", endpointEUI,
				"bs_eui", session.BaseStationEUI,
				"tenant_id", endpointTenantID,
				"error", err)
			// Don't return error - persistence failure shouldn't abort protocol handshake
		}
	}

	// Send with rollback guard (BSSCI §5.2)
	if err := s.sendMessage(session, message); err != nil {
		// CRITICAL: Rollback on send failure to maintain operation ID consistency
		session.mu.Lock()
		session.LastScOpId++
		session.mu.Unlock()

		// Clean up persisted operation (guard for community builds)
		if session.DbSessionID > 0 {
			if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
					"sessionID", session.DbSessionID,
					"opId", opId,
					"error", cleanupErr)
			}
		}

		return err
	}

	// Success - persist counter to DB for session resume
	if s.sessionSvc != nil {
		if err := s.sessionSvc.UpdateSessionCounters(s.safeCtx(), session); err != nil {
			s.logger.Error(LogBSSCIFailedToUpdateDatabaseSession,
				"error", err,
				"sessionID", session.ID,
				"opId", opId)
			// Don't fail the operation - message was sent successfully
		}
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
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIDetachPropagateRejectedByBaseStation,
			"baseStation", session.BaseStationEUI,
			"opId", msg.OpId,
			"result", result)

		// Mark the pending operation as failed and persist to database
		if session.DbSessionID > 0 {
			// Get existing metadata or create new
			// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
			var pendingOp *PendingOperation
			if s.statusSvc != nil {
				pendingOp, _ = s.statusSvc.GetPendingOperation(session, int64(msg.OpId))
			}

			metadata := make(map[string]interface{})
			if pendingOp != nil && pendingOp.Metadata != nil {
				// Copy existing metadata
				for k, v := range pendingOp.Metadata {
					metadata[k] = v
				}
			}

			// Add failure information
			metadata["failed"] = true
			metadata["failureReason"] = fmt.Sprintf("Base station rejected with code %d", result)
			metadata["failedAt"] = time.Now().UnixNano()
			metadata["failureCode"] = result

			// Persist to database
			if err := s.updatePendingOperationMetadata(session, msg.OpId, metadata); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistFailureMetadata,
					"error", err,
					"opId", msg.OpId)
			}

			// Create system event for failure visibility in UI
			if s.eventStore != nil {
				// Extract owner tenant/org from metadata for roaming support
				ownerTenantID := resolvedTenant(session, s.tenantID)
				switch v := metadata["tenantId"].(type) {
				case int64:
					ownerTenantID = v
				case float64:
					ownerTenantID = int64(v)
				default:
					s.logger.Warn(LogBSSCIMissingTenantInMetadata, "opId", msg.OpId)
				}

				var ownerOrg uuid.UUID
				if orgIDStr, ok := metadata["organizationId"].(string); ok {
					ownerOrg, _ = uuid.Parse(orgIDStr) // Ignore parse error, Nil UUID is valid fallback
				}

				// Build owner context for event creation
				ownerCtx := pkgcontext.WithTenantID(ctx, ownerTenantID)
				if ownerOrg != uuid.Nil {
					ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrg)
				}

				// Extract endpoint EUI if available
				var epEUI uint64
				if v, ok := metadata["epEui"]; ok {
					epEUI, _ = parseMetadataEUI(v)
				}
				// Also check endpointEUI field for detach
				if epEUI == 0 {
					if v, ok := metadata["endpointEUI"]; ok {
						epEUI, _ = parseMetadataEUI(v)
					}
				}

				// Determine operation type
				operationType := "propagate_failed"
				eventType := "endpoint_operation_failed"
				if pendingOp != nil {
					switch pendingOp.OperationType {
					case mioty.CmdAttachPropagate:
						operationType = "attach_propagate_failed"
						eventType = "endpoint_attach_failed"
					case mioty.CmdDetachPropagate:
						operationType = "detach_propagate_failed"
						eventType = "endpoint_detach_failed"
					}
				}

				// Extract operation ID safely (avoid nil dereference)
				opID := msg.OpId
				if pendingOp != nil {
					opID = pendingOp.OperationID
				}

				details := map[string]interface{}{
					"epEui":        pkgmioty.FormatEUI64(epEUI),
					"bsEui":        pkgmioty.FormatEUI64(session.BaseStationEUI),
					"operation":    operationType,
					"operation_id": fmt.Sprintf("%d", opID), // For operation filtering
					"failureCode":  result,
					"reason":       fmt.Sprintf("Base station rejected with code %d", result),
				}
				detailsJSON, _ := json.Marshal(details)

				title := fmt.Sprintf("Operation failed for endpoint %s on BS %s", pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI))
				if pendingOp != nil && pendingOp.OperationType == mioty.CmdAttachPropagate {
					title = fmt.Sprintf("Attach propagate failed for endpoint %s on BS %s", pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI))
				} else if pendingOp != nil && pendingOp.OperationType == mioty.CmdDetachPropagate {
					title = fmt.Sprintf("Detach propagate failed for endpoint %s on BS %s", pkgmioty.FormatEUI64(epEUI), pkgmioty.FormatEUI64(session.BaseStationEUI))
				}

				// Create endpoint event with source fields for UI visibility (use owner tenant for roaming)
				if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
					TenantID:    fmt.Sprintf("%d", ownerTenantID),
					EventType:   eventType,
					Category:    mioty.CategoryEndpoint,
					Severity:    SeverityError,
					Title:       title,
					Description: fmt.Sprintf("Base station rejected operation with result code %d", result),
					Details:     detailsJSON,
					Status:      EventStatusNew,
					SourceType:  mioty.SourceTypeEndpoint,
					SourceName:  pkgmioty.FormatEUI64(epEUI), // Critical for UI filtering
					CreatedAt:   time.Now(),
				}); err != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateFailureEvent, "error", err)
				}

				// Also create base station event for symmetric tracking (use owner tenant for roaming)
				if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
					TenantID:    fmt.Sprintf("%d", ownerTenantID),
					EventType:   eventType,
					Category:    mioty.CategoryEndpoint,
					Severity:    SeverityError,
					Title:       fmt.Sprintf("Rejected %s for endpoint %s", operationType, pkgmioty.FormatEUI64(epEUI)),
					Description: fmt.Sprintf("Base station rejected operation with result code %d", result),
					Details:     detailsJSON,
					Status:      EventStatusNew,
					SourceType:  mioty.SourceTypeBaseStation,
					SourceName:  pkgmioty.FormatEUI64(session.BaseStationEUI), // Critical for UI filtering
					CreatedAt:   time.Now(),
				}); err != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateBaseStationFailureEvent, "error", err)
				}
			}
		}

		// Send error response to base station per BSSCI-4-01
		// MUST use POSIX error code, not the base station's result code
		errorMsg := map[string]interface{}{
			"command": mioty.CmdError, // Use constant instead of literal
			"opId":    msg.OpId,
			"code":    POSIX_EPROTO,                                  // Protocol error per BSSCI spec
			"message": ResolveErrorMessage(errDetachPropagateFailed), // Use error catalog
		}

		if err := s.sendMessage(session, errorMsg); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorMessageToBaseStation,
				"error", err)
		}

		// Return nil to keep connection open - individual operation failure shouldn't close session
		return nil
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

				// BSSCI §5.9.3: Persist detPrpCmp to messages table
				// Gated by hasEUI to ensure usable audit rows with valid ep_eui
				if s.storage != nil && s.storage.MIOTYMessages() != nil {
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

					if err := s.storage.MIOTYMessages().CreateDetachPropagateMessage(ownerCtx, completionMsg); err != nil {
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

// persistPendingOperation stores a pending operation via StatusService (BSSCI §5.11-5.12.3 single writer)
// StatusService handles both DB persistence and in-memory map update using SessionOpKey composite key
func (s *Server) persistPendingOperation(session *Session, opId int64, opType string, message map[string]interface{}, euiBytes []byte, metadata map[string]interface{}) error {
	ctx := s.safeCtx()

	if session.DbSessionID == 0 {
		s.logger.WarnContext(ctx, LogBSSCIDatabaseNotAvailableForPendingOpPersistence)
		return nil
	}

	// Extract MACType and Data from metadata if present (for VM operations)
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

	// Build complete PendingOperation struct
	pendingOp := &PendingOperation{
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

	// StatusService is the single path for pending operation persistence
	if err := s.statusSvc.RecordPendingOperation(ctx, session, opId, pendingOp, session.DbSessionID); err != nil {
		// Log but don't fail - pending operations are best effort
		s.logger.WarnContext(ctx, LogBSSCIFailedToPersistPendingOperationMigrationNeeded,
			"error", err,
			"sessionID", session.DbSessionID,
			"opId", opId)
		return nil
	}

	s.logger.DebugContext(ctx, LogBSSCIPersistedPendingOperation,
		"sessionID", session.DbSessionID,
		"opId", opId,
		"opType", opType)

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

	// Update metadata in database using repository
	err = s.storage.PendingOperations().UpdateMetadata(s.safeCtx(), sessionID, opId, json.RawMessage(metadataJSON))

	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToUpdatePendingOperationMetadata,
			"error", err,
			"sessionID", sessionID,
			"opId", opId)
		return err
	}

	// Also update in memory
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
	if s.statusSvc != nil {
		if pendingOp, err := s.statusSvc.GetPendingOperation(session, opId); err == nil {
			pendingOp.Metadata = metadata
		}
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

// loadPendingOperations loads pending operations from database for session resume (Issue #3: accepts *Session for SessionSlug field)
func (s *Server) loadPendingOperations(session *Session) ([]*PendingOperation, error) {
	sessionID := session.DbSessionID
	if sessionID == 0 {
		s.logger.DebugContext(s.safeCtx(), LogBSSCIDatabaseNotAvailableForPendingOpsLoad)
		return nil, nil
	}

	// Retrieve pending operations via repository
	repoOps, err := s.storage.PendingOperations().GetBySession(s.safeCtx(), sessionID)
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

		// Deserialize operation data
		var operationData map[string]interface{}
		if err := json.Unmarshal(operationDataJSON, &operationData); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUnmarshalOperationData,
				"error", err,
				"opId", opId)
			continue
		}

		// Deserialize metadata
		var metadata map[string]interface{}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUnmarshalMetadata,
					"error", err,
					"opId", opId)
				metadata = make(map[string]interface{})
			}
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
				s.logger.DebugContext(s.safeCtx(), "Normalized detach metadata on resume",
					"opId", opId,
					"epEui", typedMeta.EpEui)
			} else {
				s.logger.WarnContext(s.safeCtx(), "Failed to normalize detach metadata on resume, using raw data",
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

// getFloat64Field extracts a float64 field from data
func getFloat64Field(data map[string]interface{}, key string) (float64, bool) {
	value, exists := data[key]
	if !exists || value == nil {
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

// toFloat64 converts an interface{} value to float64, handling common numeric types.
// Used for parsing array elements in geoLocation and similar fields.
func toFloat64(value interface{}) (float64, bool) {
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

	return s.sendMessage(session, errorMsg)
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
		s.logger.ErrorContext(ctx, "Failed to send catalog error", "error", sendErr)
	}
	return fmt.Errorf("%s", err.Token)
}

// ParseVersion parses a version string "major.minor.patch" (BSSCI-2.1-01)
// Returns specific CatalogError for each failure type to preserve diagnostic precision
// Exported for use by service layer implementations
func ParseVersion(version string) (major, minor, patch int, cerr *CatalogError) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, 0, 0, NewCatalogError(errInvalidVersionFormat, POSIX_EPROTO)
	}

	var err error
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, NewCatalogError(errInvalidMajorVersion, POSIX_EPROTO)
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, NewCatalogError(errInvalidMinorVersion, POSIX_EPROTO)
	}

	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, NewCatalogError(errInvalidPatchVersion, POSIX_EPROTO)
	}

	return major, minor, patch, nil
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
			switch b := val.(type) {
			case float64:
				uuid[i] = byte(b)
			case int:
				uuid[i] = byte(b)
			case int64:
				uuid[i] = byte(b)
			case uint8:
				uuid[i] = b
			case int8:
				// Handle negative int8 values properly
				uuid[i] = byte(b)
			default:
				return nil, &CatalogError{Token: errInvalidUUIDByteType, Posix: POSIX_EPROTO}
			}
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
	if session.DbSessionID == 0 {
		return
	}

	// Update the counters and last message time via repository
	// Convert SessionUUID from []byte to [16]byte
	var sessionUUID [16]byte
	copy(sessionUUID[:], session.SessionUUID)
	err := s.storage.BaseStationSessions().UpdateCountersAndTimestamp(
		s.safeCtx(),
		sessionUUID,
		session.LastBsOpId,
		session.LastScOpId)

	if err != nil {
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
