package bssciservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// endedDownlinkOutcomes answers every station outcome as the queue does for a
// downlink that already ended.
type endedDownlinkOutcomes struct {
	mockMIOTYDownlinksForDispatch
}

func (endedDownlinkOutcomes) UpdateDownlinkResult(context.Context, int64, uint64, *mioty.DLDataResult) (*storage.DownlinkMessage, error) {
	return nil, storage.ErrDownlinkFinished
}

func (endedDownlinkOutcomes) RevokeDownlink(context.Context, storage.DownlinkRevocation) (bool, error) {
	return false, nil
}

func newEndedDownlinkService(t *testing.T, f *reporterFixture, resolver bssci.TenantResolver) bssci.DownlinkService {
	t.Helper()
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: resolver, Outcomes: &endedDownlinkOutcomes{}, Holders: &mockMIOTYDownlinksForDispatch{},
		Results: f.reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	})
	require.NoError(t, err)
	return svc
}

// TestProcessDLDataResult_AnEndedDownlinkKeepsItsOutcome: a station that
// transmits or drops a downlink the service center already expired or revoked
// gets its dlDataResRsp, and the originators are not told a second result.
func TestProcessDLDataResult_AnEndedDownlinkKeepsItsOutcome(t *testing.T) {
	f := newReporterFixture(t)
	resolver := NewTenantResolver(nil)
	resolver.RegisterQueueTenant(reporterQueueID, "3")
	svc := newEndedDownlinkService(t, f, resolver)

	response, err := svc.ProcessDLDataResult(testutil.TestContext(), reporterStation(),
		&mioty.DLDataResult{EpEui: reporterEndpointEUI, QueId: uint64(reporterQueueID), Result: mioty.ResultSent})
	f.stop(t)

	require.NoError(t, err)
	assert.Equal(t, mioty.CmdDLDataResultResponse, response["command"])
	assert.Empty(t, f.acs.delivered)
	assert.Empty(t, f.mqtt.published)
	assert.Nil(t, f.events.lastEvent)
	_, stillMapped := resolver.ResolveTenant(testutil.TestContext(), reporterQueueID)
	assert.Error(t, stillMapped, "the queue-to-tenant mapping is released")
}

// TestProcessRevokeResponse_AnEndedDownlinkIsNotRevoked: the confirmation of
// a revoke sent after the downlink expired completes the operation and
// reports that nothing was revoked.
func TestProcessRevokeResponse_AnEndedDownlinkIsNotRevoked(t *testing.T) {
	resolver := NewTenantResolver(nil)
	resolver.RegisterQueueTenant(reporterQueueID, "3")
	svc := newEndedDownlinkService(t, newReporterFixture(t), resolver)

	response, revoked, err := svc.ProcessRevokeResponse(testutil.TestContext(), reporterStation(), -12, reporterQueueID, reporterEndpointEUI)

	require.NoError(t, err)
	assert.False(t, revoked)
	assert.Equal(t, mioty.CmdDLDataRevokeComplete, response["command"])
}

// answeredRevocations records every revocation and answers it as not held.
type answeredRevocations struct {
	endedDownlinkOutcomes
	revocations []storage.DownlinkRevocation
}

func (a *answeredRevocations) RevokeDownlink(_ context.Context, revocation storage.DownlinkRevocation) (bool, error) {
	a.revocations = append(a.revocations, revocation)
	return false, nil
}

// TestRevokeAnswers_NameTheAnsweringStation: the confirmation of a dlDataRev
// (BSSCI §3.13) and its refusal (§3.17) both revoke only what the answering
// station holds, under the downlink's owner tenant.
func TestRevokeAnswers_NameTheAnsweringStation(t *testing.T) {
	resolver := NewTenantResolver(nil)
	outcomes := &answeredRevocations{}
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: resolver, Outcomes: outcomes, Holders: &mockMIOTYDownlinksForDispatch{},
		Results: newReporterFixture(t).reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	})
	require.NoError(t, err)

	resolver.RegisterQueueTenant(reporterQueueID, "3")
	_, _, err = svc.ProcessRevokeResponse(testutil.TestContext(), reporterStation(), -12, reporterQueueID, reporterEndpointEUI)
	require.NoError(t, err)
	resolver.RegisterQueueTenant(reporterQueueID, "3")
	_, err = svc.ProcessRevokeRefusal(testutil.TestContext(), reporterStation(), bssci.RevokeRefusal{QueueID: reporterQueueID, EndpointEUI: reporterEndpointEUI})
	require.NoError(t, err)

	station := reporterStationEUI
	answered := storage.DownlinkRevocation{QueID: reporterQueueID, TenantID: 3, Station: &station}
	assert.Equal(t, []storage.DownlinkRevocation{answered, answered}, outcomes.revocations)
}
