package grpc

import (
	"context"
	"errors"
	"strconv"
	"time"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/roles"
)

// RoleResolver answers the roles a user holds in an organization; uuid.Nil as
// the organization asks for the organization-independent roles, and an unknown
// user is reported as roles.ErrUserNotFound.
type RoleResolver interface {
	Resolve(ctx context.Context, userID, orgID uuid.UUID) (authz.Roles, error)
}

// PrincipalRoleResolver answers the roles of every kind of caller KC-Core
// authorizes; an unknown service-account key is reported as roles.ErrAPIKeyNotFound.
type PrincipalRoleResolver interface {
	RoleResolver
	ResolveServiceAccount(ctx context.Context, keyID, orgID uuid.UUID) (authz.Roles, error)
}

// UserLookup retrieves user information for server-admin checks; an unknown
// user is reported as admin.ErrUserNotFound.
type UserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// IdentityInternalService implements IdentityInternalServiceServer for
// peer-to-peer org/tenant resolution and API key validation.
type IdentityInternalService struct {
	pb.UnimplementedIdentityInternalServiceServer
	log         logger.Logger
	orgResolver org.Resolver
	apiKeyRepo  APIKeyLookup
	roles       PrincipalRoleResolver
	userSvc     UserLookup
	eventWriter audit.EventWriter
}

// APIKeyLookup abstracts API key repository operations for internal service.
type APIKeyLookup interface {
	GetByHash(ctx context.Context, hash string) (APIKeyInfo, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// APIKeyInfo contains API key fields needed for validation responses.
type APIKeyInfo struct {
	ID             uuid.UUID
	TenantID       int64
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	IsActive       bool
	IsExpired      bool
}

const componentIdentityInternalService = "identity-internal-service"

// NewIdentityInternalService creates a new IdentityInternalService.
// componentIdentityInternalService labels this component in structured logs.
func NewIdentityInternalService(resolver org.Resolver, apiKeyRepo APIKeyLookup, roles PrincipalRoleResolver, userSvc UserLookup, eventWriter audit.EventWriter, log logger.Logger) *IdentityInternalService {
	return &IdentityInternalService{
		log:         log.WithField(logger.FieldComponent, componentIdentityInternalService),
		orgResolver: resolver,
		apiKeyRepo:  apiKeyRepo,
		roles:       roles,
		userSvc:     userSvc,
		eventWriter: eventWriter,
	}
}

// ResolveOrg resolves an organization UUID to its tenant ID.
func (s *IdentityInternalService) ResolveOrg(ctx context.Context, req *pb.ResolveOrgRequest) (*pb.ResolveOrgResponse, error) {
	if req.OrgId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired))
	}

	orgUUID, err := uuid.Parse(req.OrgId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat))
	}

	tenantID, err := s.orgResolver.LookupTenant(ctx, orgUUID)
	if err != nil {
		s.log.ErrorContext(ctx, LogOrgResolutionFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound))
	}

	return &pb.ResolveOrgResponse{TenantId: tenantID}, nil
}

// GetDefaultOrgForTenant returns the default organization UUID for a tenant.
func (s *IdentityInternalService) GetDefaultOrgForTenant(ctx context.Context, req *pb.GetDefaultOrgForTenantRequest) (*pb.GetDefaultOrgForTenantResponse, error) {
	if req.TenantId == 0 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantRequired))
	}

	orgUUID, err := s.orgResolver.GetDefaultOrgForTenant(ctx, req.TenantId)
	if err != nil {
		s.log.ErrorContext(ctx, LogDefaultOrgLookupFailed, logger.FieldTenantIDSnake, req.TenantId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound))
	}

	return &pb.GetDefaultOrgForTenantResponse{OrgId: orgUUID.String()}, nil
}

// GetUserRoles resolves the roles a caller holds while acting in an
// organization; without one, the caller's organization-independent roles.
func (s *IdentityInternalService) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	principal, err := parseRolesPrincipal(req)
	if err != nil {
		return nil, err
	}
	orgID, err := parseRolesOrg(req.OrgId)
	if err != nil {
		return nil, err
	}
	if s.roles == nil {
		return nil, tokenStatus(grpcerrors.ErrTokenServiceNotConfigured)
	}

	resolved, err := principal.resolve(ctx, s.roles, orgID)
	switch {
	case errors.Is(err, roles.ErrUserNotFound):
		return nil, tokenStatus(grpcerrors.ErrTokenUserNotFound)
	case errors.Is(err, roles.ErrAPIKeyNotFound):
		return nil, tokenStatus(grpcerrors.ErrTokenApiKeyNotFound)
	case err != nil:
		s.log.ErrorContext(ctx, LogRoleResolutionFailed, logger.FieldOrgIDSnake, req.OrgId,
			logger.FieldUserIDSnake, req.UserId, logger.FieldKeyIDSnake, req.ServiceAccountId, logger.FieldError, err)
		return nil, tokenStatus(grpcerrors.ErrTokenInternalError)
	}
	return &pb.GetUserRolesResponse{Roles: rolesToProto(resolved)}, nil
}

