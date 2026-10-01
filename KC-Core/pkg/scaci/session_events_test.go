package scaci

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	refusalTenant       int64  = 31
	platformTenant      int64  = 1
	freshSessionID      int64  = 501
	undecodableConnect  byte   = 0xc1
	supersedingSCEui    uint64 = 0x70B3D59CD0000001
	supersedingUUIDByte byte   = 0x51
)

// sessionEventLog is the system event log the server files session events in.
type sessionEventLog struct {
	mu     sync.Mutex
	events []models.SCACISessionEvent
}

func (l *sessionEventLog) RecordSessionEvent(_ context.Context, event *models.SCACISessionEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, *event)
	return nil
}

func (l *sessionEventLog) filed() []models.SCACISessionEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]models.SCACISessionEvent(nil), l.events...)
}

func newSessionEventServer() (*Server, *sessionEventLog, *MockHandshakeService) {
	log := &sessionEventLog{}
	handshake := &MockHandshakeService{}
	server := newHeldTestServer(newHolderFake())
	server.sessionEvents = log
	server.handshakeSvc = handshake
	server.sessionValidator = &MockSessionValidator{}
	server.config.PlatformTenantID = platformTenant
	return server, log, handshake
}

func TestHandleConnectComplete_FilesTheOpenedSession(t *testing.T) {
	server, log, _ := newSessionEventServer()
	conn, _ := pipedTLS(t)
	fresh := &Session{ID: freshSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting}
	require.True(t, adopts(testutil.TestContext(), server.registry, conn, fresh))

	require.NoError(t, server.handleConnectComplete(conn, fresh, OpIDConnect))

	require.Len(t, log.filed(), 1)
	event := log.filed()[0]
	assert.Equal(t, models.EventTypeSCACISessionOpened, event.EventType)
	assert.Equal(t, heldTenant, event.TenantID)
	assert.Equal(t, freshSessionID, event.SessionID)
	assert.Equal(t, mioty.FormatEUI64(heldAcEui), event.AcEui)
	assert.Equal(t, conn.RemoteAddr().String(), event.RemoteAddr)
}

func TestHandleConnectComplete_FilesTheResumedSession(t *testing.T) {
	server, log, _ := newSessionEventServer()
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))
	conn, _ := pipedTLS(t)
	resumed := resumingSession()
	require.True(t, adopts(testutil.TestContext(), server.registry, conn, resumed))

	require.NoError(t, server.handleConnectComplete(conn, resumed, OpIDConnect))

	require.Len(t, log.filed(), 1)
	assert.Equal(t, models.EventTypeSCACISessionResumed, log.filed()[0].EventType)
	assert.Equal(t, heldSessionID, log.filed()[0].SessionID)
}

func TestHandleConnectComplete_ResumeNoLongerHeldIsFiledAsRefused(t *testing.T) {
	server, log, _ := newSessionEventServer()
	conn, _ := pipedTLS(t)

	_ = server.handleConnectComplete(conn, resumingSession(), OpIDConnect)

	require.Len(t, log.filed(), 1)
	event := log.filed()[0]
	assert.Equal(t, models.EventTypeSCACIConnectRefused, event.EventType)
	assert.Equal(t, heldTenant, event.TenantID, "the refusal belongs to the session's tenant")
	assert.Equal(t, heldSessionID, event.SessionID)
	assert.Equal(t, errNoActiveSession, event.ErrorToken)
}

func TestEndConnection_FilesTheLossOfALiveSessionOnly(t *testing.T) {
	server, log, _ := newSessionEventServer()
	live, connecting, unowned := &mockConn{}, &mockConn{}, &mockConn{}
	server.registry.sessions[live] = &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateActive}
	server.registry.sessions[connecting] = &Session{ID: freshSessionID, TenantID: heldTenant, AcEui: heldOtherAcEui, State: StateConnecting}

	server.endConnection(connecting)
	server.endConnection(unowned)
	assert.Empty(t, log.filed(), "neither a session that never went live nor a connection without one closes a session")

	server.endConnection(live)
	require.Len(t, log.filed(), 1)
	event := log.filed()[0]
	assert.Equal(t, models.EventTypeSCACISessionClosed, event.EventType)
	assert.Equal(t, heldTenant, event.TenantID)
	assert.Equal(t, heldSessionID, event.SessionID)
}

func TestHandleConnect_UndecodableConnectIsFiledUnderTheCertificatesTenant(t *testing.T) {
	server, log, handshake := newSessionEventServer()
	cert := stubCert("ac-under-test")
	handshake.On("CertificateTenant", mock.Anything, cert).Return(refusalTenant, true)
	var session *Session

	_ = server.handleConnect(&mockConn{}, &session, cert, OpIDConnect, []byte{undecodableConnect})

	require.Len(t, log.filed(), 1)
	event := log.filed()[0]
	assert.Equal(t, models.EventTypeSCACIConnectRefused, event.EventType)
	assert.Equal(t, refusalTenant, event.TenantID)
	assert.Equal(t, models.EventCategorySCACI, event.Category, "the tenant's administrators and endpoint managers read it")
	assert.Empty(t, event.AcEui, "no application center EUI was decoded")
	assert.Equal(t, errInvalidConnectFormat, event.ErrorToken)
	assert.Equal(t, GetErrorDefinition(errInvalidConnectFormat).Message, event.ErrorMessage)
	assert.Nil(t, session)
}

