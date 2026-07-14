package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// ============================================================================
// Tenant Isolation Mock: Endpoint Service
// ============================================================================

// mockEndpointSvcIsolation captures tenantID to verify tenant propagation.
type mockEndpointSvcIsolation struct {
	capturedTenantIDs []int64
	listFunc          func(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error)
	getByEUIFunc      func(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error)
	createFunc        func(ctx context.Context, ep *models.EndPoint) (*models.EndPoint, error)
	deleteFunc        func(ctx context.Context, eui []byte, tenantID int64) error
}

func (m *mockEndpointSvcIsolation) Create(ctx context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, ep.TenantID)
	if m.createFunc != nil {
		return m.createFunc(ctx, ep)
	}
	return ep, nil
}

func (m *mockEndpointSvcIsolation) GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	if m.getByEUIFunc != nil {
		return m.getByEUIFunc(ctx, eui, tenantID)
	}
	return nil, storage.ErrNotFound
}

func (m *mockEndpointSvcIsolation) Update(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (m *mockEndpointSvcIsolation) Delete(ctx context.Context, eui []byte, tenantID int64) error {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, eui, tenantID)
	}
	return nil
}

func (m *mockEndpointSvcIsolation) ListByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcIsolation) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	if m.listFunc != nil {
		return m.listFunc(ctx, tenantID, limit, offset)
	}
	return []*models.EndPoint{}, nil
}

func (m *mockEndpointSvcIsolation) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (m *mockEndpointSvcIsolation) CheckEUIGloballyUnique(_ context.Context, _ []byte) error {
	return nil
}

// ============================================================================
// Tenant Isolation Mock: Base Station Service
// ============================================================================

// mockBasestationSvcIsolation captures tenantID to verify tenant propagation.
type mockBasestationSvcIsolation struct {
	capturedTenantIDs []int64
	listFunc          func(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error)
}

func (m *mockBasestationSvcIsolation) Create(_ context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, bs.TenantID)
	return bs, nil
}

func (m *mockBasestationSvcIsolation) GetByEUI(_ context.Context, _ []byte, tenantID int64) (*models.BaseStation, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return nil, storage.ErrNotFound
}

func (m *mockBasestationSvcIsolation) Update(_ context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	return bs, nil
}

func (m *mockBasestationSvcIsolation) Delete(_ context.Context, _ []byte, tenantID int64) error {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return nil
}

func (m *mockBasestationSvcIsolation) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	if m.listFunc != nil {
		return m.listFunc(ctx, tenantID, limit, offset)
	}
	return []*models.BaseStation{}, nil
}

func (m *mockBasestationSvcIsolation) UpdateEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	return nil, nil
}
func (m *mockBasestationSvcIsolation) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// ============================================================================
// Tenant Isolation Mock: Message Listing Service
// ============================================================================

// mockMessageListingSvcIsolation captures tenantID to verify tenant propagation.
type mockMessageListingSvcIsolation struct {
	capturedTenantIDs []int64
}

func (m *mockMessageListingSvcIsolation) GetMessage(_ context.Context, tenantID int64, _ string) (*mioty.ULDataMessage, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return nil, storage.ErrNotFound
}

func (m *mockMessageListingSvcIsolation) ListMessages(_ context.Context, tenantID int64, _ *grpcservices.MessageFilters, _, _ int) ([]*mioty.ULDataMessage, int64, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return []*mioty.ULDataMessage{}, 0, nil
}

func (m *mockMessageListingSvcIsolation) StreamMessages(_ context.Context, _ int64, _ *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	ch := make(chan *mioty.ULDataMessage)
	close(ch)
	return ch, nil
}

func (m *mockMessageListingSvcIsolation) ListBaseStationMessages(_ context.Context, tenantID int64, _ []byte, _ *grpcservices.MessageFilters, _, _ int) ([]*mioty.ULDataMessage, int64, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return []*mioty.ULDataMessage{}, 0, nil
}

func (m *mockMessageListingSvcIsolation) GetBaseStationMessage(_ context.Context, tenantID int64, _ []byte, _ string) (*mioty.ULDataMessage, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return nil, storage.ErrNotFound
}

func (m *mockMessageListingSvcIsolation) GetBaseStationMessageStats(_ context.Context, tenantID int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return &mioty.BaseStationMessageStats{}, nil
}

func (m *mockMessageListingSvcIsolation) SearchBaseStationMessages(_ context.Context, tenantID int64, _ []byte, _ string, _ *grpcservices.MessageFilters, _, _ int) ([]*mioty.ULDataMessage, int64, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return []*mioty.ULDataMessage{}, 0, nil
}

