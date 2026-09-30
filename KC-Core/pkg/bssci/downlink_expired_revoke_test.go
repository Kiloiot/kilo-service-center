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
