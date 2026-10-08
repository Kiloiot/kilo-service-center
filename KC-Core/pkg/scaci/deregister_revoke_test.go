package scaci

import (
	"net"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// SCACI §3.7: a deregistered endpoint's downlinks are revoked through the one
// revoke path (BSSCI §3.13), which revokes a downlink still pending in the
// queue there and asks the base station holding one to drop it with dlDataRev.
func TestHandleDeregisterComplete_RevokesDownlinksThroughTheRevokePath(t *testing.T) {
	const (
		opID       = int64(42)
		tenantID   = int64(1)
		pendingID  = uint64(7100001)
		heldID     = uint64(7100002)
		holdingEUI = uint64(0x70B3D59CD00009E6)
	)
	downlinks := new(MockDLService)
	downlinks.On("GetDownlinkQueue", mock.Anything, tenantID, mock.Anything).Return([]*storage.DownlinkMessage{
		{QueID: int64(pendingID), Status: mioty.DLQueueStatusPending},
		{QueID: int64(heldID), Status: mioty.DLQueueStatusQueued},
	}, nil)
	downlinks.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(tenantID, numericTestEpEui, pendingID)).Return(uint64(0), "")
	downlinks.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(tenantID, numericTestEpEui, heldID)).Return(holdingEUI, "")
	endpoints := new(MockEndpointService)
	endpoints.On("PropagateDetachToAll", mock.Anything, tenantID, numericTestEpEui).Return([]error{})
	operations := answeredDeregisterLog(numericTestEpEui)
	operations.On("UpdateOperationState", mock.Anything, int64(123), opID, mock.Anything, mock.Anything).Return(nil)
	server := &Server{
		registry:      newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:         testFrameCodec,
		commands:      mustTestCommandRegistry(),
		clock:         clock.SystemClock{},
		logger:        testLogger(),
		endpointSvc:   endpoints,
		dlSvc:         downlinks,
		operationRepo: operations,
		config:        &Config{},
	}
	session := &Session{ID: 123, TenantID: tenantID, State: StateActive}

	require.NoError(t, server.handleDeregisterComplete(&mockConn{}, session, opID))
	server.wg.Wait()

	downlinks.AssertCalled(t, "RevokeDownlink", mock.Anything, tenantDownlinkRef(tenantID, numericTestEpEui, pendingID))
	downlinks.AssertCalled(t, "RevokeDownlink", mock.Anything, tenantDownlinkRef(tenantID, numericTestEpEui, heldID))
}