func (m *mockMessageListingSvcIsolation) ExportBaseStationMessages(_ context.Context, tenantID int64, _ []byte, _ *grpcservices.MessageFilters, _ string) ([]byte, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return []byte{}, nil
}

func (m *mockMessageListingSvcIsolation) StreamBaseStationMessages(_ context.Context, _ int64, _ []byte, _ *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	ch := make(chan *mioty.ULDataMessage)
	close(ch)
	return ch, nil
}

// ============================================================================
// Tenant Isolation Mock: Integration Service
// ============================================================================

// mockIntegrationSvcIsolation captures tenantID to verify tenant propagation.
type mockIntegrationSvcIsolation struct {
	capturedTenantIDs []int64
}

func (m *mockIntegrationSvcIsolation) Create(_ context.Context, tenantID int64, _ *grpcservices.IntegrationCreateRequest) (*models.Integration, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return &models.Integration{ID: 1, TenantID: tenantID, Name: "test", Type: "http", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (m *mockIntegrationSvcIsolation) GetByID(_ context.Context, tenantID int64, _ int64) (*models.Integration, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return nil, storage.ErrNotFound
}

func (m *mockIntegrationSvcIsolation) Update(_ context.Context, tenantID int64, _ int64, _ *grpcservices.IntegrationUpdateRequest) (*models.Integration, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return nil, nil
}

func (m *mockIntegrationSvcIsolation) Delete(_ context.Context, tenantID int64, _ int64) error {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return nil
}

func (m *mockIntegrationSvcIsolation) List(_ context.Context, tenantID int64, _, _ int) ([]*models.Integration, int64, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return []*models.Integration{}, 0, nil
}

// ============================================================================
// Helper: Create service with tenant-capturing mocks
// ============================================================================

type isolationTestServices struct {
	svc          *CoreService
	endpointMock *mockEndpointSvcIsolation
	bsMock       *mockBasestationSvcIsolation
	msgListMock  *mockMessageListingSvcIsolation
	integMock    *mockIntegrationSvcIsolation
}

func newIsolationTestService() *isolationTestServices {
	epMock := &mockEndpointSvcIsolation{}
	bsMock := &mockBasestationSvcIsolation{}
	msgListMock := &mockMessageListingSvcIsolation{}
	integMock := &mockIntegrationSvcIsolation{}

	svc := &CoreService{
		endpointSvc:    epMock,
		basestationSvc: bsMock,
		msgListingSvc:  msgListMock,
		integrationSvc: integMock,
		log:            &mockLogger{},
	}

	return &isolationTestServices{
		svc:          svc,
		endpointMock: epMock,
		bsMock:       bsMock,
		msgListMock:  msgListMock,
		integMock:    integMock,
	}
}

// contextForTenant returns a context with tenant, org, and user IDs set.
func contextForTenant(tenantID int64) context.Context {
	ctx := testutil.TestContextWithTenant(tenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())
	ctx = pkgcontext.WithUserID(ctx, uuid.New().String())
	return ctx
}

// ============================================================================
// Category 1: Core CRUD — Endpoint Tenant Isolation
// ============================================================================

func TestTenantIsolation_ListEndpoints_OnlyReturnsSameTenant(t *testing.T) {
	ts := newIsolationTestService()

	tenant42 := []*models.EndPoint{{TenantID: 42, Name: "EP1"}}
	ts.endpointMock.listFunc = func(_ context.Context, tenantID int64, _, _ int) ([]*models.EndPoint, error) {
		if tenantID == 42 {
			return tenant42, nil
		}
		return []*models.EndPoint{}, nil
	}

	// Tenant 42 lists endpoints
	ctx42 := contextForTenant(42)
	resp42, err := ts.svc.ListEndPoints(ctx42, &pb.ListEndPointsRequest{PageSize: 10})
	require.NoError(t, err)
	require.NotNil(t, resp42)
	assert.Len(t, resp42.Endpoints, 1)

	// Tenant 99 lists endpoints — gets empty
	ctx99 := contextForTenant(99)
	resp99, err := ts.svc.ListEndPoints(ctx99, &pb.ListEndPointsRequest{PageSize: 10})
	require.NoError(t, err)
	require.NotNil(t, resp99)
	assert.Len(t, resp99.Endpoints, 0)

	// Verify correct tenant IDs were passed to service
	require.Len(t, ts.endpointMock.capturedTenantIDs, 2)
	assert.Equal(t, int64(42), ts.endpointMock.capturedTenantIDs[0])
	assert.Equal(t, int64(99), ts.endpointMock.capturedTenantIDs[1])
}

func TestTenantIsolation_CreateEndpoint_TenantPropagation(t *testing.T) {
	ts := newIsolationTestService()

	var capturedTenantID int64
	ts.endpointMock.createFunc = func(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
		capturedTenantID = ep.TenantID
		return ep, nil
	}

	ctx42 := contextForTenant(42)
	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "0000000000000001",
			Name:     "Test EP",
			EpClass:  "A",
			NwkSnKey: make([]byte, 16),
			AppKey:   make([]byte, 16),
		},
	}

	_, err := ts.svc.CreateEndPoint(ctx42, req)
	require.NoError(t, err)
	assert.Equal(t, int64(42), capturedTenantID)

	// Same request with tenant 99
	ctx99 := contextForTenant(99)
	_, err = ts.svc.CreateEndPoint(ctx99, req)
	require.NoError(t, err)
	assert.Equal(t, int64(99), capturedTenantID)
}

