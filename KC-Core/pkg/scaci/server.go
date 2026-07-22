// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"            // Organization resolver for propagation context
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"    // BSSCI §5.8-5.8.3 attach propagation contracts
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"      // Import neutral scheduler contracts
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/encoding"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/vmihailenco/msgpack/v5"
)

// normalizeInt64 converts various integer types from MessagePack to int64
//
// MessagePack can encode integers as int8, int16, int32, int64, or uint64 depending
// on the value range. This helper normalizes all variants to int64 for consistent handling.
//
// Parameters:
//   - v: Value from MessagePack decode (interface{})
//
// Returns:
//   - int64: Normalized 64-bit signed integer
//   - bool: True if conversion succeeded
func normalizeInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int16:
		return int64(val), true
	case int8:
		return int64(val), true
	case uint64:
		if val <= math.MaxInt64 {
			return int64(val), true
		}
	case uint32:
		return int64(val), true
	case uint16:
		return int64(val), true
	case uint8:
		return int64(val), true
	}
	return 0, false
}

// allowedSublayerPrefixes contains spec-defined sublayer prefixes per SCACI §4.
// Community edition has no handlers; managed/cloud may register rc.* handlers.
var allowedSublayerPrefixes = map[string]struct{}{"rc": {}}

// DetachPropagator allows SCACI to trigger BSSCI detach propagation without import cycles
//
// This interface is satisfied by the BSSCI Server and passed to SCACI during initialization.
// When an Application Center deregisters an endpoint, SCACI uses this interface to notify
// all connected base stations so they can clear their local endpoint state.
type DetachPropagator interface {
	// SendDetachPropagateToAll sends detach propagate to all connected base stations
	// Returns a slice of errors (one per failed base station), empty if all succeeded
	SendDetachPropagateToAll(epEUI uint64) []error
}

// Scheduler interface aliases for backward compatibility
// The actual interfaces are defined in pkg/scheduler to prevent circular dependencies
type (
	// ULTransmitScheduler is imported from pkg/scheduler
	ULTransmitScheduler = scheduler.ULTransmitScheduler

	// DownlinkScheduler is imported from pkg/scheduler
	DownlinkScheduler = scheduler.DownlinkScheduler
)

// Config holds SCACI server configuration per MIOTY SCACI v1.0.0
type Config struct {
	ListenAddr            string           // Address to listen on (host:port)
	TLS                   config.TLSConfig // TLS configuration (mutual TLS required per spec)
	ServiceCenterEUI      uint64           // Service Center EUI64
	Vendor                string           // SC vendor name
	Model                 string           // SC model
	Name                  string           // SC instance name
	SoftwareVersion       string           // SC software version
	OrgEnforcementEnabled bool             // Require valid X-Organization-ID header in strict mode
	LogPingOperations     bool             // Log Ping operations to scaci_operation_log (default: true)
	LogStatusOperations   bool             // Log Status operations to scaci_operation_log (default: true)
}

// Server implements the SCACI protocol server per MIOTY SCACI v1.0.0 §1-3
//
// Design:
//   - TCP listener with mutual TLS (§1)
//   - MessagePack framing with MIOTYA01 header (§3.1)
//   - Three-way handshake for all operations (§3)
//   - Session persistence and resumption (§3.3)
//   - Tenant isolation via certificate mapping
type Server struct {
	config   *Config
	logger   logger.Logger
	listener net.Listener

	// Session management
	sessions   map[net.Conn]*Session
	sessionsMu sync.RWMutex

	// Dependencies
	sessionRepo   interfaces.SCACISessionRepository
	operationRepo interfaces.SCACIOperationRepository // Operation lifecycle tracking (three-way handshake - cross-cutting concern)

	// Domain services
	handshakeSvc HandshakeService // Connect/version negotiation/session resumption
	endpointSvc  EndpointService  // Register/deregister endpoint operations
	ulSvc        ULService        // UL transmit scheduling via BSSCI (§3.9)
	dlSvc        DLService        // DL queue/revoke operations via BSSCI (§3.10)
	statusSvc    StatusService    // Server status/uptime/BS statistics (§3.5)

	// Cross-cutting services
	sessionValidator   SessionValidator   // Connect message field validation (§3.3)
	operationRecorder  OperationRecorder  // Operation logging for resume safety (§3.4)
	sessionPersistence SessionPersistence // Async session creation/update persistence (§3.3)
	errorRecorder      ErrorRecorder      // Error persistence and events (§3.14)

	// Organization resolution
	orgResolver org.Resolver // Organization UUID → tenant ID resolution for propagation context

	// BSSCI propagation (§5.8-5.8.3)
	sessionSnapshotProvider propagation.SessionSnapshotProvider // Provides connected BS sessions
	propagationSvc          propagation.Service                 // Orchestrates attach/detach propagation

	// Sublayer handlers (SCACI §4)
	// sublayerHandlers maps sublayer commands (e.g., "rc.dir") to handlers
	// Community edition: empty map (no handlers)
	// Managed/cloud: can register handlers without transport changes
	sublayerHandlers map[string]func(conn net.Conn, session *Session, opId int64, payload []byte) error

	// Lifecycle
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewServer creates a new SCACI server instance
//
// Validates required dependencies at construction (fail-fast pattern).
//
// Parameters:
//   - cfg: Server configuration (TLS, listen address, timeouts) - REQUIRED
//   - logger: Structured logger - REQUIRED
//   - sessionRepo: SCACI session persistence repository (§3.3) - REQUIRED
//   - operationRepo: Operation lifecycle tracking (§3.2) - REQUIRED
//   - handshakeSvc: Handshake service for Connect/version negotiation/session resumption (§3.3) - REQUIRED
//   - endpointSvc: Endpoint service for Register/Deregister operations (§3.6-3.7) - REQUIRED
//   - ulSvc: UL service for uplink transmit scheduling (§3.9) - REQUIRED
//   - dlSvc: DL service for downlink queue/revoke operations (§3.10-3.11) - REQUIRED
//   - statusSvc: Status service for uptime/BS stats (§3.5) - REQUIRED
//   - sessionValidator: Session validator for Connect message field validation (§3.3) - REQUIRED
//   - operationRecorder: Operation recorder for operation logging and resume safety (§3.4) - REQUIRED
//   - sessionPersistence: Session persistence service for async session creation/update (§3.3) - REQUIRED
//   - orgResolver: Organization UUID → tenant ID resolution (via org.Resolver) - optional
//   - sessionSnapshotProvider: BSSCI session snapshot provider (avoids circular dependency, optional)
//   - propagationSvc: Propagation service for attach/detach broadcasting (§5.8-5.8.3, optional)
//
// Returns:
//   - *Server: Configured server ready to Start()
//   - error: If any required dependency is nil
func NewServer(
	cfg *Config,
	log logger.Logger,
	sessionRepo interfaces.SCACISessionRepository,
	operationRepo interfaces.SCACIOperationRepository,
	handshakeSvc HandshakeService,
	endpointSvc EndpointService,
	ulSvc ULService,
	dlSvc DLService,
	statusSvc StatusService,
	sessionValidator SessionValidator,
	operationRecorder OperationRecorder,
	sessionPersistence SessionPersistence,
	orgResolver org.Resolver,
	sessionSnapshotProvider propagation.SessionSnapshotProvider,
	propagationSvc propagation.Service,
) (*Server, error) {
	// Validate required dependencies (fail-fast pattern per BSSCI server.go:641-644)
	if cfg == nil {
		return nil, fmt.Errorf("scaci.NewServer: cfg is required")
	}
	if log == nil {
		return nil, fmt.Errorf("scaci.NewServer: logger is required")
	}
	if sessionRepo == nil {
		return nil, fmt.Errorf("scaci.NewServer: sessionRepo is required (§3.3)")
	}
	if operationRepo == nil {
		return nil, fmt.Errorf("scaci.NewServer: operationRepo is required (§3.2)")
	}
	if handshakeSvc == nil {
		return nil, fmt.Errorf("scaci.NewServer: handshakeSvc is required (§3.3)")
	}
	if endpointSvc == nil {
		return nil, fmt.Errorf("scaci.NewServer: endpointSvc is required (§3.6-3.7)")
	}
	if ulSvc == nil {
		return nil, fmt.Errorf("scaci.NewServer: ulSvc is required (§3.9)")
	}
	if dlSvc == nil {
		return nil, fmt.Errorf("scaci.NewServer: dlSvc is required (§3.8)")
	}
	if statusSvc == nil {
		return nil, fmt.Errorf("scaci.NewServer: statusSvc is required (§3.5)")
	}
	if sessionValidator == nil {
		return nil, fmt.Errorf("scaci.NewServer: sessionValidator is required (§3.3.1)")
	}
	if operationRecorder == nil {
		return nil, fmt.Errorf("scaci.NewServer: operationRecorder is required")
	}
	if sessionPersistence == nil {
		return nil, fmt.Errorf("scaci.NewServer: sessionPersistence is required")
	}

	return &Server{
		config:                  cfg,
		logger:                  log,
		sessions:                make(map[net.Conn]*Session),
		sessionRepo:             sessionRepo,
		operationRepo:           operationRepo,
		handshakeSvc:            handshakeSvc,
		endpointSvc:             endpointSvc,
		ulSvc:                   ulSvc,
		dlSvc:                   dlSvc,
		statusSvc:               statusSvc,
		sessionValidator:        sessionValidator,
		operationRecorder:       operationRecorder,
		sessionPersistence:      sessionPersistence,
		orgResolver:             orgResolver,
		sessionSnapshotProvider: sessionSnapshotProvider,
		propagationSvc:          propagationSvc,
		sublayerHandlers:        make(map[string]func(net.Conn, *Session, int64, []byte) error), // §4: empty for community edition
		shutdown:                make(chan struct{}),
	}, nil
}

// SetErrorRecorder sets the ErrorRecorder per SCACI §3.14
//
// This setter allows wiring the ErrorRecorder after server construction,
// supporting gradual rollout and testing scenarios where the recorder
// may not be available at construction time.
func (s *Server) SetErrorRecorder(er ErrorRecorder) {
	s.errorRecorder = er
}

// sessionContext creates a context enriched with SCACI session metadata for structured logging.
// The logger's extractContextFields will automatically inject tenant_id and organization_id.
func (s *Server) sessionContext(session *Session) context.Context {
	ctx := context.Background()

	// SCACI sessions carry their own tenant ID
	ctx = pkgcontext.WithTenantID(ctx, session.TenantID)

	// Add organization ID if available (may be uuid.Nil in community mode)
	if session.OrganizationID != uuid.Nil {
		ctx = pkgcontext.WithOrganizationID(ctx, session.OrganizationID)
	}

	// Note: UserID not available for SCACI (machine-to-machine protocol)

	return ctx
}

// Start starts the SCACI server per MIOTY SCACI v1.0.0 §1
//
// Performs:
//  1. Load TLS configuration (mutual TLS required)
//  2. Create TCP listener on configured address
//  3. Accept connections in background goroutine
//
// Returns:
//   - error: Startup error if listener cannot be created
//
// scaciCertsExist checks whether all required TLS certificate files are present.
func scaciCertsExist(tlsCfg config.TLSConfig) bool {
	for _, f := range []string{tlsCfg.CertFile, tlsCfg.KeyFile, tlsCfg.CAFile} {
		if _, err := os.Stat(f); err != nil {
			return false
		}
	}
	return true
}

// Start starts the SCACI server. If TLS certificates are not yet available
// (e.g., fresh deployment before certs are generated via UI), the listener
// is deferred and a background goroutine polls until the certificates appear.
func (s *Server) Start() error {
	if scaciCertsExist(s.config.TLS) {
		return s.startTLSListener()
	}

	s.logger.Warn(LogSCACICertsNotFound,
		"cert", s.config.TLS.CertFile,
		"key", s.config.TLS.KeyFile,
		"ca", s.config.TLS.CAFile)

	s.wg.Add(1)
	go s.waitForCertsAndStart()
	return nil
}

// waitForCertsAndStart polls for certificate files and starts the TLS listener
// once they become available.
func (s *Server) waitForCertsAndStart() {
	defer s.wg.Done()

	const pollInterval = 10 * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdown:
			s.logger.Info(LogSCACIDeferredListenerCancelled)
			return
		case <-ticker.C:
			if !scaciCertsExist(s.config.TLS) {
				continue
			}
			s.logger.Info(LogSCACICertsDetected)
			if err := s.startTLSListener(); err != nil {
				s.logger.Error(LogSCACIDeferredListenerFailed, "error", err)
				continue
			}
			return
		}
	}
}

