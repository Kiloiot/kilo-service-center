package bssci

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/google/uuid"
)

// testDBIDSeed offsets mock database IDs away from small fixture IDs.
const testDBIDSeed = 1000

// TestFrameWriteTimeout bounds the frame writes of test servers; tests of a
// stalled peer configure their own shorter bound.
const TestFrameWriteTimeout = 10 * time.Second

// testFrames is the BSSCI frame codec NewServer builds, for test servers
// assembled field by field.
var testFrames = func() nettransport.FrameCodec {
	codec, err := nettransport.NewFrameCodec(mioty.MIOTYFrameIdentifier, dbconfig.MaxMessageSize, TestFrameWriteTimeout)
	if err != nil {
		panic(err)
	}
	return codec
}()

// testKeyCipher is the deterministic key-material cipher every test server uses
// so the network-key encryption paths (attach propagate, session keys) run with
// a real cipher instead of a nil one.
var testKeyCipher = func() keycrypto.Cipher {
	c, err := keycrypto.NewCipher([]byte("kilocenter-test-master-key-32byt"))
	if err != nil {
		panic(err)
	}
	return c
}()

// Shared fixture values for server configuration and mock connections in tests.
const (
	detachSigValidationOn  = true
	detachSigValidationOff = false
	orgEnforcementOn       = true
	testMockLocalPort      = 5000
	testMockRemotePort     = 12345
	testAckTimeout         = 2 * time.Second
)

// NewTestServer creates a Server instance for testing with access to unexported fields.
// This helper lives in package bssci (not bssci_test) so it can set internal fields.
// Uses interfaces.Storage for dependency inversion. StatusService is mandatory (panics if nil).

// staticTestVersionNegotiator mirrors the production negotiator semantics for
// the supported set {mioty.MIOTYProtocolVersion}: same-major requests select
// the canonical service center version; other majors and lower minors return
// the production catalog errors.
type staticTestVersionNegotiator struct{}

func (staticTestVersionNegotiator) Negotiate(_ context.Context, requested string) (string, error) {
	reqMajor, reqMinor, _, cerr := ParseVersion(requested)
	if cerr != nil {
		return "", cerr
	}
	scMajor, scMinor, _, _ := ParseVersion(mioty.MIOTYProtocolVersion)
	if reqMajor != scMajor {
		return "", NewCatalogError(ErrUnsupportedMajorVersion, POSIX_EPROTO)
	}
	if reqMinor < scMinor {
		return "", NewCatalogError(ErrUnsupportedMinorVersion, POSIX_EPROTO)
	}
	return mioty.MIOTYProtocolVersion, nil
}

// testStore mirrors the accessor set the service package's CreateTestServices
// returns. Declared here rather than imported because internal/services/bssci
// imports this package; Go interfaces are structural, so the two match by shape.
type testStore interface {
	EndPoints() interfaces.EndpointRepository
	MIOTYMessages() interfaces.MIOTYMessageRepository
	DLRXStatus() interfaces.DLRXStatusRepository
	MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository
	MIOTYDownlinks() interfaces.MIOTYDownlinkRepository
}

// repositoriesTestStore adapts the eager repository set to the accessor shape
// the test server takes.
type repositoriesTestStore struct {
	repos *postgres.Repositories
}

func (r repositoriesTestStore) EndPoints() interfaces.EndpointRepository { return r.repos.Endpoints }

func (r repositoriesTestStore) MIOTYMessages() interfaces.MIOTYMessageRepository {
	return r.repos.Messages
}

func (r repositoriesTestStore) DLRXStatus() interfaces.DLRXStatusRepository {
	return r.repos.DLRXStatus
}

func (r repositoriesTestStore) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return r.repos.BaseStationStatus
}

func (r repositoriesTestStore) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository {
	return r.repos.Downlinks
}

// RepositoryTestStore exposes a live database to NewTestServer through the
// repository set.
func RepositoryTestStore(db *postgres.DB) testStore {
	return repositoriesTestStore{repos: postgres.NewRepositories(db)}
}

func NewTestServer(
	log logger.Logger,
	storage testStore,
	eventStore interfaces.SystemEventStore,
	tenantID int64,
	sessionSvc SessionService,
	downlinkSvc DownlinkService,
	statusSvc StatusService,
	connectionSvc BaseStationConnectionRegistry,
	_ SCACIBroadcaster,
	queueSerializer QueueSerializer,
	auditLogger AuditLogger,
	tenantResolver TenantResolver,
) *Server {
	// StatusService is required for all tests to persist pending operations
	if statusSvc == nil {
		panic("NewTestServer: statusSvc cannot be nil (use NewTestServerWithMemoryStatusService for automatic setup)")
	}

	s := &Server{
		clock:              clock.SystemClock{},
		logger:             log,
		eventStore:         eventStore,
		tenantID:           tenantID,
		sessionSvc:         sessionSvc,
		downlinkSvc:        downlinkSvc,
		statusSvc:          statusSvc,
		connectionRegistry: connectionSvc,
		queueSerializer:    queueSerializer,
		auditLogger:        auditLogger,
		tenantResolver:     tenantResolver,
		versionNegotiator:  staticTestVersionNegotiator{},
		cipher:             testKeyCipher,
		sessionKeys:        presharedSessionKeys{},
		commands:           mustTestCommandRegistry(),
		frames:             testFrames,
		sessions:           newSessionRegistry(),
		downlinkDispatcher: noopDownlinkDispatcher{},
		downlinkReclaimer:  noopDownlinkReclaimer{},
	}
	wireRequiredTestCollaborators(s)
	s.pendingOps = newPendingOperationJournal(s.statusSvc, s.sessionSvc, s.clock, s.logger)
	s.events = newProtocolEventRecorder(s.eventStore, s.clock, s.logger)
	// Fan the storage facade out into the narrow store views the Server
	// consumes (mirrors NewServer's wiring).
	s.SetStorageForTest(storage)
	// Initialize broadcast hook (tests can override)
	return s
}

// unregisteredStations is a base station store that knows no station, so
// events name stations by EUI.
type unregisteredStations struct{}

func (unregisteredStations) GetByEUI(context.Context, int64, []byte) (*models.BaseStation, error) {
	return nil, storage.ErrNotFound
}

