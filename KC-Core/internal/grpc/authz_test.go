package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const authzTestTenant int64 = 1

func TestRequireAdmin_ReadsTheResolvedRoles(t *testing.T) {
	cases := []struct {
		name  string
		roles authz.Roles
		want  codes.Code
	}{
		{name: "no roles", roles: authz.Roles{}, want: codes.PermissionDenied},
		{name: "every manager role without admin", roles: authz.Roles{TenantManager: true, BaseStationManager: true, EndpointManager: true}, want: codes.PermissionDenied},
		{name: "admin", roles: authz.AllRoles, want: codes.OK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := requireAdmin(authz.WithRoles(testutil.TestContext(), tc.roles))
			assert.Equal(t, tc.want, status.Code(err), "%v", err)
		})
	}
}

// systemCatalogSvc records which catalog writes reach the service.
type systemCatalogSvc struct {
	grpcservices.BlueprintService
	created  bool
	existing *models.Manufacturer
}

func (s *systemCatalogSvc) CreateManufacturer(_ context.Context, req *grpcservices.ManufacturerCreateRequest) (*models.Manufacturer, error) {
	s.created = true
	return &models.Manufacturer{ID: uuid.New(), Name: req.Name, IsSystem: req.IsSystem}, nil
}

func (s *systemCatalogSvc) GetManufacturer(context.Context, uuid.UUID) (*models.Manufacturer, error) {
	return s.existing, nil
}

// TestSystemCatalog_WritesNeedAnAdministrator: an endpoint manager curates
// the tenant catalog, only an administrator the system catalog.
func TestSystemCatalog_WritesNeedAnAdministrator(t *testing.T) {
	endpointManager := authz.Roles{EndpointManager: true}
	cases := []struct {
		name     string
		roles    authz.Roles
		isSystem bool
		want     codes.Code
	}{
		{name: "endpoint manager adds a tenant entry", roles: endpointManager, isSystem: false, want: codes.OK},
		{name: "endpoint manager cannot add a system entry", roles: endpointManager, isSystem: true, want: grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired)},
		{name: "administrator adds a system entry", roles: authz.AllRoles, isSystem: true, want: codes.OK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			catalog := &systemCatalogSvc{}
			svc := testCoreService(coreFields{blueprintSvc: catalog, log: &mockLogger{}})
			ctx := authz.WithRoles(testutil.TestContextWithTenant(authzTestTenant), tc.roles)

			_, err := svc.CreateManufacturer(ctx, &pb.CreateManufacturerRequest{Name: t.Name(), IsSystem: tc.isSystem})

			assert.Equal(t, tc.want, status.Code(err), "%v", err)
			assert.Equal(t, tc.want == codes.OK, catalog.created, "only an admitted write reaches the catalog")
		})
	}

	t.Run("endpoint manager cannot delete a system entry", func(t *testing.T) {
		catalog := &systemCatalogSvc{existing: &models.Manufacturer{ID: uuid.New(), IsSystem: true}}
		svc := testCoreService(coreFields{blueprintSvc: catalog, log: &mockLogger{}})
		ctx := authz.WithRoles(testutil.TestContextWithTenant(authzTestTenant), endpointManager)

		_, err := svc.DeleteManufacturer(ctx, &pb.DeleteManufacturerRequest{Id: catalog.existing.ID.String()})

		assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), status.Code(err), "%v", err)
	})
}
