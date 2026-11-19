package grpc

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	endpointpkg "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	statusTestOwnerTenant   = int64(31)
	statusTestForeignTenant = int64(32)
	statusTestEndpointID    = int64(12)
	statusTestEUI           = "70B3D56770111505"
)

// statusEndpoints is a tenant-scoped endpoint table shared by the gRPC
// endpoint service and the attachment service, as the endpoints table is.
type statusEndpoints struct {
	mu                 sync.Mutex
	endpoints          map[int64]*models.EndPoint
	directStatusWrites []string
	updates            int
}

func newStatusEndpoints(epStatus string) *statusEndpoints {
	ep := &models.EndPoint{
		ID: statusTestEndpointID, TenantID: statusTestOwnerTenant, EUI: models.EUIFromString(statusTestEUI),
		Name: "status", EpStatus: epStatus, Bidi: true,
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}
	return &statusEndpoints{endpoints: map[int64]*models.EndPoint{statusTestOwnerTenant: ep}}
}

func (s *statusEndpoints) lookup(tenantID int64, eui []byte) (*models.EndPoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep, ok := s.endpoints[tenantID]
	if !ok || !bytes.Equal(eui, ep.EUI[:]) {
		return nil, storage.ErrNotFound
	}
	clone := *ep
	return &clone, nil
}

func (s *statusEndpoints) status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.endpoints[statusTestOwnerTenant].EpStatus
}

// statusEndpointService is the gRPC endpoint service over the table; its
// update, like the repository's, never writes the attachment status.
type statusEndpointService struct {
	mockEndpointSvcForUpdate
	table *statusEndpoints
}

func (s *statusEndpointService) GetByEUI(_ context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
	return s.table.lookup(tenantID, eui)
}

func (s *statusEndpointService) Update(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	s.table.mu.Lock()
	defer s.table.mu.Unlock()
	stored := s.table.endpoints[ep.TenantID]
	s.table.updates++
	if ep.EpStatus != stored.EpStatus {
		s.table.directStatusWrites = append(s.table.directStatusWrites, ep.EpStatus)
	}
	updated := *ep
	updated.EpStatus = stored.EpStatus
	s.table.endpoints[ep.TenantID] = &updated
	clone := updated
	return &clone, nil
}

// statusAttachmentStore is the attachment service's view of the table.
type statusAttachmentStore struct{ table *statusEndpoints }

func (s statusAttachmentStore) GetByEUI(_ context.Context, tenantID int64, eui []byte) (*models.EndPoint, error) {
	return s.table.lookup(tenantID, eui)
}

func (s statusAttachmentStore) EndpointDetachStateUpdate(context.Context, int64, int64, models.EndpointDetachStateParams) error {
	return nil
}

func (s statusAttachmentStore) TransitionEndpointStatus(_ context.Context, tenantID, endpointID int64, epStatus string) (bool, error) {
	s.table.mu.Lock()
	defer s.table.mu.Unlock()
	ep, ok := s.table.endpoints[tenantID]
	if !ok || ep.ID != endpointID {
		return false, nil
	}
	changed := ep.EpStatus != epStatus
	ep.EpStatus = epStatus
	return changed, nil
}

func (s statusAttachmentStore) RestateEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error) {
	return s.TransitionEndpointStatus(ctx, tenantID, endpointID, status)
}

// statusStations records what the base stations and application centers are sent.
type statusStations struct {
	mu        sync.Mutex
	attaches  int
	detaches  int
	announced []string
}

func (s *statusStations) SendAttachPropagateToAll(uint64, []byte, uint16, bool, uint32, bool, uint8, bool, bool) []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attaches++
	return nil
}

func (s *statusStations) SendDetachPropagateToAll(uint64) []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detaches++
	return nil
}

func (s *statusStations) GetConnectedSessionEUIs() []string { return nil }

func (s *statusStations) NotifyEndpointStatus(_ context.Context, notice bssciservices.EndpointStatusNotice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.announced = append(s.announced, notice.Status.EpStatus)
}

func (s *statusStations) sent() (attaches, detaches int, announced []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attaches, s.detaches, append([]string(nil), s.announced...)
}

