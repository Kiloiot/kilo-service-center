package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// The traffic and log screens narrow their listings server-side; these tests
// pin the request-to-filter translation and the validation of each predicate.

const (
	filterTestEpEUIHex = "70B3D59CD0000002"
	filterTestBsEUIHex = "70B3D59CD0000001"
	filterTestQueID    = int64(4242)
	filterTestOpID     = int64(-77)
	filterTestProfile  = "eu1"
	filterTestMode     = "ulp"
	filterTestSearch   = "attach"
	filterBadOutcome   = "sideways"
	filterBadEUI       = "zz"
	filterTestFromUnix = int64(1_700_000_000)
	filterTestPageSize = int32(20)
	filterTestOffset   = 40
)

type filterMessageListing struct {
	retainedMessageListing
	last *grpcservices.MessageFilters
}

func (m *filterMessageListing) ListMessages(ctx context.Context, tenantID int64, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	m.last = filters
	return m.retainedMessageListing.ListMessages(ctx, tenantID, filters, limit, offset)
}

type filterMessageSvc struct {
	fakeMessageSvc
	calls     int
	lastOrgID *uuid.UUID
	queue     storage.DownlinkQueueFilter
	results   storage.DownlinkResultFilter
	limit     int
	offset    int
}

func (m *filterMessageSvc) ListDownlinkQueue(_ context.Context, _ int64, filter storage.DownlinkQueueFilter, limit, offset int) ([]*storage.DownlinkMessage, int64, error) {
	m.queue, m.limit, m.offset = filter, limit, offset
	return nil, 0, nil
}

func (m *filterMessageSvc) GetDownlinkResults(_ context.Context, _ int64, orgID *uuid.UUID, filter storage.DownlinkResultFilter, limit, offset int) ([]*storage.DownlinkMessage, int, error) {
	m.calls++
	m.lastOrgID = orgID
	m.results, m.limit, m.offset = filter, limit, offset
	return nil, 0, nil
}

type filterEventSvc struct {
	last *grpcservices.EventFilters
}

func (e *filterEventSvc) List(_ context.Context, _ int64, filters *grpcservices.EventFilters, _, _ int) ([]*grpcservices.Event, int64, error) {
	e.last = filters
	if filters.Outcome == filterBadOutcome {
		return nil, 0, eventsservice.ErrInvalidOutcome
	}
	if filters.EpEUI == filterBadEUI {
		return nil, 0, eventsservice.ErrInvalidEndpointEUI
	}
	return nil, 0, nil
}

func (e *filterEventSvc) ListByBaseStation(context.Context, int64, []byte, *grpcservices.EventFilters, int, int) ([]*grpcservices.Event, int64, error) {
	return nil, 0, nil
}

func (e *filterEventSvc) ListByEndPoint(context.Context, int64, []byte, *grpcservices.EventFilters, int, int) ([]*grpcservices.Event, int64, error) {
	return nil, 0, nil
}

func (e *filterEventSvc) Stream(context.Context, int64, *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	return nil, nil
}

func TestListMessages_PassesReceptionFilters(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	listing := &filterMessageListing{retainedMessageListing: retainedMessageListing{f: f}}
	svc.msgListingSvc = listing
	duplicate, dlOpen := true, false

	_, err := svc.ListMessages(ownerCtx(), &pb.ListMessagesRequest{
		EpEui: filterTestEpEUIHex, Duplicate: &duplicate, DlOpen: &dlOpen, Profile: filterTestProfile, Mode: filterTestMode,
	})
	require.NoError(t, err)
	require.NotNil(t, listing.last)
	assert.Equal(t, &duplicate, listing.last.Duplicate)
	assert.Equal(t, &dlOpen, listing.last.DlOpen)
	assert.Equal(t, filterTestProfile, listing.last.Profile)
	assert.Equal(t, filterTestMode, listing.last.Mode)
	assert.Len(t, listing.last.EpEui, 8)

	plain, err := svc.ListMessages(ownerCtx(), &pb.ListMessagesRequest{})
	require.NoError(t, err)
	assert.Nil(t, listing.last.Duplicate, "an unset optional flag is not a filter")
	assert.Nil(t, listing.last.DlOpen)
	assert.Len(t, plain.Messages, 1)
}

// TestListDownlinkQueue_ListsOnlyTheRequestOrganization: the queue listing is
// scoped like enqueue, edit and results - a tenant's other organizations'
// downlinks are not listed.
func TestListDownlinkQueue_ListsOnlyTheRequestOrganization(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	msgSvc := &filterMessageSvc{}
	svc.useDownlinks(downlinkFakes{messages: msgSvc})

	_, err := svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{})
	require.NoError(t, err)
	require.NotNil(t, msgSvc.queue.OrganizationID)
	assert.Equal(t, retainedOwnerOrg, *msgSvc.queue.OrganizationID)

	_, err = svc.ListDownlinkQueue(testutil.TestContextWithTenant(retainedOwnerTenant), &pb.ListDownlinkQueueRequest{})
	require.Error(t, err, "a request without an organization lists nothing")
}

