package downlinks

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	serviceTenant  = int64(4)
	serviceEpEUI   = uint64(0x70b3d56770111505)
	serviceStation = uint64(0x70b3d59cd00009e6)
	serviceQueID   = int64(77)
)

var serviceOrg = uuid.MustParse("fe7fe002-6880-4ea6-84ed-a69911dbdf8c")

type queuerFake struct {
	req    *mioty.DLDataQueue
	orgID  *uuid.UUID
	result *scaci.DLDataQueueResult
	err    error
}

func (f *queuerFake) QueueDownlinkInternal(_ context.Context, _ int64, orgID *uuid.UUID, req *mioty.DLDataQueue) (*scaci.DLDataQueueResult, error) {
	f.req, f.orgID = req, orgID
	return f.result, f.err
}

type editorFake struct {
	calls   int
	orgID   *uuid.UUID
	epEUI   []byte
	patch   storage.DownlinkPatch
	updated *storage.DownlinkMessage
	err     error
}

func (f *editorFake) UpdatePendingDownlink(_ context.Context, _ int64, orgID *uuid.UUID, epEUI []byte, _ int64, patch storage.DownlinkPatch) (*storage.DownlinkMessage, error) {
	f.calls++
	f.orgID, f.epEUI, f.patch = orgID, epEUI, patch
	if f.err != nil {
		return nil, f.err
	}
	return f.updated, nil
}

// updateEventsFake records the pending downlinks announced as updated.
type updateEventsFake struct {
	updated []*storage.DownlinkMessage
	err     error
}

func (f *updateEventsFake) RecordPendingUpdated(_ context.Context, downlink *storage.DownlinkMessage) error {
	f.updated = append(f.updated, downlink)
	return f.err
}

type revokerFake struct {
	ref     *scheduler.DownlinkRef
	station uint64
	err     error
}

func (f *revokerFake) RevokeDownlink(_ context.Context, ref scheduler.DownlinkRef) (uint64, error) {
	f.ref = &ref
	return f.station, f.err
}

type endpointsFake struct{ err error }

func (f endpointsFake) GetByEUI(context.Context, []byte, int64) (*models.EndPoint, error) {
	return &models.EndPoint{}, f.err
}

type auditFake struct{ events []audit.Event }

func (f *auditFake) Record(_ context.Context, ev audit.Event) { f.events = append(f.events, ev) }

type serviceFakes struct {
	queuer    *queuerFake
	editor    *editorFake
	revoker   *revokerFake
	endpoints endpointsFake
	events    *updateEventsFake
	audit     *auditFake
}

func newServiceFakes() serviceFakes {
	return serviceFakes{
		queuer:  &queuerFake{result: &scaci.DLDataQueueResult{QueID: uint64(serviceQueID), BsEui: serviceStation, Status: mioty.DLQueueStatusQueued}},
		editor:  &editorFake{updated: &storage.DownlinkMessage{QueID: serviceQueID, EPEUI: mioty.FormatEUI64(serviceEpEUI), TenantID: "4"}},
		revoker: &revokerFake{},
		events:  &updateEventsFake{},
		audit:   &auditFake{},
	}
}

func (f serviceFakes) deps() Deps {
	return Deps{Queuer: f.queuer, Editor: f.editor, Revoker: f.revoker, Endpoints: f.endpoints, Events: f.events, Audit: f.audit, Log: logger.NewNop()}
}

func (f serviceFakes) service(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(f.deps())
	require.NoError(t, err)
	return svc
}

var serviceOwner = Owner{TenantID: serviceTenant, OrganizationID: serviceOrg, EpEUI: serviceEpEUI}

func TestNewService_RefusesAMissingCollaborator(t *testing.T) {
	for name, unset := range map[string]func(*Deps){
		"audit":  func(d *Deps) { d.Audit = nil },
		"events": func(d *Deps) { d.Events = nil },
	} {
		t.Run(name, func(t *testing.T) {
			deps := newServiceFakes().deps()
			unset(&deps)
			_, err := NewService(deps)
			assert.ErrorIs(t, err, ErrMissingDependency)
		})
	}
}

func TestQueue_QueuesUnderTheOwnerAndAudits(t *testing.T) {
	f := newServiceFakes()

	result, err := f.service(t).Queue(testutil.TestContext(), serviceOwner, Content{Payloads: [][]byte{{0x01}}, Priority: 2})

	require.NoError(t, err)
	assert.Equal(t, uint64(serviceQueID), result.QueID)
	require.NotNil(t, f.queuer.orgID)
	assert.Equal(t, serviceOrg, *f.queuer.orgID)
	assert.Equal(t, serviceEpEUI, f.queuer.req.EpEui)
	require.Len(t, f.audit.events, 1)
	assert.Equal(t, models.EventTypeDownlinkQueued, f.audit.events[0].EventType)
	assert.Equal(t, mioty.FormatEUI64(serviceStation), f.audit.events[0].Details["bsEui"])
}

func TestQueue_RefusedContentReachesNoQueue(t *testing.T) {
	f := newServiceFakes()

	_, err := f.service(t).Queue(testutil.TestContext(), serviceOwner, Content{Payloads: [][]byte{{0x01}, {0x02}}})

	assert.ErrorIs(t, err, ErrTooManyPayloads)
	assert.Nil(t, f.queuer.req)
	assert.Empty(t, f.audit.events)
}

