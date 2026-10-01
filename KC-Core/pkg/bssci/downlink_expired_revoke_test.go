package bssci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// expiredHeldDownlink is a downlink the service center expired while the
// roaming station TestBsEui04 held it.
func expiredHeldDownlink() *storage.DownlinkMessage {
	row := heldDownlink(mioty.DLQueueStatusExpired)
	row.TenantID = "3"
	return row
}

// TestRevokeHeldDownlink_AsksTheHoldingStationToDropIt pins BSSCI §3.13: a
// downlink that expired while a station held it is revoked at that station
// under the endpoint owner, without reading the queue row again.
func TestRevokeHeldDownlink_AsksTheHoldingStationToDropIt(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{err: storage.ErrNotFound}, &pendingRevocations{})
	session, conn := registerRoamingStation(server)

	require.NoError(t, server.RevokeHeldDownlink(testutil.TestContext(), expiredHeldDownlink()))

	assert.True(t, conn.errorSent, "dlDataRev is written to the holding station")
	op, err := server.statusSvc.GetPendingOperation(session, session.LastScOpId)
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.Equal(t, mioty.CmdDLDataRevoke, op.OperationType)
	assert.Equal(t, "3", op.Metadata[models.EventDetailKeyTenantID])
}

func TestRevokeHeldDownlink_DisconnectedHoldingStationIsMissing(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{err: storage.ErrNotFound}, &pendingRevocations{})

	err := server.RevokeHeldDownlink(testutil.TestContext(), expiredHeldDownlink())

	require.ErrorIs(t, err, scheduler.ErrSchedulerResourceMissing)
}

// finishedRevocations answers every revoke confirmation for a downlink that
// had already ended.
type finishedRevocations struct{ mqttTestDownlinkService }

func (*finishedRevocations) ProcessRevokeResponse(_ context.Context, _ *Session, opId int64, _ int64, _ uint64) (map[string]interface{}, bool, error) {
	return map[string]interface{}{"command": mioty.CmdDLDataRevokeComplete, "opId": opId}, false, nil
}

// revokeAudit records every revocation filed in the events.
type revokeAudit struct {
	noopAuditLogger
	revoked []int64
}

func (a *revokeAudit) RecordDLRevokeResponse(_ context.Context, _ string, _ *Session, _ uint64, queueID int64, _ int64) error {
	a.revoked = append(a.revoked, queueID)
	return nil
}

// TestDLDataRevokeResponse_OfAnEndedDownlinkRecordsNoRevocation: the station
// confirming the revoke of a downlink that expired while it held it completes
// the operation, and the events keep the expiry as its only outcome.
func TestDLDataRevokeResponse_OfAnEndedDownlinkRecordsNoRevocation(t *testing.T) {
	server := NewTestServerWithMemoryStatusService(logger.NewNop(), nil, nil, 1)
	server.downlinkSvc = &finishedRevocations{}
	audit := &revokeAudit{}
	server.auditLogger = audit
	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{ID: "revoke-station", BaseStationEUI: TestBsEui04, Encoding: EncodingJSON, DbSessionID: 1},
		Conn:                 conn,
	}
	server.statusSvc.RestorePendingOperation(session, -9, &PendingOperation{
		OperationID: -9, OperationType: mioty.CmdDLDataRevoke,
		Endpoint: []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x03, 0x41},
		Metadata: map[string]interface{}{models.EventDetailKeyQueID: int64(revokeQueueID), models.EventDetailKeyTenantID: "3"},
	})

	require.NoError(t, server.handleDLDataRevokeResponse(session, &Message{Command: mioty.CmdDLDataRevokeResponse, OpId: -9}, nil))

	assert.True(t, conn.SeenCommand(mioty.CmdDLDataRevokeComplete), "the operation completes")
	assert.Empty(t, audit.revoked, "an expired downlink is not also recorded as revoked")
}

// stationRevocations lists the overdue downlinks a station is asked to drop.
type stationRevocations struct {
	queueRowStore
	revoking []*storage.DownlinkMessage
}

func (s stationRevocations) ListStationRevocations(context.Context, uint64) ([]*storage.DownlinkMessage, error) {
	return s.revoking, nil
}

// revokingAt is an overdue downlink of tenant 3 the roaming station TestBsEui04 holds.
func revokingAt(queID int64) *storage.DownlinkMessage {
	row := heldDownlink(mioty.DLQueueStatusRevoking)
	row.QueID = queID
	row.TenantID = "3"
	return row
}

// inFlightRevoke restores on the session a dlDataRev for queID that awaits
// the station's answer, as a resumed session reissues it.
func inFlightRevoke(server *Server, session *Session, queID uint64) {
	server.statusSvc.RestorePendingOperation(session, inFlightRevokeOpID, &PendingOperation{
		OperationID: inFlightRevokeOpID, OperationType: mioty.CmdDLDataRevoke,
		Metadata: map[string]interface{}{models.EventDetailKeyQueID: queID, models.EventDetailKeyTenantID: "3"},
	})
}

// inFlightRevokeOpID is the operation of the dlDataRev inFlightRevoke restores.
const inFlightRevokeOpID int64 = -50

// TestAskAgainToDrop_SendsOnlyTheRevokesNotInFlight pins BSSCI §3.13 across a
// reconnect: every overdue downlink the station holds is asked for again,
// under its owner tenant, except the one whose dlDataRev the resumed session
// reissued.
func TestAskAgainToDrop_SendsOnlyTheRevokesNotInFlight(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{}, &pendingRevocations{})
	server.downlinkQueueStore = stationRevocations{revoking: []*storage.DownlinkMessage{revokingAt(901), revokingAt(902)}}
	session, conn := registerRoamingStation(server)
	inFlightRevoke(server, session, 901)

	server.askAgainToDrop(testutil.TestContext(), session)

	assert.True(t, conn.errorSent, "a dlDataRev is written")
	op, err := server.statusSvc.GetPendingOperation(session, session.LastScOpId)
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.Equal(t, mioty.CmdDLDataRevoke, op.OperationType)
	assert.EqualValues(t, 902, op.Metadata[models.EventDetailKeyQueID], "only the revoke not in flight is sent")
	assert.Equal(t, "3", op.Metadata[models.EventDetailKeyTenantID])
	_, err = server.statusSvc.GetPendingOperation(session, session.LastScOpId+1)
	assert.Error(t, err, "one dlDataRev is sent")
}

// TestRevokeHeldDownlink_ARevokeInFlightIsNotRepeated: a downlink whose
// dlDataRev awaits the holder's answer is not asked for a second time; once
// that operation ended without settling the downlink, it is asked again.
func TestRevokeHeldDownlink_ARevokeInFlightIsNotRepeated(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{err: storage.ErrNotFound}, &pendingRevocations{})
	session, conn := registerRoamingStation(server)
	inFlightRevoke(server, session, revokeQueueID)

	require.NoError(t, server.RevokeHeldDownlink(testutil.TestContext(), expiredHeldDownlink()))
	assert.False(t, conn.errorSent, "no second dlDataRev while the first awaits its answer")

	require.NoError(t, server.statusSvc.RemovePendingOperation(testutil.TestContext(), session, inFlightRevokeOpID))
	require.NoError(t, server.RevokeHeldDownlink(testutil.TestContext(), expiredHeldDownlink()))
	assert.True(t, conn.errorSent, "the downlink is asked for again")
}
