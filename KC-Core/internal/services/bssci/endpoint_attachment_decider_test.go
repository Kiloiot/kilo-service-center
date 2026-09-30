package bssciservices

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	deciderTestStation = uint64(0x70B3D59CD00009E6)
	deciderTestEpEUI   = uint64(0x70B3D56770111505)
)

var errDeciderTestStore = errors.New("status store unavailable")

func newTestDecider(t *testing.T, epStatus string) (*endpointAttachmentDecider, *decisionEndpoints, *recordingNotifier) {
	t.Helper()
	endpoints := &decisionEndpoints{endpoint: models.EndPoint{ID: decisionTestEndpoint, EpStatus: epStatus}}
	notices := &recordingNotifier{}
	decider, err := NewEndpointAttachmentDecider(endpoints, notices)
	require.NoError(t, err)
	return decider.(*endpointAttachmentDecider), endpoints, notices
}

func overTheAir(status string) pkgbssci.AttachmentDecision {
	snr := 12.5
	nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
	return pkgbssci.AttachmentDecision{
		TenantID: decisionTestTenant, EndpointID: decisionTestEndpoint, EpEUI: deciderTestEpEUI, Status: status,
		OverTheAir: &pkgbssci.OverTheAirReport{
			BaseStationEUI: deciderTestStation,
			Status:         &pkgbssci.EPStatusData{Snr: &snr, Nonce: &nonce},
		},
	}
}

// Every over-the-air attach starts a new attachment, with its own nonce and
// signature, so its owner is told of each one (SCACI §3.13.1) with the
// fields the station reported and the station that heard it.
func TestDeciderAnnouncesEveryOverTheAirAttach(t *testing.T) {
	decider, endpoints, notices := newTestDecider(t, endpoint.EndpointStatusDetached)

	for range 2 {
		_, err := decider.Decide(testutil.TestContext(), overTheAir(pkgbssci.EndpointStatusAttached))
		require.NoError(t, err)
	}

	assert.Equal(t, pkgbssci.EndpointStatusAttached, endpoints.status())
	told := notices.told()
	require.Len(t, told, 2)
	notice := told[1]
	assert.Equal(t, decisionTestTenant, notice.TenantID)
	assert.Equal(t, deciderTestEpEUI, notice.Status.EpEui)
	assert.Equal(t, pkgbssci.EndpointStatusAttached, notice.Status.EpStatus)
	require.NotNil(t, notice.Status.Nonce)
	assert.Equal(t, mioty.Numeric4{0x01, 0x02, 0x03, 0x04}, *notice.Status.Nonce)
	require.NotNil(t, notice.HeardBy)
	assert.Equal(t, deciderTestStation, *notice.HeardBy)
}

// A detach heard by several stations detaches the endpoint once and is
// announced once.
func TestDeciderAnnouncesAnOverTheAirDetachOnce(t *testing.T) {
	decider, endpoints, notices := newTestDecider(t, pkgbssci.EndpointStatusAttached)

	first, err := decider.Decide(testutil.TestContext(), overTheAir(endpoint.EndpointStatusDetached))
	require.NoError(t, err)
	second, err := decider.Decide(testutil.TestContext(), overTheAir(endpoint.EndpointStatusDetached))
	require.NoError(t, err)

	assert.True(t, first)
	assert.False(t, second)
	assert.Equal(t, endpoint.EndpointStatusDetached, endpoints.status())
	assert.Equal(t, []string{endpoint.EndpointStatusDetached}, notices.statuses())
}

type failingStatusStore struct{}

func (failingStatusStore) TransitionEndpointStatus(context.Context, int64, int64, string) (bool, error) {
	return false, errDeciderTestStore
}

func (f failingStatusStore) RestateEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error) {
	return f.TransitionEndpointStatus(ctx, tenantID, endpointID, status)
}

func (failingStatusStore) EndpointDetachStateUpdate(context.Context, int64, int64, models.EndpointDetachStateParams) error {
	return errDeciderTestStore
}

// A decision that could not be recorded is not announced.
func TestDeciderAnnouncesNoDecisionItCouldNotRecord(t *testing.T) {
	notices := &recordingNotifier{}
	decider, err := NewEndpointAttachmentDecider(failingStatusStore{}, notices)
	require.NoError(t, err)

	for _, decision := range []pkgbssci.AttachmentDecision{overTheAir(pkgbssci.EndpointStatusAttached), overTheAir(endpoint.EndpointStatusDetached)} {
		_, err := decider.Decide(testutil.TestContext(), decision)
		require.ErrorIs(t, err, errDeciderTestStore)
	}
	assert.Empty(t, notices.told())
}