// startTLSListener loads certificates and starts the SCACI TLS listener.
func (s *Server) startTLSListener() error {
	tlsConfig, err := LoadTLSConfig(s.config.TLS)
	if err != nil {
		return fmt.Errorf("SCACI TLS configuration failed: %w", err)
	}

	listener, err := tls.Listen("tcp", s.config.ListenAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create TLS listener on %s: %w", s.config.ListenAddr, err)
	}

	s.listener = listener
	s.logger.Info(LogSCACIServerListening,
		"address", s.config.ListenAddr,
		"tls", "required",
		"spec", "MIOTY SCACI v1.0.0")

	s.wg.Add(1)
	go s.acceptConnections()

	s.wg.Add(1)
	go s.monitorIdleSessions()

	return nil
}

// Stop gracefully shuts down the SCACI server
//
// Performs:
//  1. Signal shutdown to all goroutines
//  2. Close listener to reject new connections
//  3. Wait for all connections to terminate
//
// Returns:
//   - error: Shutdown error (currently always nil)
func (s *Server) Stop() error {
	s.logger.Info(LogSCACIServerStopping)
	close(s.shutdown)

	if s.listener != nil {
		_ = s.listener.Close()
	}

	s.wg.Wait()
	s.logger.Info(LogSCACIServerStopped)
	return nil
}

// acceptConnections accepts incoming TLS connections
func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdown:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return // Expected during shutdown
			default:
				s.logger.Error(LogSCACIAcceptConnectionFailed, "error", err)
				continue
			}
		}

		// Handle connection in separate goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// monitorIdleSessions sends keepalive pings to idle SCACI sessions per SCACI §3.4
//
// This background loop monitors all active sessions and initiates SC-to-AC pings
// when a session has been idle for longer than PingInterval.
//
// The loop:
//  1. Collects idle sessions while holding sessionsMu.RLock()
//  2. Releases lock before network I/O (prevents blocking other operations)
//  3. Sends pings outside the lock
//  4. Uses shared PingInterval constant from KC-DB/common/config
//
// Lifecycle: Launched from Start(), stopped via s.shutdown channel
func (s *Server) monitorIdleSessions() {
	defer s.wg.Done()

	ticker := time.NewTicker(dbconfig.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdown:
			return
		case <-ticker.C:
			// Collect idle sessions while holding lock (no I/O under lock)
			var idleSessions []struct {
				conn    net.Conn
				session *Session
			}

			s.sessionsMu.RLock()
			now := time.Now()
			for conn, session := range s.sessions {
				if session.State == StateActive && now.Sub(session.LastSeen) > dbconfig.PingInterval {
					idleSessions = append(idleSessions, struct {
						conn    net.Conn
						session *Session
					}{conn, session})
				}
			}
			s.sessionsMu.RUnlock()

			// Send pings outside the lock
			for _, idle := range idleSessions {
				if err := s.initiatePing(idle.conn, idle.session); err != nil {
					// SCACI §1: Use session context for tenant-aware logging
					s.logger.WarnContext(s.sessionContext(idle.session), LogSCACISendKeepaliveFailed,
						"acEui", pkgmioty.FormatEUI64(idle.session.AcEui),
						"error", err)
				}
			}
		}
	}
}

// handleConnection processes a single SCACI connection per MIOTY SCACI v1.0.0 §3
//
// Flow:
//  1. Extract client certificate for service-layer tenant resolution
//  2. Wait for Connect message (must be first, opId=0)
//  3. Thread certificate to HandshakeService (service resolves tenant)
//  4. Process subsequent messages via routing
//  5. Clean up session on disconnect
//
// Note: Tenant resolution happens in HandshakeService, NOT at transport layer
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() { _ = conn.Close() }()

	// Clean up session mapping when connection closes
	defer func() {
		s.sessionsMu.Lock()
		delete(s.sessions, conn)
		s.sessionsMu.Unlock()
	}()

	// Extract client certificate from TLS connection
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		s.logger.Error(LogSCACIConnectionNotTLS)
		return
	}

	// Force TLS handshake to get client certificate
	if err := tlsConn.Handshake(); err != nil {
		s.logger.Error(LogSCACITLSHandshakeFailed, "error", err)
		return
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		s.logger.Error(LogSCACINoClientCertificate)
		return
	}

	clientCert := state.PeerCertificates[0]
	s.logger.Info(LogSCACIConnectionEstablished,
		"remote", conn.RemoteAddr().String(),
		"cert_cn", clientCert.Subject.CommonName)

	// Session will be created after Connect message is received
	// Per spec §3.3: Connect must be first message with opId=0
	var session *Session

	// Process messages until connection closes
	for {
		select {
		case <-s.shutdown:
			return
		default:
		}

		// Read framed message
		frame, err := s.readFrame(conn)
		if err != nil {
			if err == io.EOF {
				s.logger.Info(LogSCACIConnectionClosed,
					"remote", conn.RemoteAddr().String())
			} else {
				s.logger.Error(LogSCACIReadFrameFailed, "error", err)
			}
			return
		}

		// Decode payload: try MessagePack first, fallback to JSON per SCACI §1 framing requirements
		var msg map[string]interface{}
		if err := msgpack.Unmarshal(frame.Payload, &msg); err != nil {
			// Fallback to JSON decode per SCACI §1 dual-codec support
			if jsonErr := json.Unmarshal(frame.Payload, &msg); jsonErr != nil {
				s.logger.Error(LogSCACIDecodeMessagePackFailed,
					"msgpack_error", err,
					"json_error", jsonErr)
				_ = s.sendErrorWithCatalog(conn, session, 0, POSIX_EINVAL, errInvalidMessageFormat)
				continue
			}
		}

		// Extract command and opId (required per §3.2)
		command, ok := msg["command"].(string)
		if !ok {
			s.logger.Error(LogSCACIMissingCommandField)
			_ = s.sendErrorWithCatalog(conn, session, 0, POSIX_EINVAL, errMissingCommandField)
			continue
		}

		opId, ok := normalizeInt64(msg["opId"])
		if !ok {
			s.logger.Error(LogSCACIMissingOpIDField)
			_ = s.sendErrorWithCatalog(conn, session, 0, POSIX_EINVAL, errMissingOpIdField)
			continue
		}

		s.logger.Debug(LogSCACIReceivedMessage,
			"command", command,
			"opId", opId)

		// Special handling for Connect (must be first message)
		if session == nil && command != CmdConnect {
			s.logger.Error(LogSCACIFirstMessageMustBeConnect,
				"command", command)
			_ = s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errConnectRequired)
			return
		}

		// Route message to handler
		if err := s.routeMessage(conn, &session, clientCert, command, opId, frame.Payload); err != nil {
			s.logger.Error(LogSCACIHandlerError,
				"command", command,
				"opId", opId,
				"error", err)
			// Handler should have sent error response
		}
	}
}

// readFrame reads a single SCACI frame from the connection per §3.1
//
// Returns:
//   - *Frame: Parsed frame
//   - error: Read or parse error
func (s *Server) readFrame(conn net.Conn) (*Frame, error) {
	// Read header (12 bytes)
	header := make([]byte, MinFrameSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	// Parse header to get payload size
	payloadSize, err := ParseFrameHeader(header)
	if err != nil {
		return nil, err
	}

	// Create frame
	frame := &Frame{
		Identifier:  MIOTYA01Identifier,
		PayloadSize: payloadSize,
	}

	// Read payload if present
	if payloadSize > 0 {
		payload := make([]byte, payloadSize)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return nil, err
		}
		frame.Payload = payload
	}

	return frame, nil
}

// sendFrame sends a SCACI frame to the connection per §3.1
//
// Parameters:
//   - conn: Connection to send on
//   - payload: MessagePack encoded message
//
// Returns:
//   - error: Frame construction error or write error
func (s *Server) sendFrame(conn net.Conn, payload []byte) error {
	frame, err := NewFrame(payload)
	if err != nil {
		return fmt.Errorf("NewFrame failed: %w", err)
	}

	data, err := frame.Serialize()
	if err != nil {
		return fmt.Errorf("Serialize failed: %w", err)
	}

	_, err = conn.Write(data)
	return err
}