func TestTenantIsolation_DeleteEndpoint_TenantPropagation(t *testing.T) {
	ts := newIsolationTestService()

	ctx42 := contextForTenant(42)
	_, err := ts.svc.DeleteEndPoint(ctx42, &pb.DeleteEndPointRequest{EpEui: "0000000000000001"})
	require.NoError(t, err)

	ctx99 := contextForTenant(99)
	_, err = ts.svc.DeleteEndPoint(ctx99, &pb.DeleteEndPointRequest{EpEui: "0000000000000001"})
	require.NoError(t, err)

	require.Len(t, ts.endpointMock.capturedTenantIDs, 2)
	assert.Equal(t, int64(42), ts.endpointMock.capturedTenantIDs[0])
	assert.Equal(t, int64(99), ts.endpointMock.capturedTenantIDs[1])
}

// ============================================================================
// Category 1: Core CRUD — Base Station Tenant Isolation
// ============================================================================

func TestTenantIsolation_ListBaseStations_OnlyReturnsSameTenant(t *testing.T) {
	ts := newIsolationTestService()

	ts.bsMock.listFunc = func(_ context.Context, tenantID int64, _, _ int) ([]*models.BaseStation, error) {
		if tenantID == 42 {
			return []*models.BaseStation{{TenantID: 42, Name: "BS1"}}, nil
		}
		return []*models.BaseStation{}, nil
	}

	// Tenant 42 sees its base station
	ctx42 := contextForTenant(42)
	resp42, err := ts.svc.ListBaseStations(ctx42, &pb.ListBaseStationsRequest{PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, resp42.Basestations, 1)

	// Tenant 99 sees nothing
	ctx99 := contextForTenant(99)
	resp99, err := ts.svc.ListBaseStations(ctx99, &pb.ListBaseStationsRequest{PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, resp99.Basestations, 0)

	require.Len(t, ts.bsMock.capturedTenantIDs, 2)
	assert.Equal(t, int64(42), ts.bsMock.capturedTenantIDs[0])
	assert.Equal(t, int64(99), ts.bsMock.capturedTenantIDs[1])
}

// ============================================================================
// Category 2: Messages — Tenant Isolation
// ============================================================================

func TestTenantIsolation_ListMessages_OnlyReturnsSameTenant(t *testing.T) {
	ts := newIsolationTestService()

	ctx42 := contextForTenant(42)
	_, err := ts.svc.ListMessages(ctx42, &pb.ListMessagesRequest{PageSize: 10})
	require.NoError(t, err)

	ctx99 := contextForTenant(99)
	_, err = ts.svc.ListMessages(ctx99, &pb.ListMessagesRequest{PageSize: 10})
	require.NoError(t, err)

	require.Len(t, ts.msgListMock.capturedTenantIDs, 2)
	assert.Equal(t, int64(42), ts.msgListMock.capturedTenantIDs[0])
	assert.Equal(t, int64(99), ts.msgListMock.capturedTenantIDs[1])
}

func TestTenantIsolation_GetMessage_TenantPropagation(t *testing.T) {
	ts := newIsolationTestService()

	ctx42 := contextForTenant(42)
	_, err := ts.svc.GetMessage(ctx42, &pb.GetMessageRequest{Id: "msg-001"})
	// Will return not-found error since mock returns ErrNotFound
	require.Error(t, err)

	ctx99 := contextForTenant(99)
	_, err = ts.svc.GetMessage(ctx99, &pb.GetMessageRequest{Id: "msg-001"})
	require.Error(t, err)

	require.Len(t, ts.msgListMock.capturedTenantIDs, 2)
	assert.Equal(t, int64(42), ts.msgListMock.capturedTenantIDs[0])
	assert.Equal(t, int64(99), ts.msgListMock.capturedTenantIDs[1])
}

// ============================================================================
// Category 3: No Tenant Context — Fails Fast
// ============================================================================

func TestTenantIsolation_MissingTenant_FailsClosed(t *testing.T) {
	ts := newIsolationTestService()
	ctx := context.Background() // No tenant

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "ListEndPoints",
			call: func() error {
				_, err := ts.svc.ListEndPoints(ctx, &pb.ListEndPointsRequest{PageSize: 10})
				return err
			},
		},
		{
			name: "ListBaseStations",
			call: func() error {
				_, err := ts.svc.ListBaseStations(ctx, &pb.ListBaseStationsRequest{PageSize: 10})
				return err
			},
		},
		{
			name: "ListMessages",
			call: func() error {
				_, err := ts.svc.ListMessages(ctx, &pb.ListMessagesRequest{PageSize: 10})
				return err
			},
		},
		{
			name: "GetMessage",
			call: func() error {
				_, err := ts.svc.GetMessage(ctx, &pb.GetMessageRequest{Id: "test"})
				return err
			},
		},
		{
			name: "ListIntegrations",
			call: func() error {
				_, err := ts.svc.ListIntegrations(ctx, &pb.ListIntegrationsRequest{PageSize: 10})
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok, "expected gRPC status error")
			// Missing tenant returns Unauthenticated
			assert.Equal(t, codes.Unauthenticated, st.Code(),
				"missing tenant should return Unauthenticated, got %v: %s", st.Code(), st.Message())
		})
	}

	// Service mocks should never have been called
	assert.Empty(t, ts.endpointMock.capturedTenantIDs, "endpoint service should not be called without tenant")
	assert.Empty(t, ts.bsMock.capturedTenantIDs, "base station service should not be called without tenant")
	assert.Empty(t, ts.msgListMock.capturedTenantIDs, "message service should not be called without tenant")
	assert.Empty(t, ts.integMock.capturedTenantIDs, "integration service should not be called without tenant")
}

