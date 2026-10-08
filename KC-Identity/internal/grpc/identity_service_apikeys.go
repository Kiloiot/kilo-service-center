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
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// apiKeyDeletedAuditEvent builds the audit event for a deleted API key.
func apiKeyDeletedAuditEvent(tenantID int64, orgID, keyID uuid.UUID, keyName string) audit.Event {
	sourceID := orgID
	return audit.Event{
		TenantID:    tenantID,
		SourceID:    &sourceID,
		EventType:   models.EventTypeAPIKeyDeleted,
		Title:       models.EventTitleAPIKeyDeleted,
		Description: fmt.Sprintf(models.EventDescriptionAPIKeyDeletedFmt, keyName),
		SourceName:  keyName,
		Details:     map[string]any{"keyId": keyID.String(), auditKeyOrgID: orgID.String()},
	}
}

// CreateApiKey creates a new API key.
// Tenant-isolated API key creation with ownership enforcement. The raw key is
// returned only once its creation is recorded; a key whose record could not be
// written stays unusable, since only its hash is stored.
//
//revive:disable-next-line:var-naming
func (s *IdentityService) CreateApiKey(ctx context.Context, req *pb.CreateApiKeyRequest) (*pb.CreateApiKeyResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.apiKeySvc == nil || s.orgDirectory == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	orgID, err := s.resolveAPIKeyOrgID(ctx, req)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}

	userID, err := s.resolveAPIKeyUserID(ctx, req.KeyType)
	if err != nil {
		return nil, err
	}

	tenantID, err := s.apiKeyTenant(ctx, orgID)
	if err != nil {
		return nil, err
	}

	createReq := &grpcservices.APIKeyCreateRequest{
		TenantID: tenantID,
		OrgID:    orgID,
		UserID:   userID,
		Name:     req.Name,
		KeyType:  req.KeyType,
	}

	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		createReq.ExpiresAt = &t
	}

	resp, err := s.apiKeySvc.Create(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogCreateAPIKeyFailed, logger.FieldName, req.Name, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateApiKeyFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateApiKeyFailed))
	}

	if err := s.disclosures.RecordRequired(ctx, apiKeyCreatedAuditEvent(req, resp, orgID, tenantID, userID)); err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	return &pb.CreateApiKeyResponse{
		RawKey: resp.Key,
		ApiKey: apiKeyModelToProto(resp.APIKey),
	}, nil
}

// apiKeyTenant returns the tenant of the organization a key is scoped to, not
// the caller's: a base station created with the key inherits the key's tenant,
// so per-organization tenants keep them isolated.
func (s *IdentityService) apiKeyTenant(ctx context.Context, orgID uuid.UUID) (int64, error) {
	org, err := s.orgDirectory.GetByIDUnscoped(ctx, orgID)
	if err != nil {
		s.log.ErrorContext(ctx, LogResolveTenantForAPIKeyFailed, auditKeyOrgID, orgID.String(), logger.FieldError, err)
		return 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateApiKeyFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateApiKeyFailed))
	}
	return org.TenantID, nil
}

// resolveAPIKeyOrgID returns the org the key is scoped to: the request's org when
// set, otherwise the caller's context org.
func (s *IdentityService) resolveAPIKeyOrgID(ctx context.Context, req *pb.CreateApiKeyRequest) (uuid.UUID, error) {
	if req.OrganizationId != "" {
		orgID, err := uuid.Parse(req.OrganizationId)
		if err != nil {
			return uuid.Nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderInvalid),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderInvalid))
		}
		return orgID, nil
	}
	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil {
		return uuid.Nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderRequired))
	}
	return orgID, nil
}

// resolveAPIKeyUserID validates key_type and returns the bound user: nil for
// service-account keys, the authenticated user for user keys.
func (s *IdentityService) resolveAPIKeyUserID(ctx context.Context, keyType string) (*uuid.UUID, error) {
	switch keyType {
	case models.KeyTypeServiceAccount:
		return nil, nil
	case models.KeyTypeUser:
		uid, err := grpcerrors.GetUserFromContext(ctx)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingUserCtx),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMissingUserCtx))
		}
		return &uid, nil
	default:
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidApiKeyType),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidApiKeyType))
	}
}