// sendError sends an error message per SCACI §3.14.1
//
// Parameters:
//   - conn: Connection to send on
//   - session: Current session (may be nil)
//   - opId: Operation ID from failed message
//   - code: POSIX error code
//   - message: Error description
//   - errorToken: Optional error catalog token (pass nil for legacy usage)
//
// Returns:
//   - error: Send error (logged, not returned to avoid error inception)
//
// sendError is the internal transport helper for sendErrorWithCatalog.
// All protocol errors should go through sendErrorWithCatalog for catalog consistency.
func (s *Server) sendError(conn net.Conn, _ *Session, opId int64, code int, message string, errorToken *string) error {
	// Dereference errorToken if not nil (ErrorToken is string, not *string, per spec)
	tokenStr := ""
	if errorToken != nil {
		tokenStr = *errorToken
	}

	errMsg := Error{
		BaseMessage: BaseMessage{
			Command: CmdError,
			OpId:    opId,
		},
		Code:       code,
		Message:    message,
		ErrorToken: tokenStr,
	}

	payload, err := msgpack.Marshal(&errMsg)
	if err != nil {
		s.logger.Error(LogSCACIMarshalErrorFailed, "error", err)
		return err
	}

	if err := s.sendFrame(conn, payload); err != nil {
		s.logger.Error(LogSCACISendErrorFailed, "error", err)
		return err
	}

	// Log with token if available
	logFields := []interface{}{
		"opId", opId,
		"code", code,
		"message", message,
	}
	if errorToken != nil {
		logFields = append(logFields, "errorToken", *errorToken)
	}
	s.logger.Debug(LogSCACIErrorMessageSent, logFields...)

	return nil
}

// sendErrorWithCatalog sends an error message using centralized error catalog
//
// POSIX Code Resolution: If the error token has a non-zero POSIXCode in the catalog,
// that code is used instead of defaultCode. Currently only version-related errors
// (§2.1-2.3) have POSIXCode in the catalog; all others use the caller's defaultCode.
//
// Parameters:
//   - conn: Connection to send on
//   - session: Current session (may be nil for pre-session errors)
//   - opId: Operation ID to include in error
//   - defaultCode: Default POSIX error code (overridden by catalog POSIXCode if non-zero)
//   - errorToken: Error token from errors_catalog.go
//   - contextDetail: Optional context detail (logged only, not sent on wire)
//
// Returns:
//   - error: Send error (logged, not returned to avoid error inception)
func (s *Server) sendErrorWithCatalog(conn net.Conn, session *Session, opId int64, defaultCode int, errorToken string, contextDetail ...string) error {
	ctx := context.Background()
	detail := ""
	if len(contextDetail) > 0 {
		detail = contextDetail[0]
	}

	// Use ErrorRecorder for persistence + events if available (§3.14)
	var posixCode int
	var message string
	if s.errorRecorder != nil && session != nil {
		var err error
		posixCode, message, err = s.errorRecorder.RecordOutboundError(
			ctx, session, opId, CmdError, errorToken, defaultCode, detail)
		if err != nil {
			s.logger.WarnContext(ctx, LogSCACIRecordEventFailed, "error", err)
		}
	} else {
		// Fallback to catalog lookup only (no persistence - legacy or nil session)
		def := GetErrorDefinition(errorToken)
		posixCode = defaultCode
		if def.POSIXCode != 0 {
			posixCode = def.POSIXCode
		}
		message = def.Message
	}

	// Log with spec traceability - handle nil session for connect failures
	def := GetErrorDefinition(errorToken)
	logFields := []interface{}{
		"token", def.Token,
		"spec", def.SpecSection,
		"severity", def.Severity,
		"opId", opId,
		"posixCode", posixCode,
	}

	// Only add acEui if session is not nil (connect failures may have no session yet)
	if session != nil {
		logFields = append(logFields, "acEui", pkgmioty.FormatEUI64(session.AcEui))
		if session.TenantID > 0 {
			logFields = append(logFields, "tenantId", session.TenantID)
		}
	}

	// Add optional context detail to logs (NOT sent on wire)
	if detail != "" {
		logFields = append(logFields, "context", detail)
	}

	s.logger.WarnContext(ctx, LogSCACIErrorReceived, logFields...)

	// Send on wire (errorToken NOT included per spec - internal only)
	return s.sendError(conn, session, opId, posixCode, message, nil)
}

// sendErrorAck sends error acknowledgement per SCACI §3.14.2
//
// Called when SC receives CmdError from AC to complete the error sequence.
//
// Parameters:
//   - conn: Connection to send on
//   - session: Session for logging context
//   - opId: Operation ID from the error message (echoed back)
//
// Returns:
//   - error: Send error
func (s *Server) sendErrorAck(conn net.Conn, _ *Session, opId int64) error {
	ack := ErrorAck{
		BaseMessage: BaseMessage{
			Command: CmdErrorAck,
			OpId:    opId,
		},
	}

	if err := s.sendResponse(conn, &ack); err != nil {
		s.logger.Error(LogSCACISentErrorAck, "opId", opId, "error", err)
		return err
	}

	s.logger.Debug(LogSCACISentErrorAck, "opId", opId)
	return nil
}

// sendResponse sends a message to the AC per SCACI §3
//
// Parameters:
//   - conn: Connection to send on
//   - msg: Message struct (must have Command and OpId)
//
// Returns:
//   - error: Serialization or send error
func (s *Server) sendResponse(conn net.Conn, msg interface{}) error {
	payload, err := msgpack.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := s.sendFrame(conn, payload); err != nil {
		return fmt.Errorf("failed to send frame: %w", err)
	}

	// Log response for debugging
	if msgJSON, err := json.Marshal(msg); err == nil {
		s.logger.Debug(LogSCACIResponseSent, "message", string(msgJSON))
	}

	return nil
}

// isACHandshakeCompletion returns true for AC-initiated messages that reuse their request's opId
//
// Per MIOTY SCACI three-way handshake pattern, AC sends:
//  1. Request (e.g., dlDataQue) with opId=N
//  2. SC responds (e.g., dlDataQueRsp) with same opId=N
//  3. AC completes (e.g., dlDataQueCmp) with same opId=N
//
// These "complete" and certain "response" messages must bypass opId monotonicity validation
// because they legitimately reuse the request's opId rather than incrementing.
func isACHandshakeCompletion(command string) bool {
	switch command {
	// Three-way handshake completions (AC reuses request opId)
	case CmdPingComplete,
		CmdStatusComplete,
		CmdRegisterComplete,
		CmdDeregisterComplete,
		CmdULDataComplete,
		CmdULDataTransmitComplete,
		CmdDLDataQueueComplete,
		CmdDLDataRevokeComplete,
		CmdDLDataResultResponse, // AC sends this as part of dlDataRes handshake
		CmdDLDataResultComplete,
		CmdEPStatusComplete,
		CmdErrorAck: // AC echoes triggering opId per §3.14.2
		return true
	default:
		return false
	}
}