func TestListDownlinkQueue_FiltersAndPaginatesInTheStore(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	msgSvc := &filterMessageSvc{}
	svc.useDownlinks(downlinkFakes{messages: msgSvc})
	priority := float32(0.5)
	queID := filterTestQueID

	_, err := svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{
		EpEui: filterTestEpEUIHex, Status: string(mioty.DLQueueStatusQueued), Priority: &priority, QueId: &queID,
		PageSize: filterTestPageSize, PageToken: grpcerrors.GeneratePaginationToken(filterTestOffset),
	})
	require.NoError(t, err)
	require.NotNil(t, msgSvc.queue.Status)
	assert.Equal(t, mioty.DLQueueStatusQueued, *msgSvc.queue.Status)
	assert.Equal(t, &priority, msgSvc.queue.Priority)
	assert.Equal(t, &queID, msgSvc.queue.QueID)
	require.NotNil(t, msgSvc.queue.EpEUI)
	assert.Equal(t, [8]byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x02}, *msgSvc.queue.EpEUI)
	assert.Equal(t, int(filterTestPageSize), msgSvc.limit)
	assert.Equal(t, filterTestOffset, msgSvc.offset)

	_, err = svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{Status: "parked"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidDownlinkQueueStatus)

	_, err = svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{EpEui: "zz"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidEndpointEUIFormat)

	_, err = svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{BsEui: filterTestBsEUIHex})
	require.NoError(t, err)
	require.NotNil(t, msgSvc.queue.BsEUI, "the holding station narrows the queue")
	assert.Equal(t, [8]byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x01}, *msgSvc.queue.BsEUI)
	require.NotNil(t, msgSvc.queue.OrganizationID, "a station filter stays within the caller's organization")

	_, err = svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{BsEui: "zz"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidBasestationEUIFormat)

	_, err = svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{PageToken: "not-a-token"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidPageToken)
}

func TestGetDownlinkResults_PassesBaseStationAndQueueID(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	msgSvc := &filterMessageSvc{}
	svc.useDownlinks(downlinkFakes{messages: msgSvc})
	queID := filterTestQueID
	from := timestamppb.New(time.Unix(filterTestFromUnix, 0))

	_, err := svc.GetDownlinkResults(ownerCtx(), &pb.GetDownlinkResultsRequest{
		EpEui: filterTestEpEUIHex, BsEui: filterTestBsEUIHex, QueId: &queID, StatusFilter: mioty.DLDataResultSent, TimeFrom: from,
	})
	require.NoError(t, err)
	assert.Len(t, msgSvc.results.EpEUI, 8)
	assert.Len(t, msgSvc.results.BsEUI, 8)
	assert.Equal(t, &queID, msgSvc.results.QueID)
	assert.Equal(t, string(mioty.DLQueueStatusTransmitted), msgSvc.results.Status, "BSSCI result names map onto the stored status")
	require.NotNil(t, msgSvc.results.From)
	assert.Nil(t, msgSvc.results.To)

	_, err = svc.GetDownlinkResults(ownerCtx(), &pb.GetDownlinkResultsRequest{BsEui: "nope"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidBasestationEUIFormat)
}

func TestListEvents_PassesLogFiltersAndMapsOutcomeErrors(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	events := &filterEventSvc{}
	svc.eventSvc = events
	opID := filterTestOpID

	_, err := svc.ListEvents(ownerCtx(), &pb.ListEventsRequest{
		OpId: &opID, EpEui: filterTestEpEUIHex, BsEui: filterTestBsEUIHex, Outcome: grpcservices.EventOutcomeFailure, Search: filterTestSearch,
	})
	require.NoError(t, err)
	require.NotNil(t, events.last)
	assert.Equal(t, &opID, events.last.OpID)
	assert.Equal(t, filterTestEpEUIHex, events.last.EpEUI)
	assert.Equal(t, filterTestBsEUIHex, events.last.BsEUI)
	assert.Equal(t, grpcservices.EventOutcomeFailure, events.last.Outcome)
	assert.Equal(t, filterTestSearch, events.last.Search)

	_, err = svc.ListEvents(ownerCtx(), &pb.ListEventsRequest{Outcome: filterBadOutcome})
	assertCode(t, err, grpcerrors.ErrTokenInvalidEventOutcome)

	_, err = svc.ListEvents(ownerCtx(), &pb.ListEventsRequest{EpEui: filterBadEUI})
	assertCode(t, err, grpcerrors.ErrTokenInvalidEndpointEUIFormat)
}

// filterTestFormat stands in for a userData format identifier (SCACI §3.8.1).
const filterTestFormat = uint8(7)

func TestULDataMessageToProto_CarriesTheUserDataFormat(t *testing.T) {
	format := filterTestFormat
	withFormat := ulDataMessageToProto(&mioty.ULDataMessage{Format: &format})
	require.NotNil(t, withFormat.Format, "a stored format reaches the listing")
	assert.Equal(t, uint32(filterTestFormat), withFormat.GetFormat())

	withoutFormat := ulDataMessageToProto(&mioty.ULDataMessage{})
	assert.Nil(t, withoutFormat.Format, "an uplink without a format leaves the field unset")
}

// filterTestActorID stands in for the user an audit event records.
const filterTestActorID = "7a1d9c52-4f0e-4b8e-9d7a-3f2c1b0e5d44"

// filterTestActorEmail stands in for the email resolved for filterTestActorID.
const filterTestActorEmail = "operator@tenant.example"

func TestEventToProto_CarriesActingUser(t *testing.T) {
	audited := eventToProto(&grpcservices.Event{ID: "1", UserID: filterTestActorID, UserEmail: filterTestActorEmail})
	assert.Equal(t, filterTestActorID, audited.GetUserId(), "the audit log names who acted")
	assert.Equal(t, filterTestActorEmail, audited.GetUserEmail(), "by email when the store resolved one")

	raised := eventToProto(&grpcservices.Event{ID: "2"})
	assert.Empty(t, raised.GetUserId(), "a service-raised event has no actor")
	assert.Empty(t, raised.GetUserEmail())
}