type statusSessionKeys struct{}

func (statusSessionKeys) NetworkSessionKey(_ context.Context, ep *models.EndPoint) ([]byte, error) {
	return ep.NwkSnKey, nil
}

type statusEvents struct{}

func (statusEvents) CreateEvent(context.Context, *models.SystemEvent) error { return nil }

// statusCreator stands in for the creation of attached endpoints, which no
// status update performs.
type statusCreator struct{}

func (statusCreator) CreateWithStatus(_ context.Context, ep *models.EndPoint, _ string) (*models.EndPoint, error) {
	return ep, nil
}

// statusRunner runs the background work when it is started.
type statusRunner struct{}

func (statusRunner) Go(ctx context.Context, fn func(context.Context)) { fn(ctx) }

func statusService(table *statusEndpoints, stations *statusStations) *CoreService {
	decider, err := bssciservices.NewEndpointAttachmentDecider(statusAttachmentStore{table}, stations)
	if err != nil {
		panic(err)
	}
	propagation, err := bssciservices.NewAttachmentPropagation(stations, statusSessionKeys{}, statusEvents{}, statusRunner{}, clock.SystemClock{}, logger.NewNop())
	if err != nil {
		panic(err)
	}
	attachments, err := bssciservices.NewEndpointAttachmentService(statusAttachmentStore{table}, statusCreator{}, decider, stations, propagation)
	if err != nil {
		panic(err)
	}
	return testCoreService(coreFields{
		endpointSvc:           &statusEndpointService{table: table},
		endpointAttachmentSvc: attachments,
		log:                   &mockLogger{},
	})
}

func updateStatus(svc *CoreService, tenantID int64, epStatus string, paths ...string) (*pb.EndPoint, error) {
	return svc.UpdateEndPoint(testutil.TestContextWithTenant(tenantID), &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: statusTestEUI, Status: epStatus, AttachStatus: endpointpkg.EndpointStatusAttaching, Name: "renamed"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: append([]string{fieldMaskStatus}, paths...)},
	})
}

// BSSCI §3.8, SCACI §3.13: a status change through UpdateEndPoint is an
// attach decision, made by the attachment service like AttachEndPoint: the
// stations are sent attPrp and the application centers one epStat.
func TestUpdateEndPoint_StatusChangeAttachesThroughTheAttachmentService(t *testing.T) {
	table := newStatusEndpoints(endpointpkg.EndpointStatusDetached)
	stations := &statusStations{}

	resp, err := updateStatus(statusService(table, stations), statusTestOwnerTenant, endpointpkg.EndpointStatusAttached, fieldMaskName)

	require.NoError(t, err)
	assert.Empty(t, table.directStatusWrites, "the status is never written directly")
	assert.Equal(t, endpointpkg.EndpointStatusAttached, table.status())
	assert.Equal(t, endpointpkg.EndpointStatusAttached, resp.AttachStatus, "the response reports the attachment")
	assert.Equal(t, "renamed", resp.Name, "the other masked fields are applied")
	require.Eventually(t, func() bool { attaches, _, _ := stations.sent(); return attaches == 1 }, testServerWait, testServerPoll,
		"the stations are sent one attPrp")
	_, detaches, announced := stations.sent()
	assert.Zero(t, detaches)
	assert.Equal(t, []string{pkgmioty.EPStatusAttached}, announced, "one epStat for one attachment")
}

// A detached status through UpdateEndPoint detaches the endpoint through the
// attachment service: one detPrp to the stations, one epStat.
func TestUpdateEndPoint_StatusChangeDetachesThroughTheAttachmentService(t *testing.T) {
	table := newStatusEndpoints(endpointpkg.EndpointStatusAttached)
	stations := &statusStations{}

	_, err := updateStatus(statusService(table, stations), statusTestOwnerTenant, endpointpkg.EndpointStatusDetached)

	require.NoError(t, err)
	assert.Equal(t, endpointpkg.EndpointStatusDetached, table.status())
	require.Eventually(t, func() bool { _, detaches, _ := stations.sent(); return detaches == 1 }, testServerWait, testServerPoll,
		"the stations are sent one detPrp")
	_, _, announced := stations.sent()
	assert.Equal(t, []string{pkgmioty.EPStatusDetached}, announced, "one epStat for one detachment")
}