// routeMessage dispatches messages to appropriate handlers per SCACI §3
//
// # This function is the central dispatcher for all SCACI operations
//
// Design:
//   - Connect handler receives certificate (service resolves tenant)
//   - Other handlers use session.TenantID (already resolved)
//
// Parameters:
//   - cert: Client certificate (only used by Connect handler)
//   - payload: Raw MessagePack payload from frame (handlers decode directly)
func (s *Server) routeMessage(conn net.Conn, session **Session, cert *x509.Certificate, command string, opId int64, payload []byte) error {
	// SCACI §3.3-01: Connect must complete before other operations
	if *session != nil && (*session).State == StateConnecting {
		if command != CmdConnect && command != CmdConnectComplete {
			return s.sendErrorWithCatalog(conn, *session, opId, POSIX_EINVAL, errConnectWaitingCmp)
		}
	}

	// SCACI §3.3.2: opId=0 is reserved for connect handshake only
	// Allowed inbound: CmdConnect, CmdConnectComplete, CmdErrorAck (echoes triggering opId)
	// CmdConnectResponse is SC→AC only, so inbound conRsp with opId=0 is rejected
	if opId == 0 {
		if command != CmdConnect && command != CmdConnectComplete && command != CmdErrorAck {
			// Nil-safe session pointer for sendErrorWithCatalog
			var sess *Session
			if session != nil {
				sess = *session
			}
			_ = s.sendErrorWithCatalog(conn, sess, opId, POSIX_EINVAL, ErrOpIDZeroReserved)
			return nil // Reject but keep connection open (recoverable error)
		}
	}

	// SCACI §3.2: Validate opId sign matches command initiator
	// Must occur BEFORE monotonicity validation and opId persistence
	if errToken := ValidateOpIDSign(command, opId); errToken != "" {
		var sess *Session
		if session != nil {
			sess = *session
		}
		_ = s.sendErrorWithCatalog(conn, sess, opId, POSIX_EINVAL, errToken)
		return nil // Reject, do NOT persist opId counters
	}

	// Validate AC opId monotonicity (skip Connect/ConnectComplete/ErrorAck and handshake completions)
	// ErrorAck echoes the opId from the original error message (SCACI §3.14.2)
	// Handshake completions reuse their request's opId per MIOTY SCACI three-way handshake pattern
	if *session != nil && opId > 0 &&
		command != CmdConnect && command != CmdConnectComplete && command != CmdErrorAck &&
		!isACHandshakeCompletion(command) {

		s.sessionsMu.Lock()
		err := (*session).ValidateAcOpId(opId)
		if err == nil {
			(*session).UpdateAcOpId(opId)
		}
		// Capture values for persistence before releasing lock
		sessionID := (*session).ID
		tenantID := (*session).TenantID
		newAcOpID := (*session).AcOpIdCounter
		scOpID := (*session).ScOpIdCounter
		s.sessionsMu.Unlock()

		if err != nil {
			return s.sendErrorWithCatalog(conn, *session, opId, POSIX_EINVAL, errOpIdOutOfOrder, err.Error())
		}

		// Persist opId pair atomically per §3.2
		s.persistOpIDsPair(sessionID, tenantID, newAcOpID, scOpID)
	}

	// SCACI §4 Sublayer Guard: Handle dotted sublayer commands
	// Runs after opId validation, before handler switch
	if idx := strings.IndexByte(command, '.'); idx > 0 {
		prefix := command[:idx]

		// Extract session safely (double pointer pattern per lines 871-873)
		var sess *Session
		if session != nil {
			sess = *session
		}

		// Build log fields (nil-safe: only add acEui when session exists)
		logFields := []interface{}{"command", command, "prefix", prefix}
		if sess != nil {
			logFields = append(logFields, "acEui", pkgmioty.FormatEUI64(sess.AcEui))
		}

		// Case 1: Unknown prefix (not in spec allowlist)
		if _, allowed := allowedSublayerPrefixes[prefix]; !allowed {
			s.logger.Warn(LogSCACIUnsupportedSublayerPrefix, logFields...)
			return s.sendErrorWithCatalog(conn, sess, opId, POSIX_ENOTSUP, errUnsupportedSublayerPrefix, command)
		}

		// Case 2: Allowed prefix - check if handler exists for the full command
		// Future: managed/cloud can register handlers in s.sublayerHandlers map
		handler, hasHandler := s.sublayerHandlers[command]
		if !hasHandler {
			// No handler registered for this command
			s.logger.Warn(LogSCACIUnsupportedSublayerPrefix, logFields...)
			return s.sendErrorWithCatalog(conn, sess, opId, POSIX_ENOTSUP, errUnsupportedSublayerPrefix, command)
		}

		// Case 3: Handler exists - dispatch directly and return
		// (bypasses switch to avoid double-error from default case)
		s.logger.Debug("SCACI sublayer handler invoked", logFields...)
		return handler(conn, sess, opId, payload)
	}

	switch command {
	// Connection operations (§3.3)
	case CmdConnect:
		return s.handleConnect(conn, session, cert, opId, payload)
	case CmdConnectComplete:
		return s.handleConnectComplete(conn, *session, opId)

	// Ping operations (§3.4)
	case CmdPing:
		return s.handlePing(conn, *session, opId)
	case CmdPingResponse:
		return s.handlePingResponse(conn, *session, opId)
	case CmdPingComplete:
		return s.handlePingComplete(conn, *session, opId)

	// Status operations (§3.5)
	case CmdStatus:
		return s.handleStatus(conn, *session, opId)
	case CmdStatusComplete:
		return s.handleStatusComplete(conn, *session, opId)

	// Register operations (§3.6)
	case CmdRegister:
		return s.handleRegister(conn, *session, opId, payload)
	case CmdRegisterComplete:
		return s.handleRegisterComplete(conn, *session, opId)

	// Deregister operations (§3.7)
	case CmdDeregister:
		return s.handleDeregister(conn, *session, opId, payload)
	case CmdDeregisterComplete:
		return s.handleDeregisterComplete(conn, *session, opId)

	// UL data operations (§3.8)
	case CmdULDataResponse:
		return s.handleULDataResponse(conn, *session, opId)
	case CmdULDataComplete:
		// AC should never send ulDataCmp - SC sends it per SCACI §3.8.3
		s.logger.Error(LogSCACIUnexpectedULDataCmp,
			"opId", opId,
			"acEui", pkgmioty.FormatEUI64((*session).AcEui))

		// Mark operation failed if it exists
		if (*session).ID > 0 && s.operationRepo != nil {
			errCtx, errCancel := context.WithTimeout(context.Background(), dbconfig.DefaultQueryTimeout)
			defer errCancel()

			if err := s.operationRepo.UpdateOperationState(errCtx, (*session).ID, opId,
				models.OperationStateFailed, map[string]interface{}{
					"errorToken":  errProtocolViolationULCmp,
					"errorDetail": "AC sent ulDataCmp but SC should send it",
				}); err != nil {
				s.logger.Error(LogSCACIUpdateOperationStateFailed, "error", err)
			}
		}

		return s.sendErrorWithCatalog(conn, *session, opId, POSIX_EPROTO, errProtocolViolationULCmp)

	// UL data transmit operations (§3.9)
	case CmdULDataTransmit:
		return s.handleULDataTransmit(conn, *session, opId, payload)
	case CmdULDataTransmitResponse:
		// AC cannot send ulDataTxRsp - protocol violation per SCACI §3.9.2
		s.logger.Error(LogSCACIUnexpectedULDataTxRsp,
			"opId", opId,
			"acEui", pkgmioty.FormatEUI64((*session).AcEui))

		// Mark operation failed if it exists
		if (*session).ID > 0 && s.operationRepo != nil {
			errCtx, errCancel := context.WithTimeout(context.Background(), dbconfig.DefaultQueryTimeout)
			defer errCancel()

			if err := s.operationRepo.UpdateOperationState(errCtx, (*session).ID, opId,
				models.OperationStateFailed, map[string]interface{}{
					"errorToken":  errProtocolViolationULRsp,
					"errorDetail": "AC sent ulDataTxRsp but SC should send it",
				}); err != nil {
				s.logger.Error(LogSCACIUpdateOperationStateFailed, "error", err)
			}
		}

		return s.sendErrorWithCatalog(conn, *session, opId, POSIX_EPROTO, errProtocolViolationULRsp)
	case CmdULDataTransmitComplete:
		return s.handleULDataTransmitComplete(conn, *session, opId)

	// DL data operations (§3.10-3.12)
	case CmdDLDataQueue:
		return s.handleDLDataQueue(conn, *session, opId, payload)
	case CmdDLDataQueueComplete:
		return s.handleDLDataQueueComplete(conn, *session, opId)

	case CmdDLDataRevoke:
		return s.handleDLDataRevoke(conn, *session, opId, payload)
	case CmdDLDataRevokeComplete:
		return s.handleDLDataRevokeComplete(conn, *session, opId)

	case CmdDLDataResultResponse:
		return s.handleDLDataResultResponse(conn, *session, opId)
	case CmdDLDataResultComplete:
		return s.handleDLDataResultComplete(conn, *session, opId)

	// EP status operations (§3.13)
	case CmdEPStatusResponse:
		return s.handleEPStatusResponse(conn, *session, opId)
	case CmdEPStatusComplete:
		return s.handleEPStatusComplete(conn, *session, opId)

	// Error handling (§3.14)
	case CmdError:
		return s.handleInboundError(conn, *session, opId, payload)

	// Error acknowledgement (§3.14.2)
	case CmdErrorAck:
		return s.handleErrorAck(conn, *session, opId)

	default:
		s.logger.Warn(LogSCACIUnknownCommand, "command", command)
		return s.sendErrorWithCatalog(conn, *session, opId, POSIX_ENOTSUP, errUnsupportedCommand)
	}
}

// persistOpIDsPair persists both AC and SC operation IDs atomically per SCACI §3.2
//
// This helper ensures that both operation ID counters are persisted together in a single
// database transaction, preventing race conditions during session resume. If a crash
// occurs between separate persists, the session may fail to resume correctly.
//
// Parameters:
//   - sessionID: Database session ID (skip if <= 0)
//   - tenantID: Tenant ID for multi-tenant isolation
//   - acOpId: AC operation ID counter to persist
//   - scOpId: SC operation ID counter to persist
func (s *Server) persistOpIDsPair(sessionID, tenantID, acOpId, scOpId int64) {
	if sessionID <= 0 {
		return
	}
	go func(sid, tid, acId, scId int64) {
		ctx, cancel := context.WithTimeout(context.Background(), SessionPersistTimeout)
		defer cancel()
		if err := s.sessionRepo.UpdateOperationIDs(ctx, tid, sid, acId, scId); err != nil {
			s.logger.ErrorContext(ctx, LogSCACIPersistOpIDsPairFailed, "sessionID", sid, "error", err)
		}
	}(sessionID, tenantID, acOpId, scOpId)
}

// initiatePing sends SC-initiated ping to detect connection failures per SCACI §3.4
//
// This method is called by the keepalive monitor when a session has been idle
// for longer than the ping interval. It initiates the three-way handshake:
//  1. SC → AC: Ping (negative opId)
//  2. AC → SC: PingResponse (same opId)
//  3. SC → AC: PingComplete (same opId)
//
// Parameters:
//   - conn: TLS connection to Application Center
//   - session: Active SCACI session
//
// Returns:
//   - error: Send error, or nil on success
func (s *Server) initiatePing(conn net.Conn, session *Session) error {
	session.WriteMu.Lock()
	opId := session.NextScOpId()

	ping := Ping{
		BaseMessage: BaseMessage{
			Command: CmdPing,
			OpId:    opId,
		},
	}

	// Capture while mutex is still held
	sessionID := session.ID
	tenantID := session.TenantID
	acOpId := session.AcOpIdCounter
	scOpId := session.ScOpIdCounter

	err := s.sendResponse(conn, &ping)
	session.WriteMu.Unlock()

	if err == nil {
		s.persistOpIDsPair(sessionID, tenantID, acOpId, scOpId)

		// Persist heartbeat after successful ping send (async, best-effort per §3.4)
		if s.sessionPersistence != nil && session != nil && session.ID > 0 {
			s.sessionPersistence.PersistHeartbeatAsync(s.sessionContext(session), session)
		}

		// Record SC-initiated Ping for audit trail (§3.4) if configured
		if s.config.LogPingOperations && sessionID > 0 && s.operationRecorder != nil {
			recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
			defer recCancel()

			requestData := map[string]interface{}{
				"opId":      opId,
				"initiator": "sc",
			}
			if err := s.operationRecorder.Record(recCtx, session, opId, CmdPing, models.OperationDirectionOutbound, requestData); err != nil {
				s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingOpFailed, "error", err)
			}
		}
	}

	return err
}

// buildFallbackBaseStationReception creates a single-BS reception array from legacy ULDataMessage fields
// Deep-copies optional fields and sanitizes per SCACI §3.8.1
func buildFallbackBaseStationReception(data *mioty.ULDataMessage) []mioty.BaseStationReception {
	bs := mioty.BaseStationReception{
		BsEui:  data.BsEui,
		RxTime: data.RxTime,
		Snr:    data.SNR,
		Rssi:   data.RSSI,
	}

	if data.RxDuration != nil {
		copied := *data.RxDuration
		bs.RxDuration = &copied
	}
	if data.EqSnr != nil {
		copied := *data.EqSnr
		bs.EqSnr = &copied
	}
	if data.Profile != nil {
		copied := *data.Profile
		bs.Profile = &copied
	}
	if data.Mode != nil {
		copied := *data.Mode
		bs.Mode = &copied
	}
	if data.Subpackets != nil {
		bs.Subpackets = &mioty.Subpackets{
			SNR:       append([]float64{}, data.Subpackets.SNR...),
			RSSI:      append([]float64{}, data.Subpackets.RSSI...),
			Frequency: append([]int64{}, data.Subpackets.Frequency...),
			Phase:     append([]float64{}, data.Subpackets.Phase...),
		}
	}

	return []mioty.BaseStationReception{bs}
}