func TestQueue_KeepsTheSCACIFailure(t *testing.T) {
	f := newServiceFakes()
	failure := &scaci.DLDataQueueError{Token: scaci.ErrEndpointNotFound}
	f.queuer.err = failure

	_, err := f.service(t).Queue(testutil.TestContext(), serviceOwner, Content{})

	assert.ErrorIs(t, err, ErrQueue)
	var queueErr *scaci.DLDataQueueError
	require.ErrorAs(t, err, &queueErr)
	assert.Equal(t, scaci.ErrEndpointNotFound, queueErr.Token)
	assert.Empty(t, f.audit.events)
}

func TestUpdate_RewritesTheOwnersPendingDownlinkAndAudits(t *testing.T) {
	f := newServiceFakes()

	_, err := f.service(t).Update(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID}, Content{Payloads: [][]byte{{0xAA}}, Format: 3})

	require.NoError(t, err)
	require.NotNil(t, f.editor.orgID)
	assert.Equal(t, serviceOrg, *f.editor.orgID)
	assert.Equal(t, mioty.EUI64Bytes(serviceEpEUI), f.editor.epEUI)
	assert.Equal(t, uint8(3), f.editor.patch.Format)
	require.Len(t, f.audit.events, 1)
	assert.Equal(t, models.EventTypeDownlinkUpdated, f.audit.events[0].EventType)
	assert.Equal(t, []*storage.DownlinkMessage{f.editor.updated}, f.events.updated,
		"the rewritten downlink is announced to every viewer of the endpoint's queue")
}

// The rewrite stands when its announcement fails; the operator is answered
// with the rewritten downlink.
func TestUpdate_StandsWhenItsEventFails(t *testing.T) {
	f := newServiceFakes()
	f.events.err = errors.New("event store down")

	updated, err := f.service(t).Update(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID}, Content{Payloads: [][]byte{{0xAA}}})

	require.NoError(t, err)
	assert.Same(t, f.editor.updated, updated)
	assert.Len(t, f.audit.events, 1)
}

func TestUpdate_ReportsWhyNothingWasRewritten(t *testing.T) {
	for _, sentinel := range []error{storage.ErrDownlinkNotFound, storage.ErrDownlinkNotPending} {
		f := newServiceFakes()
		f.editor.err = sentinel

		_, err := f.service(t).Update(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID}, Content{})

		assert.ErrorIs(t, err, sentinel)
		assert.NotErrorIs(t, err, ErrUpdate)
		assert.Empty(t, f.audit.events)
		assert.Empty(t, f.events.updated, "nothing rewritten, nothing announced")
	}

	f := newServiceFakes()
	f.editor.err = errors.New("connection reset")
	_, err := f.service(t).Update(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID}, Content{})
	assert.ErrorIs(t, err, ErrUpdate)
}

func TestUpdate_RefusedContentReachesNoStore(t *testing.T) {
	f := newServiceFakes()

	_, err := f.service(t).Update(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID}, Content{Format: 256})

	assert.ErrorIs(t, err, ErrFormatInvalid)
	assert.Zero(t, f.editor.calls)
}

func TestRevoke_NamesTheOwnersDownlink(t *testing.T) {
	f := newServiceFakes()

	result, err := f.service(t).Revoke(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID})

	require.NoError(t, err)
	assert.Equal(t, RevokeStatusRevoked, result.Status)
	require.NotNil(t, f.revoker.ref)
	assert.Equal(t, scheduler.DownlinkRef{TenantID: serviceTenant, QueID: uint64(serviceQueID), OrganizationID: f.revoker.ref.OrganizationID, EpEUI: f.revoker.ref.EpEUI}, *f.revoker.ref)
	assert.Equal(t, serviceOrg, *f.revoker.ref.OrganizationID)
	assert.Equal(t, serviceEpEUI, *f.revoker.ref.EpEUI)
	require.Len(t, f.audit.events, 1)
	assert.Equal(t, RevokeStatusRevoked, f.audit.events[0].Details[models.EventDetailKeyStatus])
}

func TestRevoke_AtTheHoldingStationIsInitiated(t *testing.T) {
	f := newServiceFakes()
	f.revoker.station = serviceStation

	result, err := f.service(t).Revoke(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID})

	require.NoError(t, err)
	assert.Equal(t, RevokeStatusInitiated, result.Status)
	assert.Equal(t, mioty.FormatEUI64(serviceStation), f.audit.events[0].Details["bsEui"])
}

func TestRevoke_Refusals(t *testing.T) {
	f := newServiceFakes()
	f.endpoints = endpointsFake{err: storage.ErrNotFound}
	_, err := f.service(t).Revoke(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID})
	assert.ErrorIs(t, err, ErrEndpointNotFound)
	assert.Nil(t, f.revoker.ref)

	f = newServiceFakes()
	f.endpoints = endpointsFake{err: errors.New("connection reset")}
	_, err = f.service(t).Revoke(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID})
	assert.ErrorIs(t, err, ErrEndpointLookup)

	f = newServiceFakes()
	f.revoker.err = scheduler.ErrSchedulerQueueNotFound
	_, err = f.service(t).Revoke(testutil.TestContext(), Target{Owner: serviceOwner, QueID: serviceQueID})
	assert.ErrorIs(t, err, ErrRevoke)
	assert.ErrorIs(t, err, scheduler.ErrSchedulerQueueNotFound)
	assert.Empty(t, f.audit.events)

	f = newServiceFakes()
	_, err = f.service(t).Revoke(testutil.TestContext(), Target{Owner: serviceOwner, QueID: -1})
	assert.ErrorIs(t, err, scheduler.ErrSchedulerQueueNotFound)
	assert.Nil(t, f.revoker.ref)
}
