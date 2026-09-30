// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

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
	listener *nettransport.Listener
	codec    nettransport.FrameCodec

	// Session management
	clock clock.Clock
	// registry owns which connection serves which session and the sessions
	// held without one (SCACI §1).
	registry *SessionRegistry

	// Dependencies (narrow consumer-owned storage contracts)
	operationRepo OperationStore // Operation lifecycle tracking (three-way handshake - cross-cutting concern)

	// Domain services
	handshakeSvc HandshakeService // Connect/version negotiation/session resumption
	endpointSvc  EndpointService  // Register/deregister endpoint operations
	ulSvc        ULService        // UL transmit scheduling via BSSCI (§3.9)
	dlSvc        DLService        // DL queue/revoke operations via BSSCI (§3.10)
	statusSvc    StatusService    // Server status/uptime/BS statistics (§3.5)

	// Cross-cutting services
	sessionValidator   SessionValidator   // Connect message field validation (§3.3)
	operationRecorder  OperationRecorder  // Operation logging for resume safety (§3.4)
	sessionPersistence SessionPersistence // Session creation, heartbeat and operation ID persistence (§3.2-§3.4)
	errorRecorder      ErrorRecorder      // Error persistence and events (§3.14)
	sessionEvents      SessionEventStore  // Session lifecycle events (§1, §3.3)

	// Organization resolution
	orgResolver OrganizationDirectory // Default organization lookup for propagation context

	// BSSCI propagation (§5.8-5.8.3)
	sessionSnapshotProvider SessionSnapshotSource // Provides connected BS sessions
	propagationSvc          EndpointPropagator    // Orchestrates attach propagation

	// Sublayer handlers (SCACI §4)
	// sublayerHandlers maps sublayer commands (e.g., "rc.dir") to handlers
	// Community edition: empty map (no handlers)
	// Managed/cloud: can register handlers without transport changes
	sublayerHandlers map[string]SublayerHandler
	commands         *commandRegistry

	// Lifecycle
	ctx      context.Context
	cancel   context.CancelFunc
	shutdown chan struct{}
	wg       sync.WaitGroup

	// Detached persistence: session-state writes run on their own context and
	// task group so shutdown can drain them instead of aborting in-flight
	// database work when the main context is cancelled.
	persistCtx    context.Context
	persistCancel context.CancelFunc
	persistTasks  taskGroup
}

// Dependencies is everything the composition root wires into a server.
type Dependencies struct {
	Registry     *SessionRegistry
	Operations   OperationStore
	Handshake    HandshakeService
	Endpoints    EndpointService
	UL           ULService
	DL           DLService
	Status       StatusService
	Validator    SessionValidator
	Recorder     OperationRecorder
	Persistence  SessionPersistence
	OrgDirectory OrganizationDirectory
	Snapshots    SessionSnapshotSource
	Propagation  EndpointPropagator
	Errors       ErrorRecorder
	Clock        clock.Clock
	// SessionEvents files session opens, resumes, losses and refused connects.
	SessionEvents SessionEventStore
}

// NewServer creates a SCACI server; every dependency is required.
func NewServer(cfg *Config, log logger.Logger, deps Dependencies) (*Server, error) {
	if err := requireDependencies(cfg, log, deps); err != nil {
		return nil, err
	}
	commands, err := newCommandRegistry(commandTable)
	if err != nil {
		return nil, err
	}
	codec, err := newFrameCodec(cfg.SocketWriteTimeout)
	if err != nil {
		return nil, fmt.Errorf(errFmtScaciNewServerCause, err)
	}

	srvCtx, cancel := context.WithCancel(context.Background()) // context-root: process
	persistCtx, persistCancel := context.WithCancel(context.WithoutCancel(srvCtx))

	return &Server{
		ctx:                     srvCtx,
		cancel:                  cancel,
		persistCtx:              persistCtx,
		persistCancel:           persistCancel,
		config:                  cfg,
		logger:                  log,
		codec:                   codec,
		clock:                   deps.Clock,
		registry:                deps.Registry,
		operationRepo:           deps.Operations,
		handshakeSvc:            deps.Handshake,
		endpointSvc:             deps.Endpoints,
		ulSvc:                   deps.UL,
		dlSvc:                   deps.DL,
		statusSvc:               deps.Status,
		sessionValidator:        deps.Validator,
		operationRecorder:       deps.Recorder,
		sessionPersistence:      deps.Persistence,
		errorRecorder:           deps.Errors,
		sessionEvents:           deps.SessionEvents,
		orgResolver:             deps.OrgDirectory,
		sessionSnapshotProvider: deps.Snapshots,
		propagationSvc:          deps.Propagation,
		sublayerHandlers:        make(map[string]SublayerHandler), // §4: empty until an edition registers handlers
		commands:                commands,
		shutdown:                make(chan struct{}),
	}, nil
}