func (unregisteredStations) Update(context.Context, int64, int64, map[string]interface{}) error {
	return storage.ErrNotFound
}

// wireRequiredTestCollaborators gives a test server the collaborators NewServer
// requires; tests override them through the setters.
func wireRequiredTestCollaborators(s *Server) {
	s.endpointOwners = repositoryEndpointOwners{server: s}
	s.orgResolver = unresolvedOrganizations{}
	s.attachmentDecider = &repositoryDecider{server: s, epStat: discardingEPStatusBroadcaster{}}
	s.protocolMessages = discardingProtocolMessages{}
	s.propagationSvc = noopPropagationService{}
	s.pendingDownlinks = noPendingDownlinks{}
	s.stationCertificates = anyStationCertificate{}
	s.basestationRepo = unregisteredStations{}
	s.stationEvents = discardingStationEvents{}
	if s.eventStore == nil {
		s.eventStore = discardingEvents{}
	}
}

// anyStationCertificate binds every connect to the station it claims, for
// tests that do not exercise the certificate binding.
type anyStationCertificate struct{}

func (anyStationCertificate) BindStationCertificate(context.Context, StationCertificateClaim) error {
	return nil
}

// noPendingDownlinks has no downlink waiting for a base station.
type noPendingDownlinks struct{}

func (noPendingDownlinks) ListPendingDownlinks(context.Context) ([]storage.PendingDownlink, error) {
	return nil, nil
}

// discardingStationEvents drops every base station activity event for tests
// that do not observe them.
type discardingStationEvents struct{}

func (discardingStationEvents) RecordEvent(context.Context, [8]byte, string, time.Time, map[string]interface{}) error {
	return nil
}

// discardingEvents drops every system event for tests that do not observe them.
type discardingEvents struct{}

func (discardingEvents) CreateEvent(context.Context, *models.SystemEvent) error { return nil }

// discardingProtocolMessages drops every protocol message row for tests that
// do not observe them.
type discardingProtocolMessages struct{}

func (discardingProtocolMessages) CreateDetachMessage(context.Context, *mioty.DetachMessage, map[string]interface{}) error {
	return nil
}

func (discardingProtocolMessages) CreateAttachPropagateMessage(context.Context, *mioty.AttachPropagateMessage) error {
	return nil
}

func (discardingProtocolMessages) CreateDetachPropagateMessage(context.Context, *mioty.DetachPropagateMessage) error {
	return nil
}

// repositoryEndpointOwners resolves owners through whichever endpoint
// repository the test server holds when the lookup runs.
type repositoryEndpointOwners struct {
	server *Server
}

func (o repositoryEndpointOwners) ResolveOwner(ctx context.Context, eui models.EUI) (EndpointOwner, error) {
	if o.server.endpointRepo == nil {
		return EndpointOwner{}, storage.ErrNotFound
	}
	endpoint, err := o.server.endpointRepo.Get(ctx, eui)
	if err != nil {
		return EndpointOwner{}, err
	}
	owner := endpoint.OwnerTenantID
	if owner == 0 {
		owner = endpoint.TenantID
	}
	return EndpointOwner{TenantID: owner, Endpoint: endpoint}, nil
}

// errNoCertificateOrganization is the certificate lookup of a test server
// whose organizations are unknown.
var errNoCertificateOrganization = errors.New("no organization is known for the certificate")

// unresolvedOrganizations knows no organization for any tenant or certificate.
type unresolvedOrganizations struct{}

func (unresolvedOrganizations) ResolveCert(context.Context, *x509.Certificate) (uuid.UUID, int64, error) {
	return uuid.Nil, 0, errNoCertificateOrganization
}

func (unresolvedOrganizations) GetDefaultOrgForTenant(context.Context, int64) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// epStatBroadcaster receives the epStat of an announced attachment decision.
type epStatBroadcaster interface {
	BroadcastEPStatus(ctx context.Context, tenantID int64, data *EPStatusData) error
}

// repositoryDecider records attachment decisions through the test server's
// endpoint repository and announces them the way the service center's
// decider does: once per change of the status, and for every over-the-air
// attach.
type repositoryDecider struct {
	server    *Server
	epStat    epStatBroadcaster
	mu        sync.Mutex
	decisions []AttachmentDecision
}

func (d *repositoryDecider) Decide(ctx context.Context, decision AttachmentDecision) (bool, error) {
	d.mu.Lock()
	d.decisions = append(d.decisions, decision)
	d.mu.Unlock()
	var changed bool
	var err error
	switch {
	case d.server.endpointRepo == nil:
		// A test server without endpoints records no status.
	case decision.Status == EndpointStatusAttached:
		changed, err = d.server.endpointRepo.TransitionEndpointStatus(ctx, decision.TenantID, decision.EndpointID, decision.Status)
	default:
		var telemetry *endpoint.DetachTelemetry
		if decision.OverTheAir != nil {
			telemetry = decision.OverTheAir.Telemetry
		}
		changed, err = endpoint.DetachEndpoint(ctx, d.server.endpointRepo, decision.TenantID, decision.EndpointID, telemetry)
	}
	if err != nil {
		return false, err
	}
	if changed || (decision.OverTheAir != nil && decision.Status == EndpointStatusAttached) {
		status := &EPStatusData{EpEui: decision.EpEUI, EpStatus: decision.Status}
		if decision.OverTheAir != nil && decision.OverTheAir.Status != nil {
			status = decision.OverTheAir.Status
		}
		_ = d.epStat.BroadcastEPStatus(ctx, decision.TenantID, status)
	}
	return changed, nil
}

// discardingEPStatusBroadcaster drops every epStat for tests that do not observe it.
type discardingEPStatusBroadcaster struct{}

func (discardingEPStatusBroadcaster) BroadcastEPStatus(context.Context, int64, *EPStatusData) error {
	return nil
}

// EndpointOwnedBy is an owner rule under which every endpoint is provisioned
// for one tenant.
type EndpointOwnedBy int64

