package scaci_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// The retry scenarios: Application Center A stays reachable, Application
// Center B of the same tenant has lost its connection when the uplink is
// delivered, and the delivery is attempted twice, as the outbox retries a
// row one of whose sessions failed.
const (
	retryAcEuiB     = uint64(0x70B3D59CD0000A0B)
	retryUUIDB      = byte(0x31)
	retryPacketCnt  = uint32(40)
	retryAcOpID     = int64(1)
	retryMessageID  = "3c9e2a71-5d84-4f06-9b1e-7a2c4d6e8f10"
	retryFirstScOp  = int64(-1)
	retryOperations = 1
)

func retryUplink() *mioty.ULDataMessage {
	return &mioty.ULDataMessage{
		ID:           retryMessageID,
		EpEui:        offlineEpEui,
		PacketCnt:    retryPacketCnt,
		UserData:     []byte{0x42},
		BaseStations: []mioty.BaseStationReception{{BsEui: offlineBsEui, RxTime: offlineRxTime, Snr: 12, Rssi: -80}},
	}
}

// acSession reads the session of the harness tenant's Application Center
// acEui opened as snAcUUID.
func (h *scaciHarness) acSession(t *testing.T, acEui uint64, snAcUUID scaci.UUID16) *models.SCACISession {
	t.Helper()
	ac := scaci.ApplicationCenter{TenantID: harnessTenant, AcEui: acEui}
	session, err := h.sessions.GetSessionByAcUUID(testutil.TestContext(), ac.Key(), snAcUUID)
	require.NoError(t, err)
	return session
}

func (h *scaciHarness) waitForACStatus(t *testing.T, acEui uint64, snAcUUID scaci.UUID16, status string) *models.SCACISession {
	t.Helper()
	var session *models.SCACISession
	require.Eventually(t, func() bool {
		session = h.acSession(t, acEui, snAcUUID)
		return session.Status == status
	}, harnessWaitTimeout, harnessWaitStep, "session of %x never became %s", acEui, status)
	return session
}

// uplinkOperations counts the ulData operations the session holds for the
// retry scenario's uplink.
func (h *scaciHarness) uplinkOperations(t *testing.T, sessionID int64) int64 {
	t.Helper()
	return h.countUplinkOperations(t, retryUplinkOperations, sessionID)
}

// waitForUplinkCompleted waits until the service center recorded the
// completion of the session's ulData, which it writes after sending ulDataCmp
// (SCACI §3.8.3).
func (h *scaciHarness) waitForUplinkCompleted(t *testing.T, sessionID int64) {
	t.Helper()
	require.Eventually(t, func() bool {
		return h.countUplinkOperations(t, retryUplinkOperations+` AND state = $5`, sessionID, string(models.OperationStateCompleted)) == retryOperations
	}, harnessWaitTimeout, harnessWaitStep, "the ulData of session %d was never completed", sessionID)
}

const retryUplinkOperations = `SELECT count(*) FROM scaci_operation_log
	WHERE session_id = $1 AND command = $2 AND direction = $3 AND (request_data->>'packetCnt')::bigint = $4`

func (h *scaciHarness) countUplinkOperations(t *testing.T, query string, sessionID int64, more ...interface{}) int64 {
	t.Helper()
	args := append([]interface{}{sessionID, scaci.CmdULData, string(models.OperationDirectionOutbound), retryPacketCnt}, more...)
	var count int64
	require.NoError(t, h.db.QueryRow(testutil.TestContext(), query, args...).Scan(&count))
	return count
}

// retrySessions connects A and B, completes their connect operations and
// leaves B without its connection; it returns A and the IDs of both sessions.
func retrySessions(t *testing.T, h *scaciHarness) (*acClient, int64, int64) {
	t.Helper()
	a := h.dial(t)
	a.connect(harnessUUID(harnessUUIDFirst))
	require.Equal(t, scaci.CmdPingResponse, a.ping(retryAcOpID)["command"], "the service center completed A's connect")
	b := h.dialAs(t, h.pki.client, retryAcEuiB)
	b.connect(harnessUUID(retryUUIDB))
	require.Equal(t, scaci.CmdPingResponse, b.ping(retryAcOpID)["command"], "the service center completed B's connect")
	require.NoError(t, b.conn.Close())
	sessionB := h.waitForACStatus(t, retryAcEuiB, harnessUUID(retryUUIDB), models.SCACISessionStatusDisconnected)
	return a, h.acSession(t, harnessAcEui, harnessUUID(harnessUUIDFirst)).ID, sessionB.ID
}

// resumeB resumes B's session and returns the opId of the one ulData its
// resume reissues; nothing else follows it.
func resumeB(t *testing.T, h *scaciHarness) int64 {
	t.Helper()
	resumed := h.dialAs(t, h.pki.client, retryAcEuiB)
	require.Equal(t, true, resumed.resume(harnessUUID(retryUUIDB), retryAcOpID, 0)["snResume"])
	opID := resumed.completeULData()
	resumed.assertQuiet(offlineQuietPeriod)
	return opID
}

// A retried delivery reaches only the session that lacks the uplink: A,
// which received and completed it, gets nothing more, and B's resume
// reissues its one operation under its original opId - never a second,
// non-duplicate operation for one telegram (SCACI §1 l.118-123, §3.2
// l.203-209, §3.8.1 l.417).
func TestSCACIServer_RetriedUplinkReachesEverySessionOnce(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	a, sessionA, sessionB := retrySessions(t, h)

	require.NoError(t, h.server.BroadcastULData(testutil.TestContext(), harnessTenant, retryUplink()))
	a.completeULData()
	require.NoError(t, h.server.BroadcastULData(testutil.TestContext(), harnessTenant, retryUplink()))

	a.assertQuiet(offlineQuietPeriod)
	assert.Equal(t, retryFirstScOp, resumeB(t, h), "B's operation keeps its opId")
	assert.Equal(t, int64(retryOperations), h.uplinkOperations(t, sessionA))
	assert.Equal(t, int64(retryOperations), h.uplinkOperations(t, sessionB))
}

// The identity outlives the service center: after a restart the retry finds
// what each session holds in the operation log, so neither resume receives
// the uplink as a new operation.
func TestSCACIServer_RetriedUplinkAfterARestartReachesEverySessionOnce(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	a, sessionA, sessionB := retrySessions(t, h)
	require.NoError(t, h.server.BroadcastULData(testutil.TestContext(), harnessTenant, retryUplink()))
	seen := a.completeULData()
	h.waitForUplinkCompleted(t, sessionA)

	h.restart(t)
	require.NoError(t, h.server.BroadcastULData(testutil.TestContext(), harnessTenant, retryUplink()))

	resumedA := h.dial(t)
	require.Equal(t, true, resumedA.resume(harnessUUID(harnessUUIDFirst), retryAcOpID, seen)["snResume"])
	resumedA.assertQuiet(offlineQuietPeriod)
	assert.Equal(t, retryFirstScOp, resumeB(t, h), "B's operation keeps its opId across the restart")
	assert.Equal(t, int64(retryOperations), h.uplinkOperations(t, sessionA))
	assert.Equal(t, int64(retryOperations), h.uplinkOperations(t, sessionB))
}
