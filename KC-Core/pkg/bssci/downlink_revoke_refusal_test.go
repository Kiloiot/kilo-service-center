package bssci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// revokeRefusalMessage is the message of the stations' refusal in these tests.
const revokeRefusalMessage = "downlink not held"

// refusalRecorder records every revoke a base station refused and answers it
// as the queue does: revoked, unless the downlink had already ended.
type refusalRecorder struct {
	mqttTestDownlinkService
	ended    bool
	refusals []RevokeRefusal
}

func (r *refusalRecorder) ProcessRevokeRefusal(_ context.Context, _ *Session, refusal RevokeRefusal) (bool, error) {
	r.refusals = append(r.refusals, refusal)
	return !r.ended, nil
}

// refuseRevoke revokes the held downlink at its station and delivers the
// station's error answer to that dlDataRev.
func refuseRevoke(t *testing.T, server *Server, session *Session, revoke func() error) {
	t.Helper()
	require.NoError(t, revoke())
	opID := session.LastScOpId
	refusal := map[string]interface{}{"command": mioty.CmdError, "opId": opID, "code": int64(POSIX_ENOENT), "message": revokeRefusalMessage}
	require.NoError(t, server.handleError(session, &Message{Command: mioty.CmdError, OpId: opID, Data: refusal}, refusal))
	_, err := server.statusSvc.GetPendingOperation(session, opID)
	assert.Error(t, err, "the error finalizes the dlDataRev")
}

// BSSCI §3.13, §3.17: a station's error answer to a dlDataRev reaches the
// downlink service with its code and message; when the service ends the
// downlink revoked, the revocation is recorded as a confirmed one is.
func TestRevokeRefusedByTheStationRevokesTheDownlink(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{row: heldDownlink(mioty.DLQueueStatusQueued)}, &pendingRevocations{})
	recorder := &refusalRecorder{}
	server.downlinkSvc = recorder
	audit := &revokeAudit{}
	server.auditLogger = audit
	session, _ := registerRoamingStation(server)

	refuseRevoke(t, server, session, func() error {
		_, err := server.RevokeDownlink(testutil.TestContext(), revokeRef)
		return err
	})

	assert.Equal(t, []RevokeRefusal{{QueueID: int64(revokeQueueID), EndpointEUI: 0x70b3d59cd0000341, Code: POSIX_ENOENT}},
		recorder.refusals)
	assert.Equal(t, []int64{int64(revokeQueueID)}, audit.revoked)
}

// A downlink the expiry sweep expired while a station held it keeps its
// expiry as its only outcome when the station refuses the sweep's revoke.
func TestRevokeRefusedForAnExpiredDownlinkKeepsTheExpiry(t *testing.T) {
	server, _ := newRevokeServer(t, queueRowStore{}, &pendingRevocations{})
	recorder := &refusalRecorder{ended: true}
	server.downlinkSvc = recorder
	audit := &revokeAudit{}
	server.auditLogger = audit
	session, _ := registerRoamingStation(server)

	refuseRevoke(t, server, session, func() error {
		return server.RevokeHeldDownlink(testutil.TestContext(), expiredHeldDownlink())
	})

	require.Len(t, recorder.refusals, 1)
	assert.Empty(t, audit.revoked, "an expired downlink is not also recorded as revoked")
}