func (t EndpointOwnedBy) ResolveOwner(_ context.Context, eui models.EUI) (EndpointOwner, error) {
	tenantID := int64(t)
	return EndpointOwner{TenantID: tenantID, Endpoint: &models.EndPoint{EUI: eui, TenantID: tenantID, OwnerTenantID: tenantID}}, nil
}

// SetEndpointOwnerResolver replaces the owner rule of a test server.
func (s *Server) SetEndpointOwnerResolver(r EndpointOwnerResolver) { s.endpointOwners = r }

// CallHandleDLRXStatus exposes the unexported handleDLRXStatus method for testing.
func (s *Server) CallHandleDLRXStatus(session *Session, msg *Message, data map[string]interface{}) error {
	// Mirror production dispatch: handleMessage normalizes the payload before
	// the handler runs, and the handler's type assertions rely on it.
	normalized, err := normalizePayload(s.sessionContext(session), s.logger, msg.Command, data)
	if err != nil {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, err.Error())
	}
	return s.handleDLRXStatus(session, msg, normalized)
}

// CallHandleDLDataResult exposes the unexported handleDLDataResult method for testing.
func (s *Server) CallHandleDLDataResult(session *Session, msg *Message, data map[string]interface{}) error {
	// Mirror production dispatch: handleMessage normalizes the payload before
	// the handler runs, and the handler's type assertions rely on it.
	normalized, err := normalizePayload(s.sessionContext(session), s.logger, msg.Command, data)
	if err != nil {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, err.Error())
	}
	return s.handleDLDataResult(session, msg, normalized)
}

// CallCommand dispatches one inbound command through the server's command
// table exactly as handleMessage would after the gates.
func (s *Server) CallCommand(command string, session *Session, msg *Message, data map[string]interface{}) error {
	handler, ok := s.commands.inboundHandler(command)
	if !ok {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errUnsupportedCommand), command)
	}
	return handler(s, session, msg, data)
}

// commandHandler exposes the table lookup to tests.
func (s *Server) commandHandler(command string) (HandlerFunc, bool) {
	return s.commands.inboundHandler(command)
}

// overrideHandlerForTest swaps a command's handler on this server only.
func (s *Server) overrideHandlerForTest(command string, handler HandlerFunc) {
	s.commands.override(command, handler)
}

// newTestCommandRegistry builds the production command table for direct
// registry assertions.
func newTestCommandRegistry(t *testing.T) *commandRegistry {
	t.Helper()
	registry, err := newCommandRegistry(commandTable, commandDirectionMap)
	require.NoError(t, err)
	return registry
}

// mustTestCommandRegistry is newTestCommandRegistry for scaffolding that has
// no testing.T at hand.
func mustTestCommandRegistry() *commandRegistry {
	registry, err := newCommandRegistry(commandTable, commandDirectionMap)
	if err != nil {
		panic(err)
	}
	return registry
}

// emptyTestCommandRegistry keeps the direction table but no handlers, so a
// test can register exactly the handlers it wants observed.
func emptyTestCommandRegistry() *commandRegistry {
	return &commandRegistry{byCommand: map[string]*CommandSpec{}, direction: commandDirectionMap}
}

// GetDownlinkSvc exposes the downlinkSvc field for testing.
func (s *Server) GetDownlinkSvc() DownlinkService {
	return s.downlinkSvc
}

// Handlers exposes the unexported handlers map for testing.
// Returns the map as interface{} to avoid exposing unexported types.
func (s *Server) Handlers() map[string]interface{} {
	result := make(map[string]interface{})
	for command, spec := range s.commands.byCommand {
		if spec.Handler != nil {
			result[command] = spec.Handler
		}
	}
	return result
}

// Int64Ptr returns a pointer to the given int64 value (helper for tests).
func Int64Ptr(v int64) *int64 {
	return &v
}

// Uint32Ptr returns a pointer to the given uint32 value (helper for tests).
func Uint32Ptr(v uint32) *uint32 {
	return &v
}

// CallSequenceOperation exposes the unexported sequenceOperation method for testing.
func (s *Server) CallSequenceOperation(session *Session, command string, opId int64) string {
	return s.sequenceOperation(session, command, opId)
}

// CallHandleConnect exposes the unexported handleConnect method for testing.
func (s *Server) CallHandleConnect(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleConnect(session, msg, data)
}

// CallHandleConnectComplete exposes the unexported handleConnectComplete method for testing.
func (s *Server) CallHandleConnectComplete(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleConnectComplete(session, msg, data)
}

// SetConfig sets the server config for testing.
func (s *Server) SetConfig(config *Config) {
	s.config = config
}

// SetEndpointRepository sets the endpoint repository dependency for testing.
func (s *Server) SetEndpointRepository(repo EndpointDirectory) {
	s.endpointRepo = repo
}

// SetConnectionManager is a no-op retained for test compatibility: the
// Server no longer holds the concrete connection manager (live-connection
// operations go through BaseStationConnectionRegistry).
func (s *Server) SetConnectionManager(_ *basestation.ConnectionManager) {}

// Test-only wiring setters mirroring the removed production setters: the
// production Server is wired exclusively through Dependencies and
// ConfigureRuntime.

func (s *Server) SetMQTTPublisher(pub MQTTEventPublisher) { s.mqttPublisher = pub }

func (s *Server) SetUplinkIngestService(svc UplinkIngestService) { s.uplinkIngestSvc = svc }

func (s *Server) SetDetachValidator(validator DetachSignatureValidator) {
	s.detachValidator = validator
}

// SetSCACIEPStatusBroadcaster records the epStat of every decision the test
// server's attachment decider announces.
func (s *Server) SetSCACIEPStatusBroadcaster(b epStatBroadcaster) {
	s.attachmentDecider = &repositoryDecider{server: s, epStat: b}
}

// SetAttachmentDecider replaces the attachment decider of a test server.
func (s *Server) SetAttachmentDecider(d AttachmentDecider) { s.attachmentDecider = d }

func (s *Server) SetCertificateIdentityResolver(r CertificateIdentityResolver) {
	s.certIdentityResolver = r
}

func (s *Server) SetDownlinkDispatcher(d DownlinkDispatcher) { s.downlinkDispatcher = d }

func (s *Server) SetDownlinkReclaimer(r DownlinkReclaimer) { s.downlinkReclaimer = r }

