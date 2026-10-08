package scaci

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	deliveryTestSessionID = int64(7)
	deliveryTestEpEui     = uint64(0x70B3D5677011150A)
	deliveryTestQueID     = uint64(99)
	deliveryTestAcEui     = uint64(0x70B3D59CD0000A0D)
)

var errDeliveryRecordFailed = errors.New("operation log unavailable")

// orderingRecorder notes, for every recorded operation, whether the
// connection was still empty: the operation must be on record before its
// request reaches the application center.
type orderingRecorder struct {
	mockOperationRepoStub
	conn *mockConn
	fail error

	mu          sync.Mutex
	beforeWrite []bool
}

func (r *orderingRecorder) note() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beforeWrite = append(r.beforeWrite, len(r.conn.written) == 0)
	return r.fail
}

func (r *orderingRecorder) Record(context.Context, *Session, int64, string, models.OperationDirection, map[string]interface{}) error {
	return r.note()
}

func (r *orderingRecorder) EnsureUplinkOperation(context.Context, *Session, int64, string, map[string]interface{}) (*models.SCACIOperation, bool, error) {
	return &models.SCACIOperation{}, true, r.note()
}

func (r *orderingRecorder) RecordOperation(context.Context, *models.SCACIOperationRequest) (*models.SCACIOperation, error) {
	return &models.SCACIOperation{}, r.note()
}

func (r *orderingRecorder) recordedBeforeWrite() []bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]bool(nil), r.beforeWrite...)
}

func newDeliveryServer(conn *mockConn, recorder *orderingRecorder, config *Config) *Server {
	session := activeBroadcastSession(broadcastULDataTestTenant)
	session.ID = deliveryTestSessionID
	session.AcEui = deliveryTestAcEui
	server := newBroadcastULDataServer(map[net.Conn]*Session{conn: session})
	server.config = config
	server.operationRecorder = recorder
	server.operationRepo = recorder
	counters := new(mockSessionRepository)
	counters.On("UpdateOperationIDs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.sessionPersistence = sessionRowsOver{repo: counters}
	return server
}

// scBroadcasts are the service-center operations fanned out to a tenant's
// application centers; each must be resumable once written (SCACI §1).
var scBroadcasts = []struct {
	command   string
	broadcast func(*Server) error
}{
	{command: CmdULData, broadcast: func(s *Server) error {
		return s.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, broadcastULDataFixture())
	}},
	{command: CmdDLDataResult, broadcast: func(s *Server) error {
		return s.BroadcastDLDataResult(testutil.TestContext(), ApplicationCenter{TenantID: broadcastULDataTestTenant, AcEui: deliveryTestAcEui}, deliveryTestQueID,
			&mioty.DLDataResult{EpEui: deliveryTestEpEui, QueId: deliveryTestQueID, Result: ResultExpired})
	}},
	{command: CmdEPStatus, broadcast: func(s *Server) error {
		return s.BroadcastEPStatus(testutil.TestContext(), broadcastULDataTestTenant,
			&EPStatusData{EpEui: deliveryTestEpEui, EpStatus: EPStatusDetached, Sign: &mioty.Numeric4{}})
	}},
}

func TestBroadcasts_RecordTheOperationBeforeWritingIt(t *testing.T) {
	for _, tc := range scBroadcasts {
		t.Run(tc.command, func(t *testing.T) {
			conn := &mockConn{}
			recorder := &orderingRecorder{conn: conn}
			server := newDeliveryServer(conn, recorder, &Config{})

			require.NoError(t, tc.broadcast(server))

			assert.Equal(t, []bool{true}, recorder.recordedBeforeWrite(), "the response may arrive as soon as the request is written")
			assert.NotEmpty(t, conn.written)
		})
	}
}

func TestBroadcasts_UnrecordableOperationIsNotDelivered(t *testing.T) {
	for _, tc := range scBroadcasts {
		t.Run(tc.command, func(t *testing.T) {
			conn := &mockConn{}
			recorder := &orderingRecorder{conn: conn, fail: errDeliveryRecordFailed}
			server := newDeliveryServer(conn, recorder, &Config{})

			err := tc.broadcast(server)

			require.Error(t, err, "a session whose operation cannot be recorded counts as a failed delivery")
			assert.ErrorIs(t, err, errDeliveryRecordFailed)
			assert.Empty(t, conn.written, "an operation a resume could not reissue is not sent")
		})
	}
}

// SCACI §3.8: telegrams carrying only control data are empty data, and
// userData "might be empty".
func TestBroadcastULData_DeliversEmptyUserData(t *testing.T) {
	for name, userData := range map[string][]byte{"empty": {}, "nil": nil} {
		t.Run(name, func(t *testing.T) {
			conn := &mockConn{}
			server := newBroadcastULDataServer(map[net.Conn]*Session{conn: activeBroadcastSession(broadcastULDataTestTenant)})
			uplink := broadcastULDataFixture()
			uplink.UserData = userData

			require.NoError(t, server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, uplink))

			var sent ULData
			require.NoError(t, decodeResponse(conn.written, &sent))
			assert.Equal(t, CmdULData, sent.Command)
			assert.NotNil(t, sent.UserData, "userData is mandatory on the wire")
			assert.Empty(t, sent.UserData)
		})
	}
}

func TestReplayULData_EmptyStoredUserData(t *testing.T) {
	conn := &mockConn{}
	server := newBroadcastULDataServer(nil)
	uplink := broadcastULDataFixture()
	op := &models.SCACIOperation{
		OpId:    -deliveryTestSessionID,
		Command: CmdULData,
		RequestData: map[string]interface{}{
			MetadataKeyEpEui: mioty.FormatEUI64(uplink.EpEui),
			"userData":       "",
			"baseStations": []interface{}{map[string]interface{}{
				"bsEui": mioty.FormatEUI64(uplink.BaseStations[0].BsEui), "rxTime": float64(uplink.BaseStations[0].RxTime),
			}},
		},
	}

	require.NoError(t, server.replayULData(conn, &Session{ID: deliveryTestSessionID}, op))

	var sent ULData
	require.NoError(t, decodeResponse(conn.written, &sent))
	assert.NotNil(t, sent.UserData)
	assert.Empty(t, sent.UserData)
}

// The keepalive's audit row must exist before its response can arrive, or
// the acknowledgement finds nothing and the ping is counted as missed.
func TestInitiatePing_RecordsBeforeWriting(t *testing.T) {
	conn := &mockConn{}
	recorder := &orderingRecorder{conn: conn}
	server := newDeliveryServer(conn, recorder, &Config{LogPingOperations: true})
	session := server.registry.sessions[conn]

	require.NoError(t, server.initiatePing(conn, session))

	assert.Equal(t, []bool{true}, recorder.recordedBeforeWrite())
	assert.NotEmpty(t, conn.written)
}

// Ping audit is optional; a keepalive still goes out when it cannot be audited.
func TestInitiatePing_AuditFailureStillSendsThePing(t *testing.T) {
	conn := &mockConn{}
	recorder := &orderingRecorder{conn: conn, fail: errDeliveryRecordFailed}
	server := newDeliveryServer(conn, recorder, &Config{LogPingOperations: true})

	require.NoError(t, server.initiatePing(conn, server.registry.sessions[conn]))

	var sent Ping
	require.NoError(t, decodeResponse(conn.written, &sent))
	assert.Equal(t, CmdPing, sent.Command)
}
