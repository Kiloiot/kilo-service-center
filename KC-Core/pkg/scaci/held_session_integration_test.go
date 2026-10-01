package scaci_test

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// The endpoint and downlink the offline scenarios report on.
const (
	offlineEpEui          = uint64(0x70B3D5677011150A)
	offlineBsEui          = uint64(0x70B3D59CD00009E6)
	offlineACQueID        = uint64(4242)
	offlineInternalQueID  = int64(7_300_000_000_000_042)
	offlineFirstPacketCnt = uint32(10)
	offlineRxTime         = int64(1_700_000_000_000_000_000)
	offlineQuietPeriod    = 500 * time.Millisecond
	// resumeWithSecondConCmpFrames is the reissued uplink and the refusal.
	resumeWithSecondConCmpFrames = 2
)

func offlineUplink(packetCnt uint32) *mioty.ULDataMessage {
	return &mioty.ULDataMessage{
		ID:           uuid.NewString(),
		EpEui:        offlineEpEui,
		PacketCnt:    packetCnt,
		UserData:     []byte{0x42},
		BaseStations: []mioty.BaseStationReception{{BsEui: offlineBsEui, RxTime: offlineRxTime, Snr: 12, Rssi: -80}},
	}
}

// intField reads an integer field of a decoded frame, in whatever width
// MessagePack chose.
func intField(t *testing.T, msg map[string]interface{}, key string) int64 {
	t.Helper()
	switch v := msg[key].(type) {
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v) //nolint:gosec // test values fit
	}
	t.Fatalf("%s is not an integer: %v", key, msg[key])
	return 0
}

// expect reads the next frame and requires its command.
func (c *acClient) expect(command string) map[string]interface{} {
	c.t.Helper()
	msg, err := c.receive()
	require.NoError(c.t, err)
	require.Equal(c.t, command, msg["command"], "got %v", msg)
	return msg
}

// completeULData answers an ulData and waits for the service center to
// complete the operation (SCACI §3.8.2, §3.8.3); it returns the opId.
func (c *acClient) completeULData() int64 {
	c.t.Helper()
	opID := intField(c.t, c.expect(scaci.CmdULData), "opId")
	c.send(scaci.ULDataResponse{BaseMessage: scaci.BaseMessage{Command: scaci.CmdULDataResponse, OpId: opID}})
	c.expect(scaci.CmdULDataComplete)
	return opID
}

// assertQuiet requires that nothing reaches the application center for a while.
func (c *acClient) assertQuiet(period time.Duration) {
	c.t.Helper()
	require.NoError(c.t, c.conn.SetReadDeadline(time.Now().Add(period)))
	_, err := frameCodec.Read(c.conn)
	var netErr net.Error
	require.True(c.t, errors.As(err, &netErr) && netErr.Timeout() || errors.Is(err, os.ErrDeadlineExceeded),
		"the application center received a frame: %v", err)
}

// produceWhileOffline produces an uplink and a downlink result while no
// connection holds the application center's session.
func (h *scaciHarness) produceWhileOffline(t *testing.T) {
	t.Helper()
	ctx := testutil.TestContext()
	require.NoError(t, h.server.BroadcastULData(ctx, harnessTenant, offlineUplink(offlineFirstPacketCnt+1)))
	require.NoError(t, h.server.BroadcastDLDataResult(ctx, harnessApplicationCenter, offlineACQueID,
		&mioty.DLDataResult{EpEui: offlineEpEui, QueId: uint64(offlineInternalQueID), Result: mioty.ResultExpired}))
}

// loseConnection closes the application center's connection and waits until
// the service center recorded the loss.
func (h *scaciHarness) loseConnection(t *testing.T, client *acClient, snAcUUID scaci.UUID16) {
	t.Helper()
	require.NoError(t, client.conn.Close())
	h.waitForStatus(t, snAcUUID, models.SCACISessionStatusDisconnected)
}

// An application center that resumes its session receives the uplink and the
// downlink result produced while it was disconnected, reissued after the
// connect operation with SC opIds that continue from before the loss, and
// before any operation started after the resume (SCACI §1 l.118-123, §3.2,
// §3.8, §3.12).
func TestSCACIServer_ResumedSessionReceivesOperationsProducedWhileOffline(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	snAcUUID := harnessUUID(harnessUUIDFirst)
	first := h.dial(t)
	first.connect(snAcUUID)
	h.waitForStatus(t, snAcUUID, models.SCACISessionStatusActive)
	require.Equal(t, scaci.CmdPingResponse, first.ping(harnessFirstAcOpID)["command"], "the service center completed the connect")
	require.NoError(t, h.server.BroadcastULData(testutil.TestContext(), harnessTenant, offlineUplink(offlineFirstPacketCnt)))
	seen := first.completeULData()
	h.loseConnection(t, first, snAcUUID)

	h.produceWhileOffline(t)

	resumed := h.dial(t)
	rsp := resumed.resume(snAcUUID, harnessFirstAcOpID, seen)
	require.Equal(t, true, rsp["snResume"])
	require.NoError(t, h.server.BroadcastULData(testutil.TestContext(), harnessTenant, offlineUplink(offlineFirstPacketCnt+2)))

	uplink := resumed.expect(scaci.CmdULData)
	result := resumed.expect(scaci.CmdDLDataResult)
	next := resumed.expect(scaci.CmdULData)
	assert.Equal(t, seen-1, intField(t, uplink, "opId"), "SC opIds continue from the last one before the loss")
	assert.Equal(t, int64(offlineFirstPacketCnt+1), intField(t, uplink, "packetCnt"))
	assert.Equal(t, seen-2, intField(t, result, "opId"))
	assert.Equal(t, int64(offlineACQueID), intField(t, result, "queId"), "the result carries the queue id the application center assigned")
	assert.Equal(t, seen-3, intField(t, next, "opId"), "an operation started after the resume follows the reissued ones")
	assert.Equal(t, int64(offlineFirstPacketCnt+2), intField(t, next, "packetCnt"))
}