func (s *Server) SetPendingDownlinks(l PendingDownlinkLister) { s.pendingDownlinks = l }

func (s *Server) SetServingStations(l ServingStationLocator) { s.servingStations = l }

func (s *Server) SetPropagationService(svc propagation.Service) { s.propagationSvc = svc }

func (s *Server) SetRoamingService(svc RoamingService) { s.roamingSvc = svc }

func (s *Server) SetDispositionResolver(r IngressDispositionResolver) { s.dispositionResolver = r }

func (s *Server) SetRelayOutboxWriter(w RelayOutboxWriter) { s.relayOutbox = w }

func (s *Server) SetBlueprintDecoder(d BlueprintDecoder) { s.blueprintDecoder = d }

func (s *Server) SetBlueprintResolver(r BlueprintResolver) { s.blueprintResolver = r }

// CallHandleStatusResponse exposes the unexported handleStatusResponse method for testing.
func (s *Server) CallHandleStatusResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleStatusResponse(session, msg, data)
}

// LEGACY: Old PendingOperation helper removed after Issue #3 composite key implementation.
// Use s.pendingOps[makeSessionOpKey(session, opID)] directly in tests (requires session context).
// func (s *Server) PendingOperation(opID int64) *PendingOperation {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()
// 	if op, ok := s.pendingOps[opID]; ok {
// 		clone := *op
// 		return &clone
// 	}
// 	return nil
// }

// CallHandleAttachPropagateResponse exposes the unexported handleAttachPropagateResponse method for testing.
func (s *Server) CallHandleAttachPropagateResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleAttachPropagateResponse(session, msg, data)
}

// CallHandleDetachPropagateResponse exposes the unexported handleDetachPropagateResponse method for testing.
func (s *Server) CallHandleDetachPropagateResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDetachPropagateResponse(session, msg, data)
}

// CallHandlePingResponse exposes the unexported handlePingResponse method for testing.
func (s *Server) CallHandlePingResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handlePingResponse(session, msg, data)
}

// RegisterSession adds a session to the server's sessions map for testing.
func (s *Server) RegisterSession(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions.add(session)
}

// GetSession retrieves a session from the server's sessions map for testing.
// Returns nil if the session is not found.
// This helper ensures thread-safe read access and helps prevent future race conditions.
func (s *Server) GetSession(sessionID string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, _ := s.sessions.get(sessionID)
	return session
}

// CallHandleDLDataResultResponse exposes the unexported handleDLDataResultResponse method for testing.
func (s *Server) CallHandleDLDataResultResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDLDataResultResponse(session, msg, data)
}

// CallHandleDLRXStatusResponse exposes the unexported handleDLRXStatusResponse method for testing.
func (s *Server) CallHandleDLRXStatusResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDLRXStatusResponse(session, msg, data)
}

// CallHandleMessage exposes the unexported handleMessage method for testing.
// Used to test sublayer error handling per BSSCI §4-01.
func (s *Server) CallHandleMessage(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleMessage(session, msg, data)
}

// CallNormalizePayload exposes normalizePayload for tests.
func CallNormalizePayload(ctx context.Context, log logger.Logger, command string, data map[string]interface{}) (map[string]interface{}, error) {
	return normalizePayload(ctx, log, command, data)
}

// CallHandleAttach exposes the unexported handleAttach method for testing.
func (s *Server) CallHandleAttach(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleAttach(session, msg, data)
}

// CallHandleDetach exposes the unexported handleDetach method for testing.
func (s *Server) CallHandleDetach(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDetach(session, msg, data)
}

// CallHandleDetachComplete exposes the unexported handleDetachComplete method for testing.
func (s *Server) CallHandleDetachComplete(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDetachComplete(session, msg, data)
}

// CallHandleAttachComplete exposes the unexported handleAttachComplete method for testing.
func (s *Server) CallHandleAttachComplete(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleAttachComplete(session, msg, data)
}

// CallHandleULData exposes the unexported handleULData method for testing.
func (s *Server) CallHandleULData(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleULData(session, msg, data)
}

// ulData refusal tokens the external ingest tests assert on.
const (
	ErrTokenInvalidSubpackets = errInvalidSubpackets
	ErrTokenULUserDataTooLong = errULUserDataTooLong
)

// NewTestServerWithBaseStationRepo creates a Server instance for testing with a custom base station repository.
// This is useful for tests that need to verify tenant isolation by capturing repository call arguments.
// StatusService is mandatory (panics if nil).
func NewTestServerWithBaseStationRepo(
	log logger.Logger,
	storage testStore,
	basestationRepo interfaces.BaseStationRepository,
	tenantID int64,
	sessionSvc SessionService,
	downlinkSvc DownlinkService,
	statusSvc StatusService,
	connectionSvc BaseStationConnectionRegistry,
	_ SCACIBroadcaster,
	queueSerializer QueueSerializer,
	auditLogger AuditLogger,
	tenantResolver TenantResolver,
) *Server {
	// StatusService is required for all tests to persist pending operations
	if statusSvc == nil {
		panic("NewTestServerWithBaseStationRepo: statusSvc cannot be nil")
	}

	s := &Server{
		clock:              clock.SystemClock{},
		logger:             log,
		basestationRepo:    basestationRepo,
		tenantID:           tenantID,
		sessionSvc:         sessionSvc,
		downlinkSvc:        downlinkSvc,
		statusSvc:          statusSvc,
		connectionRegistry: connectionSvc,
		queueSerializer:    queueSerializer,
		auditLogger:        auditLogger,
		tenantResolver:     tenantResolver,
		versionNegotiator:  staticTestVersionNegotiator{},
		sessionKeys:        presharedSessionKeys{},
		frames:             testFrames,
		sessions:           newSessionRegistry(),
		downlinkDispatcher: noopDownlinkDispatcher{},
		downlinkReclaimer:  noopDownlinkReclaimer{},
	}
	wireRequiredTestCollaborators(s)
	s.SetStorageForTest(storage)
	s.basestationRepo = basestationRepo
	s.pendingOps = newPendingOperationJournal(s.statusSvc, s.sessionSvc, s.clock, s.logger)
	s.events = newProtocolEventRecorder(s.eventStore, s.clock, s.logger)
	return s
}

