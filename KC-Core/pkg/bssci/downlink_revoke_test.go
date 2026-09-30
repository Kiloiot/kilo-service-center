package bssci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// revokeRef names the owner tenant's downlink the revoke tests revoke.
var revokeRef = scheduler.DownlinkRef{TenantID: revokeOwnerTenant, QueID: revokeQueueID}

const (
	revokeOwnerTenant   = int64(3)
	revokeStationTenant = int64(7)
	revokeQueueID       = uint64(900)
	revokeEndpointEUI   = "70b3d59cd0000341"
)

// queueRowStore is a downlink queue holding one row, or failing every read.
type queueRowStore struct {
	row *storage.DownlinkMessage
	err error
}

func (f queueRowStore) GetDownlinkByRevocation(context.Context, storage.DownlinkRevocation) (*storage.DownlinkMessage, error) {
	return f.row, f.err
}

// pendingRevocations answers every local revoke with one outcome and records the calls.
type pendingRevocations struct {
	pending bool
	calls   []storage.DownlinkRevocation
}

func (f *pendingRevocations) RevokeDownlink(_ context.Context, revocation storage.DownlinkRevocation) (bool, error) {
	f.calls = append(f.calls, revocation)
	return f.pending, nil
}

func newRevokeServer(t *testing.T, queue queueRowStore, revocations *pendingRevocations) (*Server, *recordingEventStore) {
	t.Helper()
	events := &recordingEventStore{}
	server := NewTestServerWithMemoryStatusService(logger.NewNop(), nil, events, 1)
	server.downlinkQueueStore = queue
	server.downlinkRevoke = revocations
	return server, events
}

// registerRoamingStation connects the base station holding the downlink; it
// belongs to another tenant than the endpoint.
func registerRoamingStation(server *Server) (*Session, *testConn) {
	conn := &testConn{}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "revoke-station",
			BaseStationEUI:    TestBsEui04,
			Encoding:          EncodingMessagePack,
			LastScOpId:        -1,
			HandshakeComplete: true,
			DbSessionID:       1,
			ResolvedTenantID:  revokeStationTenant,
		},
		Conn:          conn,
		Bidirectional: true,
	}
	server.RegisterSession(session)
	return session, conn
}

func heldDownlink(status mioty.DLQueueStatus) *storage.DownlinkMessage {
	return &storage.DownlinkMessage{
		EPEUI:  revokeEndpointEUI,
		Status: status,
		QueID:  int64(revokeQueueID),
		BsEui:  TestBsEui04,
	}
}

// TestRevokeDownlink_PendingDownlinkIsRevokedInTheQueue: a downlink no base
// station holds yet is revoked in the queue; nothing is sent to a station.
func TestRevokeDownlink_PendingDownlinkIsRevokedInTheQueue(t *testing.T) {
	revocations := &pendingRevocations{pending: true}
	server, _ := newRevokeServer(t, queueRowStore{row: &storage.DownlinkMessage{
		EPEUI: revokeEndpointEUI, Status: mioty.DLQueueStatusPending, QueID: int64(revokeQueueID),
	}}, revocations)
	_, conn := registerRoamingStation(server)

	bsEui, err := server.RevokeDownlink(testutil.TestContext(), revokeRef)

	require.NoError(t, err)
	assert.Zero(t, bsEui)
	assert.Equal(t, []storage.DownlinkRevocation{{QueID: int64(revokeQueueID), TenantID: revokeOwnerTenant}}, revocations.calls,
		"a revoke in the queue names no station")
	assert.False(t, conn.errorSent, "no dlDataRev for a downlink no base station holds")
}

// TestRevokeDownlink_HeldDownlinkIsRevokedAtItsStationUnderTheOwnerTenant: a
// roaming station revokes the downlink on behalf of the endpoint owner, so
// the revoke event and the pending operation belong to the owner tenant.
func TestRevokeDownlink_HeldDownlinkIsRevokedAtItsStationUnderTheOwnerTenant(t *testing.T) {
	server, events := newRevokeServer(t, queueRowStore{row: heldDownlink(mioty.DLQueueStatusQueued)}, &pendingRevocations{})
	session, conn := registerRoamingStation(server)

	bsEui, err := server.RevokeDownlink(testutil.TestContext(), revokeRef)

	require.NoError(t, err)
	assert.Equal(t, TestBsEui04, bsEui)
	assert.True(t, conn.errorSent, "dlDataRev is written to the holding station")
	require.Len(t, events.created, 1)
	assert.Equal(t, "3", events.created[0].TenantID)
	op, err := server.statusSvc.GetPendingOperation(session, session.LastScOpId)
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.Equal(t, "3", op.Metadata[models.EventDetailKeyTenantID])
}

// TestRevokeDownlink_FinishedDownlinkIsNotFound: a downlink that already
// finished cannot be revoked and no dlDataRev reaches the station.
func TestRevokeDownlink_FinishedDownlinkIsNotFound(t *testing.T) {
	for _, status := range []mioty.DLQueueStatus{mioty.DLQueueStatusTransmitted, mioty.DLQueueStatusRevoked} {
		t.Run(string(status), func(t *testing.T) {
			server, _ := newRevokeServer(t, queueRowStore{row: heldDownlink(status)}, &pendingRevocations{})
			_, conn := registerRoamingStation(server)

			_, err := server.RevokeDownlink(testutil.TestContext(), revokeRef)

			require.ErrorIs(t, err, scheduler.ErrSchedulerQueueNotFound)
			assert.False(t, conn.errorSent)
		})
	}
}

// TestRevokeDownlink_UnknownDownlinkIsNotFound: a queue id the tenant does not
// own reads as not found.
func TestRevokeDownlink_UnknownDownlinkIsNotFound(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{err: storage.ErrNotFound}, &pendingRevocations{})

	_, err := server.RevokeDownlink(testutil.TestContext(), revokeRef)

	require.ErrorIs(t, err, scheduler.ErrSchedulerQueueNotFound)
}

// TestRevokeDownlink_DisconnectedHoldingStationIsMissing: the station holding
// the downlink must be connected to take the revoke.
func TestRevokeDownlink_DisconnectedHoldingStationIsMissing(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{row: heldDownlink(mioty.DLQueueStatusQueued)}, &pendingRevocations{})

	bsEui, err := server.RevokeDownlink(testutil.TestContext(), revokeRef)

	require.ErrorIs(t, err, scheduler.ErrSchedulerResourceMissing)
	assert.Equal(t, TestBsEui04, bsEui)
}
