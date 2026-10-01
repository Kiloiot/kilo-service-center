package bssci

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	decisionTestTenant   = int64(64)
	decisionTestEndpoint = int64(6401)
	decisionTestStation  = uint64(0x70B3D59CD00009E6)
	decisionTestEpEUI    = uint64(0x0001020304050607)
)

// recordingDecider records the decisions the handlers take.
type recordingDecider struct {
	mu        sync.Mutex
	decisions []AttachmentDecision
}

func (d *recordingDecider) Decide(_ context.Context, decision AttachmentDecision) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.decisions = append(d.decisions, decision)
	return true, nil
}

func (d *recordingDecider) taken() []AttachmentDecision {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]AttachmentDecision(nil), d.decisions...)
}

func newDecisionTestServer(t *testing.T) (*Server, *recordingDecider, *Session) {
	t.Helper()
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, decisionTestTenant,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, nil, queueSerializer, auditLogger, tenantResolver)
	server.SetEndpointRepository(detachableEndpointRepo(decisionTestEpEUI, decisionTestTenant, decisionTestEndpoint))
	decider := &recordingDecider{}
	server.SetAttachmentDecider(decider)
	session := &Session{ProtocolSessionState: ProtocolSessionState{
		ID: "decision-station", BaseStationEUI: decisionTestStation, ResolvedTenantID: decisionTestTenant, DbSessionID: 64, Encoding: EncodingJSON,
	}}
	return server, decider, session
}

// An attach a base station completed is the owner's attachment decision,
// reported with the station that heard it and the fields it reported
// (BSSCI §3.6.3, SCACI §3.13.1).
func TestCompletedAttachIsAnOverTheAirAttachmentDecision(t *testing.T) {
	server, decider, session := newDecisionTestServer(t)
	const opID = int64(6402)
	require.NoError(t, server.statusSvc.RecordPendingOperation(testutil.TestContext(), session, opID, &PendingOperation{
		OperationID: opID, OperationType: mioty.CmdAttach,
		Metadata: map[string]interface{}{
			"epEui": int64(decisionTestEpEUI), metadataKeyEndpointID: decisionTestEndpoint, metadataKeyEndpointTenantID: decisionTestTenant,
			"attachCnt": int64(3), "snr": 11.5, "rssi": -70.0,
		},
	}, decisionTestTenant))

	require.NoError(t, server.CallHandleAttachComplete(session, &Message{Command: mioty.CmdAttachComplete, OpId: opID}, nil))
	server.wg.Wait()

	decisions := decider.taken()
	require.Len(t, decisions, 1)
	decision := decisions[0]
	assert.Equal(t, decisionTestTenant, decision.TenantID)
	assert.Equal(t, decisionTestEndpoint, decision.EndpointID)
	assert.Equal(t, decisionTestEpEUI, decision.EpEUI)
	assert.Equal(t, EndpointStatusAttached, decision.Status)
	require.NotNil(t, decision.OverTheAir)
	assert.Equal(t, decisionTestStation, decision.OverTheAir.BaseStationEUI)
	require.NotNil(t, decision.OverTheAir.Status.AttachCnt)
	assert.Equal(t, uint32(3), *decision.OverTheAir.Status.AttachCnt)
}

// A detach a base station completed is the owner's detachment decision,
// reported with the station that heard it and the det's telemetry.
func TestCompletedDetachIsAnOverTheAirDetachmentDecision(t *testing.T) {
	server, decider, session := newDecisionTestServer(t)
	const opID = int64(6403)
	require.NoError(t, server.statusSvc.RecordPendingOperation(testutil.TestContext(), session, opID, &PendingOperation{
		OperationID: opID, OperationType: mioty.CmdDetach,
		Metadata: map[string]interface{}{
			"epEui": float64(decisionTestEpEUI), "packetCnt": float64(100), "rxTime": float64(1699876543000),
			"snr": 12.5, "rssi": -75.0, "signature": []byte{1, 2, 3, 4},
			"endpointID": float64(decisionTestEndpoint), "tenantId": float64(decisionTestTenant),
		},
	}, decisionTestTenant))

	require.NoError(t, server.CallHandleDetachComplete(session, &Message{Command: mioty.CmdDetachComplete, OpId: opID}, nil))
	server.wg.Wait()

	decisions := decider.taken()
	require.Len(t, decisions, 1)
	decision := decisions[0]
	assert.Equal(t, decisionTestTenant, decision.TenantID)
	assert.Equal(t, decisionTestEndpoint, decision.EndpointID)
	assert.Equal(t, endpoint.EndpointStatusDetached, decision.Status)
	require.NotNil(t, decision.OverTheAir)
	assert.Equal(t, decisionTestStation, decision.OverTheAir.BaseStationEUI)
	require.NotNil(t, decision.OverTheAir.Telemetry)
	assert.Equal(t, []byte{1, 2, 3, 4}, decision.OverTheAir.Telemetry.Sign)
	require.NotNil(t, decision.OverTheAir.Telemetry.PacketCnt)
	assert.Equal(t, uint32(100), *decision.OverTheAir.Telemetry.PacketCnt)
}