// NewTestServerWithMemoryStatusService creates a Server instance for testing with all services automatically wired.
// This is the recommended helper for most tests - it creates a complete service stack with memory-backed StatusService.
// Convenience wrapper that ensures StatusService is always present.
//
// Usage:
//
//	server := bssci.NewTestServerWithMemoryStatusService(logger, mockStorage, nil, 1)
func NewTestServerWithMemoryStatusService(
	log logger.Logger,
	storage testStore,
	eventStore interfaces.SystemEventStore,
	tenantID int64,
) *Server {
	// Use local CreateTestServices to avoid import cycle
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(log, eventStore)

	// Use provided storage if available, otherwise use mock
	if storage == nil {
		storage = mockStorage
	}

	return NewTestServer(
		log,
		storage,
		eventStore,
		tenantID,
		sessionSvc,
		downlinkSvc,
		statusSvc,
		connectionSvc,
		broadcaster,
		queueSerializer,
		auditLogger,
		tenantResolver,
	)
}

// Local test service implementations to avoid import cycle with internal/services/bssci

// mockSessionService implements SessionService for tests with stateful behavior
type mockSessionService struct {
	mu             sync.RWMutex
	sessions       map[string]*Session // Keyed by session ID
	sessionsByDBID map[int64]*Session  // Keyed by database session ID
	sessionsByUUID map[string]*Session // Keyed by uppercase hex(SessionUUID)
	lastPing       map[int64]time.Time // Keyed by database session ID
	dbIDCounter    int64               // Simulates DB auto-increment
	// terminatedRows and disconnectedRows record the session rows the server
	// retired or made resumable, for ownership assertions.
	terminatedRows   []int64
	disconnectedRows []int64
}

// newMockSessionService creates a new mock session service with state
func newMockSessionService() *mockSessionService {
	return &mockSessionService{
		sessions:       make(map[string]*Session),
		sessionsByDBID: make(map[int64]*Session),
		sessionsByUUID: make(map[string]*Session),
		lastPing:       make(map[int64]time.Time),
		dbIDCounter:    testDBIDSeed, // Start at 1000 for test IDs
	}
}

func (m *mockSessionService) HandleResume(_ context.Context, _ *Session, bsUUID []byte, bsOpId, scOpId *int64, _ uint64) ResumeOutcome {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Look up session by UUID using uppercase hex (matches production)
	key := strings.ToUpper(hex.EncodeToString(bsUUID))
	if existingSession, exists := m.sessionsByUUID[key]; exists {
		// Mirror production constraint semantics: absent constraints pass;
		// required BS operation ID beyond known state or claimed SC
		// operation ID beyond issued state is an inconsistent resume
		if bsOpId != nil && *bsOpId > existingSession.LastBsOpId {
			return ResumeOutcome{Disposition: ResumeInconsistent, Previous: existingSession, Err: ErrResumeCounterMismatch}
		}
		if scOpId != nil && *scOpId < existingSession.LastScOpId {
			return ResumeOutcome{Disposition: ResumeInconsistent, Previous: existingSession, Err: ErrResumeCounterMismatch}
		}
		return ResumeOutcome{Disposition: ResumeCompatible, Previous: existingSession}
	}
	return ResumeOutcome{Disposition: ResumeNoMatch} // No valid resume
}

func (m *mockSessionService) PersistSession(_ context.Context, session *Session, _ *basestation.BaseStation, isResume bool, connectInfo json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// A fresh session gets a new row; a resume claims the row stored under its
	// SC session UUID, as the production claim does
	if !isResume && session.DbSessionID == 0 {
		m.dbIDCounter++
		session.DbSessionID = m.dbIDCounter
	}
	if isResume && session.SessionUUID != nil {
		if prior, ok := m.sessionsByUUID[strings.ToUpper(hex.EncodeToString(session.SessionUUID))]; ok {
			session.DbSessionID = prior.DbSessionID
		}
	}

	// Persist ConnectInfo for resume validation
	if connectInfo != nil {
		session.ConnectInfo = connectInfo
	}

	// Store in sessions map by session ID
	if session.ID != "" {
		m.sessions[session.ID] = session
	}

	// Store by database ID for lookups
	if session.DbSessionID != 0 {
		m.sessionsByDBID[session.DbSessionID] = session
	}

	// Store by UUID if present (via StoreSessionByUUID which will be called separately)
	// The HandleResume test flow relies on explicit StoreSessionByUUID calls

	return nil
}

func (m *mockSessionService) StoreSessionByUUID(session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session.SessionUUID != nil {
		key := strings.ToUpper(hex.EncodeToString(session.SessionUUID))
		m.sessionsByUUID[key] = session
	}
}

func (m *mockSessionService) MarkHandshakeComplete(session *Session) {
	session.HandshakeComplete = true
}

// RemoveSession mirrors production: a closing connection evicts only its own
// entry, so a provisional connection never erases the resumable session it
// offered.
func (m *mockSessionService) RemoveSession(session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, session.ID)
	if session.DbSessionID != 0 {
		delete(m.sessionsByDBID, session.DbSessionID)
		delete(m.lastPing, session.DbSessionID)
	}
	if session.SessionUUID != nil {
		key := strings.ToUpper(hex.EncodeToString(session.SessionUUID))
		if stored, ok := m.sessionsByUUID[key]; ok && stored.ID == session.ID {
			delete(m.sessionsByUUID, key)
		}
	}
}

// forget drops a session from every index, as terminating its row ends its
// resumability regardless of which connection held it.
func (m *mockSessionService) forget(session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, session.ID)
	delete(m.sessionsByDBID, session.DbSessionID)
	delete(m.lastPing, session.DbSessionID)
	delete(m.sessionsByUUID, strings.ToUpper(hex.EncodeToString(session.SessionUUID)))
}

func (m *mockSessionService) MarkDisconnected(_ context.Context, session *Session) error {
	if session.DbSessionID == 0 {
		return errors.New(errFmtCannotMarkSessionDisconnectedNotPersisted)
	}
	m.mu.Lock()
	m.disconnectedRows = append(m.disconnectedRows, session.DbSessionID)
	m.mu.Unlock()
	return nil
}