// buildBaseStationsRequestData converts BaseStationReception array to operation log format
// Returns array of maps with hex EUIs and numeric metrics for SCACI operation tracking
func buildBaseStationsRequestData(basestations []mioty.BaseStationReception) []map[string]interface{} {
	result := make([]map[string]interface{}, len(basestations))
	for i, bs := range basestations {
		entry := map[string]interface{}{
			"bsEui":  pkgmioty.FormatEUI64(bs.BsEui),
			"rxTime": bs.RxTime,
			"snr":    bs.Snr,
			"rssi":   bs.Rssi,
		}
		if bs.RxDuration != nil {
			entry["rxDuration"] = *bs.RxDuration
		}
		if bs.EqSnr != nil {
			entry["eqSnr"] = *bs.EqSnr
		}
		if bs.DlRxSnr != nil {
			entry["dlRxSnr"] = *bs.DlRxSnr
		}
		if bs.DlRxRssi != nil {
			entry["dlRxRssi"] = *bs.DlRxRssi
		}
		if bs.Profile != nil {
			entry["profile"] = *bs.Profile
		}
		if bs.Mode != nil {
			entry["mode"] = *bs.Mode
		}
		// Include subpackets for evidence fidelity per SCACI §3.8.1
		if bs.Subpackets != nil {
			subpkts := map[string]interface{}{
				"snr":       bs.Subpackets.SNR,
				"rssi":      bs.Subpackets.RSSI,
				"frequency": bs.Subpackets.Frequency,
			}
			if len(bs.Subpackets.Phase) > 0 {
				subpkts["phase"] = bs.Subpackets.Phase
			}
			entry["subpackets"] = subpkts
		}
		result[i] = entry
	}
	return result
}

// derefBool safely dereferences a *bool, returning false if nil.
// Used for operation log JSON serialization where we need a bool value (not a pointer).
func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// BroadcastULData forwards UL data to all active ACs for tenant (synchronous)
//
// Parameters:
//   - ctx: Context for operation
//   - tenantID: Tenant ID to filter sessions
//   - data: UL data message from BSSCI (typed mioty.ULDataMessage)
//
// Returns:
//   - error: If any sends fail
func (s *Server) BroadcastULData(_ context.Context, tenantID int64, data *mioty.ULDataMessage) error {
	if data == nil {
		return fmt.Errorf("nil UL data message")
	}

	s.sessionsMu.RLock()
	type target struct {
		conn    net.Conn
		session *Session
	}
	var targets []target
	for conn, session := range s.sessions {
		if session.TenantID == tenantID && session.State == StateActive {
			targets = append(targets, target{conn, session})
		}
	}
	s.sessionsMu.RUnlock()

	if len(targets) == 0 {
		return nil
	}

	// Build canonical BaseStationReception array
	var baseStations []mioty.BaseStationReception
	if len(data.BaseStations) > 0 {
		baseStations = data.BaseStations
	} else {
		baseStations = buildFallbackBaseStationReception(data)
	}

	// Force responseExp=false when dlOpen=false per SCACI §3.8.1 semantics
	responseExp := data.ResponseExp && data.DlOpen

	for _, t := range targets {
		t.session.WriteMu.Lock()
		opId := t.session.NextScOpId()

		msg := ULData{
			BaseMessage: BaseMessage{
				Command: CmdULData,
				OpId:    opId,
			},
			EpEui:        data.EpEui,
			BaseStations: baseStations,
			PacketCnt:    data.PacketCnt,
			UserData:     append([]byte(nil), data.UserData...),
			Format:       data.Format,
			DlOpen:       data.DlOpen,
			ResponseExp:  responseExp,
			DlAck:        data.DlAck,
			Duplicate:    data.Duplicate, // Copy pointer directly - both are *bool per SCACI §3.8.1
		}

		// Use SendULData for validation per SCACI §3.8.1 - validates mandatory fields
		err := s.SendULData(t.conn, t.session, &msg)

		// Capture values before unlock
		sessionID := t.session.ID
		tenantID := t.session.TenantID
		acOpID := t.session.AcOpIdCounter
		newScOpID := t.session.ScOpIdCounter

		t.session.WriteMu.Unlock()

		if err != nil {
			// Log with context and continue to next session - don't abort broadcast to other ACs
			s.logger.ErrorContext(s.sessionContext(t.session), LogSCACISendULDataFailed,
				"epEui", pkgmioty.FormatEUI64(data.EpEui),
				"error", err)
			continue // Continue to next session, don't fail entire broadcast
		}

		// Update session after successful send
		t.session.UpdateLastSeen()

		// Persist opId pair atomically per §3.2
		s.persistOpIDsPair(sessionID, tenantID, acOpID, newScOpID)

		// Record operation after successful send (for resume/audit)
		if sessionID > 0 && s.operationRepo != nil {
			opCtx, opCancel := context.WithTimeout(context.Background(), dbconfig.DefaultQueryTimeout)

			requestData := map[string]interface{}{
				"epEui":        pkgmioty.FormatEUI64(data.EpEui),
				"packetCnt":    data.PacketCnt,
				"dlOpen":       data.DlOpen,
				"responseExp":  responseExp,
				"dlAck":        data.DlAck,
				"baseStations": buildBaseStationsRequestData(baseStations),
				"duplicate":    derefBool(data.Duplicate), // Dereference *bool to bool for JSON per SCACI §3.8.1
			}
			if data.Format != nil {
				requestData["format"] = *data.Format
			}
			if data.UserData != nil {
				requestData["userData"] = encoding.EncodeUserData(data.UserData)
			}

			_, err := s.operationRepo.RecordOperation(opCtx, &models.SCACIOperationRequest{
				SessionID:   sessionID,
				TenantID:    tenantID,
				OpId:        opId,
				Command:     CmdULData,
				Direction:   string(models.OperationDirectionOutbound),
				RequestData: requestData,
			})

			opCancel() // Cancel immediately after operation, not deferred

			if err != nil {
				s.logger.Warn(LogSCACIRecordULDataOpFailed,
					"opId", opId,
					"error", err)
				// Continue - operation tracking is for resume, not critical path
			}
		}
	}

	return nil
}

// BroadcastDLDataResult forwards DL data results to all active ACs for tenant (synchronous)
//
// Parameters:
//   - ctx: Context for operation
//   - tenantID: Tenant ID to filter sessions
//   - result: DL data result from BSSCI (typed mioty.DLDataResult)
//
// Returns:
//   - error: If any sends fail
func (s *Server) BroadcastDLDataResult(_ context.Context, tenantID int64, result *mioty.DLDataResult) error {
	if result == nil {
		return fmt.Errorf("nil DL data result")
	}

	s.sessionsMu.RLock()
	type target struct {
		conn    net.Conn
		session *Session
	}
	var targets []target
	for conn, session := range s.sessions {
		if session.TenantID == tenantID && session.State == StateActive {
			targets = append(targets, target{conn, session})
		}
	}
	s.sessionsMu.RUnlock()

	if len(targets) == 0 {
		return nil
	}

	for _, t := range targets {
		t.session.WriteMu.Lock()
		opId := t.session.NextScOpId()

		// Copy optional pointer fields to avoid aliasing
		var txTimeCopy *int64
		if result.TxTime != nil {
			val := *result.TxTime
			txTimeCopy = &val
		}
		var packetCntCopy *uint32
		if result.PacketCnt != nil {
			val := *result.PacketCnt
			packetCntCopy = &val
		}
		var bsEuiCopy *uint64
		if result.BsEui != nil {
			val := *result.BsEui
			bsEuiCopy = &val
		}

		msg := DLDataResult{
			BaseMessage: BaseMessage{
				Command: CmdDLDataResult,
				OpId:    opId,
			},
			EpEui:     result.EpEui,
			QueID:     result.QueId,
			Result:    result.Result,
			TxTime:    txTimeCopy,
			PacketCnt: packetCntCopy,
			BsEui:     bsEuiCopy,
		}

		// Validate and send via helper (SCACI §3.12.1)
		err := s.SendDLDataResult(t.conn, t.session, &msg)

		// Capture values before unlock
		sessionID := t.session.ID
		tenantID := t.session.TenantID
		acOpID := t.session.AcOpIdCounter
		newScOpID := t.session.ScOpIdCounter

		t.session.WriteMu.Unlock()

		if err != nil {
			// Log with context and continue to next session - don't abort broadcast to other ACs
			s.logger.ErrorContext(s.sessionContext(t.session), LogSCACISendDLResultToACFailed,
				"acEui", pkgmioty.FormatEUI64(t.session.AcEui),
				"error", err)
			continue // Continue to next session, don't fail entire broadcast
		}

		// Update session after successful send
		t.session.UpdateLastSeen()

		// Persist opId pair atomically per §3.2
		s.persistOpIDsPair(sessionID, tenantID, acOpID, newScOpID)

		// Record operation after successful send (for resume/audit)
		if sessionID > 0 && s.operationRepo != nil {
			opCtx, opCancel := context.WithTimeout(context.Background(), dbconfig.DefaultQueryTimeout)

			requestData := map[string]interface{}{
				"epEui":  pkgmioty.FormatEUI64(result.EpEui),
				"queId":  result.QueId,
				"result": result.Result,
			}
			if result.TxTime != nil {
				requestData["txTime"] = *result.TxTime
			}
			if result.PacketCnt != nil {
				requestData["packetCnt"] = *result.PacketCnt
			}
			if result.BsEui != nil {
				requestData["bsEui"] = pkgmioty.FormatEUI64(*result.BsEui)
			}

			_, err := s.operationRepo.RecordOperation(opCtx, &models.SCACIOperationRequest{
				SessionID:   sessionID,
				TenantID:    tenantID,
				OpId:        opId,
				Command:     CmdDLDataResult,
				Direction:   string(models.OperationDirectionOutbound),
				RequestData: requestData,
			})

			opCancel() // Cancel immediately after operation, not deferred

			if err != nil {
				s.logger.Warn(LogSCACIRecordDLResultOpFailed,
					"opId", opId,
					"error", err)
			}
		}
	}

	return nil
}

// EPStatusData holds the data for an EPStatus broadcast
// This struct is used as input to BroadcastEPStatus and mirrors the EPStatus message fields.
// BSSCI→SCACI integration uses this to pass attach/detach status data.
type EPStatusData struct {
	EpEui      uint64
	EpStatus   string            // EPStatusAttached or EPStatusDetached
	AttachCnt  *uint32           // OTA attach only
	Nonce      *mioty.Numeric4   // OTA attach only
	Sign       *mioty.Numeric4   // OTA attach/detach only
	Snr        *float64          // OTA attach/detach only
	Rssi       *float64          // OTA attach/detach only
	EqSnr      *float64          // Optional
	Subpackets *mioty.Subpackets // Optional
}

