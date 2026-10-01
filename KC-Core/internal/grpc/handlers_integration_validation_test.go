package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/integrations"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

func integrationTestContext(tenantID int64) context.Context {
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), tenantID)
	return pkgcontext.WithOrganizationID(ctx, uuid.New())
}

func assertIntegrationStatus(t *testing.T, err error, token string) {
	t.Helper()
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(token), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(token), st.Message())
}

func TestCreateIntegration_InvalidTypeIsAnInvalidArgument(t *testing.T) {
	svc := createTestIntegrationService()
	svc.integrationSvc = &mockIntegrationService{
		createFunc: func(context.Context, int64, *grpcservices.IntegrationCreateRequest) (*models.Integration, error) {
			return nil, integrations.ErrInvalidType
		},
	}
	config, err := structpb.NewStruct(map[string]interface{}{"url": "https://example.com"})
	require.NoError(t, err)

	_, err = svc.CreateIntegration(integrationTestContext(100), &pb.CreateIntegrationRequest{Name: "n", Type: "smtp", Config: config})

	assertIntegrationStatus(t, err, grpcerrors.ErrTokenInvalidIntegrationType)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUpdateIntegration_InvalidStatusIsAnInvalidArgument(t *testing.T) {
	svc := createTestIntegrationService()
	svc.integrationSvc = &mockIntegrationService{
		updateFunc: func(context.Context, int64, int64, *grpcservices.IntegrationUpdateRequest) (*models.Integration, error) {
			return nil, integrations.ErrInvalidStatus
		},
	}

	_, err := svc.UpdateIntegration(integrationTestContext(100), &pb.UpdateIntegrationRequest{Id: 1, Status: "running"})

	assertIntegrationStatus(t, err, grpcerrors.ErrTokenInvalidIntegrationStatus)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUpdateIntegration_ForeignTenantGetsNotFound(t *testing.T) {
	owned := createTestIntegration()
	svc := createTestIntegrationService()
	svc.integrationSvc = &mockIntegrationService{
		updateFunc: func(_ context.Context, tenantID int64, id int64, _ *grpcservices.IntegrationUpdateRequest) (*models.Integration, error) {
			if tenantID != owned.TenantID || id != owned.ID {
				return nil, integrations.ErrIntegrationNotFound
			}
			return owned, nil
		},
	}

	_, err := svc.UpdateIntegration(integrationTestContext(owned.TenantID+1), &pb.UpdateIntegrationRequest{Id: owned.ID, Name: "renamed"})

	assertIntegrationStatus(t, err, grpcerrors.ErrTokenIntegrationNotFound)
}