// apiKeyCreatedAuditEvent builds the audit event for a created API key.
func apiKeyCreatedAuditEvent(req *pb.CreateApiKeyRequest, resp *grpcservices.APIKeyCreateResponse, orgID uuid.UUID, tenantID int64, userID *uuid.UUID) audit.Event {
	details := map[string]any{
		"keyId":       resp.APIKey.ID.String(),
		"keyPrefix":   resp.APIKey.KeyPrefix,
		"keyType":     req.KeyType,
		auditKeyOrgID: orgID.String(),
	}
	var auditUserID string
	if userID != nil {
		details[auditKeyUserID] = userID.String()
		auditUserID = userID.String()
	}
	sourceID := orgID
	return audit.Event{
		TenantID:    tenantID,
		SourceID:    &sourceID,
		EventType:   models.EventTypeAPIKeyCreated,
		Title:       models.EventTitleAPIKeyCreated,
		Description: fmt.Sprintf(models.EventDescriptionAPIKeyCreatedFmt, req.Name, orgID.String()),
		SourceName:  req.Name,
		UserID:      auditUserID,
		Details:     details,
	}
}

// GetApiKey returns an API key by ID.
// Tenant-isolated API key retrieval with ownership check.
//
//revive:disable-next-line:var-naming
func (s *IdentityService) GetApiKey(ctx context.Context, req *pb.GetApiKeyRequest) (*pb.GetApiKeyResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.apiKeySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Extract org context for tenant isolation.
	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderRequired))
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	keyID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidAPIKeyIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidAPIKeyIDFormat))
	}

	// Use org-scoped query to enforce ownership.
	key, err := s.apiKeySvc.GetByIDAndOrg(ctx, keyID, orgID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetAPIKeyFailed, logger.FieldKeyIDSnake, req.Id, logger.FieldOrgIDSnake, orgID, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenApiKeyNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenApiKeyNotFound))
	}

	return &pb.GetApiKeyResponse{
		ApiKey: apiKeyModelToProto(key),
	}, nil
}

// DeleteApiKey deletes an API key.
// Tenant-isolated API key deletion with ownership check.
//
//revive:disable-next-line:var-naming
func (s *IdentityService) DeleteApiKey(ctx context.Context, req *pb.DeleteApiKeyRequest) (*pb.DeleteApiKeyResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.apiKeySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Extract org context for tenant isolation.
	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderRequired))
	}
	tenantID, err := grpcerrors.GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	keyID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidAPIKeyIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidAPIKeyIDFormat))
	}

	// Prefetch key metadata before deletion for audit event.
	var keyName string
	existingKey, prefetchErr := s.apiKeySvc.GetByIDAndOrg(ctx, keyID, orgID)
	if prefetchErr == nil && existingKey != nil {
		keyName = existingKey.Name
	}

	// Use org-scoped deletion to enforce ownership.
	if err := s.apiKeySvc.DeleteByIDAndOrg(ctx, keyID, orgID); err != nil {
		s.log.ErrorContext(ctx, LogDeleteAPIKeyFailed, logger.FieldKeyIDSnake, req.Id, logger.FieldOrgIDSnake, orgID, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteApiKeyFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteApiKeyFailed))
	}

	// Emit audit event for API key deletion.
	if keyName == "" {
		keyName = req.Id
	}
	s.audit.Record(ctx, apiKeyDeletedAuditEvent(tenantID, orgID, keyID, keyName))

	return &pb.DeleteApiKeyResponse{
		Success: true,
	}, nil
}

// ListApiKeys returns a list of API keys.
//
//revive:disable-next-line:var-naming
func (s *IdentityService) ListApiKeys(ctx context.Context, req *pb.ListApiKeysRequest) (*pb.ListApiKeysResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.apiKeySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := grpcerrors.GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderRequired))
	}

	limit := grpcerrors.ClampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := grpcerrors.ParsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	var userID *uuid.UUID
	if req.UserId != "" {
		id, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
		}
		userID = &id
	}

	keys, total, err := s.apiKeySvc.List(ctx, tenantID, orgID, userID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListAPIKeysFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListApiKeysFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListApiKeysFailed))
	}

	var pbKeys []*pb.ApiKey
	for _, k := range keys {
		pbKeys = append(pbKeys, apiKeyModelToProto(k))
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
	return &pb.ListApiKeysResponse{
		ApiKeys:       pbKeys,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}