// ============================================================================
// Category 4: Integrations — Tenant Isolation
// ============================================================================

func TestTenantIsolation_ListIntegrations_OnlyReturnsSameTenant(t *testing.T) {
	ts := newIsolationTestService()

	ctx42 := contextForTenant(42)
	_, err := ts.svc.ListIntegrations(ctx42, &pb.ListIntegrationsRequest{PageSize: 10})
	require.NoError(t, err)

	ctx99 := contextForTenant(99)
	_, err = ts.svc.ListIntegrations(ctx99, &pb.ListIntegrationsRequest{PageSize: 10})
	require.NoError(t, err)

	require.Len(t, ts.integMock.capturedTenantIDs, 2)
	assert.Equal(t, int64(42), ts.integMock.capturedTenantIDs[0])
	assert.Equal(t, int64(99), ts.integMock.capturedTenantIDs[1])
}

// ============================================================================
// Category 6: Cross-Tenant Access — No Information Leakage
// ============================================================================

func TestTenantIsolation_CrossTenantEndpointAccess_ReturnsNotFound(t *testing.T) {
	ts := newIsolationTestService()

	// GetByEUI returns data only for tenant 42
	ts.endpointMock.getByEUIFunc = func(_ context.Context, _ []byte, tenantID int64) (*models.EndPoint, error) {
		if tenantID == 42 {
			return &models.EndPoint{TenantID: 42, Name: "Tenant42 EP"}, nil
		}
		return nil, storage.ErrNotFound
	}

	// Tenant 42 can access its own endpoint
	ctx42 := contextForTenant(42)
	resp42, err := ts.svc.GetEndPoint(ctx42, &pb.GetEndPointRequest{EpEui: "0000000000000001"})
	require.NoError(t, err)
	require.NotNil(t, resp42)

	// Tenant 99 trying to access the same endpoint gets NotFound
	ctx99 := contextForTenant(99)
	_, err = ts.svc.GetEndPoint(ctx99, &pb.GetEndPointRequest{EpEui: "0000000000000001"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code(), "cross-tenant access should return NotFound, not PermissionDenied")
}

func TestTenantIsolation_CrossTenantBaseStationAccess_ReturnsNotFound(t *testing.T) {
	ts := newIsolationTestService()

	// Tenant 99 tries to get a base station that belongs to tenant 42
	ctx99 := contextForTenant(99)
	_, err := ts.svc.GetBaseStation(ctx99, &pb.GetBaseStationRequest{BsEui: "1122334455667788"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code(), "cross-tenant BS access should return NotFound")
}
