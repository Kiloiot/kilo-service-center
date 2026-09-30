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
			s.logger.ErrorContext(ctx, LogOrgCallerTenantNotFound, logger.FieldTenantID, req.TenantID, logger.FieldError, err)
			return nil, fmt.Errorf("%s: %w", errOpValidateTenant, err)
		}
		tenantID = tenant.ID
	} else {
		// Create a new tenant for the organization
		tenant, err := s.tenantStore.CreateTenant(ctx, req.Name, "")
		if err != nil {
			s.logger.ErrorContext(ctx, LogOrgTenantCreateFailed, logger.FieldName, req.Name, logger.FieldError, err)
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
		Tags:        models.HstoreMap(req.Tags),
	}

	if err := s.orgStore.Create(ctx, org); err != nil {
		if rollbackTenant {
			s.logger.WarnContext(ctx, LogOrgTenantRollback,
				logger.FieldTenantIDCamel, tenantID, logger.FieldOrgName, req.Name)
			if delErr := s.tenantStore.DeleteTenant(ctx, tenantID); delErr != nil {
				s.logger.ErrorContext(ctx, LogOrgTenantRollbackFailed, logger.FieldTenantIDCamel, tenantID, logger.FieldError, delErr)
			}
		}
		return nil, fmt.Errorf("%s: %w", errOpCreateOrganization, err)
	}

	s.logger.InfoContext(ctx, LogOrgCreatedWithTenant, logger.FieldOrgIDCamel, org.OrgID, logger.FieldTenantIDCamel, tenantID)
	return org, nil
}

// GetByID retrieves an organization by ID, scoped to a tenant.
func (s *OrganizationAdminService) GetByID(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error) {
	org, err := s.orgStore.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrOrganizationNotFound
		}
		s.logger.ErrorContext(ctx, LogOrgGetFailed, logger.FieldOrgIDCamel, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetOrganization, err)
	}
	return org, nil
}

// Update modifies an existing organization, scoped to a tenant.
func (s *OrganizationAdminService) Update(ctx context.Context, id uuid.UUID, tenantID int64, req *grpcservices.OrganizationUpdateRequest) (*models.Organization, error) {
	existing, err := s.orgStore.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetOrganizationForUpdate, err)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if len(updates) == 0 {
		return existing, nil
	}

	if err := s.orgStore.Update(ctx, id, tenantID, updates); err != nil {
		s.logger.ErrorContext(ctx, LogOrgUpdateFailed, logger.FieldOrgIDCamel, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpUpdateOrganization, err)
	}

	org, err := s.orgStore.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpGetUpdatedOrganization, err)
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
		return fmt.Errorf("%s: %w", errOpGetOrganizationForDelete, err)
	}

	if err := s.orgStore.Delete(ctx, id, tenantID); err != nil {
		s.logger.ErrorContext(ctx, LogOrgDeleteFailed, logger.FieldOrgIDCamel, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpDeleteOrganization, err)
	}

	s.logger.InfoContext(ctx, LogOrgDeleted, logger.FieldOrgIDCamel, id)
	return nil
}

// List returns paginated organizations scoped to a tenant.
func (s *OrganizationAdminService) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Organization, int64, error) {
	orgs, total, err := s.orgStore.ListOrganizations(ctx, &tenantID, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogOrgListFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpListOrganizations, err)
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
		s.logger.ErrorContext(ctx, LogOrgGetUnscopedFailed, logger.FieldOrgIDCamel, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetOrganizationUnscoped, err)
	}
	return org, nil
}

// ListAll returns paginated organizations across all tenants.
func (s *OrganizationAdminService) ListAll(ctx context.Context, limit, offset int) ([]*models.Organization, int64, error) {
	return s.orgStore.ListOrganizations(ctx, nil, limit, offset)
}

// Ensure OrganizationAdminService implements grpcservices.OrganizationService
var _ grpcservices.OrganizationService = (*OrganizationAdminService)(nil)
