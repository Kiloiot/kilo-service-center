package bssciservices

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	reconcileTenant             = int64(100)
	reconcileDetachedWhileAway  = uint64(0x70B3D56770111505)
	reconcileDetachedBeforehand = uint64(0x70B3D56770111506)
	reconcileNeverAttached      = uint64(0x70B3D56770111507)
	reconcileAttached           = uint64(0x70B3D56770111508)
	reconcileAttachedWhileAway  = uint64(0x70B3D56770111509)
	reconcileAttachedUnrecorded = uint64(0x70B3D5677011150A)
	// reconcileHeardByOtherStation hears an over-the-air attach while the reconciled station is away.
	reconcileHeardByOtherStation = uint64(0x70B3D59CD00009E7)
)

func reconcileEndpoint(id int64, eui uint64, epStatus string) *models.EndPoint {
	return &models.EndPoint{ID: id, TenantID: reconcileTenant, EUI: models.EUI(mioty.EUI64(eui).ToBytes()), EpStatus: epStatus, Bidi: true}
}

// BSSCI §1, §3.9: a station that resumes its session kept the endpoints it
// held, so it is sent detPrp for each endpoint the service center detached
// while it was away. A station that was still connected when the endpoint was
// detached was sent the detachment then, and a new session discarded what it
// held; neither is sent a detachment again.
func TestReconcileSendsAResumedStationOnlyTheDetachmentsItMissed(t *testing.T) {
	stationBLeft := time.Unix(1_800_000_000, 0)
	detachedBeforehand := stationBLeft.Add(-time.Minute)
	detachedMeanwhile := stationBLeft.Add(time.Minute)
	stationALeft := detachedMeanwhile.Add(time.Minute)

	env := newPropagationTestEnv(t,
		reconcileEndpoint(1, reconcileDetachedWhileAway, endpoint.EndpointStatusDetached),
		reconcileEndpoint(2, reconcileDetachedBeforehand, endpoint.EndpointStatusDetached),
		reconcileEndpoint(3, reconcileNeverAttached, endpoint.EndpointStatusDetached),
		reconcileEndpoint(4, reconcileAttached, pkgbssci.EndpointStatusAttached),
	)
	env.repo.attachmentChanged(1, detachedMeanwhile)
	env.repo.attachmentChanged(2, detachedBeforehand)
	stations := []propagation.BaseStationSession{
		{ID: "station-b-away", TenantID: reconcileTenant, Resumed: true, DisconnectedAt: &stationBLeft},
		{ID: "station-a-stayed", TenantID: reconcileTenant, Resumed: true, DisconnectedAt: &stationALeft},
		{ID: "station-c-new", TenantID: reconcileTenant},
	}

	for _, station := range stations {
		require.NoError(t, env.service.ReconcileBaseStation(testutil.TestContext(), station, nil))
	}

	detached := map[string][]uint64{}
	for _, call := range env.sender.detaches {
		detached[call.sessionID] = append(detached[call.sessionID], call.endpointEUI)
	}
	assert.Equal(t, map[string][]uint64{"station-b-away": {reconcileDetachedWhileAway}}, detached,
		"only the station away during the detachment is sent it, once")
	assert.Equal(t, map[string][]int64{"station-c-new": {4}}, attachedBySession(env.sender),
		"only the new session is sent the endpoint attached before either station left")
}

// BSSCI §1, §3.8: a resumed station kept the attachments it held and the
// downlinks it queued for them, and an attPrp for an endpoint it holds makes
// it discard those downlinks. It is sent attPrp only for the endpoints whose
// attachment changed while it was away, or for every attached endpoint when
// that time is unknown; a new session holds nothing and is sent them all.
func TestReconcileSendsAResumedStationOnlyTheAttachmentsItMissed(t *testing.T) {
	stationLeft := time.Unix(1_800_000_000, 0)
	env := newPropagationTestEnv(t,
		reconcileEndpoint(4, reconcileAttached, pkgbssci.EndpointStatusAttached),
		reconcileEndpoint(5, reconcileAttachedWhileAway, pkgbssci.EndpointStatusAttached),
		reconcileEndpoint(6, reconcileAttachedUnrecorded, pkgbssci.EndpointStatusAttached),
	)
	env.repo.attachmentChanged(4, stationLeft.Add(-time.Minute))
	env.repo.attachmentChanged(5, stationLeft.Add(time.Minute))
	stations := []propagation.BaseStationSession{
		{ID: "station-resumed", TenantID: reconcileTenant, Resumed: true, DisconnectedAt: &stationLeft},
		{ID: "station-resumed-disconnect-unknown", TenantID: reconcileTenant, Resumed: true},
		{ID: "station-new", TenantID: reconcileTenant},
	}

	for _, station := range stations {
		require.NoError(t, env.service.ReconcileBaseStation(testutil.TestContext(), station, nil))
	}

	assert.Equal(t, map[string][]int64{
		"station-resumed":                    {5},
		"station-resumed-disconnect-unknown": {4, 5, 6},
		"station-new":                        {4, 5, 6},
	}, attachedBySession(env.sender))
}

// attachedBySession lists the endpoints each station was sent attPrp for.
func attachedBySession(sender *mockAttachPropagateSender) map[string][]int64 {
	attached := map[string][]int64{}
	for _, call := range sender.calls {
		attached[call.sessionID] = append(attached[call.sessionID], call.endpoint.ID)
	}
	return attached
}

// A resumed station whose disconnect time is unknown is sent every detachment
// the service center recorded, so none it may still hold is missed.
func TestReconcileSendsAResumedStationOfUnknownDisconnectEveryDetachment(t *testing.T) {
	detachedAt := time.Unix(1_800_000_000, 0)
	env := newPropagationTestEnv(t,
		reconcileEndpoint(1, reconcileDetachedWhileAway, endpoint.EndpointStatusDetached),
		reconcileEndpoint(3, reconcileNeverAttached, endpoint.EndpointStatusDetached),
	)
	env.repo.attachmentChanged(1, detachedAt)

	require.NoError(t, env.service.ReconcileBaseStation(testutil.TestContext(),
		propagation.BaseStationSession{ID: "station-legacy", TenantID: reconcileTenant, Resumed: true}, nil))

	require.Len(t, env.sender.detaches, 1)
	assert.Equal(t, reconcileDetachedWhileAway, env.sender.detaches[0].endpointEUI)
}

// The resumed session carries the time its previous connection was lost, as
// the session row recorded it.
func TestHandleResume_CarriesThePreviousDisconnectTime(t *testing.T) {
	svc, repo := newResumeTestService(t, reconcileTenant)
	scUUID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	bsUUID := [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20}
	disconnectedAt := time.Unix(1_800_000_000, 0).UTC()
	row := seedResumableSession(repo, 42, reconcileTenant, bsUUID, scUUID, 1, -1)
	row.EndedAt = &disconnectedAt

	outcome := svc.HandleResume(testutil.TestContext(),
		&pkgbssci.Session{ProtocolSessionState: pkgbssci.ProtocolSessionState{ResolvedTenantID: reconcileTenant}},
		bsUUID[:], nil, nil, 0xAABB)

	require.Equal(t, pkgbssci.ResumeCompatible, outcome.Disposition)
	require.NotNil(t, outcome.Previous.DisconnectedAt, "the resumed session knows when its connection was lost")
	assert.True(t, disconnectedAt.Equal(*outcome.Previous.DisconnectedAt))
}
