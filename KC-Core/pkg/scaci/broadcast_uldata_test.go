package scaci

import (
	"errors"
	"net"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	broadcastULDataTestTenant    = int64(42)
	broadcastULDataTestMessageID = "5b0d4c9e-7f1a-4c2b-9d3e-1a2b3c4d5e6f"
)

var errBroadcastULDataTestWrite = errors.New("write failed")

func newBroadcastULDataServer(sessions map[net.Conn]*Session) *Server {
	return &Server{
		registry: newTestRegistry(sessions, newHolderFake()),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   logger.NewNop(),
		config:   &Config{},
	}
}

func activeBroadcastSession(tenantID int64) *Session {
	return withOpIDs(&Session{
		TenantID: tenantID,
		State:    StateActive,
	}, OpIDPair{SC: -1})
}

func broadcastULDataFixture() *mioty.ULDataMessage {
	return &mioty.ULDataMessage{
		ID:        broadcastULDataTestMessageID,
		EpEui:     0x70B3D5677011150A,
		PacketCnt: 9,
		UserData:  []byte{0x42},
		BaseStations: []mioty.BaseStationReception{{
			BsEui:  0x70B3D59CD00009E6,
			RxTime: 1_700_000_000_000_000_000,
			Rssi:   -80,
			Snr:    12,
		}},
	}
}

// One failed Application Center session must fail the broadcast: the outbox
// keeps the row pending so the delivery worker retries every session.
func TestBroadcastULData_PartialSendFailureIsReported(t *testing.T) {
	t.Parallel()
	healthy := &mockConn{}
	broken := &failingMockConn{writeErr: errBroadcastULDataTestWrite}
	server := newBroadcastULDataServer(map[net.Conn]*Session{
		healthy: activeBroadcastSession(broadcastULDataTestTenant),
		broken:  activeBroadcastSession(broadcastULDataTestTenant),
	})

	err := server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, broadcastULDataFixture())

	require.Error(t, err)
	assert.ErrorIs(t, err, errBroadcastSessionFailed)
	assert.ErrorIs(t, err, errBroadcastULDataTestWrite)
	assert.NotEmpty(t, healthy.written, "the healthy session still receives the uplink")
}

func TestBroadcastULData_AllSessionsDeliveredIsSuccess(t *testing.T) {
	t.Parallel()
	first := &mockConn{}
	second := &mockConn{}
	otherTenant := &mockConn{}
	server := newBroadcastULDataServer(map[net.Conn]*Session{
		first:       activeBroadcastSession(broadcastULDataTestTenant),
		second:      activeBroadcastSession(broadcastULDataTestTenant),
		otherTenant: activeBroadcastSession(broadcastULDataTestTenant + 1),
	})

	require.NoError(t, server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, broadcastULDataFixture()))

	assert.NotEmpty(t, first.written)
	assert.NotEmpty(t, second.written)
	assert.Empty(t, otherTenant.written, "tenant isolation: foreign sessions receive nothing")
}

// A telegram is delivered once: its SCACI duplicate flag reports a reused
// packet counter (§3.8.1), never the receptions merged into the stored
// message, and the stored message is not rewritten for it.
func TestBroadcastULData_DuplicateReportsAReusedPacketCounter(t *testing.T) {
	merged := true
	for name, tc := range map[string]struct {
		reused bool
		want   bool
	}{
		"merged receptions are new data":             {reused: false, want: false},
		"a counter reused outside the window is not": {reused: true, want: true},
	} {
		t.Run(name, func(t *testing.T) {
			conn := &mockConn{}
			server := newBroadcastULDataServer(map[net.Conn]*Session{conn: activeBroadcastSession(broadcastULDataTestTenant)})
			uplink := broadcastULDataFixture()
			uplink.Duplicate = &merged
			uplink.PacketCntReused = tc.reused

			require.NoError(t, server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, uplink))

			var sent ULData
			require.NoError(t, decodeResponse(conn.written, &sent))
			require.NotNil(t, sent.Duplicate)
			assert.Equal(t, tc.want, *sent.Duplicate)
			assert.Same(t, &merged, uplink.Duplicate, "the stored message keeps its own duplicate flag")
		})
	}
}

func TestBroadcastULData_NoActiveSessionIsSuccess(t *testing.T) {
	t.Parallel()
	server := newBroadcastULDataServer(map[net.Conn]*Session{})

	require.NoError(t, server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, broadcastULDataFixture()))
}
