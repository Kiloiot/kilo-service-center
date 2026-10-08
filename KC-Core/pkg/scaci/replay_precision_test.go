package scaci

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// Values above 2^53 that a float64 cannot represent exactly.
const (
	precisionQueID uint64 = 1<<53 + 1
	// beyondSignedQueID is 2^63 + 5, an Application Center queue id BIGINT cannot hold.
	beyondSignedQueID uint64 = 1<<63 + 5
	precisionTxTime   int64  = 1_700_000_000_123_456_789
	precisionRxTime   int64  = 1_700_000_000_987_654_321
	precisionTenant   int64  = 7

	precisionInternalQueID int64 = 1<<53 + 3
)

// recordedOperation is one operation a broadcast recorded before sending it.
type recordedOperation struct {
	opID        int64
	command     string
	requestData map[string]interface{}
}

// operationLogCapture keeps what the broadcast records so the test can
// hand it back to replay the way the JSONB operation log does.
type operationLogCapture struct {
	recorded []recordedOperation
}

func (r *operationLogCapture) Record(_ context.Context, _ *Session, opId int64, command string, _ models.OperationDirection, data map[string]interface{}) error {
	r.recorded = append(r.recorded, recordedOperation{opID: opId, command: command, requestData: data})
	return nil
}

func (r *operationLogCapture) EnsureUplinkOperation(_ context.Context, _ *Session, opId int64, sourceMessageID string, data map[string]interface{}) (*models.SCACIOperation, bool, error) {
	stored := map[string]interface{}{models.OperationRequestKeySourceMessageID: sourceMessageID}
	for key, value := range data {
		stored[key] = value
	}
	r.recorded = append(r.recorded, recordedOperation{opID: opId, command: CmdULData, requestData: stored})
	return &models.SCACIOperation{OpId: opId}, true, nil
}

// storedOperation returns the recorded operation as it reads back from the
// JSONB operation log: marshalled and decoded into a generic map.
func (r *operationLogCapture) storedOperation(t *testing.T) *models.SCACIOperation {
	t.Helper()
	require.Len(t, r.recorded, 1)
	rec := r.recorded[0]
	encoded, err := json.Marshal(rec.requestData)
	require.NoError(t, err)
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	return &models.SCACIOperation{
		OpId:        rec.opID,
		TenantID:    precisionTenant,
		Command:     rec.command,
		Direction:   string(models.OperationDirectionOutbound),
		RequestData: decoded,
	}
}

// acceptingCounterStore persists session state without error.
type acceptingCounterStore struct{}

func (acceptingCounterStore) UpdateOperationIDs(_ context.Context, _, _, _, _ int64) error {
	return nil
}

func (acceptingCounterStore) MarkSessionDisconnected(_ context.Context, _, _ int64) error {
	return nil
}

func precisionServer(repo *operationLogCapture) (*Server, *Session) {
	session := &Session{ID: 55, TenantID: precisionTenant, State: StateActive, AcEui: 0x0102030405060708}
	server := &Server{
		registry:           newTestRegistry(map[net.Conn]*Session{&mockConn{}: session}, newHolderFake()),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		logger:             testLogger(),
		operationRecorder:  repo,
		sessionPersistence: sessionRowsOver{repo: acceptingCounterStore{}},
		config:             &Config{},
	}
	return server, session
}

// TestReplayDLDataResult_Preserves64BitFields: the Application Center gets
// back the queue id it chose (SCACI §3.10.1, §3.12.1) exactly, live and on
// replay, whether a float64 cannot hold it or a signed 64-bit integer cannot.
func TestReplayDLDataResult_Preserves64BitFields(t *testing.T) {
	for name, acQueID := range map[string]uint64{
		"beyond float64 precision": precisionQueID,
		"beyond signed 64-bit":     beyondSignedQueID,
	} {
		t.Run(name, func(t *testing.T) {
			repo := &operationLogCapture{}
			server, session := precisionServer(repo)
			liveConn := &mockConn{}
			server.registry.sessions = map[net.Conn]*Session{liveConn: session}
			txTime := precisionTxTime
			packetCnt := uint32(9)
			bsEUI := uint64(0x70B3D59CD00009E6)

			require.NoError(t, server.BroadcastDLDataResult(testutil.TestContext(), session.ApplicationCenter(), acQueID, &mioty.DLDataResult{
				EpEui:     0x70B3D59CD00009E7,
				QueId:     uint64(precisionInternalQueID),
				Result:    mioty.ResultSent,
				TxTime:    &txTime,
				PacketCnt: &packetCnt,
				BsEui:     &bsEUI,
			}))
			var live DLDataResult
			require.NoError(t, decodeResponse(liveConn.written, &live))
			assert.Equal(t, acQueID, live.QueID)

			replayConn := &mockConn{}
			require.NoError(t, server.replayDLDataResult(replayConn, session, repo.storedOperation(t)))

			var replayed DLDataResult
			require.NoError(t, decodeResponse(replayConn.written, &replayed))
			assert.Equal(t, acQueID, replayed.QueID)
			require.NotNil(t, replayed.TxTime)
			assert.Equal(t, precisionTxTime, *replayed.TxTime)
		})
	}
}

