package scaci

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// originTestSessionID is the persisted Application Center session the result tests broadcast to.
const originTestSessionID = int64(56)

// Application Centers of coreTenantID: the one that queued the downlink and another one.
const (
	originQueuerAcEui = uint64(0x70B3D59CD0000A01)
	originOtherAcEui  = uint64(0x70B3D59CD0000A02)
)

// originTestServer serves one active, persisted Application Center session of
// coreTenantID and records every operation it initiates.
func originTestServer() (*Server, *mockConn, *operationLogCapture) {
	server := coreTestServer(new(MockDLService))
	recorder := &operationLogCapture{}
	server.operationRecorder = recorder
	server.sessionPersistence = sessionRowsOver{repo: acceptingCounterStore{}}
	conn := &mockConn{}
	server.registry.sessions[conn] = &Session{ID: originTestSessionID, TenantID: coreTenantID, AcEui: originQueuerAcEui, State: StateActive}
	return server, conn, recorder
}

// TestBroadcastDLDataResult_ForeignTenantDownlinkIsNotReported: a result is
// broadcast to the sessions of the downlink's owner tenant only, so another
// tenant's downlink, whatever its Application Center id, never reaches this
// tenant's sessions.
func TestBroadcastDLDataResult_ForeignTenantDownlinkIsNotReported(t *testing.T) {
	server, conn, recorder := originTestServer()
	const foreignTenant = coreTenantID + 1

	require.NoError(t, server.BroadcastDLDataResult(testutil.TestContext(), ApplicationCenter{TenantID: foreignTenant, AcEui: originQueuerAcEui}, coreACQueID, &mioty.DLDataResult{
		EpEui: coreEpEUI, QueId: uint64(coreInternalQueID), Result: mioty.ResultSent,
	}))

	assert.Empty(t, conn.written)
	assert.Empty(t, recorder.recorded)
}

// TestBroadcastDLDataResult_ReachesOnlyTheQueuingApplicationCenter: the queId
// of a dlDataRes is the one the queuing Application Center assigned (SCACI
// §3.10.1, §3.12.1), so another Application Center of the tenant, which never
// assigned it, is not sent the result.
func TestBroadcastDLDataResult_ReachesOnlyTheQueuingApplicationCenter(t *testing.T) {
	server, queuerConn, recorder := originTestServer()
	otherConn := &mockConn{}
	server.registry.sessions[otherConn] = &Session{ID: originTestSessionID + 1, TenantID: coreTenantID, AcEui: originOtherAcEui, State: StateActive}

	require.NoError(t, server.BroadcastDLDataResult(testutil.TestContext(), ApplicationCenter{TenantID: coreTenantID, AcEui: originQueuerAcEui}, coreACQueID, &mioty.DLDataResult{
		EpEui: coreEpEUI, QueId: uint64(coreInternalQueID), Result: mioty.ResultExpired,
	}))

	var sent DLDataResult
	require.NoError(t, decodeResponse(queuerConn.written, &sent))
	assert.Equal(t, coreACQueID, sent.QueID)
	assert.Empty(t, otherConn.written, "an Application Center that did not queue the downlink is not sent its result")
	assert.Len(t, recorder.recorded, 1)
}

// TestBroadcastDLDataResult_ReachesOnlyTheQueuingOrganization: a session of
// another organization's Application Center with the same acEui, connected
// or held for resume, is never sent the result of a downlink the owner
// queued.
func TestBroadcastDLDataResult_ReachesOnlyTheQueuingOrganization(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	server, ownerConn, recorder := originTestServer()
	server.registry.sessions[ownerConn].OrganizationID = owner
	otherConn := &mockConn{}
	server.registry.sessions[otherConn] = &Session{ID: originTestSessionID + 1, TenantID: coreTenantID, OrganizationID: other, AcEui: originQueuerAcEui, State: StateActive}
	holder, ok := server.registry.holder.(*holderFake)
	require.True(t, ok)
	holder.Hold(testutil.TestContext(), &Session{ID: originTestSessionID + 2, TenantID: coreTenantID, OrganizationID: other, AcEui: originQueuerAcEui})

	queuer := ApplicationCenter{TenantID: coreTenantID, OrganizationID: owner, AcEui: originQueuerAcEui}
	require.NoError(t, server.BroadcastDLDataResult(testutil.TestContext(), queuer, coreACQueID, &mioty.DLDataResult{
		EpEui: coreEpEUI, QueId: uint64(coreInternalQueID), Result: mioty.ResultExpired,
	}))

	assert.NotEmpty(t, ownerConn.written, "the queuing organization's Application Center gets its result")
	assert.Empty(t, otherConn.written, "another organization's session of the acEui is not sent it")
	assert.Empty(t, holder.recorded(), "nor is it recorded for another organization's held session")
	assert.Len(t, recorder.recorded, 1)
}