// BroadcastEPStatus forwards endpoint status to all active ACs for tenant (synchronous)
// Per SCACI §3.13, this is SC-initiated: SC sends epStatus, AC responds/completes
//
// Parameters:
//   - ctx: Context for operation
//   - tenantID: Tenant ID to filter sessions
//   - data: EPStatus data (epEui, epStatus, optional OTA fields)
//
// Returns:
//   - error: If any sends fail
func (s *Server) BroadcastEPStatus(ctx context.Context, tenantID int64, data *EPStatusData) error {
	if data == nil {
		return fmt.Errorf("nil EPStatus data")
	}

	// Collect targets under read lock (same pattern as BroadcastDLDataResult:1197-1208)
	s.sessionsMu.RLock()
	type target struct {
		conn    net.Conn
		session *Session
	}
	var targets []target
	for conn, session := range s.sessions {
		if session.TenantID == tenantID && session.State == StateActive {
			targets = append(targets, target{conn, session})
		}
	}
	s.sessionsMu.RUnlock()

	if len(targets) == 0 {
		s.logger.DebugContext(ctx, LogSCACINoActiveACsForTenant, "tenantId", tenantID)
		return nil // No connected ACs, not an error
	}

	for _, t := range targets {
		t.session.WriteMu.Lock()
		opId := t.session.NextScOpId()

		msg := &EPStatus{
			BaseMessage: BaseMessage{
				Command: CmdEPStatus, // MUST set command explicitly
				OpId:    opId,
			},
			EpEui:    data.EpEui,
			EpStatus: data.EpStatus,
		}

		// Copy optional OTA fields if present (avoid pointer aliasing)
		if data.AttachCnt != nil {
			v := *data.AttachCnt
			msg.AttachCnt = &v
		}
		if data.Nonce != nil {
			msg.Nonce = data.Nonce
		}
		if data.Sign != nil {
			msg.Sign = data.Sign
		}
		if data.Snr != nil {
			v := *data.Snr
			msg.Snr = &v
		}
		if data.Rssi != nil {
			v := *data.Rssi
			msg.Rssi = &v
		}
		if data.EqSnr != nil {
			v := *data.EqSnr
			msg.EqSnr = &v
		}
		if data.Subpackets != nil {
			msg.Subpackets = data.Subpackets
		}

		// Use SendEPStatus (NOT sendResponse) - validates and logs per §2.5
		err := s.SendEPStatus(t.conn, t.session, msg)

		// Capture values before unlock
		sessionID := t.session.ID
		sessionTenantID := t.session.TenantID
		acOpID := t.session.AcOpIdCounter
		newScOpID := t.session.ScOpIdCounter

		t.session.WriteMu.Unlock()

		if err != nil {
			s.logger.ErrorContext(ctx, LogSCACISendEPStatusFailed,
				"acEui", pkgmioty.FormatEUI64(t.session.AcEui),
				"error", err)
			return err
		}

		t.session.UpdateLastSeen()
		s.persistOpIDsPair(sessionID, sessionTenantID, acOpID, newScOpID)

		// Record operation for replay (SCACI §1 resume requirement)
		// Use incoming context for tenant/org propagation
		if sessionID > 0 && s.operationRepo != nil {
			opCtx, opCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)

			requestData := map[string]interface{}{
				"epEui":    pkgmioty.FormatEUI64(data.EpEui),
				"epStatus": data.EpStatus,
			}
			// Store OTA fields if present
			// NOTE: Store nonce/sign as base64 strings (matching encoding pattern)
			if data.AttachCnt != nil {
				requestData["attachCnt"] = *data.AttachCnt // uint32 stored as number
			}
			if data.Nonce != nil {
				requestData["nonce"] = encoding.EncodeUserData(data.Nonce[:])
			}
			if data.Sign != nil {
				requestData["sign"] = encoding.EncodeUserData(data.Sign[:])
			}
			if data.Snr != nil {
				requestData["snr"] = *data.Snr
			}
			if data.Rssi != nil {
				requestData["rssi"] = *data.Rssi
			}
			if data.EqSnr != nil {
				requestData["eqSnr"] = *data.EqSnr
			}
			// Store subpackets as JSON-serializable structure
			if data.Subpackets != nil {
				requestData["subpackets"] = data.Subpackets
			}

			_, err := s.operationRepo.RecordOperation(opCtx, &models.SCACIOperationRequest{
				SessionID:   sessionID,
				TenantID:    sessionTenantID,
				OpId:        opId,
				Command:     CmdEPStatus,
				Direction:   string(models.OperationDirectionOutbound),
				RequestData: requestData,
			})

			opCancel() // Cancel immediately after operation, not deferred

			if err != nil {
				s.logger.WarnContext(ctx, LogSCACIRecordEPStatusOpFailed,
					"opId", opId,
					"error", err)
				// Continue - operation tracking is for resume, not critical path
			}
		}
	}

	return nil
}

// ========================================================================
// SCACI §1: Pending Operation Replay on Session Resume
// ========================================================================

// replayableCommands defines which SC-originated operations are replayable
// Only commands that are recorded with OperationDirectionOutbound and negative opIds
var replayableCommands = map[string]bool{
	CmdULData:       true, // server.go:1143-1149
	CmdDLDataResult: true, // server.go:1269-1276
	CmdEPStatus:     true, // BroadcastEPStatus - SC-initiated per §3.13
}

// nonReplayableReasons documents why commands are skipped (for logging)
// NOTE: EPStatus is NOT in this list - it IS SC-initiated and replayable per SCACI §3.13
//
// Connect (§3.3) and Ping (§3.4) are non-replayable because:
// - Connect: Session establishment handshake only, uses opId=0
// - Ping: Keepalive mechanism with no persistent state change
// Both are logged to scaci_operation_log for audit trail but excluded from replay.
var nonReplayableReasons = map[string]string{
	CmdConnect:         "Session establishment handshake per §3.3",
	CmdConnectResponse: "Session establishment handshake per §3.3",
	CmdConnectComplete: "Session establishment handshake per §3.3",
	CmdPing:            "Keepalive operation per §3.4, no state change",
	CmdPingResponse:    "Keepalive operation per §3.4, no state change",
	CmdPingComplete:    "Keepalive operation per §3.4, no state change",
	CmdStatus:          "AC-initiated per §3.5, SC only responds",
	CmdError:           "Error notifications are ephemeral",
}

// replayPendingOperations retrieves and resends incomplete SC-initiated operations per SCACI §1
// "In the case of a connection loss and reestablishment... only operations, which had not been
// completed before the connection loss are reissued."
//
// Uses session context to preserve tenant/org isolation during replay.
func (s *Server) replayPendingOperations(conn net.Conn, session *Session) {
	if s.operationRepo == nil || session.ID <= 0 {
		return
	}

	// Use session context for tenant isolation
	ctx, cancel := context.WithTimeout(s.sessionContext(session), ReplayOperationTimeout)
	defer cancel()

	pendingOps, err := s.operationRepo.GetPendingOperations(ctx, session.ID)
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIGetPendingOpsFailed, "error", err)
		return
	}

	for _, op := range pendingOps {
		// Only replay SC-initiated (negative opId) outbound operations
		if op.OpId >= 0 || op.Direction != string(models.OperationDirectionOutbound) {
			continue
		}

		// Verify tenant matches session (cross-tenant protection)
		if op.TenantID != session.TenantID {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACICrossTenantReplayRejected,
				"opTenantID", op.TenantID, "sessionTenantID", session.TenantID)
			continue
		}

		s.logger.InfoContext(s.sessionContext(session), LogSCACIReplayingPendingOp,
			"opId", op.OpId, "command", op.Command)
		if err := s.replaySingleOperation(conn, session, op); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACIReplayOpFailed,
				"opId", op.OpId, "command", op.Command, "error", err)
		}
	}
}

// replaySingleOperation dispatches replay to command-specific handlers
func (s *Server) replaySingleOperation(conn net.Conn, session *Session, op *models.SCACIOperation) error {
	if !replayableCommands[op.Command] {
		if reason, known := nonReplayableReasons[op.Command]; known {
			s.logger.DebugContext(s.sessionContext(session), LogSCACISkipNonReplayable,
				"command", op.Command, "reason", reason)
		} else {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUnknownReplayCommand,
				"command", op.Command)
		}
		return nil
	}

	switch op.Command {
	case CmdULData:
		return s.replayULData(conn, session, op)
	case CmdDLDataResult:
		return s.replayDLDataResult(conn, session, op)
	case CmdEPStatus:
		return s.replayEPStatus(conn, session, op)
	default:
		return nil // Should not reach here due to replayableCommands check
	}
}