func TestDeciderRefusesAnUnknownDecision(t *testing.T) {
	decider, endpoints, notices := newTestDecider(t, endpoint.EndpointStatusDetached)

	_, err := decider.Decide(testutil.TestContext(), pkgbssci.AttachmentDecision{TenantID: decisionTestTenant, Status: endpoint.EndpointStatusAttaching})

	require.Error(t, err)
	assert.Equal(t, endpoint.EndpointStatusDetached, endpoints.status())
	assert.Empty(t, notices.told())
}

func TestNewEndpointAttachmentDeciderRefusesMissingCollaborators(t *testing.T) {
	_, err := NewEndpointAttachmentDecider(nil, &recordingNotifier{})
	assert.ErrorIs(t, err, errNilStatusTransitioner)
	_, err = NewEndpointAttachmentDecider(failingStatusStore{}, nil)
	assert.ErrorIs(t, err, errNilEndpointStatusNotifier)
}

// contextNotifier records the context each notice was told on.
type contextNotifier struct {
	mu       sync.Mutex
	contexts []context.Context
}

func (n *contextNotifier) NotifyEndpointStatus(ctx context.Context, _ EndpointStatusNotice) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.contexts = append(n.contexts, ctx)
}

// The fan-out tells every notifier on the background work, scoped to the
// owner and still alive after the request that took the decision ended.
func TestEndpointStatusFanoutTellsEveryNotifierDetachedFromTheRequest(t *testing.T) {
	first, second := &contextNotifier{}, &contextNotifier{}
	work := NewBackgroundWork()
	fanout, err := NewEndpointStatusFanout(work, first, second)
	require.NoError(t, err)

	request, finish := context.WithCancel(testutil.TestContext())
	finish()
	fanout.NotifyEndpointStatus(request, EndpointStatusNotice{TenantID: decisionTestTenant, Status: &pkgbssci.EPStatusData{EpEui: deciderTestEpEUI}})
	require.NoError(t, work.Stop(testutil.TestContext()))

	for _, n := range []*contextNotifier{first, second} {
		require.Len(t, n.contexts, 1)
		assert.NoError(t, n.contexts[0].Err(), "the request's cancellation does not reach the notifier")
		tenant, err := pkgcontext.GetTenantID(n.contexts[0])
		require.NoError(t, err)
		assert.Equal(t, decisionTestTenant, tenant)
	}
}

func TestNewEndpointStatusFanoutRefusesMissingCollaborators(t *testing.T) {
	_, err := NewEndpointStatusFanout(nil)
	assert.ErrorIs(t, err, errNilBackgroundRunner)
	_, err = NewEndpointStatusFanout(NewBackgroundWork(), nil)
	assert.ErrorIs(t, err, errNilEndpointStatusNotifier)
}

type recordingEPStat struct {
	tenants  []int64
	statuses []*pkgbssci.EPStatusData
}

func (r *recordingEPStat) BroadcastEPStatus(_ context.Context, tenantID int64, data *pkgbssci.EPStatusData) error {
	r.tenants = append(r.tenants, tenantID)
	r.statuses = append(r.statuses, data)
	return nil
}

func TestEPStatNotifierSendsTheOwnerItsEPStat(t *testing.T) {
	broadcaster := &recordingEPStat{}
	notifier, err := NewEPStatNotifier(broadcaster, logger.NewNop())
	require.NoError(t, err)
	status := &pkgbssci.EPStatusData{EpEui: deciderTestEpEUI, EpStatus: pkgbssci.EndpointStatusAttached}

	notifier.NotifyEndpointStatus(testutil.TestContext(), EndpointStatusNotice{TenantID: decisionTestTenant, Status: status})

	assert.Equal(t, []int64{decisionTestTenant}, broadcaster.tenants)
	assert.Equal(t, []*pkgbssci.EPStatusData{status}, broadcaster.statuses)
}

type attachmentEventCall struct {
	event   string
	org     string
	epEUI   uint64
	heardBy *uint64
}

type recordingAttachmentEvents struct {
	calls []attachmentEventCall
}

func (r *recordingAttachmentEvents) PublishAttach(_ context.Context, org string, epEUI uint64, heardBy *uint64) error {
	r.calls = append(r.calls, attachmentEventCall{event: pkgbssci.EndpointStatusAttached, org: org, epEUI: epEUI, heardBy: heardBy})
	return nil
}

