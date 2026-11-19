package bssci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// newServiceCenterStatusFixture connects two stations of the endpoint's
// tenant and records every epStat the server forwards.
func newServiceCenterStatusFixture(t *testing.T, epStatus string) (*attachFixture, *mockSCACIEPStatusBroadcaster, *bsscitest.TestConn) {
	t.Helper()
	f := newOwnAttachFixture(t)
	f.endpoints.endpoints[TestEpEui01].EpStatus = epStatus
	broadcaster := NewMockSCACIEPStatusBroadcaster()
	f.server.SetSCACIEPStatusBroadcaster(broadcaster)
	f.server.RegisterSession(f.session)
	_, second := f.newSecondStation()
	return f, broadcaster, second
}

// completeOperation answers the last command the station received with its
// response, as a base station accepting it would.
func (f *attachFixture) completeOperation(t *testing.T, sessionID string, conn *bsscitest.TestConn, command, response string) {
	t.Helper()
	sent := conn.LastMessage(command)
	require.NotNil(t, sent, "%s was sent", command)
	opID := int64(sent["opId"].(float64))
	data := map[string]interface{}{"command": response, "opId": opID}
	session := f.server.GetSession(sessionID)
	msg := &Message{Command: response, OpId: opID, Data: data}
	if response == mioty.CmdAttachPropagateResponse {
		require.NoError(t, f.server.CallHandleAttachPropagateResponse(session, msg, data))
		return
	}
	require.NoError(t, f.server.CallHandleDetachPropagateResponse(session, msg, data))
}

// The service center decides an endpoint's attachment (SCACI §3.13, BSSCI
// §3.8): a station confirming an attPrp sent before the endpoint was detached
// neither attaches it again nor announces it, whichever station confirms.
func TestLateAttachPropagateDoesNotUndoADetachment(t *testing.T) {
	f, broadcaster, second := newServiceCenterStatusFixture(t, endpoint.EndpointStatusDetached)
	_, foreign := f.newStation("attach-foreign-station", TestBsEui03, attachRoamingOwnerTenant)

	require.Empty(t, f.server.SendAttachPropagateToAll(TestEpEui01, testPresharedKey(), 0x1505, false, 0, false, 0, false, false))
	f.completeOperation(t, "attach-foreign-station", foreign, mioty.CmdAttachPropagate, mioty.CmdAttachPropagateResponse)
	f.completeOperation(t, f.session.ID, f.conn, mioty.CmdAttachPropagate, mioty.CmdAttachPropagateResponse)
	f.completeOperation(t, "attach-second-station", second, mioty.CmdAttachPropagate, mioty.CmdAttachPropagateResponse)
	f.server.wg.Wait()

	assert.Equal(t, endpoint.EndpointStatusDetached, f.endpoints.endpoints[TestEpEui01].EpStatus, "the endpoint stays detached")
	assert.Zero(t, broadcaster.CallCount(), "no attachment is announced")
}

// A station confirming a detPrp sent before the endpoint was attached again
// neither detaches it nor announces it.
func TestLateDetachPropagateDoesNotUndoAnAttachment(t *testing.T) {
	f, broadcaster, second := newServiceCenterStatusFixture(t, EndpointStatusAttached)

	require.Empty(t, f.server.SendDetachPropagateToAll(TestEpEui01))
	f.completeOperation(t, f.session.ID, f.conn, mioty.CmdDetachPropagate, mioty.CmdDetachPropagateResponse)
	f.completeOperation(t, "attach-second-station", second, mioty.CmdDetachPropagate, mioty.CmdDetachPropagateResponse)
	f.server.wg.Wait()

	assert.Equal(t, EndpointStatusAttached, f.endpoints.endpoints[TestEpEui01].EpStatus, "the endpoint stays attached")
	assert.Zero(t, broadcaster.CallCount(), "no detachment is announced")
}

// attachStatePersistence applies an attach's endpoint updates to the endpoint
// directory, as the attach transaction does.
type attachStatePersistence struct {
	recordingAttachPersistence
	endpoints *fakeEndpointRepo
}

func (p *attachStatePersistence) PersistAttachSession(ctx context.Context, rec AttachSessionRecord) error {
	if err := p.endpoints.EndpointAttachmentStateUpdate(ctx, rec.TenantID, rec.EndpointID, rec.EndpointUpdates); err != nil {
		return err
	}
	return p.recordingAttachPersistence.PersistAttachSession(ctx, rec)
}

// An over-the-air attach is reported once with its over-the-air fields; the
// propagation that follows is not a second attachment.
func TestOverTheAirAttachIsNotReportedAgainByItsPropagation(t *testing.T) {
	f, broadcaster, second := newServiceCenterStatusFixture(t, endpoint.EndpointStatusDetached)
	f.server.attachPersistence = &attachStatePersistence{endpoints: f.endpoints}
	f.attach(t)
	complete := map[string]interface{}{"command": mioty.CmdAttachComplete, "opId": int64(1)}
	require.NoError(t, f.server.CallHandleMessage(f.session, &Message{Command: mioty.CmdAttachComplete, OpId: 1, Data: complete}, complete))
	f.server.wg.Wait()
	require.Equal(t, 1, broadcaster.CallCount())
	require.NotNil(t, broadcaster.LastCall().Data.AttachCnt, "the over-the-air attach carries its fields")

	require.NoError(t, f.server.SendAttachPropagate("attach-second-station", TestEpEui01, testPresharedKey(), 0x1505, false, 0, false, 0, false, false))
	f.completeOperation(t, "attach-second-station", second, mioty.CmdAttachPropagate, mioty.CmdAttachPropagateResponse)
	f.server.wg.Wait()

	assert.Equal(t, 1, broadcaster.CallCount(), "the propagation of an attached endpoint is not reported")
}