// replayULData reconstructs and resends ulData from stored RequestData
// Storage format: map[string]interface{} with epEui as hex string (FormatEUI64)
func (s *Server) replayULData(conn net.Conn, session *Session, op *models.SCACIOperation) error {
	data := op.RequestData

	// epEui stored as hex string via FormatEUI64 (see server.go:1129)
	epEuiStr, ok := data["epEui"].(string)
	if !ok || epEuiStr == "" {
		return fmt.Errorf("missing/invalid epEui in stored RequestData")
	}
	epEui, err := ParseEUI64(epEuiStr)
	if err != nil {
		return fmt.Errorf("parse epEui %q: %w", epEuiStr, err)
	}

	// Reconstruct ULData message with preserved opId
	session.WriteMu.Lock()
	defer session.WriteMu.Unlock()

	// Extract optional fields from stored data
	packetCnt, _ := extractInt64FromJSON(data, "packetCnt")
	dlOpen, _ := data["dlOpen"].(bool)
	responseExp, _ := data["responseExp"].(bool)
	dlAck, _ := data["dlAck"].(bool)

	// Reconstruct userData from hex-encoded storage
	var userData []byte
	if userDataStr, ok := data["userData"].(string); ok && userDataStr != "" {
		// userData is stored as hex-encoded string via encoding.EncodeUserData
		var err error
		userData, err = encoding.DecodeUserData(userDataStr)
		if err != nil {
			// Data corruption - log with context and abort replay per SCACI §1
			s.logger.ErrorContext(s.sessionContext(session), LogSCACIReplayUserDataCorrupted,
				"opId", op.OpId, "storedValue", userDataStr, "error", err)
			return fmt.Errorf("decode userData for replay: %w", err)
		}
	}

	// Reconstruct baseStations array if present
	var baseStations []mioty.BaseStationReception
	if bsData, ok := data["baseStations"].([]interface{}); ok {
		for _, bsItem := range bsData {
			if bsMap, ok := bsItem.(map[string]interface{}); ok {
				bs := mioty.BaseStationReception{}
				if bsEuiStr, ok := bsMap["bsEui"].(string); ok {
					bs.BsEui, _ = ParseEUI64(bsEuiStr)
				}
				bs.RxTime, _ = extractInt64FromJSON(bsMap, "rxTime")
				if snr, ok := bsMap["snr"].(float64); ok {
					bs.Snr = snr
				}
				if rssi, ok := bsMap["rssi"].(float64); ok {
					bs.Rssi = rssi
				}
				baseStations = append(baseStations, bs)
			}
		}
	}

	msg := ULData{
		BaseMessage: BaseMessage{
			Command: CmdULData,
			OpId:    op.OpId, // Preserve original opId for replay
		},
		EpEui:        epEui,
		BaseStations: baseStations,
		PacketCnt:    uint32(packetCnt), // #nosec G115 - SCACI §3.8 defines PacketCnt as 32-bit
		UserData:     userData,
		DlOpen:       dlOpen,
		ResponseExp:  responseExp,
		DlAck:        dlAck,
	}

	// Extract format if present (stored as float64 from JSON, needs to be uint8)
	if formatVal, ok := extractInt64FromJSON(data, "format"); ok {
		formatU8 := uint8(formatVal) // #nosec G115 - SCACI §3.8 defines Format as 8-bit
		msg.Format = &formatU8
	}

	return s.SendULData(conn, session, &msg)
}

// replayDLDataResult reconstructs and resends dlDataRes from stored RequestData
// Storage format: map[string]interface{} with epEui/queId/result fields
func (s *Server) replayDLDataResult(conn net.Conn, session *Session, op *models.SCACIOperation) error {
	data := op.RequestData

	epEuiStr, ok := data["epEui"].(string)
	if !ok || epEuiStr == "" {
		return fmt.Errorf("missing/invalid epEui in stored RequestData")
	}
	epEui, err := ParseEUI64(epEuiStr)
	if err != nil {
		return fmt.Errorf("parse epEui %q: %w", epEuiStr, err)
	}

	queId, _ := extractInt64FromJSON(data, "queId")
	result, _ := data["result"].(string)

	session.WriteMu.Lock()
	defer session.WriteMu.Unlock()

	msg := DLDataResult{
		BaseMessage: BaseMessage{
			Command: CmdDLDataResult,
			OpId:    op.OpId, // Preserve original opId for replay
		},
		EpEui:  epEui,
		QueID:  uint64(queId), // #nosec G115 - SCACI §3.12 defines QueID as 64-bit unsigned
		Result: result,
	}

	// Extract optional fields
	if txTime, ok := extractInt64FromJSON(data, "txTime"); ok {
		msg.TxTime = &txTime
	}
	if packetCnt, ok := extractInt64FromJSON(data, "packetCnt"); ok {
		packetCntU32 := uint32(packetCnt) // #nosec G115 - SCACI §3.12 defines PacketCnt as 32-bit
		msg.PacketCnt = &packetCntU32
	}
	if bsEuiStr, ok := data["bsEui"].(string); ok && bsEuiStr != "" {
		if bsEui, err := ParseEUI64(bsEuiStr); err == nil {
			msg.BsEui = &bsEui
		}
	}

	// Validate and send via helper (SCACI §3.12.1)
	return s.SendDLDataResult(conn, session, &msg)
}

// extractInt64FromJSON handles JSON numeric unmarshaling (float64 → int64)
// JSON standard unmarshals numbers as float64; this helper normalizes them.
func extractInt64FromJSON(data map[string]interface{}, key string) (int64, bool) {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case float64:
			return int64(v), true
		case int64:
			return v, true
		case int:
			return int64(v), true
		}
	}
	return 0, false
}

// replayEPStatus reconstructs and resends epStatus from stored RequestData
// Handles base64-encoded nonce/sign and validates before send via SendEPStatus
func (s *Server) replayEPStatus(conn net.Conn, session *Session, op *models.SCACIOperation) error {
	data := op.RequestData

	// epEui stored as hex string via FormatEUI64
	epEuiStr, ok := data["epEui"].(string)
	if !ok || epEuiStr == "" {
		s.logger.Error(LogSCACIReplayEPStatusInvalidData, "opId", op.OpId, "field", "epEui")
		return fmt.Errorf("missing/invalid epEui in stored RequestData")
	}
	epEui, err := ParseEUI64(epEuiStr)
	if err != nil {
		s.logger.Error(LogSCACIReplayEPStatusInvalidData, "opId", op.OpId, "field", "epEui", "error", err)
		return fmt.Errorf("parse epEui %q: %w", epEuiStr, err)
	}

	// epStatus stored as string enum
	epStatus, ok := data["epStatus"].(string)
	if !ok || epStatus == "" {
		s.logger.Error(LogSCACIReplayEPStatusInvalidData, "opId", op.OpId, "field", "epStatus")
		return fmt.Errorf("missing/invalid epStatus in stored RequestData")
	}

	session.WriteMu.Lock()
	defer session.WriteMu.Unlock()

	// Reconstruct EPStatus with preserved opId AND command
	msg := &EPStatus{
		BaseMessage: BaseMessage{
			Command: CmdEPStatus, // MUST set command explicitly
			OpId:    op.OpId,
		},
		EpEui:    epEui,
		EpStatus: epStatus,
	}

	// Restore optional OTA fields
	// attachCnt stored as float64 (JSON number)
	if attachCntF, ok := data["attachCnt"].(float64); ok {
		v := uint32(attachCntF)
		msg.AttachCnt = &v
	}

	// nonce/sign stored as base64 strings (via encoding.EncodeUserData)
	if nonceStr, ok := data["nonce"].(string); ok && nonceStr != "" {
		nonceBytes, err := encoding.DecodeUserData(nonceStr)
		if err == nil && len(nonceBytes) == 4 {
			var nonce mioty.Numeric4
			copy(nonce[:], nonceBytes)
			msg.Nonce = &nonce
		} else {
			s.logger.Warn(LogSCACIReplayEPStatusFieldDecodeErr, "opId", op.OpId, "field", "nonce")
		}
	}
	if signStr, ok := data["sign"].(string); ok && signStr != "" {
		signBytes, err := encoding.DecodeUserData(signStr)
		if err == nil && len(signBytes) == 4 {
			var sign mioty.Numeric4
			copy(sign[:], signBytes)
			msg.Sign = &sign
		} else {
			s.logger.Warn(LogSCACIReplayEPStatusFieldDecodeErr, "opId", op.OpId, "field", "sign")
		}
	}

	// snr/rssi/eqSnr stored as float64
	if snr, ok := data["snr"].(float64); ok {
		msg.Snr = &snr
	}
	if rssi, ok := data["rssi"].(float64); ok {
		msg.Rssi = &rssi
	}
	if eqSnr, ok := data["eqSnr"].(float64); ok {
		msg.EqSnr = &eqSnr
	}

	// Reconstruct subpackets if present (JSON array → mioty.Subpackets)
	if subpacketsRaw, ok := data["subpackets"]; ok && subpacketsRaw != nil {
		subpackets, err := reconstructSubpackets(subpacketsRaw)
		if err != nil {
			s.logger.Warn(LogSCACIReplayEPStatusFieldDecodeErr, "opId", op.OpId, "field", "subpackets", "error", err)
			// Continue without subpackets - non-fatal
		} else {
			msg.Subpackets = subpackets
		}
	}

	s.logger.InfoContext(s.sessionContext(session), LogSCACIReplayingEPStatus,
		"opId", op.OpId, "epEui", pkgmioty.FormatEUI64(epEui), "epStatus", epStatus)

	// Use SendEPStatus for validation and logging (NOT sendResponse)
	return s.SendEPStatus(conn, session, msg)
}

// reconstructSubpackets converts JSON-decoded subpackets back to mioty.Subpackets
// mioty.Subpackets is a struct with SNR, RSSI, Frequency, Phase slices
func reconstructSubpackets(raw interface{}) (*mioty.Subpackets, error) {
	// Handle JSON-decoded map[string]interface{} → mioty.Subpackets
	rawMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("subpackets not a map")
	}

	result := &mioty.Subpackets{}

	// Extract SNR slice
	if snrRaw, ok := rawMap["snr"].([]interface{}); ok {
		for _, v := range snrRaw {
			if f, ok := v.(float64); ok {
				result.SNR = append(result.SNR, f)
			}
		}
	}

	// Extract RSSI slice
	if rssiRaw, ok := rawMap["rssi"].([]interface{}); ok {
		for _, v := range rssiRaw {
			if f, ok := v.(float64); ok {
				result.RSSI = append(result.RSSI, f)
			}
		}
	}

	// Extract Frequency slice
	if freqRaw, ok := rawMap["frequency"].([]interface{}); ok {
		for _, v := range freqRaw {
			if f, ok := v.(float64); ok {
				result.Frequency = append(result.Frequency, int64(f))
			}
		}
	}

	// Extract Phase slice (optional)
	if phaseRaw, ok := rawMap["phase"].([]interface{}); ok {
		for _, v := range phaseRaw {
			if f, ok := v.(float64); ok {
				result.Phase = append(result.Phase, f)
			}
		}
	}

	return result, nil
}

// ============================================================================
// Assembly Validation Helpers (SCACI §§2.5, 3.12.1, 3.13.1)
// ============================================================================

// SendDLDataResult validates and sends a DLDataResult message.
// All DLDataResult senders MUST call this helper to ensure validation.
// DO NOT bypass this function—validation is mandatory per SCACI §3.12.1.
func (s *Server) SendDLDataResult(conn net.Conn, session *Session, msg *DLDataResult) error {
	if errToken := ValidateDLDataResult(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDLResultValidationFailed,
			"spec", "§3.12.1", "token", errToken, "opId", msg.OpId, "epEui", pkgmioty.FormatEUI64(msg.EpEui))
		return fmt.Errorf("DLDataResult validation failed: %s", errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendEPStatus validates and sends an EPStatus message.
// All future EP status notifiers MUST call this helper to ensure validation.
// DO NOT bypass this function—validation is mandatory per SCACI §§3.2, 3.13.1.
func (s *Server) SendEPStatus(conn net.Conn, session *Session, msg *EPStatus) error {
	if errToken := ValidateEPStatus(msg, msg.OpId); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIEPStatusValidationFailed,
			"spec", "§3.13.1", "token", errToken, "opId", msg.OpId, "epEui", pkgmioty.FormatEUI64(msg.EpEui))
		return fmt.Errorf("EPStatus validation failed: %s", errToken)
	}
	return s.sendResponse(conn, msg)
}