func (r *recordingAttachmentEvents) PublishDetach(_ context.Context, org string, epEUI uint64, heardBy *uint64) error {
	r.calls = append(r.calls, attachmentEventCall{event: endpoint.EndpointStatusDetached, org: org, epEUI: epEUI, heardBy: heardBy})
	return nil
}

type tenantOrganizations map[int64]uuid.UUID

func (o tenantOrganizations) GetDefaultOrgForTenant(_ context.Context, tenantID int64) (uuid.UUID, error) {
	return o[tenantID], nil
}

// An attach decided in the service center is published once to the owner's
// MQTT event/attach, without the station fields an over-the-air attach has.
func TestAttachThroughTheServiceCenterPublishesOneMQTTAttachEvent(t *testing.T) {
	endpoints := &decisionEndpoints{endpoint: models.EndPoint{
		ID: decisionTestEndpoint, EUI: models.EUIFromString(decisionTestEUI), EpStatus: endpoint.EndpointStatusDetached, NwkSnKey: decisionKey(),
	}}
	org := uuid.New()
	events := &recordingAttachmentEvents{}
	mqttNotifier, err := NewMQTTAttachmentNotifier(events, tenantOrganizations{decisionTestTenant: org}, logger.NewNop())
	require.NoError(t, err)
	work := NewBackgroundWork()
	fanout, err := NewEndpointStatusFanout(work, mqttNotifier)
	require.NoError(t, err)
	decider, err := NewEndpointAttachmentDecider(endpoints, fanout)
	require.NoError(t, err)
	propagation, err := NewAttachmentPropagation(idleStations{}, fixedSessionKeys{}, discardedEvents{}, work, clock.SystemClock{}, logger.NewNop())
	require.NoError(t, err)
	svc, err := NewEndpointAttachmentService(endpoints, endpoints, decider, fanout, propagation)
	require.NoError(t, err)

	for range 2 {
		_, err := svc.AttachEndPoint(testutil.TestContext(), decisionTestEUI, decisionTestTenant)
		require.NoError(t, err)
	}
	require.NoError(t, work.Stop(testutil.TestContext()))

	require.Len(t, events.calls, 1, "one attachment publishes one MQTT event/attach")
	assert.Equal(t, attachmentEventCall{event: pkgbssci.EndpointStatusAttached, org: org.String(), epEUI: deciderTestEpEUI}, events.calls[0])
}

// An over-the-air detach is published to the owner's MQTT event/detach with
// the station that heard it; an owner without an organization gets none.
func TestMQTTAttachmentNotifierPublishesUnderTheOwnersOrganization(t *testing.T) {
	org := uuid.New()
	events := &recordingAttachmentEvents{}
	notifier, err := NewMQTTAttachmentNotifier(events, tenantOrganizations{decisionTestTenant: org}, logger.NewNop())
	require.NoError(t, err)
	heardBy := deciderTestStation
	detached := &pkgbssci.EPStatusData{EpEui: deciderTestEpEUI, EpStatus: endpoint.EndpointStatusDetached}

	notifier.NotifyEndpointStatus(testutil.TestContext(), EndpointStatusNotice{TenantID: decisionTestTenant, Status: detached, HeardBy: &heardBy})
	notifier.NotifyEndpointStatus(testutil.TestContext(), EndpointStatusNotice{TenantID: decisionTestTenant + 1, Status: detached, HeardBy: &heardBy})

	require.Len(t, events.calls, 1, "an owner without an organization gets no MQTT event")
	assert.Equal(t, attachmentEventCall{event: endpoint.EndpointStatusDetached, org: org.String(), epEUI: deciderTestEpEUI, heardBy: &heardBy}, events.calls[0])
}

func TestNewEndpointStatusNotifiersRefuseMissingCollaborators(t *testing.T) {
	_, err := NewEPStatNotifier(nil, logger.NewNop())
	assert.ErrorIs(t, err, errNilEPStatusBroadcaster)
	_, err = NewEPStatNotifier(&recordingEPStat{}, nil)
	assert.ErrorIs(t, err, errNilEndpointStatusLogger)
	_, err = NewMQTTAttachmentNotifier(nil, tenantOrganizations{}, logger.NewNop())
	assert.ErrorIs(t, err, errNilAttachmentEventPublisher)
	_, err = NewMQTTAttachmentNotifier(&recordingAttachmentEvents{}, nil, logger.NewNop())
	assert.ErrorIs(t, err, errNilOwnerOrganizations)
	_, err = NewMQTTAttachmentNotifier(&recordingAttachmentEvents{}, tenantOrganizations{}, nil)
	assert.ErrorIs(t, err, errNilEndpointStatusLogger)
}