func TestHandleConnect_RefusedHandshakeIsFiledWithTheApplicationCenter(t *testing.T) {
	server, log, handshake := newSessionEventServer()
	cert := stubCert("ac-under-test")
	server.sessionValidator.(*MockSessionValidator).On("ValidateConnectFields", mock.Anything).Return("")
	handshake.On("ValidateConnect", mock.Anything, mock.Anything, cert).Return(nil, nil, errMajorVersionUnsupported)
	handshake.On("CertificateTenant", mock.Anything, cert).Return(refusalTenant, true)
	payload, err := msgpack.Marshal(&Connect{BaseMessage: BaseMessage{Command: CmdConnect}, Version: "2.0.0", AcEui: heldAcEui})
	require.NoError(t, err)
	var session *Session

	_ = server.handleConnect(&mockConn{}, &session, cert, OpIDConnect, payload)

	require.Len(t, log.filed(), 1)
	event := log.filed()[0]
	assert.Equal(t, refusalTenant, event.TenantID)
	assert.Equal(t, mioty.FormatEUI64(heldAcEui), event.AcEui)
	assert.Equal(t, errMajorVersionUnsupported, event.ErrorToken)
	assert.Zero(t, event.SessionID)
}

// connectAs runs a connect that the handshake answers with next.
func connectAs(t *testing.T, server *Server, handshake *MockHandshakeService, next *Session) {
	t.Helper()
	cert := stubCert("ac-under-test")
	server.sessionValidator.(*MockSessionValidator).On("ValidateConnectFields", mock.Anything).Return("")
	version := ProtocolVersionString
	response := &ConnectResponse{Version: &version, ScEui: supersedingSCEui, SnScUUID: UUID16{supersedingUUIDByte}}
	handshake.On("ValidateConnect", mock.Anything, mock.Anything, cert).Return(next, response, "")
	payload, err := msgpack.Marshal(&Connect{BaseMessage: BaseMessage{Command: CmdConnect}, Version: ProtocolVersionString, AcEui: next.AcEui})
	require.NoError(t, err)
	var session *Session
	require.NoError(t, server.handleConnect(&mockConn{}, &session, cert, OpIDConnect, payload))
	require.Same(t, next, session)
}

// A new session of the application center replaces its live session (SCACI
// §1): the replaced session is filed as closed, superseded, once.
func TestHandleConnect_ASupersededSessionIsFiledAsClosed(t *testing.T) {
	server, log, handshake := newSessionEventServer()
	oldConn := &mockConn{}
	server.registry.sessions[oldConn] = &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateActive}

	connectAs(t, server, handshake, &Session{ID: freshSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting})

	require.Len(t, log.filed(), 1)
	event := log.filed()[0]
	assert.Equal(t, models.EventTypeSCACISessionClosed, event.EventType)
	assert.Equal(t, models.SCACISessionClosedSuperseded, event.Reason)
	assert.Equal(t, heldSessionID, event.SessionID)
	assert.Equal(t, heldTenant, event.TenantID)

	server.endConnection(oldConn)
	assert.Len(t, log.filed(), 1, "the replaced connection ending files nothing more")
}

// A resume moves the session to a new connection: nothing closed.
func TestHandleConnect_AResumeSupersedesNoSession(t *testing.T) {
	server, log, handshake := newSessionEventServer()
	server.registry.sessions[&mockConn{}] = &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateActive}

	connectAs(t, server, handshake, &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting, Resumed: true})

	assert.Empty(t, log.filed())
}

func TestEndConnection_ALostConnectionIsFiledWithItsReason(t *testing.T) {
	server, log, _ := newSessionEventServer()
	live := &mockConn{}
	server.registry.sessions[live] = &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateActive}

	server.endConnection(live)

	require.Len(t, log.filed(), 1)
	assert.Equal(t, models.SCACISessionClosedConnectionLost, log.filed()[0].Reason)
}

func TestHandleConnect_RefusalOfNoTenantIsAServerLevelSecurityEvent(t *testing.T) {
	server, log, handshake := newSessionEventServer()
	cert := stubCert("unknown-ac")
	handshake.On("CertificateTenant", mock.Anything, cert).Return(int64(0), false)
	var session *Session

	_ = server.handleConnect(&mockConn{}, &session, cert, OpIDConnect, []byte{undecodableConnect})

	require.Len(t, log.filed(), 1)
	event := log.filed()[0]
	assert.Equal(t, platformTenant, event.TenantID, "filed under the platform tenant, never a guessed one")
	assert.Equal(t, models.EventCategorySecurity, event.Category, "administrators only")
}

func TestSessionEvents_ALiveSessionIsFiledInTheSCACICategory(t *testing.T) {
	server, log, _ := newSessionEventServer()
	live := &mockConn{}
	server.registry.sessions[live] = &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateActive}

	server.endConnection(live)

	require.Len(t, log.filed(), 1)
	assert.Equal(t, models.EventCategorySCACI, log.filed()[0].Category)
}

func TestNewServer_RequiresThePlatformTenant(t *testing.T) {
	_, err := newServerWithConfig(&Config{ListenAddr: ":5001"}, stubOrgDirectory{}, stubSnapshotSource{}, stubEndpointPropagator{},
		NewErrorRecorder(nil, nil, testLogger()), &sessionEventLog{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), depMsgPlatformTenantRequired)
}

type stubErrorRecorder struct{ ErrorRecorder }

func TestNewServer_NilSessionEvents(t *testing.T) {
	_, err := newServerThroughPersistence(stubOrgDirectory{}, stubSnapshotSource{}, stubEndpointPropagator{}, stubErrorRecorder{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), depMsgSessionEventsRequired)
}
