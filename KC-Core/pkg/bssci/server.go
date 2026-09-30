package bssci

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

const (
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
	// POSIX_EEXIST reports a packet counter reused inside the duplicate window with different content.
	POSIX_EEXIST = 17

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
	config     *Config
	logger     logger.Logger
	clock      clock.Clock
	listener   *nettransport.Listener
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	commands   *commandRegistry
	frames     nettransport.FrameCodec
	deps       Dependencies
	pendingOps *pendingOperationJournal
	events     *protocolEventRecorder
	tenantID   int64
	// nextOpID removed - now per-session via session.LastScOpId (BSSCI §5.2)

	// Active sessions (keep for protocol handling)
	sessions *sessionRegistry
	// mu guards the lifecycle flags (runtimeConfigured, started, stopped).
	mu sync.RWMutex

	// Injected services
	sessionSvc         SessionService
	versionNegotiator  VersionNegotiator
	downlinkSvc        DownlinkService
	statusSvc          StatusService
	connectionRegistry BaseStationConnectionRegistry
	queueSerializer    QueueSerializer // Downlink response frame builder
	auditLogger        AuditLogger     // Downlink audit event recorder
	tenantResolver     TenantResolver  // Queue-to-tenant mapping (replaces queueTenants map)
	sessionReconciler  AbandonedSessionReconciler
	stationEvents      StationEventRecorder // Base station activity events (pings)

	// Organization resolution
	orgResolver OrganizationDirectory // Organization UUID → tenant ID resolution
	// Certificate identity enforcement (strict mode): resolver maps the TLS
	// client certificate to a tenant/org identity at accept; the directory
	// reads registered station identity for connect-time enforcement
	certIdentityResolver CertificateIdentityResolver
	stationCertificates  StationCertificateBinder
	defaultTenantID      int64 // Community mode fallback tenant ID

	// Storage-boundary contracts (narrow, consumer-owned; satisfied
	// structurally by the KC-DB repositories)
	eventStore         EventStore
	basestationRepo    BaseStationStore
	endpointRepo       EndpointDirectory
	endpointOwners     EndpointOwnerResolver
	cipher             keycrypto.Cipher
	protocolMessages   ProtocolMessageStore
	dlrxStore          DLRXStatusStore
	bsStatusStore      BaseStationStatusStore
	downlinkQueueStore DownlinkQueueStore
	downlinkRevoke     DownlinkRevocationStore
	pendingDownlinks   PendingDownlinkLister
	servingStations    ServingStationLocator

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

	// sessionKeys yields the network session key an attach propagate carries
	sessionKeys NetworkSessionKeySource

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

	// Returns to pending the downlinks a base station no longer holds
	downlinkReclaimer DownlinkReclaimer

	// Decides an endpoint's attachment when an over-the-air attach or detach completes
	attachmentDecider AttachmentDecider

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

// ProtocolServices are the collaborators that implement BSSCI behaviour on
// top of the wire: session lifecycle, version negotiation, downlink and
// status flows, queue serialization and audit.
type ProtocolServices struct {
	Session            SessionService
	VersionNegotiator  VersionNegotiator
	Downlink           DownlinkService
	Status             StatusService
	ConnectionRegistry BaseStationConnectionRegistry
	QueueSerializer    QueueSerializer
	AuditLogger        AuditLogger
	TenantResolver     TenantResolver
	SessionReconciler  AbandonedSessionReconciler
	ServingStations    ServingStationLocator
	AttachmentDecider  AttachmentDecider
	StationEvents      StationEventRecorder
}

func (p ProtocolServices) missing() string {
	return firstMissing(
		required{depNameSessionService, p.Session == nil},
		required{depNameVersionNegotiator, p.VersionNegotiator == nil},
		required{depNameDownlinkService, p.Downlink == nil},
		required{depNameStatusService, p.Status == nil},
		required{depNameConnectionRegistry, p.ConnectionRegistry == nil},
		required{depNameQueueSerializer, p.QueueSerializer == nil},
		required{depNameAuditLogger, p.AuditLogger == nil},
		required{depNameTenantResolver, p.TenantResolver == nil},
		required{depNameSessionReconciler, p.SessionReconciler == nil},
		required{depNameServingStationLocator, p.ServingStations == nil},
		required{depNameAttachmentDecider, p.AttachmentDecider == nil},
		required{depNameStationEventRecorder, p.StationEvents == nil},
	)
}

// StorageContracts are the narrow persistence views the server writes and
// reads; each is satisfied directly by the matching KC-DB repository.
type StorageContracts struct {
	Events            EventStore
	BaseStations      BaseStationStore
	Endpoints         EndpointDirectory
	EndpointOwners    EndpointOwnerResolver
	AttachPersistence EndpointAttachmentPersistence
	SessionKeys       NetworkSessionKeySource
	ProtocolMessages  ProtocolMessageStore
	DLRXStatus        DLRXStatusStore
	BaseStationStatus BaseStationStatusStore
	DownlinkQueue     DownlinkQueueStore
	DownlinkRevoke    DownlinkRevocationStore
	PendingDownlinks  PendingDownlinkLister
}

func (c StorageContracts) missing() string {
	return firstMissing(
		required{depNameEventStore, c.Events == nil},
		required{depNameBaseStationStore, c.BaseStations == nil},
		required{depNameEndpointDirectory, c.Endpoints == nil},
		required{depNameEndpointOwnerResolver, c.EndpointOwners == nil},
		required{depNameAttachPersistence, c.AttachPersistence == nil},
		required{depNameSessionKeySource, c.SessionKeys == nil},
		required{depNameProtocolMessageStore, c.ProtocolMessages == nil},
		required{depNameDLRXStatusStore, c.DLRXStatus == nil},
		required{depNameBaseStationStatusStore, c.BaseStationStatus == nil},
		required{depNameDownlinkQueueStore, c.DownlinkQueue == nil},
		required{depNameDownlinkRevocationStore, c.DownlinkRevoke == nil},
		required{depNamePendingDownlinkLister, c.PendingDownlinks == nil},
	)
}

// IdentityResolvers map certificates and organizations onto tenants and
// protect key material at rest.
type IdentityResolvers struct {
	OrgDirectory        OrganizationDirectory
	CertIdentity        CertificateIdentityResolver
	StationCertificates StationCertificateBinder
	Cipher              keycrypto.Cipher
}

func (r IdentityResolvers) missing() string {
	return firstMissing(
		required{depNameOrganizationDirectory, r.OrgDirectory == nil},
		required{depNameCertificateIdentityResolver, r.CertIdentity == nil},
		required{depNameStationCertificateBinder, r.StationCertificates == nil},
	)
}

// IngestPipeline carries an uplink from the wire to storage and to the
// application centers.
type IngestPipeline struct {
	Uplink            UplinkIngestService
	Disposition       IngressDispositionResolver
	BlueprintDecoder  BlueprintDecoder
	BlueprintResolver BlueprintResolver
}

func (p IngestPipeline) missing() string {
	return firstMissing(
		required{depNameUplinkIngestService, p.Uplink == nil},
		required{depNameDispositionResolver, p.Disposition == nil},
		required{depNameBlueprintDecoder, p.BlueprintDecoder == nil},
		required{depNameBlueprintResolver, p.BlueprintResolver == nil},
	)
}

// FeatureCollaborators are optional: a nil entry disables the feature.
// DetachValidator is the retained BSSCI-5.7-01 signature seam.
type FeatureCollaborators struct {
	Roaming         RoamingService
	RelayOutbox     RelayOutboxWriter
	DetachValidator DetachSignatureValidator
	MQTT            MQTTEventPublisher
}

// Dependencies is everything the composition root wires into a server
// before ConfigureRuntime supplies the circular collaborators.
type Dependencies struct {
	// Clock drives sweeps, deadlines and last-seen stamps.
	Clock    clock.Clock
	Protocol ProtocolServices
	Storage  StorageContracts
	Identity IdentityResolvers
	Ingest   IngestPipeline
	Features FeatureCollaborators

	TenantID        int64
	DefaultTenantID int64
}

// missing names the first absent required dependency, or "" when complete.
func (d Dependencies) missing() string {
	if d.Clock == nil {
		return depNameClock
	}
	for _, group := range []interface{ missing() string }{d.Protocol, d.Storage, d.Identity, d.Ingest} {
		if name := group.missing(); name != "" {
			return name
		}
	}
	return ""
}

// required pairs a dependency name with whether it is absent.
type required struct {
	name   string
	absent bool
}

func firstMissing(reqs ...required) string {
	for _, r := range reqs {
		if r.absent {
			return r.name
		}
	}
	return ""
}

// RuntimeDependencies carries the collaborators that are constructed against
// the live *Server (circular) and therefore cannot be constructor arguments.
type RuntimeDependencies struct {
	Propagation        propagation.Service
	DownlinkDispatcher DownlinkDispatcher
	DownlinkReclaimer  DownlinkReclaimer
}

// NewServer creates a BSSCI server from its dependency set; call
// ConfigureRuntime before Start to supply the circular collaborators.
func NewServer(cfg *Config, log logger.Logger, deps Dependencies) (*Server, error) {
	if cfg == nil {
		return nil, errServerConfigRequired
	}
	// StatusService is mandatory (single-writer architecture)
	if deps.Protocol.Status == nil {
		return nil, errStatusSvcRequired
	}

	frames, err := nettransport.NewFrameCodec(mioty.MIOTYFrameIdentifier, dbconfig.MaxMessageSize, cfg.SocketWriteTimeout)
	if err != nil {
		return nil, fmt.Errorf(errFmtBuildFrameCodec, err)
	}

	commands, err := newCommandRegistry(commandTable, commandDirectionMap)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background()) // context-root: process

	s := &Server{
		config:   cfg,
		logger:   log,
		clock:    deps.Clock,
		deps:     deps,
		sessions: newSessionRegistry(),
		ctx:      ctx,
		cancel:   cancel,
		commands: commands,
		frames:   frames,
		tenantID: deps.TenantID,

		eventStore:         deps.Storage.Events,
		basestationRepo:    deps.Storage.BaseStations,
		endpointRepo:       deps.Storage.Endpoints,
		endpointOwners:     deps.Storage.EndpointOwners,
		cipher:             deps.Identity.Cipher,
		protocolMessages:   deps.Storage.ProtocolMessages,
		dlrxStore:          deps.Storage.DLRXStatus,
		bsStatusStore:      deps.Storage.BaseStationStatus,
		downlinkQueueStore: deps.Storage.DownlinkQueue,
		downlinkRevoke:     deps.Storage.DownlinkRevoke,
		pendingDownlinks:   deps.Storage.PendingDownlinks,
		servingStations:    deps.Protocol.ServingStations,
		attachPersistence:  deps.Storage.AttachPersistence,
		sessionKeys:        deps.Storage.SessionKeys,

		sessionSvc:         deps.Protocol.Session,
		versionNegotiator:  deps.Protocol.VersionNegotiator,
		downlinkSvc:        deps.Protocol.Downlink,
		statusSvc:          deps.Protocol.Status,
		connectionRegistry: deps.Protocol.ConnectionRegistry,
		queueSerializer:    deps.Protocol.QueueSerializer,
		auditLogger:        deps.Protocol.AuditLogger,
		tenantResolver:     deps.Protocol.TenantResolver,
		sessionReconciler:  deps.Protocol.SessionReconciler,
		attachmentDecider:  deps.Protocol.AttachmentDecider,
		stationEvents:      deps.Protocol.StationEvents,

		orgResolver:     deps.Identity.OrgDirectory,
		defaultTenantID: deps.DefaultTenantID,

		certIdentityResolver: deps.Identity.CertIdentity,
		stationCertificates:  deps.Identity.StationCertificates,
		uplinkIngestSvc:      deps.Ingest.Uplink,
		roamingSvc:           deps.Features.Roaming,
		dispositionResolver:  deps.Ingest.Disposition,
		relayOutbox:          deps.Features.RelayOutbox,
		detachValidator:      deps.Features.DetachValidator,
		mqttPublisher:        deps.Features.MQTT,
		blueprintDecoder:     deps.Ingest.BlueprintDecoder,
		blueprintResolver:    deps.Ingest.BlueprintResolver,
	}
	s.pendingOps = newPendingOperationJournal(s.statusSvc, s.sessionSvc, s.clock, s.logger)
	s.events = newProtocolEventRecorder(s.eventStore, s.clock, s.logger)

	// Register command handlers

	return s, nil
}