// ============================================================================
// Send Helpers with Validators (SCACI §§2.5 Assembly Compliance)
// ============================================================================

// SendConnectResponse validates and sends a ConnectResponse message.
// DO NOT bypass this function—validation is mandatory per SCACI §3.3.2.
//
// SCACI §3.3.2: version is OPTIONAL - defaults to negotiated version from session
// or ProtocolVersionString if session is nil/empty.
func (s *Server) SendConnectResponse(conn net.Conn, session *Session, msg *ConnectResponse) error {
	// Apply version default per SCACI §3.3.2 before validation
	if msg.Version == nil || *msg.Version == "" {
		defaultVer := ProtocolVersionString
		if session != nil && session.NegotiatedVersion != "" {
			defaultVer = session.NegotiatedVersion
		}
		msg.Version = &defaultVer
	}

	if errToken := ValidateConnectResponse(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIConnectRspValidationFailed,
			"spec", "§3.3.2", "token", errToken, "opId", msg.OpId)
		return fmt.Errorf("ConnectResponse validation failed: %s", errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendStatusResponse validates and sends a StatusResponse message.
// DO NOT bypass this function—validation is mandatory per SCACI §3.5.2.
func (s *Server) SendStatusResponse(conn net.Conn, session *Session, msg *StatusResponse) error {
	if errToken := ValidateStatusResponse(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIStatusRspValidationFailed,
			"spec", "§3.5.2", "token", errToken, "opId", msg.OpId)
		return fmt.Errorf("StatusResponse validation failed: %s", errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendULData validates and sends a ULData message.
// DO NOT bypass this function—validation is mandatory per SCACI §3.8.1.
func (s *Server) SendULData(conn net.Conn, session *Session, msg *ULData) error {
	if errToken := ValidateULData(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIULDataValidationFailed,
			"spec", "§3.8.1", "token", errToken, "opId", msg.OpId, "epEui", pkgmioty.FormatEUI64(msg.EpEui))
		return fmt.Errorf("ULData validation failed: %s", errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendError validates and sends an Error message.
// DO NOT bypass this function—validation is mandatory per SCACI §3.14.1.
func (s *Server) SendError(conn net.Conn, session *Session, msg *Error) error {
	if errToken := ValidateError(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIErrorMsgValidationFailed,
			"spec", "§3.14.1", "token", errToken, "opId", msg.OpId)
		return fmt.Errorf("Error message validation failed: %s", errToken)
	}
	return s.sendResponse(conn, msg)
}

// ============================================================================
// Send Helpers with Command Validation (BaseMessage-only responses)
// ============================================================================

// SendPingResponse validates command and sends a PingResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendPingResponse(conn net.Conn, session *Session, msg *PingResponse) error {
	if msg.Command != CmdPingResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			"spec", "§3.4.2", "expected", CmdPingResponse, "got", msg.Command, "opId", msg.OpId)
		return fmt.Errorf("PingResponse command mismatch: got %s, want %s", msg.Command, CmdPingResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendRegisterResponse validates command and sends a RegisterResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendRegisterResponse(conn net.Conn, session *Session, msg *RegisterResponse) error {
	if msg.Command != CmdRegisterResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			"spec", "§3.6.2", "expected", CmdRegisterResponse, "got", msg.Command, "opId", msg.OpId)
		return fmt.Errorf("RegisterResponse command mismatch: got %s, want %s", msg.Command, CmdRegisterResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendDeregisterResponse validates command and sends a DeregisterResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendDeregisterResponse(conn net.Conn, session *Session, msg *DeregisterResponse) error {
	if msg.Command != CmdDeregisterResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			"spec", "§3.7.2", "expected", CmdDeregisterResponse, "got", msg.Command, "opId", msg.OpId)
		return fmt.Errorf("DeregisterResponse command mismatch: got %s, want %s", msg.Command, CmdDeregisterResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendULDataComplete validates command and sends a ULDataComplete message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendULDataComplete(conn net.Conn, session *Session, msg *ULDataComplete) error {
	if msg.Command != CmdULDataComplete {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			"spec", "§3.8.3", "expected", CmdULDataComplete, "got", msg.Command, "opId", msg.OpId)
		return fmt.Errorf("ULDataComplete command mismatch: got %s, want %s", msg.Command, CmdULDataComplete)
	}
	return s.sendResponse(conn, msg)
}

// SendULDataTransmitResponse validates command and sends a ULDataTransmitResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
// Note: ULDataTransmitResponse is aliased to mioty.ULDataTransmitResponse which uses CommandType.
func (s *Server) SendULDataTransmitResponse(conn net.Conn, session *Session, msg *ULDataTransmitResponse) error {
	if msg.CommandType != CmdULDataTransmitResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			"spec", "§3.9.2", "expected", CmdULDataTransmitResponse, "got", msg.CommandType, "opId", msg.OpId)
		return fmt.Errorf("ULDataTransmitResponse command mismatch: got %s, want %s", msg.CommandType, CmdULDataTransmitResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendDLDataQueueResponse validates command and sends a DLDataQueueResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
// Note: DLDataQueueResponse is aliased to mioty.DLDataQueueResponse which uses CommandType.
func (s *Server) SendDLDataQueueResponse(conn net.Conn, session *Session, msg *DLDataQueueResponse) error {
	if msg.CommandType != CmdDLDataQueueResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			"spec", "§3.10.2", "expected", CmdDLDataQueueResponse, "got", msg.CommandType, "opId", msg.OpId)
		return fmt.Errorf("DLDataQueueResponse command mismatch: got %s, want %s", msg.CommandType, CmdDLDataQueueResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendDLDataRevokeResponse validates command and sends a DLDataRevokeResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendDLDataRevokeResponse(conn net.Conn, session *Session, msg *DLDataRevokeResponse) error {
	if msg.Command != CmdDLDataRevokeResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			"spec", "§3.11.2", "expected", CmdDLDataRevokeResponse, "got", msg.Command, "opId", msg.OpId)
		return fmt.Errorf("DLDataRevokeResponse command mismatch: got %s, want %s", msg.Command, CmdDLDataRevokeResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendDLDataResultComplete validates command and sends a DLDataResultComplete message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendDLDataResultComplete(conn net.Conn, session *Session, msg *DLDataResultComplete) error {
	if msg.Command != CmdDLDataResultComplete {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			"spec", "§3.12.3", "expected", CmdDLDataResultComplete, "got", msg.Command, "opId", msg.OpId)
		return fmt.Errorf("DLDataResultComplete command mismatch: got %s, want %s", msg.Command, CmdDLDataResultComplete)
	}
	return s.sendResponse(conn, msg)
}

// QueueDownlinkInternal provides gRPC/internal access to SCACI dlDataQue flow.
//
// This method creates a synthetic session for internal calls and delegates to
// processDLDataQueueCore - the same single-source logic used by socket handlers.
// This ensures consistent validation, persistence, operation recording, and
// scheduler coordination for both SCACI socket and gRPC paths.
//
// Parameters:
//   - ctx: Context with timeout (inherits from gRPC request)
//   - tenantID: Authenticated tenant ID from gRPC context
//   - orgID: Organization UUID for audit trail (may be nil)
//   - req: MIOTY DLDataQueue request from gRPC
//
// Returns:
//   - *DLDataQueueResult: Result containing queId, bsEui, opId, status
//   - error: Non-nil if operation failed
func (s *Server) QueueDownlinkInternal(
	ctx context.Context,
	tenantID int64,
	orgID *uuid.UUID,
	req *mioty.DLDataQueue,
) (*DLDataQueueResult, error) {
	// Create synthetic session for internal calls
	var orgUUID uuid.UUID
	if orgID != nil {
		orgUUID = *orgID
	}
	session := &Session{
		TenantID:       tenantID,
		OrganizationID: orgUUID,
		// ID is 0 - skip operation recording for internal paths (no persistent session)
		// conn is nil - marks this as internal call (no socket I/O)
	}

	// Allocate positive internal opId (matches AC semantics per §3.2)
	// For internal paths, we use time-based opId since there's no AC counter
	opId := time.Now().UnixNano() // Positive, unique per call

	// Convert mioty.DLDataQueue to internal DLDataQueue for processDLDataQueueCore
	dlReq := &DLDataQueue{
		EpEui:        req.EpEui,
		QueId:        req.QueId,
		UserData:     req.UserData,
		Prio:         req.Prio,
		CntDepend:    req.CntDepend,
		PacketCnt:    req.PacketCnt,
		Format:       req.Format,
		ResponseExp:  req.ResponseExp,
		ResponsePrio: req.ResponsePrio,
		DlWindReq:    req.DlWindReq,
		ExpOnly:      req.ExpOnly,
		DlRxStatQry:  req.DlRxStatQry,
	}

	// Delegate to single-source core logic (same path as socket handler)
	result, errToken, _ := s.processDLDataQueueCore(ctx, session, opId, dlReq)
	if errToken != "" {
		s.logger.WarnContext(ctx, LogSCACIDLDataQueueFailed,
			"opId", opId,
			"tenantId", tenantID,
			"epEui", pkgmioty.FormatEUI64(req.EpEui),
			"errorToken", errToken,
			"path", "internal")
		return nil, fmt.Errorf("SCACI dlDataQue failed: %s", errToken)
	}

	s.logger.InfoContext(ctx, LogSCACIDLDataQueueProcessed,
		"opId", opId,
		"tenantId", tenantID,
		"epEui", pkgmioty.FormatEUI64(req.EpEui),
		"queId", result.QueID,
		"bsEui", pkgmioty.FormatEUI64(result.BsEui),
		"path", "internal")

	return &DLDataQueueResult{
		QueID:  result.QueID,
		BsEui:  result.BsEui,
		OpID:   opId,
		Status: bssci.DLQueueStatusQueued,
	}, nil
}
