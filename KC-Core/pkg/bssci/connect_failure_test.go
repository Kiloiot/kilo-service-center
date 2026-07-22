package bssci

import (
	"context"
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingRegistrationConnSvc resolves the base station but fails the live
// connection registration, exercising the activation compensation path.
type failingRegistrationConnSvc struct{}

func (failingRegistrationConnSvc) GetBaseStationGlobal(_ context.Context, eui [8]byte) (*basestation.BaseStation, error) {
	return &basestation.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: "Test BS"}, nil
}

func (failingRegistrationConnSvc) RegisterConnection(_ context.Context, _ *Session, _ *basestation.BaseStation) error {
	return errors.New("registration failed")
}

func (failingRegistrationConnSvc) DisconnectBaseStationIfCurrent(_ context.Context, _ [8]byte, _ string) error {
	return nil
}

func (failingRegistrationConnSvc) UpdateLastSeen(_ context.Context, _ [8]byte) error { return nil }

// terminateSpySessionSvc records TerminateSession calls to prove the
// activation compensation runs.
type terminateSpySessionSvc struct {
	SessionService
	terminated int
}

func (t *terminateSpySessionSvc) TerminateSession(ctx context.Context, session *Session) error {
	t.terminated++
	return t.SessionService.TerminateSession(ctx, session)
}

// TestConnectResponseWriteFailureNoActivation: a conRsp write failure makes
// the connect operation Terminal without any partial activation - the session
// is never published to the live registry.
func TestConnectResponseWriteFailureNoActivation(t *testing.T) {
	log := newRecordingLogger()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	server := NewTestServer(log, storage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, failingRegistrationConnSvc{}, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{MessageEncoding: EncodingJSON, ServiceCenterEUI: TestBsEui02,
		Vendor: "v", Model: "m", Name: "n", SoftwareVersion: "1.0.0"}
	server.RegisterHandlers()

	conn := &bsscitest.TestConn{Encoding: EncodingJSON, FailWrites: true}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:           "conrsp-write-failure",
			Encoding:     EncodingJSON,
			ConnectState: ConnectStateAwaitingConnect,
		},
		Conn: conn,
	}

	payload := connectPayload("1.0.0", uint64(TestBsEui01))
	msg := &Message{Command: payload["command"].(string), OpId: 0, Data: payload}

	err := server.CallHandleConnect(session, msg, payload)

	require.Error(t, err, "a conRsp write failure must surface as a handler error (closing the connection)")
	assert.Equal(t, ConnectStateTerminal, session.ConnectState,
		"the connect operation is Terminal after a conRsp write failure")
	assert.Nil(t, server.GetSession(session.ID),
		"a session whose conRsp never went out must not be published to the live registry")
}

// TestActivationCompensationOnRegistrationFailure: when the live connection
// registration fails after the session row was persisted, the persisted
// session is compensated (terminated) and nothing is published to the live
// registries.
func TestActivationCompensationOnRegistrationFailure(t *testing.T) {
	log := newRecordingLogger()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	spy := &terminateSpySessionSvc{SessionService: sessionSvc}
	server := NewTestServer(log, storage, nil, 1,
		spy, downlinkSvc, statusSvc, failingRegistrationConnSvc{}, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{MessageEncoding: EncodingJSON}
	server.RegisterHandlers()

	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "activation-compensation",
			BaseStationEUI:   TestBsEui01,
			Encoding:         EncodingJSON,
			ResolvedTenantID: 1,
			ConnectState:     ConnectStateAwaitingConnectComplete,
		},
		Conn: conn,
		pendingBaseStation: &basestation.BaseStation{
			ID: 1, TenantID: 1, Name: "Test BS",
		},
	}

	data := map[string]interface{}{"command": "conCmp", "opId": int64(0)}
	msg := &Message{Command: "conCmp", OpId: 0, Data: data}

	err := server.CallHandleConnectComplete(session, msg, data)

	require.Error(t, err, "a registration failure after conCmp must close the connection")
	assert.Equal(t, 1, spy.terminated,
		"the just-persisted session must be compensated via TerminateSession")
	assert.Equal(t, ConnectStateTerminal, session.ConnectState)
	assert.Nil(t, server.GetSession(session.ID),
		"a session whose activation failed must not be published to the live registry")
}