func TestReplayULData_PreservesReceptionTime(t *testing.T) {
	repo := &operationLogCapture{}
	server, session := precisionServer(repo)

	require.NoError(t, server.BroadcastULData(testutil.TestContext(), precisionTenant, &mioty.ULDataMessage{
		ID:        broadcastULDataTestMessageID,
		EpEui:     0x70B3D59CD00009E7,
		PacketCnt: 3,
		UserData:  []byte{0x01},
		BaseStations: []mioty.BaseStationReception{{
			BsEui:  0x70B3D59CD00009E6,
			RxTime: precisionRxTime,
			Snr:    5,
			Rssi:   -90,
		}},
	}))

	replayConn := &mockConn{}
	require.NoError(t, server.replayULData(replayConn, session, repo.storedOperation(t)))

	var replayed ULData
	require.NoError(t, decodeResponse(replayConn.written, &replayed))
	require.Len(t, replayed.BaseStations, 1)
	assert.Equal(t, precisionRxTime, replayed.BaseStations[0].RxTime)
}

// A reissued uplink is the one the Application Center would have received
// live: every reception detail and the duplicate flag survive the operation
// log, so an uplink produced while it was offline arrives in full (SCACI §1,
// §3.8.1).
func TestReplayULData_ReissuesTheRecordedUplinkInFull(t *testing.T) {
	repo := &operationLogCapture{}
	server, session := precisionServer(repo)
	liveConn := &mockConn{}
	server.registry.sessions = map[net.Conn]*Session{liveConn: session}
	rxDuration, eqSnr, dlRxSnr, dlRxRssi := int64(5_000_000), 9.5, 3.25, -101.5
	profile, mode, format := "eu1", "ulp", uint8(7)
	uplink := &mioty.ULDataMessage{
		ID: broadcastULDataTestMessageID, EpEui: 0x70B3D59CD00009E7, PacketCnt: 3, UserData: []byte{0x01, 0x02}, Format: &format,
		DlOpen: true, ResponseExp: true, DlAck: true, PacketCntReused: true,
		BaseStations: []mioty.BaseStationReception{{
			BsEui: 0x70B3D59CD00009E6, RxTime: precisionRxTime, RxDuration: &rxDuration, Snr: 5.5, Rssi: -90.25,
			EqSnr: &eqSnr, DlRxSnr: &dlRxSnr, DlRxRssi: &dlRxRssi, Profile: &profile, Mode: &mode,
			Subpackets: &mioty.Subpackets{SNR: []float64{1.5, 2.5}, RSSI: []float64{-90, -91},
				Frequency: []int64{868_000_000, 868_100_000}, Phase: []float64{10, -20}},
		}},
	}
	require.NoError(t, server.BroadcastULData(testutil.TestContext(), precisionTenant, uplink))
	var live ULData
	require.NoError(t, decodeResponse(liveConn.written, &live))

	replayConn := &mockConn{}
	require.NoError(t, server.replayULData(replayConn, session, repo.storedOperation(t)))
	var replayed ULData
	require.NoError(t, decodeResponse(replayConn.written, &replayed))

	assert.Equal(t, live, replayed)
}

func TestExtractUint64FromJSON(t *testing.T) {
	tests := map[string]struct {
		value interface{}
		want  uint64
		ok    bool
	}{
		"decimal string above 2^63": {value: "18446744073709551615", want: 1<<64 - 1, ok: true},
		"legacy JSON number":        {value: float64(12345), want: 12345, ok: true},
		"negative JSON number":      {value: float64(-1), ok: false},
		"negative decimal string":   {value: "-1", ok: false},
		"missing":                   {value: nil, ok: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := extractUint64FromJSON(map[string]interface{}{"queId": tc.value}, "queId")
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}