func (m *mockSessionService) TerminateSession(_ context.Context, session *Session) error {
	m.mu.Lock()
	m.terminatedRows = append(m.terminatedRows, session.DbSessionID)
	m.mu.Unlock()
	m.forget(session)
	return nil
}

// retiredRows returns the rows TerminateSession and MarkDisconnected touched.
func (m *mockSessionService) retiredRows() (terminated, disconnected []int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]int64(nil), m.terminatedRows...), append([]int64(nil), m.disconnectedRows...)
}

func (m *mockSessionService) UpdateEncoding(_ context.Context, _ int64, sessionID int64, encoding string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find session by DB ID and update encoding
	for _, sess := range m.sessions {
		if sess.DbSessionID == sessionID {
			sess.Encoding = encoding
			return nil
		}
	}
	return nil
}

func (m *mockSessionService) UpdateSessionCounters(_ context.Context, _ *Session) error {
	// In a real implementation, this would persist to DB
	// For tests, the counters are already updated in the session object
	return nil
}

func (m *mockSessionService) UpdatePingTimestamp(_ context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if session.DbSessionID != 0 {
		m.lastPing[session.DbSessionID] = now
		// Also update the in-memory session if found
		if sess, exists := m.sessionsByDBID[session.DbSessionID]; exists {
			sess.LastSeen = now
		}
	}
	return nil
}

// GetLastPing returns the last ping timestamp for a session (for testing)
func (m *mockSessionService) GetLastPing(dbSessionID int64) (time.Time, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, exists := m.lastPing[dbSessionID]
	return t, exists
}

// memoryStatusService implements StatusService with in-memory storage for tests
type memoryStatusService struct {
	pendingOps *map[SessionOpKey]*PendingOperation
	mu         *sync.RWMutex
	logger     logger.Logger
	// recordErr, when set, is returned by Record* calls to exercise the
	// persist-failure abort path (no wire frame, counter gap only).
	recordErr error
	// metadataUpdates records UpdatePendingOperationMetadata calls so tests
	// can assert the persistence path now owned by the StatusService.
	metadataUpdates []StatusMetadataUpdate
	// removedOps records RemovePendingOperation calls (durable deletes) so
	// tests can assert irrecoverable rows are removed during resume.
	removedOps []int64
	// deletedSessions records the database sessions whose persisted rows
	// DeletePendingOperations removed, as the production DeleteBySession does.
	deletedSessions []int64
	// persistedRows backs PersistedOperations per database session for resume
	// tests that inject raw rows (including malformed ones) into the load path.
	persistedRows map[int64][]PersistedOperation
	// loadErr, when set, is returned by PersistedOperations.
	loadErr error
}

// StatusMetadataUpdate captures one UpdatePendingOperationMetadata call.
type StatusMetadataUpdate struct {
	SessionID int64
	OpId      int64
	Metadata  json.RawMessage
}

// StatusMetadataUpdates returns the recorded metadata persistence calls of the
// in-memory status service (nil when svc is a different implementation).
func StatusMetadataUpdates(svc StatusService) []StatusMetadataUpdate {
	if ms, ok := svc.(*memoryStatusService); ok {
		ms.mu.RLock()
		defer ms.mu.RUnlock()
		return append([]StatusMetadataUpdate(nil), ms.metadataUpdates...)
	}
	return nil
}

func (m *memoryStatusService) RecordPendingOperation(_ context.Context, session *Session, opId int64, op *PendingOperation, _ int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordErr != nil {
		return m.recordErr
	}
	key := makeSessionOpKey(session, opId)
	(*m.pendingOps)[key] = op
	return nil
}

func (m *memoryStatusService) RecordPendingOperations(_ context.Context, session *Session, ops []*PendingOperation, _ int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordErr != nil {
		return m.recordErr
	}
	for _, op := range ops {
		key := makeSessionOpKey(session, op.OperationID)
		(*m.pendingOps)[key] = op
	}
	return nil
}

func (m *memoryStatusService) RestorePendingOperation(session *Session, opId int64, op *PendingOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := makeSessionOpKey(session, opId)
	(*m.pendingOps)[key] = op
}

func (m *memoryStatusService) GetPendingOperation(session *Session, opId int64) (*PendingOperation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := makeSessionOpKey(session, opId)
	op, exists := (*m.pendingOps)[key]
	if !exists {
		return nil, errors.New(errFmtPendingOperationNotFound)
	}
	return op, nil
}

func (m *memoryStatusService) RemovePendingOperation(_ context.Context, session *Session, opId int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removedOps = append(m.removedOps, opId)
	key := makeSessionOpKey(session, opId)
	delete(*m.pendingOps, key)
	return nil
}

func (m *memoryStatusService) ExtractQueueMetadata(session *Session, opId int64) (endpointEUI uint64, queueID int64, tenantID string, organizationID *uuid.UUID) {
	op, err := m.GetPendingOperation(session, opId)
	if err != nil {
		return 0, 0, "", nil
	}
	if qid, ok := op.Metadata["queId"].(uint64); ok {
		queueID = int64(qid)
	}
	if tid, ok := op.Metadata["tenantId"].(string); ok {
		tenantID = tid
	}
	return
}

func (m *memoryStatusService) UpdatePendingOperationMetadata(_ context.Context, session *Session, opId int64, metadata map[string]interface{}, metadataJSON json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadataUpdates = append(m.metadataUpdates, StatusMetadataUpdate{
		SessionID: session.DbSessionID,
		OpId:      opId,
		Metadata:  append(json.RawMessage(nil), metadataJSON...),
	})
	key := makeSessionOpKey(session, opId)
	if op, ok := (*m.pendingOps)[key]; ok {
		op.Metadata = metadata
	}
	return nil
}

func (m *memoryStatusService) PersistedOperations(_ context.Context, dbSessionID int64) ([]PersistedOperation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return append([]PersistedOperation(nil), m.persistedRows[dbSessionID]...), nil
}

// persistRows stores raw pending-operation rows of a database session.
func (m *memoryStatusService) persistRows(dbSessionID int64, rows ...PersistedOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.persistedRows == nil {
		m.persistedRows = make(map[int64][]PersistedOperation)
	}
	m.persistedRows[dbSessionID] = append(m.persistedRows[dbSessionID], rows...)
}

