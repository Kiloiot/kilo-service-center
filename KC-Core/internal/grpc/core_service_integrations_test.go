// Package grpc provides gRPC handler tests for integration CRUD.
package grpc

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// ============================================================================
// Mock Integration Service
// ============================================================================

// mockIntegrationService implements grpcservices.IntegrationService for testing
type mockIntegrationService struct {
	createFunc  func(ctx context.Context, tenantID int64, req *grpcservices.IntegrationCreateRequest) (*models.Integration, error)
	getByIDFunc func(ctx context.Context, tenantID int64, id int64) (*models.Integration, error)
	updateFunc  func(ctx context.Context, tenantID int64, id int64, req *grpcservices.IntegrationUpdateRequest) (*models.Integration, error)
	deleteFunc  func(ctx context.Context, tenantID int64, id int64) error
	listFunc    func(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error)
}

func (m *mockIntegrationService) Create(ctx context.Context, tenantID int64, req *grpcservices.IntegrationCreateRequest) (*models.Integration, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, tenantID, req)
	}
	return nil, nil
}

func (m *mockIntegrationService) GetByID(ctx context.Context, tenantID int64, id int64) (*models.Integration, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, tenantID, id)
	}
	return nil, nil
}

func (m *mockIntegrationService) Update(ctx context.Context, tenantID int64, id int64, req *grpcservices.IntegrationUpdateRequest) (*models.Integration, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, tenantID, id, req)
	}
	return nil, nil
}

func (m *mockIntegrationService) Delete(ctx context.Context, tenantID int64, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, tenantID, id)
	}
	return nil
}

func (m *mockIntegrationService) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, tenantID, limit, offset)
	}
	return nil, 0, nil
}

// ============================================================================
// Test Helpers
// ============================================================================

func createTestIntegration() *models.Integration {
	orgID := uuid.New()
	desc := "Test integration description"
	config := json.RawMessage(`{"url": "https://example.com/webhook"}`)
	return &models.Integration{
		ID:             1,
		OrgID:          orgID,
		TenantID:       100,
		Name:           "Test Integration",
		Description:    &desc,
		Type:           models.IntegrationTypeHTTP,
		Config:         config,
		DeliveryFormat: models.IntegrationDeliveryFormatJSON,
		Status:         models.IntegrationStatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func createTestIntegrationService() *CoreService {
	// Create minimal service with mocks for required dependencies
	svc := &CoreService{
		log: testLogger{},
	}
	return svc
}

// testLogger implements logger.Logger interface for testing
type testLogger struct{}

func (l testLogger) Info(_ string, _ ...interface{})                            {}
func (l testLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (l testLogger) Debug(_ string, _ ...interface{})                           {}
func (l testLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (l testLogger) Warn(_ string, _ ...interface{})                            {}
func (l testLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (l testLogger) Error(_ string, _ ...interface{})                           {}
func (l testLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (l testLogger) Fatal(_ string, _ ...interface{})                           {}
func (l testLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (l testLogger) WithField(_ string, _ interface{}) logger.Logger            { return l }
func (l testLogger) WithFields(_ map[string]interface{}) logger.Logger          { return l }

// ============================================================================
// CreateIntegration Tests
// ============================================================================

func TestCreateIntegration_ServiceNotConfigured(t *testing.T) {
	svc := createTestIntegrationService()
	// integrationSvc is nil by default

	ctx := pkgcontext.WithTenantID(context.Background(), 100)

	req := &pb.CreateIntegrationRequest{
		Name: "Test",
		Type: "http",
	}

	_, err := svc.CreateIntegration(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured), st.Code())
}

func TestCreateIntegration_MissingTenant(t *testing.T) {
	mockSvc := &mockIntegrationService{}
	svc := createTestIntegrationService()
	svc.integrationSvc = mockSvc

	// Context without tenant ID
	ctx := context.Background()

	req := &pb.CreateIntegrationRequest{
		Name: "Test",
		Type: "http",
	}

	_, err := svc.CreateIntegration(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestCreateIntegration_MissingName(t *testing.T) {
	mockSvc := &mockIntegrationService{}
	svc := createTestIntegrationService()
	svc.integrationSvc = mockSvc

	ctx := pkgcontext.WithTenantID(context.Background(), 100)
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())

	req := &pb.CreateIntegrationRequest{
		Name: "", // Empty name
		Type: "http",
	}

	_, err := svc.CreateIntegration(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired), st.Code())
}

func TestCreateIntegration_MissingConfig(t *testing.T) {
	mockSvc := &mockIntegrationService{}
	svc := createTestIntegrationService()
	svc.integrationSvc = mockSvc

	ctx := pkgcontext.WithTenantID(context.Background(), 100)
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())

	req := &pb.CreateIntegrationRequest{
		Name:   "Test Integration",
		Type:   "http",
		Config: nil, // Nil config should fail validation
	}

	_, err := svc.CreateIntegration(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenConfigRequired), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenConfigRequired))
}

func TestCreateIntegration_Success(t *testing.T) {
	integration := createTestIntegration()
	mockSvc := &mockIntegrationService{
		createFunc: func(_ context.Context, tenantID int64, req *grpcservices.IntegrationCreateRequest) (*models.Integration, error) {
			assert.Equal(t, int64(100), tenantID)
			assert.Equal(t, "Test Integration", req.Name)
			return integration, nil
		},
	}
	svc := createTestIntegrationService()
	svc.integrationSvc = mockSvc

	ctx := pkgcontext.WithTenantID(context.Background(), 100)
	ctx = pkgcontext.WithOrganizationID(ctx, integration.OrgID)

	configStruct, err := structpb.NewStruct(map[string]interface{}{
		"url":    "https://example.com/webhook",
		"method": "POST",
	})
	require.NoError(t, err)

	req := &pb.CreateIntegrationRequest{
		Name:   "Test Integration",
		Type:   "http",
		Config: configStruct,
	}

	resp, err := svc.CreateIntegration(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "Test Integration", resp.Name)
	assert.Equal(t, models.IntegrationTypeHTTP, resp.Type)
	assert.Equal(t, models.IntegrationStatusActive, resp.Status)
}

// ============================================================================
// GetIntegration Tests
// ============================================================================

func TestGetIntegration_ServiceNotConfigured(t *testing.T) {
	svc := createTestIntegrationService()

	ctx := pkgcontext.WithTenantID(context.Background(), 100)

	req := &pb.GetIntegrationRequest{Id: 1}

	_, err := svc.GetIntegration(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured), st.Code())
}

func TestGetIntegration_MissingID(t *testing.T) {
	mockSvc := &mockIntegrationService{}
	svc := createTestIntegrationService()
	svc.integrationSvc = mockSvc

	ctx := pkgcontext.WithTenantID(context.Background(), 100)

	req := &pb.GetIntegrationRequest{Id: 0}

	_, err := svc.GetIntegration(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired), st.Code())
}

func TestGetIntegration_Success(t *testing.T) {
	integration := createTestIntegration()
	mockSvc := &mockIntegrationService{
		getByIDFunc: func(_ context.Context, tenantID int64, id int64) (*models.Integration, error) {
			assert.Equal(t, int64(100), tenantID)
			assert.Equal(t, int64(1), id)
			return integration, nil
		},
	}
	svc := createTestIntegrationService()
	svc.integrationSvc = mockSvc

	ctx := pkgcontext.WithTenantID(context.Background(), 100)

	req := &pb.GetIntegrationRequest{Id: 1}

	resp, err := svc.GetIntegration(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "Test Integration", resp.Name)
}

// ============================================================================
// ListIntegrations Tests
// ============================================================================

func TestListIntegrations_ServiceNotConfigured(t *testing.T) {
	svc := createTestIntegrationService()

	ctx := pkgcontext.WithTenantID(context.Background(), 100)

	req := &pb.ListIntegrationsRequest{}

	_, err := svc.ListIntegrations(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured), st.Code())
}