// parseRolesOrg reads the acting organization; an empty one is uuid.Nil.
func parseRolesOrg(orgID string) (uuid.UUID, error) {
	if orgID == "" {
		return uuid.Nil, nil
	}
	parsed, err := uuid.Parse(orgID)
	if err != nil {
		return uuid.Nil, tokenStatus(grpcerrors.ErrTokenInvalidOrgIDFormat)
	}
	return parsed, nil
}

// rolesPrincipal is the caller GetUserRoles answers for: a user, or an organization service-account key.
type rolesPrincipal struct {
	userID         uuid.UUID
	serviceAccount uuid.UUID
}

// parseRolesPrincipal accepts exactly one of user_id and service_account_id.
func parseRolesPrincipal(req *pb.GetUserRolesRequest) (rolesPrincipal, error) {
	switch {
	case req.UserId != "" && req.ServiceAccountId != "":
		return rolesPrincipal{}, tokenStatus(grpcerrors.ErrTokenInvalidRequest)
	case req.ServiceAccountId != "":
		keyID, err := uuid.Parse(req.ServiceAccountId)
		if err != nil {
			return rolesPrincipal{}, tokenStatus(grpcerrors.ErrTokenInvalidAPIKeyIDFormat)
		}
		return rolesPrincipal{serviceAccount: keyID}, nil
	case req.UserId == "":
		return rolesPrincipal{}, tokenStatus(grpcerrors.ErrTokenUserIDRequired)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return rolesPrincipal{}, tokenStatus(grpcerrors.ErrTokenInvalidUserIDFormat)
	}
	return rolesPrincipal{userID: userID}, nil
}

func (p rolesPrincipal) resolve(ctx context.Context, resolver PrincipalRoleResolver, orgID uuid.UUID) (authz.Roles, error) {
	if p.serviceAccount != uuid.Nil {
		return resolver.ResolveServiceAccount(ctx, p.serviceAccount, orgID)
	}
	return resolver.Resolve(ctx, p.userID, orgID)
}

func tokenStatus(token string) error {
	return status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
}

// CheckServerAdmin checks whether a user has the server-level IsAdmin flag set.
func (s *IdentityInternalService) CheckServerAdmin(ctx context.Context, req *pb.CheckServerAdminRequest) (*pb.CheckServerAdminResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserIDRequired))
	}

	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
	}

	if s.userSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	user, err := s.userSvc.GetByID(ctx, userUUID)
	if err != nil {
		s.log.ErrorContext(ctx, LogServerAdminCheckFailed, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
		if errors.Is(err, admin.ErrUserNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserNotFound))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	return &pb.CheckServerAdminResponse{IsAdmin: authz.Effective(user, nil).Admin}, nil
}

// RecordPlatformEvent persists a platform-level event from stateless services (e.g., KC-Gateway).
func (s *IdentityInternalService) RecordPlatformEvent(ctx context.Context, req *pb.RecordPlatformEventRequest) (*pb.RecordPlatformEventResponse, error) {
	if s.eventWriter == nil {
		return &pb.RecordPlatformEventResponse{}, nil
	}

	event := &models.SystemEvent{
		TenantID:    strconv.FormatInt(req.TenantId, 10),
		EventType:   req.EventType,
		Category:    req.Category,
		Severity:    req.Severity,
		SourceType:  req.SourceType,
		SourceName:  req.SourceName,
		UserID:      req.UserId,
		Title:       req.Title,
		Description: req.Description,
		Details:     req.Data,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.eventWriter.CreateEvent(ctx, event); err != nil {
		s.log.ErrorContext(ctx, LogRecordPlatformEventFailed, logger.FieldError, err, logger.FieldEventType, req.EventType)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	return &pb.RecordPlatformEventResponse{}, nil
}
