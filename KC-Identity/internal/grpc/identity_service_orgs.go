package grpc

import (
	"context"
	"fmt"
	"math"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

// auditKeyOrgID is the audit event detail key for organization identifiers.
const (
	auditKeyOrgID  = "orgId"
	auditKeyUserID = "userId"
)

// CreateOrganization creates a new organization.
func (s *IdentityService) CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}

	// TenantID 0 makes the service provision a dedicated tenant per organization,
	// so base stations (scoped only by tenant_id) stay isolated between organizations.
	createReq := &grpcservices.OrganizationCreateRequest{
		Name:        req.Name,
		Description: req.Description,
		TenantID:    0,
		Tags:        req.Tags,
	}

	org, err := s.orgSvc.Create(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogCreateOrganizationFailed, logger.FieldName, req.Name, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateOrgFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateOrgFailed))
	}

	// Emit CRUD event
	sourceID := org.OrgID
	s.audit.Record(ctx, audit.Event{
		TenantID:    org.TenantID,
		SourceID:    &sourceID,
		EventType:   models.EventTypeOrgCreated,
		Title:       models.EventTitleOrgCreated,
		Description: fmt.Sprintf(models.EventDescriptionOrgCreated, org.Name),
		SourceName:  org.Name,
		Details:     map[string]any{auditKeyOrgID: org.OrgID.String(), "name": org.Name},
	})

	return &pb.CreateOrganizationResponse{
		Organization: orgModelToProto(org),
	}, nil
}

// GetOrganization returns an organization by ID.
func (s *IdentityService) GetOrganization(ctx context.Context, req *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	orgID, tenantID, err := s.validateOrgAccessUnscoped(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	org, err := s.orgSvc.GetByID(ctx, orgID, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetOrganizationFailed, logger.FieldOrgIDSnake, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound))
	}

	return &pb.GetOrganizationResponse{
		Organization: orgModelToProto(org),
	}, nil
}

// UpdateOrganization updates an organization.
func (s *IdentityService) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	// Tags are fixed at creation, so an integration can tell the organization it created apart.
	if len(req.Tags) > 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgTagsImmutable),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgTagsImmutable))
	}
	orgID, tenantID, err := s.validateOrgAccessUnscoped(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	updateReq := &grpcservices.OrganizationUpdateRequest{}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}

	org, err := s.orgSvc.Update(ctx, orgID, tenantID, updateReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogUpdateOrganizationFailed, logger.FieldOrgIDSnake, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateOrgFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateOrgFailed))
	}

	// Emit CRUD event
	sourceID := org.OrgID
	s.audit.Record(ctx, audit.Event{
		TenantID:    tenantID,
		SourceID:    &sourceID,
		EventType:   models.EventTypeOrgUpdated,
		Title:       models.EventTitleOrgUpdated,
		Description: fmt.Sprintf(models.EventDescriptionOrgUpdated, org.Name),
		SourceName:  org.Name,
		Details:     map[string]any{auditKeyOrgID: org.OrgID.String(), "name": org.Name},
	})

	return &pb.UpdateOrganizationResponse{
		Organization: orgModelToProto(org),
	}, nil
}

// DeleteOrganization deletes an organization.
func (s *IdentityService) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	orgID, tenantID, err := s.validateOrgAccessUnscoped(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// Snapshot the org's keys before Delete removes them in-tx, so each can be audited.
	deletedKeys := s.listOrgAPIKeys(ctx, tenantID, orgID)

	if err := s.orgSvc.Delete(ctx, orgID, tenantID); err != nil {
		s.log.ErrorContext(ctx, LogDeleteOrganizationFailed, logger.FieldOrgIDSnake, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteOrgFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteOrgFailed))
	}

	// The org's tenant is removed with its last org, so the platform tenant keeps the record.
	for _, k := range deletedKeys {
		s.audit.Record(ctx, apiKeyDeletedAuditEvent(s.platformTenantID, orgID, k.ID, k.Name))
	}
	sourceID := orgID
	s.audit.Record(ctx, audit.Event{
		TenantID:    s.platformTenantID,
		SourceID:    &sourceID,
		EventType:   models.EventTypeOrgDeleted,
		Title:       models.EventTitleOrgDeleted,
		Description: fmt.Sprintf(models.EventDescriptionOrgDeleted, req.Id),
		Details:     map[string]any{auditKeyOrgID: req.Id, models.EventDetailKeyTenantID: tenantID},
	})

	return &pb.DeleteOrganizationResponse{Success: true}, nil
}

// apiKeyListPageSize is the page size for enumerating an org's API keys for the audit trail.
const apiKeyListPageSize = 200

// listOrgAPIKeys returns all API keys of an org (fully paginated), for the deletion audit.
func (s *IdentityService) listOrgAPIKeys(ctx context.Context, tenantID int64, orgID uuid.UUID) []*models.APIKey {
	if s.apiKeySvc == nil {
		return nil
	}
	var all []*models.APIKey
	for offset := 0; ; offset += apiKeyListPageSize {
		batch, _, err := s.apiKeySvc.List(ctx, tenantID, orgID, nil, apiKeyListPageSize, offset)
		if err != nil {
			s.log.WarnContext(ctx, LogListOrgAPIKeysForAuditFailed, logger.FieldOrgIDSnake, orgID.String(), logger.FieldError, err)
			break
		}
		all = append(all, batch...)
		if len(batch) < apiKeyListPageSize {
			break
		}
	}
	return all
}

// ListOrganizations returns a list of organizations.
func (s *IdentityService) ListOrganizations(ctx context.Context, req *pb.ListOrganizationsRequest) (*pb.ListOrganizationsResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	limit := grpcerrors.ClampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := grpcerrors.ParsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	orgs, total, err := s.orgSvc.ListAll(ctx, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListOrganizationsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListOrgsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListOrgsFailed))
	}

	var pbOrgs []*pb.Organization
	for _, o := range orgs {
		pbOrgs = append(pbOrgs, orgModelToProto(o))
	}

	var nextToken string
	if offset+limit < int(total) {
		nextToken = grpcerrors.GeneratePaginationToken(offset + limit)
	}

	if total > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	totalCount := int32(total) //nolint:gosec // bounds checked above
	return &pb.ListOrganizationsResponse{
		Organizations: pbOrgs,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}