// A new session starts from discarded state (SCACI §1 l.121-123): nothing
// produced while the previous session was disconnected reaches it.
func TestSCACIServer_NewSessionReceivesNothingProducedWhileOffline(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	snAcUUID := harnessUUID(harnessUUIDFirst)
	first := h.dial(t)
	first.connect(snAcUUID)
	h.waitForStatus(t, snAcUUID, models.SCACISessionStatusActive)
	h.loseConnection(t, first, snAcUUID)

	h.produceWhileOffline(t)

	fresh := h.dial(t)
	rsp := fresh.connect(harnessUUID(harnessUUIDSecond))
	require.Equal(t, false, rsp["snResume"])
	fresh.assertQuiet(offlineQuietPeriod)
	assert.False(t, h.session(t, snAcUUID).CanResume, "the previous session and what it held are discarded")
}

// Resumable sessions survive a service center restart (scaci_sessions,
// scaci_operation_log), so the restarted service center keeps recording for a
// disconnected session and its resume receives what was produced after the
// restart.
func TestSCACIServer_RestartKeepsRecordingForResumableSessions(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	snAcUUID := harnessUUID(harnessUUIDFirst)
	first := h.dial(t)
	first.connect(snAcUUID)
	h.waitForStatus(t, snAcUUID, models.SCACISessionStatusActive)
	h.loseConnection(t, first, snAcUUID)

	h.restart(t)
	h.produceWhileOffline(t)

	resumed := h.dial(t)
	rsp := resumed.resume(snAcUUID, 0, 0)
	require.Equal(t, true, rsp["snResume"])
	assert.Equal(t, int64(-1), intField(t, resumed.expect(scaci.CmdULData), "opId"))
	assert.Equal(t, int64(-2), intField(t, resumed.expect(scaci.CmdDLDataResult), "opId"))
}

// The connect operation completes once (SCACI §3.3): a second conCmp on a
// resumed session is refused with a protocol error, and the session keeps its
// connection and what it reissues.
func TestSCACIServer_SecondConnectCompleteIsRefused(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	snAcUUID := harnessUUID(harnessUUIDFirst)
	first := h.dial(t)
	first.connect(snAcUUID)
	h.waitForStatus(t, snAcUUID, models.SCACISessionStatusActive)
	h.loseConnection(t, first, snAcUUID)
	require.NoError(t, h.server.BroadcastULData(testutil.TestContext(), harnessTenant, offlineUplink(offlineFirstPacketCnt)))

	resumed := h.dial(t)
	require.Equal(t, true, resumed.resume(snAcUUID, 0, 0)["snResume"])
	resumed.send(scaci.ConnectComplete{BaseMessage: scaci.BaseMessage{Command: scaci.CmdConnectComplete, OpId: scaci.OpIDConnect}})

	frames := make(map[interface{}]map[string]interface{}, resumeWithSecondConCmpFrames)
	for range resumeWithSecondConCmpFrames {
		msg, err := resumed.receive()
		require.NoError(t, err, "the resumed session keeps its connection")
		frames[msg["command"]] = msg
	}
	require.Contains(t, frames, scaci.CmdError, "the second conCmp is refused")
	assert.Equal(t, int64(scaci.POSIX_EPROTO), intField(t, frames[scaci.CmdError], "code"))
	assert.Contains(t, frames, scaci.CmdULData, "the held uplink is still reissued")
}

// An uplink produced between the conRsp and the conCmp of a connect operation
// reaches the Application Center once the operation completes, for a new
// session and for a resumed one alike (SCACI §1, §3.3).
func TestSCACIServer_OperationDuringTheConnectReachesTheSession(t *testing.T) {
	for name, connect := range map[string]func(h *scaciHarness, snAcUUID scaci.UUID16) scaci.Connect{
		"new session": func(*scaciHarness, scaci.UUID16) scaci.Connect {
			return scaci.Connect{SnAcUUID: harnessUUID(harnessUUIDSecond)}
		},
		"resumed session": func(h *scaciHarness, snAcUUID scaci.UUID16) scaci.Connect {
			first := h.dial(t)
			first.connect(snAcUUID)
			require.Equal(t, scaci.CmdPingResponse, first.ping(harnessFirstAcOpID)["command"])
			h.loseConnection(t, first, snAcUUID)
			noneSeen := int64(0)
			return scaci.Connect{SnAcUUID: snAcUUID, SnAcOpId: &noneSeen, SnScOpId: &noneSeen}
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newSCACIHarness(t, scaci.Config{})
			con := connect(h, harnessUUID(harnessUUIDFirst))
			client := h.dial(t)
			client.startConnect(con)

			require.NoError(t, h.server.BroadcastULData(testutil.TestContext(), harnessTenant, offlineUplink(offlineFirstPacketCnt)))
			client.completeConnect()

			uplink := client.expect(scaci.CmdULData)
			assert.Equal(t, int64(-1), intField(t, uplink, "opId"), "the first SC operation of the session")
			assert.Equal(t, int64(offlineFirstPacketCnt), intField(t, uplink, "packetCnt"))
		})
	}
}
