package bssci

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	pingTestTenant   = int64(1)
	pingTestOperator = "be8a02c9-1470-4314-8a86-48712761aca9"
	pingTestRspOpID  = int64(-17)
)

var errPingEventsDown = errors.New("station events unavailable")

// recordedStationEvent is one event the ping path handed the recorder, with
// the tenant and operator its context carried.
type recordedStationEvent struct {
	eui        [8]byte
	eventType  string
	occurredAt time.Time
	data       map[string]interface{}
	tenantID   int64
	userID     string
}

type stationEventLog struct {
	events []recordedStationEvent
	err    error
}

func (l *stationEventLog) RecordEvent(ctx context.Context, eui [8]byte, eventType string, occurredAt time.Time, data map[string]interface{}) error {
	tenantID, tenantErr := pkgcontext.GetTenantID(ctx)
	if tenantErr != nil {
		tenantID = 0
	}
	userID, userErr := pkgcontext.GetUserID(ctx)
	if userErr != nil {
		userID = ""
	}
	l.events = append(l.events, recordedStationEvent{eui, eventType, occurredAt, data, tenantID, userID})
	return l.err
}

// newPingServer is a server holding one live, connected session of TestBsEui01.
func newPingServer(t *testing.T, events *stationEventLog) (*Server, *Session, *bsscitest.TestConn) {
	t.Helper()
	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	server := NewTestServer(log, storage, nil, pingTestTenant,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.stationEvents = events
	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "ping-session",
			BaseStationEUI:    uint64(TestBsEui01),
			Encoding:          EncodingJSON,
			LastScOpId:        -1,
			HandshakeComplete: true,
			DbSessionID:       1,
		},
		Conn: conn,
	}
	server.RegisterSession(session)
	return server, session, conn
}

// The operator's ping is recorded for the station, its tenant and the
// operator, with the opId the frame carried.
func TestInitiatePing_RecordsPingSent(t *testing.T) {
	events := &stationEventLog{}
	server, _, conn := newPingServer(t, events)
	ctx := pkgcontext.WithUserID(testutil.TestContext(), pingTestOperator)

	opID, err := server.InitiatePing(ctx, uint64(TestBsEui01), pingTestTenant)

	require.NoError(t, err)
	require.True(t, conn.SeenCommand(mioty.CmdPing), "the ping frame is written")
	require.Len(t, events.events, 1)
	sent := events.events[0]
	assert.Equal(t, models.EventTypeBaseStationPingSent, sent.eventType)
	assert.Equal(t, mioty.EUI64(uint64(TestBsEui01)).ToBytes(), sent.eui)
	assert.Equal(t, opID, sent.data[models.EventDetailKeyOpID])
	assert.Equal(t, pingTestTenant, sent.tenantID)
	assert.Equal(t, pingTestOperator, sent.userID)
}

// A station's pingRsp is recorded against the station with its opId and the
// result it reported, and the handshake still completes.
func TestHandlePingResponse_RecordsPingAnswered(t *testing.T) {
	events := &stationEventLog{}
	server, session, conn := newPingServer(t, events)
	data := map[string]interface{}{"result": float64(0)}

	require.NoError(t, server.handlePingResponse(session, &Message{Command: mioty.CmdPingResponse, OpId: pingTestRspOpID}, data))

	assert.True(t, conn.SeenCommand(mioty.CmdPingComplete), "pingCmp completes the handshake")
	require.Len(t, events.events, 1)
	answered := events.events[0]
	assert.Equal(t, models.EventTypeBaseStationPingAnswered, answered.eventType)
	assert.Equal(t, mioty.EUI64(uint64(TestBsEui01)).ToBytes(), answered.eui)
	assert.Equal(t, pingTestRspOpID, answered.data[models.EventDetailKeyOpID])
	assert.Equal(t, int64(0), answered.data[models.EventDetailKeyResult])
	assert.Equal(t, pingTestTenant, answered.tenantID)
}

// A pingRsp without a result (BSSCI §5.4.2 defines none) records no result.
func TestHandlePingResponse_OmitsAbsentResult(t *testing.T) {
	events := &stationEventLog{}
	server, session, _ := newPingServer(t, events)

	require.NoError(t, server.handlePingResponse(session, &Message{Command: mioty.CmdPingResponse, OpId: pingTestRspOpID}, map[string]interface{}{}))

	require.Len(t, events.events, 1)
	assert.NotContains(t, events.events[0].data, models.EventDetailKeyResult)
}

// The activity record never decides the ping: a recorder failure neither
// fails the operator's ping nor the station's handshake.
func TestPing_RecorderFailureDoesNotFailThePing(t *testing.T) {
	events := &stationEventLog{err: errPingEventsDown}
	server, session, conn := newPingServer(t, events)

	opID, err := server.InitiatePing(testutil.TestContext(), uint64(TestBsEui01), pingTestTenant)
	require.NoError(t, err)
	assert.Less(t, opID, int64(0), "the SC-assigned opId is returned")

	require.NoError(t, server.handlePingResponse(session, &Message{Command: mioty.CmdPingResponse, OpId: opID}, map[string]interface{}{}))
	assert.True(t, conn.SeenCommand(mioty.CmdPingComplete))
	assert.Len(t, events.events, 2, "both records were attempted")
}