// ConfigureRuntime supplies the circular dependencies once, before Start:
// the propagation service and the downlink dispatcher and reclaimer are
// constructed against the live *Server and injected back here.
func (s *Server) ConfigureRuntime(deps RuntimeDependencies) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return errRuntimeReconfigureAfterStart
	}
	if s.runtimeConfigured {
		return errRuntimeAlreadyConfigured
	}
	if deps.Propagation == nil {
		return errPropagationServiceRequired
	}
	if deps.DownlinkDispatcher == nil {
		return errDownlinkDispatcherRequired
	}
	if deps.DownlinkReclaimer == nil {
		return errDownlinkReclaimerRequired
	}
	if s.config != nil && s.config.DetachSignatureValidationEnabled && s.detachValidator == nil {
		return errDetachValidatorNotWired
	}
	s.propagationSvc = deps.Propagation
	s.downlinkDispatcher = deps.DownlinkDispatcher
	s.downlinkReclaimer = deps.DownlinkReclaimer
	s.runtimeConfigured = true
	return nil
}

func (s *Server) validateRuntimeWiring() error {
	if name := s.deps.missing(); name != "" {
		return fmt.Errorf(errFmtDependencyRequired, name)
	}
	return nil
}

// Start starts the BSSCI server. If TLS certificates are not yet available
// (e.g., fresh deployment before certs are generated via UI), the listener
// is deferred and polls until the certificates appear. A validation failure
// leaves the server in its pre-Start state so the composition root can
// complete the wiring and try again; a successful Start is committed exactly
// once and cannot follow Stop.
func (s *Server) Start() error {
	// The composition root must supply the circular dependencies before the
	// server accepts traffic; an incompletely wired server refuses to start.
	s.mu.Lock()
	switch {
	case s.stopped:
		s.mu.Unlock()
		return errServerAlreadyStopped
	case s.started:
		s.mu.Unlock()
		return errServerAlreadyStarted
	case !s.runtimeConfigured:
		s.mu.Unlock()
		return errServerRuntimeNotConfigured
	}
	if err := s.validateRuntimeWiring(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf(errFmtServerWiringIncomplete, err)
	}
	s.started = true
	s.mu.Unlock()

	if err := s.sessionReconciler.ReconcileAbandonedSessions(s.ctx); err != nil {
		return fmt.Errorf(errFmtReconcileAbandonedSessions, err)
	}

	// BSSCI mandates TLS with mutual certificates and sets no minimum version;
	// the floor is protocol.bsci_tls.min_version and applies to every base station.
	s.listener = nettransport.NewListener(nettransport.ListenerConfig{
		Name:             listenerName,
		Addr:             s.config.ListenAddr,
		Files:            nettransport.TLSFiles{Cert: s.config.TLSCert, Key: s.config.TLSKey, CA: s.config.TLSCACert},
		TLS:              nettransport.TLSOptions{MinVersion: s.config.TLSMinVersion},
		CertPollInterval: s.config.CertificatePollInterval,
		// Background work starts only once the listener is committed, so an
		// immediate TLS failure leaves nothing running.
		OnListening: s.startDLRXQueryExpiryWorker,
	}, s.logger, s.serveConnection)
	if err := s.listener.Start(s.ctx); err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToStartTLS), err)
	}
	return nil
}