// An unchanged status is no decision: nothing is sent.
func TestUpdateEndPoint_UnchangedStatusDecidesNothing(t *testing.T) {
	table := newStatusEndpoints(endpointpkg.EndpointStatusAttached)
	stations := &statusStations{}

	resp, err := updateStatus(statusService(table, stations), statusTestOwnerTenant, endpointpkg.EndpointStatusAttached, fieldMaskName)

	require.NoError(t, err)
	assert.Equal(t, "renamed", resp.Name)
	assert.Equal(t, endpointpkg.EndpointStatusAttached, table.status())
	attaches, detaches, announced := stations.sent()
	assert.Zero(t, attaches)
	assert.Zero(t, detaches)
	assert.Empty(t, announced, "no epStat for an unchanged status")
}

// Every composition root wires the attachment service, so endpoint handlers
// without one are refused when they are built.
func TestNewEndpointHandlersRefusesAMissingAttachmentService(t *testing.T) {
	_, err := NewEndpointHandlers(EndpointHandlerDeps{Endpoints: &fakeEndpointSvc{}, Clock: clock.SystemClock{}}, &captureAuditRecorder{}, &mockLogger{})

	require.EqualError(t, err, errMsgEndpointAttachmentCannotBeNil)
}

func TestNewEndpointHandlers_StatisticsNeedTheRegistrationWindow(t *testing.T) {
	_, err := NewEndpointHandlers(EndpointHandlerDeps{
		Endpoints:  &fakeEndpointSvc{},
		Attachment: &mockEndpointAttachmentSvc{},
		Clock:      clock.SystemClock{},
		Stats:      uplinkLedger{},
	}, &captureAuditRecorder{}, &mockLogger{})
	require.EqualError(t, err, errMsgEndpointStatsWindowCannotBeNil)
}

// A status decision is the service center's: attaching is not one.
func TestUpdateEndPoint_AttachingIsNotAStatusDecision(t *testing.T) {
	table := newStatusEndpoints(endpointpkg.EndpointStatusDetached)
	stations := &statusStations{}

	_, err := updateStatus(statusService(table, stations), statusTestOwnerTenant, endpointpkg.EndpointStatusAttaching)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointStatus), st.Message())
	assert.Zero(t, table.updates)
	assert.Equal(t, endpointpkg.EndpointStatusDetached, table.status())
}

// A caller acting in another tenant cannot move an endpoint's status: the
// endpoint is not found and its owner's stations and application centers
// are sent nothing.
func TestUpdateEndPoint_StatusOfAnotherTenantsEndpointIsNotFound(t *testing.T) {
	table := newStatusEndpoints(endpointpkg.EndpointStatusDetached)
	stations := &statusStations{}

	_, err := updateStatus(statusService(table, stations), statusTestForeignTenant, endpointpkg.EndpointStatusAttached)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound), st.Code())
	assert.Equal(t, endpointpkg.EndpointStatusDetached, table.status())
	attaches, _, announced := stations.sent()
	assert.Zero(t, attaches)
	assert.Empty(t, announced)
}

// Attaching or detaching an endpoint the tenant does not have is answered
// with not found, not with a failed attach or detach.
func TestAttachAndDetachOfAnUnknownEndpointAreNotFound(t *testing.T) {
	svc := statusService(newStatusEndpoints(endpointpkg.EndpointStatusDetached), &statusStations{})
	ctx := testutil.TestContextWithTenant(statusTestForeignTenant)

	_, attachErr := svc.AttachEndPoint(ctx, &pb.AttachEndPointRequest{EpEui: statusTestEUI})
	_, detachErr := svc.DetachEndPoint(ctx, &pb.DetachEndPointRequest{EpEui: statusTestEUI})

	for _, err := range []error{attachErr, detachErr} {
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound), st.Code())
		assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound), st.Message())
	}
}