// DeletePendingOperations mirrors production: the session's persisted rows
// are deleted and its cached operations evicted.
func (m *memoryStatusService) DeletePendingOperations(_ context.Context, session *Session) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedSessions = append(m.deletedSessions, session.DbSessionID)
	count := int64(len(m.persistedRows[session.DbSessionID]))
	delete(m.persistedRows, session.DbSessionID)
	for key := range *m.pendingOps {
		if key.SessionID == session.ID {
			delete(*m.pendingOps, key)
		}
	}
	return count, nil
}

// persistedRowsOf returns the persisted rows a database session still holds.
func (m *memoryStatusService) persistedRowsOf(dbSessionID int64) []PersistedOperation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]PersistedOperation(nil), m.persistedRows[dbSessionID]...)
}

// retiredPendingOperations returns the database sessions whose persisted
// rows were deleted.
func (m *memoryStatusService) retiredPendingOperations() []int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]int64(nil), m.deletedSessions...)
}

func (m *memoryStatusService) EvictCachedOperations(session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key := range *m.pendingOps {
		if key.SessionID == session.ID {
			delete(*m.pendingOps, key)
		}
	}
}

func (m *memoryStatusService) ProcessOperationStatusUpdate(_ context.Context, _ *Session, _ int64, _ string) error {
	return nil
}

// Canonical test mocks defined in test_mocks_test.go (same package, test-only compilation).
// Mocks remain in pkg/bssci to avoid import cycle (mocks implement bssci interfaces).
// The _test.go suffix ensures test-only compilation, following Go best practices.

// CreateTestServices creates all service dependencies for tests
// Minimal local implementation to avoid internal/services/bssci import cycle.
// Returns nil for most services - tests needing real implementations should provide their own.
func CreateTestServices(log logger.Logger, _ interfaces.SystemEventStore) (
	SessionService,
	DownlinkService,
	StatusService,
	BaseStationConnectionRegistry,
	SCACIBroadcaster,
	QueueSerializer,
	AuditLogger,
	TenantResolver,
	testStore,
) {
	// Create StatusService with in-memory map (most tests only need this)
	ops := make(map[SessionOpKey]*PendingOperation)
	var mu sync.RWMutex
	statusSvc := &memoryStatusService{
		pendingOps: &ops,
		mu:         &mu,
		logger:     log,
	}

	// Create stateful mock SessionService for VM tests
	sessionSvc := newMockSessionService()

	// Create canonical mock TenantResolver (from test_mocks_test.go)
	tenantResolver := NewMockTenantResolver(log)

	// Create canonical mock QueueSerializer (from test_mocks_test.go)
	queueSerializer := NewMockQueueSerializer()

	// Tests needing storage or other dependencies should inject their own implementations
	// This avoids ad-hoc fakes that break dependency boundaries
	return sessionSvc, nil, statusSvc, nil, nil, queueSerializer, noopAuditLogger{}, tenantResolver, nil
}

// SetStorageForTest fans a storage facade out into the narrow store views the
// Server consumes, mirroring NewServer's production wiring. Tests that inject
// a custom storage fake after construction must use this instead of assigning
// the storage field directly.
func (s *Server) SetStorageForTest(storage testStore) {
	// Only fakes that can open a transaction drive the attach persister; the
	// rest still get the narrow store views below.
	if txStore, ok := storage.(txStorage); ok {
		s.attachPersistence = &storageTxPersister{server: s, storage: txStore}
	}
	if storage == nil {
		return
	}
	// Nil accessor results never clobber a fake a test wired directly, which
	// matches the old direct-assignment semantics of server.storage.
	if repo := storage.EndPoints(); repo != nil {
		s.endpointRepo = repo
	}
	if repo := storage.MIOTYMessages(); repo != nil {
		s.protocolMessages = repo
	}
	if repo := storage.DLRXStatus(); repo != nil {
		s.dlrxStore = repo
	}
	if repo := storage.MIOTYBaseStationStatus(); repo != nil {
		s.bsStatusStore = repo
	}
	if repo := storage.MIOTYDownlinks(); repo != nil {
		if lookup, ok := repo.(DownlinkQueueStore); ok {
			s.downlinkQueueStore = lookup
		}
		if revoke, ok := repo.(DownlinkRevocationStore); ok {
			s.downlinkRevoke = revoke
		}
	}
}

// storageTxPersister mirrors the production attachment persister over the
// Server's live storage and base station repository fields so tests keep
// injecting transaction failures through their storage fakes.
// AttachTx is the transaction surface the test attach persister drives: the
// two repositories of an attach operation plus the lifecycle. Deliberately
// mirrors the narrow handle the production adapters use.
type AttachTx interface {
	EndPoints() interfaces.EndpointRepository
	EndPointSessions() interfaces.EndPointSessionRepository
	Commit() error
	Rollback() error
}

// txStorage is the facade plus the transaction entry point. Production code
// reaches transactions through the KC-DB adapters instead; this helper keeps
// the older shape because it exists to drive the server, not to model wiring.
type txStorage interface {
	testStore
	EndPointSessions() interfaces.EndPointSessionRepository
	BeginTx(ctx context.Context) (AttachTx, error)
}

// errAttachPersistenceUnavailable reports a test server built without storage.
var errAttachPersistenceUnavailable = errors.New("endpoint attachment persistence not configured")

type storageTxPersister struct {
	server  *Server
	storage txStorage
}

