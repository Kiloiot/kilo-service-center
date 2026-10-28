package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	bpresolver "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// recordingPinner answers a fixed result and records whose endpoints it was asked to pin.
type recordingPinner struct {
	results []bpresolver.PinResult
	err     error
	tenants []int64
	dryRuns []bool
}

func (p *recordingPinner) Pin(_ context.Context, tenantID int64, dryRun bool) ([]bpresolver.PinResult, error) {
	p.tenants = append(p.tenants, tenantID)
	p.dryRuns = append(p.dryRuns, dryRun)
	return p.results, p.err
}

var pinResults = []bpresolver.PinResult{
	{EpEUI: "70B3D56770111501", Outcome: bpresolver.PinOutcomePinned, BlueprintID: "8c1f8f6e-3b0e-4d4c-9d8e-0f5a3f0c2b11"},
	{EpEUI: "70B3D56770111502", Outcome: bpresolver.PinOutcomeNothingToPin},
	{EpEUI: "70B3D56770111503", Outcome: bpresolver.PinOutcomeError, Error: "blueprint lookup failed"},
}

// TC-CM-101: pinning acts on the caller's tenant only and is recorded.
func TestPinCurrentBlueprints_PinsTheCallersTenantAndIsRecorded(t *testing.T) {
	pinner := &recordingPinner{results: pinResults}
	recorder := &captureAuditRecorder{}
	svc := testCoreService(coreFields{pinner: pinner, audit: recorder, log: &mockLogger{}})

	resp, err := svc.PinCurrentBlueprints(testutil.TestContextWithTenant(modeTenant), &pb.PinCurrentBlueprintsRequest{})

	require.NoError(t, err)
	assert.Equal(t, []int64{modeTenant}, pinner.tenants, "only the caller's tenant")
	require.Len(t, resp.GetResults(), 3)
	assert.Equal(t, "70B3D56770111501", resp.GetResults()[0].GetEpEui())
	assert.Equal(t, "pinned", resp.GetResults()[0].GetOutcome())
	assert.Equal(t, pinResults[0].BlueprintID, resp.GetResults()[0].GetBlueprintId())
	assert.Equal(t, "nothing_to_pin", resp.GetResults()[1].GetOutcome())
	assert.Equal(t, "error", resp.GetResults()[2].GetOutcome())
	assert.Equal(t, "blueprint lookup failed", resp.GetResults()[2].GetError())
	require.Len(t, recorder.events, 1)
	ev := recorder.events[0]
	assert.Equal(t, modeTenant, ev.TenantID)
	assert.Equal(t, models.EventCategoryEndpoint, ev.Category)
	assert.Equal(t, models.EventTypeBlueprintsPinned, ev.EventType)
	assert.Equal(t, 1, ev.Details[models.EventDetailKeyPinnedCount])
}

// TC-CM-108: a pass that pins nothing is recorded too, with a zero count.
func TestPinCurrentBlueprints_APassThatPinsNothingIsRecorded(t *testing.T) {
	for name, results := range map[string][]bpresolver.PinResult{
		"no endpoint without a snapshot": nil,
		"nothing to pin and a failure":   pinResults[1:],
	} {
		t.Run(name, func(t *testing.T) {
			recorder := &captureAuditRecorder{}
			svc := testCoreService(coreFields{pinner: &recordingPinner{results: results}, audit: recorder, log: &mockLogger{}})

			resp, err := svc.PinCurrentBlueprints(testutil.TestContextWithTenant(modeTenant), &pb.PinCurrentBlueprintsRequest{})

			require.NoError(t, err)
			assert.Len(t, resp.GetResults(), len(results))
			require.Len(t, recorder.events, 1)
			ev := recorder.events[0]
			assert.Equal(t, modeTenant, ev.TenantID)
			assert.Equal(t, models.EventTypeBlueprintsPinned, ev.EventType)
			assert.Equal(t, 0, ev.Details[models.EventDetailKeyPinnedCount])
		})
	}
}

func TestPinCurrentBlueprints_ADryRunIsNotRecorded(t *testing.T) {
	pinner := &recordingPinner{results: pinResults}
	recorder := &captureAuditRecorder{}
	svc := testCoreService(coreFields{pinner: pinner, audit: recorder, log: &mockLogger{}})

	resp, err := svc.PinCurrentBlueprints(testutil.TestContextWithTenant(modeTenant), &pb.PinCurrentBlueprintsRequest{DryRun: true})

	require.NoError(t, err)
	assert.Equal(t, []bool{true}, pinner.dryRuns)
	assert.Len(t, resp.GetResults(), 3)
	assert.Empty(t, recorder.events)
}

func TestPinCurrentBlueprints_AFailedPassIsRefusedAndNotRecorded(t *testing.T) {
	recorder := &captureAuditRecorder{}
	svc := testCoreService(coreFields{pinner: &recordingPinner{err: errDbDown}, audit: recorder, log: &mockLogger{}})

	_, err := svc.PinCurrentBlueprints(testutil.TestContextWithTenant(modeTenant), &pb.PinCurrentBlueprintsRequest{})

	requireCode(t, err, codes.Internal)
	assert.Empty(t, recorder.events)
}

// TC-CM-101: pinning needs the endpoint manager role, which an organization's key has.
func TestPinCurrentBlueprints_IsAnEndpointManagerCall(t *testing.T) {
	pin := pb.CoreService_PinCurrentBlueprints_FullMethodName
	assert.True(t, admitted(t, matrixInterceptor(authz.ServiceAccountRoles), pin, false), "organization key")
	assert.False(t, admitted(t, matrixInterceptor(authz.Roles{BaseStationManager: true}), pin, false), "base station manager")
}