func TestListIntegrations_Success(t *testing.T) {
	integration := createTestIntegration()
	mockSvc := &mockIntegrationService{
		listFunc: func(_ context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error) {
			assert.Equal(t, int64(100), tenantID)
			assert.Equal(t, 20, limit) // Default limit
			assert.Equal(t, 0, offset)
			return []*models.Integration{integration}, 1, nil
		},
	}
	svc := createTestIntegrationService()
	svc.integrationSvc = mockSvc

	ctx := pkgcontext.WithTenantID(context.Background(), 100)

	req := &pb.ListIntegrationsRequest{}

	resp, err := svc.ListIntegrations(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, int64(1), resp.TotalCount)
	require.Len(t, resp.Integrations, 1)
	assert.Equal(t, int64(1), resp.Integrations[0].Id)
}

// ============================================================================
// DeleteIntegration Tests
// ============================================================================

func TestDeleteIntegration_ServiceNotConfigured(t *testing.T) {
	svc := createTestIntegrationService()

	ctx := pkgcontext.WithTenantID(context.Background(), 100)

	req := &pb.DeleteIntegrationRequest{Id: 1}

	_, err := svc.DeleteIntegration(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured), st.Code())
}

func TestDeleteIntegration_Success(t *testing.T) {
	mockSvc := &mockIntegrationService{
		deleteFunc: func(_ context.Context, tenantID int64, id int64) error {
			assert.Equal(t, int64(100), tenantID)
			assert.Equal(t, int64(1), id)
			return nil
		},
	}
	svc := createTestIntegrationService()
	svc.integrationSvc = mockSvc

	ctx := pkgcontext.WithTenantID(context.Background(), 100)

	req := &pb.DeleteIntegrationRequest{Id: 1}

	resp, err := svc.DeleteIntegration(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}