// requireDependencies refuses a missing configuration, logger or collaborator.
func requireDependencies(cfg *Config, log logger.Logger, deps Dependencies) error {
	presence := []struct {
		present bool
		message string
	}{
		{cfg != nil, depMsgCfgRequired},
		{log != nil, depMsgLoggerRequired},
		{deps.Registry != nil, depMsgSessionRegistryRequired},
		{deps.Operations != nil, depMsgOperationRepoRequired},
		{deps.Handshake != nil, depMsgHandshakeSvcRequired},
		{deps.Endpoints != nil, depMsgEndpointSvcRequired},
		{deps.UL != nil, depMsgULSvcRequired},
		{deps.DL != nil, depMsgDLSvcRequired},
		{deps.Status != nil, depMsgStatusSvcRequired},
		{deps.Validator != nil, depMsgSessionValidatorRequired},
		{deps.Recorder != nil, depMsgOperationRecorderRequired},
		{deps.Persistence != nil, depMsgSessionPersistenceRequired},
		{deps.OrgDirectory != nil, depMsgOrgResolverRequired},
		{deps.Snapshots != nil, depMsgSessionSnapshotProviderRequired},
		{deps.Propagation != nil, depMsgPropagationSvcRequired},
		{deps.Errors != nil, depMsgErrorRecorderRequired},
		{deps.Clock != nil, depMsgClockRequired},
		{deps.SessionEvents != nil, depMsgSessionEventsRequired},
	}
	for _, dep := range presence {
		if !dep.present {
			return fmt.Errorf(errFmtScaciNewServer, dep.message)
		}
	}
	if cfg.PlatformTenantID <= 0 {
		return fmt.Errorf(errFmtScaciNewServer, depMsgPlatformTenantRequired)
	}
	return nil
}

// safeCtx returns the server lifecycle context for pre-session and lifecycle
// logging and for detached work that must stop when the server stops. Stop()
// cancels it. A directly constructed Server (in tests) has no context yet, so a
// nil receiver context falls back to a plain background one.
func (s *Server) safeCtx() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background() // context-root: lifecycle-fallback
}

// persistContext returns the detached persistence context, falling back to
// the lifecycle context for test servers built without one.
func (s *Server) persistContext() context.Context {
	if s.persistCtx != nil {
		return s.persistCtx
	}
	return s.safeCtx()
}

// sessionContext creates a context enriched with SCACI session metadata for structured logging.
// The logger's extractContextFields will automatically inject tenant_id and organization_id.
// Rooted in the server lifecycle context so it stops with the server while carrying values.
// Nil-safe: a nil session yields the plain pre-session context.
func (s *Server) sessionContext(session *Session) context.Context {
	return withSessionValues(s.safeCtx(), session)
}

// withSessionValues decorates parent with the session's tenant and, when
// resolved, organization. A nil session leaves parent unchanged.
func withSessionValues(parent context.Context, session *Session) context.Context {
	if session == nil {
		return parent
	}
	ctx := pkgcontext.WithTenantID(parent, session.TenantID)
	if session.OrganizationID != uuid.Nil {
		ctx = pkgcontext.WithOrganizationID(ctx, session.OrganizationID)
	}
	return ctx
}

// runTracked runs background work a connection handler starts on the server
// wait group, so Stop waits for it. Only a running handler may call it: its
// own count keeps the wait group open.
func (s *Server) runTracked(task func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		task()
	}()
}

// persistDetached runs a session-state write detached from the connection:
// it outlives the connection, is bounded by timeout and is drained by Stop.
func (s *Server) persistDetached(session *Session, timeout time.Duration, write func(ctx context.Context)) {
	s.persistTasks.start(func() {
		ctx, cancel := context.WithTimeout(withSessionValues(s.persistContext(), session), timeout)
		defer cancel()
		write(ctx)
	})
}
