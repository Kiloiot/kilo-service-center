package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/google/uuid"
)

// OrganizationAdminService implements grpcservices.OrganizationService.
type OrganizationAdminService struct {
	orgStore    OrganizationStore
	tenantStore TenantStore
	logger      logger.Logger
}

// NewOrganizationAdminService creates a new organization admin service.
func NewOrganizationAdminService(orgStore OrganizationStore, tenantStore TenantStore, log logger.Logger) *OrganizationAdminService {
	return &OrganizationAdminService{
		orgStore:    orgStore,
		tenantStore: tenantStore,
		logger:      log,
	}
}

// Create creates a new organization, using an existing tenant if TenantID is provided.
func (s *OrganizationAdminService) Create(ctx context.Context, req *grpcservices.OrganizationCreateRequest) (*models.Organization, error) {
	if req.Name == "" {
		return nil, ErrOrganizationNameRequired
	}

	var tenantID int64
	var rollbackTenant bool

	if req.TenantID > 0 {
		// Validate the caller's tenant exists
		tenant, err := s.tenantStore.GetTenant(ctx, req.TenantID)
		if err != nil {
			s.logger.ErrorContext(ctx, "caller tenant not found", "tenantID", req.TenantID, "error", err)
			return nil, fmt.Errorf("validate tenant: %w", err)
		}
		tenantID = tenant.ID
	} else {
		// Create a new tenant for the organization
		tenant, err := s.tenantStore.CreateTenant(ctx, req.Name, "")
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create tenant for organization", "name", req.Name, "error", err)
			return nil, ErrTenantCreationFailed
		}
		tenantID = tenant.ID
		rollbackTenant = true
	}

	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	org := &models.Organization{
		OrgID:       uuid.New(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: description,
		State:       models.OrganizationStateActive,
	}

	if err := s.orgStore.Create(ctx, org); err != nil {
		if rollbackTenant {
			s.logger.WarnContext(ctx, "rolling back tenant creation after org create failure",
				"tenantId", tenantID, "orgName", req.Name)
			if delErr := s.tenantStore.DeleteTenant(ctx, tenantID); delErr != nil {
				s.logger.ErrorContext(ctx, "failed to rollback tenant", "tenantId", tenantID, "error", delErr)
			}
		}
		return nil, fmt.Errorf("create organization: %w", err)
	}

	s.logger.InfoContext(ctx, "created organization with tenant", "orgId", org.OrgID, "tenantId", tenantID)
	return org, nil
}

// GetByID retrieves an organization by ID, scoped to a tenant.
func (s *OrganizationAdminService) GetByID(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error) {
	org, err := s.orgStore.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrOrganizationNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get organization", "orgId", id, "error", err)
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return org, nil
}

// Update modifies an existing organization, scoped to a tenant.
func (s *OrganizationAdminService) Update(ctx context.Context, id uuid.UUID, tenantID int64, req *grpcservices.OrganizationUpdateRequest) (*models.Organization, error) {
	_, err := s.orgStore.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("get organization for update: %w", err)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if len(updates) == 0 {
		org, _ := s.orgStore.GetByID(ctx, id, tenantID)
		return org, nil
	}

	if err := s.orgStore.Update(ctx, id, tenantID, updates); err != nil {
		s.logger.ErrorContext(ctx, "failed to update organization", "orgId", id, "error", err)
		return nil, fmt.Errorf("update organization: %w", err)
	}

	org, err := s.orgStore.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get updated organization: %w", err)
	}

	return org, nil
}

// Delete permanently removes an organization, scoped to a tenant.
func (s *OrganizationAdminService) Delete(ctx context.Context, id uuid.UUID, tenantID int64) error {
	_, err := s.orgStore.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("get organization for delete: %w", err)
	}

	if err := s.orgStore.Delete(ctx, id, tenantID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete organization", "orgId", id, "error", err)
		return fmt.Errorf("delete organization: %w", err)
	}

	s.logger.InfoContext(ctx, "deleted organization", "orgId", id)
	return nil
}

// List returns paginated organizations scoped to a tenant.
func (s *OrganizationAdminService) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Organization, int64, error) {
	orgs, total, err := s.orgStore.List(ctx, &tenantID, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list organizations", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list organizations: %w", err)
	}
	return orgs, total, nil
}

// GetByIDUnscoped retrieves an organization by ID without tenant scoping.
func (s *OrganizationAdminService) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	org, err := s.orgStore.GetByIDUnscoped(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrOrganizationNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get organization unscoped", "orgId", id, "error", err)
		return nil, fmt.Errorf("get organization unscoped: %w", err)
	}
	return org, nil
}

// ListAll returns paginated organizations across all tenants.
func (s *OrganizationAdminService) ListAll(ctx context.Context, limit, offset int) ([]*models.Organization, int64, error) {
	return s.orgStore.List(ctx, nil, limit, offset)
}

// Ensure OrganizationAdminService implements grpcservices.OrganizationService
var _ grpcservices.OrganizationService = (*OrganizationAdminService)(nil)