// Listening reports whether the BSSCI listener is bound and accepting.
func (s *Server) Listening() bool {
	return s.listener != nil && s.listener.Listening()
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
		s.listener.Stop()
	}
	s.wg.Wait()
	return nil
}

// teardownContext bounds a lost connection's persistence without inheriting
// the server context, so a session lost while the server stops is still
// recorded resumable.
func (s *Server) teardownContext(session *Session) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(s.sessionContext(session)), sessionTeardownTimeout)
}

// serveConnection is the listener's connection handler; the server's wait
// group tracks the connection alongside its other background work.
func (s *Server) serveConnection(conn net.Conn) {
	s.wg.Add(1)
	s.handleConnection(conn)
}

// closeConnection closes a connection that Stop or displacement may already
// have closed.
func (s *Server) closeConnection(conn net.Conn) {
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCloseConnection,
			logger.FieldRemote, conn.RemoteAddr().String(), logger.FieldError, err)
	}
}

// handleConnection handles a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer s.closeConnection(conn)
	// An active session reads without a deadline, so stopping the server
	// closes the connection to end a read on a quiet base station.
	stopClosing := context.AfterFunc(s.safeCtx(), func() { s.closeConnection(conn) })
	defer stopClosing()

	s.logger.InfoContext(s.safeCtx(), LogBSSCINewConnection, logger.FieldRemote, conn.RemoteAddr().String())

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               uuid.New().String(),
			ResolvedTenantID: s.defaultTenantID, // Initialize to server default
			// Encoding left empty - detected on first frame per BSSCI Section 1
		},
		Conn:      conn,
		Connected: s.clock.Now(),
		LastSeen:  s.clock.Now(),
	}

	// Extract TLS client certificate and resolve organization
	if tlsConn, ok := conn.(*tls.Conn); ok {
		// The handshake belongs to connection establishment and is abandoned when the server stops
		handshakeCtx, cancelHandshake := context.WithTimeout(s.safeCtx(), s.config.connectionEstablishmentTimeout())
		err := tlsConn.HandshakeContext(handshakeCtx)
		cancelHandshake()
		if err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCITLSHandshakeFailed,
				logger.FieldRemote, conn.RemoteAddr().String(), logger.FieldError, err)
			return
		}

		state := tlsConn.ConnectionState()
		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			session.ClientCert = cert

			// Try to resolve organization from certificate
			ctx := s.safeCtx()
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
							logger.FieldError, resolveErr,
							logger.FieldCertCN, cert.Subject.CommonName)
						return
					}
					// Community fallback: default tenant + its default org
					s.logger.WarnContext(ctx, LogBSSCICertOrgResolutionFailedUsingCommunityFallback,
						logger.FieldError, resolveErr,
						logger.FieldCertCN, cert.Subject.CommonName,
						logger.FieldTenantID, session.ResolvedTenantID)
					orgID = s.defaultOrgForSessionTenant(ctx, session)
				} else {
					session.ResolvedTenantID = identity.TenantID
					session.certSubjectEUI = identity.SubjectEUI
					orgID = identity.OrganizationID
					s.logger.InfoContext(ctx, LogBSSCICertOrgResolutionSucceeded,
						logger.FieldOrgID, orgID.String(),
						logger.FieldTenantID, identity.TenantID,
						logger.FieldCertCN, cert.Subject.CommonName)
				}
			case s.orgResolver != nil:
				orgID, tenantID, err = s.orgResolver.ResolveCert(ctx, cert)
				if err != nil {
					// Certificate-based resolution failed
					if s.config != nil && s.config.OrgEnforcementEnabled {
						s.logger.ErrorContext(ctx, LogBSSCICertIdentityRejectedStrictMode,
							logger.FieldError, err,
							logger.FieldCertCN, cert.Subject.CommonName)
						return
					}
					// Community fallback: default tenant + its default org
					s.logger.WarnContext(ctx, LogBSSCICertOrgResolutionFailedUsingCommunityFallback,
						logger.FieldError, err,
						logger.FieldCertCN, cert.Subject.CommonName,
						logger.FieldTenantID, session.ResolvedTenantID)
					orgID = s.defaultOrgForSessionTenant(ctx, session)
				} else {
					// SUCCESS: cert resolution worked, use resolved tenant
					session.ResolvedTenantID = tenantID
					s.logger.InfoContext(ctx, LogBSSCICertOrgResolutionSucceeded,
						logger.FieldOrgID, orgID.String(),
						logger.FieldTenantID, tenantID,
						logger.FieldCertCN, cert.Subject.CommonName)
				}
			default:
				// No resolver - community edition
				orgID = uuid.Nil
			}

			session.OrganizationID = orgID
			s.logger.DebugContext(ctx, LogBSSCISessionOrgTenantResolved,
				logger.FieldOrgID, orgID.String(),
				logger.FieldTenantID, session.ResolvedTenantID,
				logger.FieldCertCN, cert.Subject.CommonName)
		} else {
			// No peer certificate provided
			// ResolvedTenantID already set to s.defaultTenantID in session init
			// Try to get default org for the server's default tenant
			ctx := s.safeCtx()
			var orgID uuid.UUID
			if s.orgResolver != nil {
				var err error
				orgID, err = s.orgResolver.GetDefaultOrgForTenant(ctx, session.ResolvedTenantID)
				if err != nil {
					s.logger.WarnContext(ctx, LogBSSCINoPeerCertAndFailedToResolveDefaultOrgForBSSCISession,
						logger.FieldError, err,
						logger.FieldTenantID, session.ResolvedTenantID)
					orgID = uuid.Nil
				}
			}
			session.OrganizationID = orgID
			s.logger.DebugContext(ctx, LogBSSCISessionNoPeerCertUsingDefaults,
				logger.FieldOrgID, orgID.String(),
				logger.FieldTenantID, session.ResolvedTenantID)
		}
	}

	// Ensure we update status to offline when connection ends
	defer func() {
		session.ConnectState = ConnectStateTerminal

		// Stop status mechanism safely
		session.mu.Lock()
		if session.stopStatus != nil {
			close(session.stopStatus)
			session.stopStatus = nil // Prevent double-close
		}
		session.mu.Unlock()

		ctx, cancelTeardown := s.teardownContext(session)
		defer cancelTeardown()

		// Session-map cleanup runs unconditionally so rejected or provisional
		// connections never linger in the live maps
		wasLive := s.sessions.remove(session.ID)

		// Also remove from SessionService's sessionsByUUID map to prevent stale resume
		if s.sessionSvc != nil {
			s.sessionSvc.RemoveSession(session)
		}

		// Only the live connection took its base station online, so only it
		// takes it offline; a displaced or never-activated connection has no
		// status to retire (the store also guards by connection identity)
		if wasLive && s.connectionRegistry != nil {
			euiBytes := mioty.EUI64(session.BaseStationEUI).ToBytes()

			if err := s.connectionRegistry.DisconnectBaseStationIfCurrent(ctx, euiBytes, session.ID); err != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateOfflineStatus,
					logger.FieldEui, session.BaseStationEUI,
					logger.FieldError, err)
			} else {
				s.logger.InfoContext(ctx, LogBSSCIBaseStationDisconnectedStatusOffline,
					logger.FieldEui, session.BaseStationEUI,
					logger.FieldName, session.Name)
			}
		}

		// A row belongs to this connection only once conCmp created or claimed
		// it; losing the connection hands that row back resumable with its
		// final counters and its pending operations, which resume reissues
		// with their original IDs
		// (BSSCI §3.3 / §5.3). A connection that never claimed a row retires
		// nothing. The cache is swept in every case: it is keyed by the runtime
		// session ID that dies with this connection.
		if session.DbSessionID != 0 && s.sessionSvc != nil {
			if err := s.sessionSvc.UpdateSessionCounters(ctx, session); err != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateSessionCounters,
					logger.FieldError, err,
					logger.FieldSessionID, session.DbSessionID)
			}
			if err := s.sessionSvc.MarkDisconnected(ctx, session); err != nil {
				s.logger.ErrorContext(ctx, LogBSSCIFailedToMarkSessionDisconnected,
					logger.FieldError, err,
					logger.FieldSessionID, session.DbSessionID,
					logger.FieldEui, session.BaseStationEUI)
			}
		}
		if s.statusSvc != nil {
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
			deadline = s.clock.Now().Add(s.config.connectionEstablishmentTimeout())
		case ConnectStateAwaitingConnectComplete, ConnectStateAwaitingConnectErrorAck:
			deadline = s.clock.Now().Add(s.config.operationAckTimeout())
		default:
			// Complete/Terminal: no read deadline
		}
		if err := conn.SetReadDeadline(deadline); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSetReadDeadline, logger.FieldError, err)
			return
		}

		frame, err := s.frames.Read(conn)
		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
			case errors.Is(err, nettransport.ErrInvalidFrameIdentifier):
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIInvalidProtocolIdentifier, logger.FieldError, err)
			case errors.Is(err, nettransport.ErrPayloadTooLarge):
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIPayloadTooLarge, logger.FieldError, err)
			default:
				s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToReadFrame,
					logger.FieldRemote, conn.RemoteAddr().String(), logger.FieldError, err)
			}
			return
		}
		payload := frame.Payload

		// Detect and persist encoding on first message (BSSCI Section 1)
		if session.Encoding == "" {
			encoding := detectEncoding(payload, s.config.MessageEncoding)
			session.Encoding = encoding

			// Persist encoding to database if session has been persisted
			if session.DbSessionID > 0 {
				if err := s.sessionSvc.UpdateEncoding(s.safeCtx(), resolvedTenant(session, s.tenantID), session.DbSessionID, encoding); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistEncoding,
						logger.FieldEncoding, encoding,
						logger.FieldSessionID, session.DbSessionID,
						logger.FieldError, err)
					// Non-fatal: continue with in-memory encoding
				}
			}

			s.logger.InfoContext(s.safeCtx(), LogBSSCIDetectedMessageEncoding,
				logger.FieldEncoding, encoding,
				logger.FieldSessionID, session.DbSessionID)
		}

		// Decode message using detected/persisted encoding
		rawMsg, err := decodeMessage(payload, session.Encoding)
		if err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToDecodeMessagePack,
				logger.FieldEncoding, session.Encoding,
				logger.FieldError, err)
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
			logger.FieldRawMessage, rawMsg,
			logger.FieldCommand, command)

		// Extract opId - MIOTY spec requires connect operation to use ID 0
		var opId int64
		if opIdValue, exists := rawMsg["opId"]; exists {
			if val, ok := parseOpID(opIdValue); ok {
				opId = val
			} else {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIInvalidOpIDType, logger.FieldValue, opIdValue)
				catalogErr := NewCatalogError(errInvalidOperationID, POSIX_EPROTO)
				s.sendCatalogError(session, 0, catalogErr)
				return
			}
		} else {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIOpIDFieldNotFound, logger.FieldMessage, rawMsg)
			return
		}

		msg := &Message{
			Command:    command,
			OpId:       opId,
			Data:       rawMsg,
			RawPayload: payload, // Capture original wire bytes for forensic analysis
		}

		// BSSCI-3.2: operation IDs are sequenced by the command's role
		if errToken := s.sequenceOperation(session, command, opId); errToken != "" {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIOperationIDValidationFailed,
				logger.FieldCommand, command,
				logger.FieldOpID, opId,
				logger.FieldErrorTokenSnake, errToken)
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
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorResponse, logger.FieldError, err)
			}
			return
		}

		// Update last seen
		session.LastSeen = s.clock.Now()

		// Update last seen in database for connected basestations
		if session.BaseStationEUI != 0 && s.connectionRegistry != nil {
			euiBytes := mioty.EUI64(session.BaseStationEUI).ToBytes()

			// Update last seen time in database
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				ctx := s.sessionContext(session)
				if err := s.connectionRegistry.UpdateLastSeen(ctx, euiBytes); err != nil {
					s.logger.DebugContext(ctx, LogBSSCIFailedToUpdateLastSeen,
						logger.FieldEui, session.BaseStationEUI,
						logger.FieldError, err)
				}
			}()
		}

		if session.DbSessionID > 0 {
			s.persistSessionCounters(session)
		}

		// Handle message
		if err := s.handleMessage(session, msg, rawMsg); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToHandleMessage,
				logger.FieldCommand, command,
				logger.FieldError, err)
			return
		}
	}
}