func (p *storageTxPersister) PersistAttachSession(ctx context.Context, rec AttachSessionRecord) error {
	st := p.storage
	if st == nil {
		return errAttachPersistenceUnavailable
	}
	tx, txErr := st.BeginTx(ctx)
	if txErr != nil {
		p.server.logger.ErrorContext(ctx, LogBSSCIFailedToBeginTransaction, logger.FieldError, txErr)
		return txErr
	}
	var commitErr error
	defer func() {
		if commitErr != nil {
			_ = tx.Rollback()
		}
	}()
	if err := tx.EndPoints().EndpointAttachmentStateUpdate(ctx, rec.TenantID, rec.EndpointID, rec.EndpointUpdates); err != nil {
		commitErr = err
		p.server.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateEndpointAttachMetadata, logger.FieldError, err)
		return err
	}
	activeSession, getErr := tx.EndPointSessions().GetActive(ctx, fmt.Sprintf("%d", rec.EndpointID))
	if getErr != nil {
		commitErr = getErr
		p.server.logger.ErrorContext(ctx, LogBSSCIFailedToLoadEndpointSession, logger.FieldError, getErr)
		return getErr
	}
	now := time.Now().UTC()
	var primaryBsID *int64
	if p.server.basestationRepo != nil {
		if bs, bsErr := p.server.basestationRepo.GetByEUI(ctx, rec.BSLookupTenantID, rec.BaseStationEUI); bsErr == nil && bs != nil {
			primaryBsID = &bs.ID
		}
	}
	if activeSession != nil {
		rec.ApplyToSession(activeSession, now, primaryBsID)
		if err := tx.EndPointSessions().Update(ctx, activeSession); err != nil {
			commitErr = err
			p.server.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateEndpointSession, logger.FieldError, err)
			return err
		}
	} else if err := tx.EndPointSessions().Create(ctx, rec.NewSession(now, primaryBsID)); err != nil {
		commitErr = err
		p.server.logger.ErrorContext(ctx, LogBSSCIFailedToCreateEndpointSession, logger.FieldError, err)
		return err
	}
	if err := tx.Commit(); err != nil {
		commitErr = err
		p.server.logger.ErrorContext(ctx, LogBSSCIFailedToCommitAttachTransaction, logger.FieldError, err)
		return err
	}
	commitErr = nil
	return nil
}

func (p *storageTxPersister) PersistAttachPropagateSession(ctx context.Context, rec AttachPropagateSessionRecord) error {
	st := p.storage
	if st == nil {
		return fmt.Errorf(errFmtBeginAttachPropagateTransaction, errAttachPersistenceUnavailable)
	}
	tx, txErr := st.BeginTx(ctx)
	if txErr != nil {
		return fmt.Errorf(errFmtBeginAttachPropagateTransaction, txErr)
	}
	var commitErr error
	defer func() {
		if commitErr != nil {
			_ = tx.Rollback()
		}
	}()
	if err := tx.EndPoints().EndpointAttachSessionUpdate(ctx, rec.TenantID, rec.EndpointID, rec.EndpointUpdates); err != nil {
		commitErr = err
		return fmt.Errorf(errFmtUpdateEndpoint, err)
	}
	activeSession, getErr := tx.EndPointSessions().GetActive(ctx, fmt.Sprintf("%d", rec.EndpointID))
	if getErr != nil {
		commitErr = getErr
		return fmt.Errorf(errFmtLoadEndpointSession, getErr)
	}
	now := time.Now().UTC()
	var primaryBsID *int64
	if p.server.basestationRepo != nil {
		if bs, bsErr := p.server.basestationRepo.GetByEUI(ctx, rec.TenantID, rec.BaseStationEUI); bsErr == nil && bs != nil {
			primaryBsID = &bs.ID
		}
	}
	if activeSession != nil {
		rec.ApplyToSession(activeSession, now, primaryBsID)
		if err := tx.EndPointSessions().Update(ctx, activeSession); err != nil {
			commitErr = err
			return fmt.Errorf(errFmtUpdateEndpointSession, err)
		}
	} else if err := tx.EndPointSessions().Create(ctx, rec.NewSession(now, primaryBsID)); err != nil {
		commitErr = err
		return fmt.Errorf(errFmtCreateEndpointSession, err)
	}
	if err := tx.Commit(); err != nil {
		commitErr = err
		return fmt.Errorf(errFmtCommitAttachPropagateTransaction, err)
	}
	commitErr = nil
	return nil
}

// makeSessionOpKey creates a composite key for pending operations map access.
// Use this helper instead of raw opID to prevent cross-session collisions (Issue #3).
func makeSessionOpKey(session *Session, opID int64) SessionOpKey {
	return SessionOpKey{
		SessionID:   session.ID,
		OperationID: opID,
	}
}

// Error format strings shared by this package's failure paths; verbs are filled at the point of failure.
const (
	errFmtCommitAttachPropagateTransaction          = "failed to commit attach propagate transaction: %w"
	errFmtCreateEndpointSession                     = "failed to create endpoint session: %w"
	errFmtUpdateEndpointSession                     = "failed to update endpoint session: %w"
	errFmtLoadEndpointSession                       = "failed to load endpoint session: %w"
	errFmtUpdateEndpoint                            = "failed to update endpoint: %w"
	errFmtBeginAttachPropagateTransaction           = "failed to begin attach propagate transaction: %w"
	errFmtPendingOperationNotFound                  = "pending operation not found"
	errFmtCannotMarkSessionDisconnectedNotPersisted = "cannot mark session disconnected: not persisted (DbSessionID=0)"
)

// add registers a session directly, bypassing displacement; test scaffolding only.
func (r *sessionRegistry) add(session *Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[session.ID] = session
}

// sessionRegistryWith builds a registry pre-populated with the given sessions.
func sessionRegistryWith(sessions ...*Session) *sessionRegistry {
	r := newSessionRegistry()
	for _, session := range sessions {
		r.add(session)
	}
	return r
}

// override replaces (or adds) a command's handler.
func (r *commandRegistry) override(command string, handler HandlerFunc) {
	if spec, ok := r.byCommand[command]; ok {
		spec.Handler = handler
		return
	}
	r.byCommand[command] = &CommandSpec{Command: command, Handler: handler}
}

// count reports the live-session total for assertions.
func (r *sessionRegistry) count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byID)
}

// noopAuditLogger discards audit events so handler tests can run the audit
// paths without an event store.
type noopAuditLogger struct{}

func (noopAuditLogger) RecordQueueAck(context.Context, string, *Session, uint64, int64, int64) error {
	return nil
}

func (noopAuditLogger) RecordDLRevokeResponse(context.Context, string, *Session, uint64, int64, int64) error {
	return nil
}

func (noopAuditLogger) RecordQueueRevoked(context.Context, *storage.DownlinkMessage) error {
	return nil
}
