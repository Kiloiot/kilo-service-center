package grpc

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// MembershipLookup retrieves membership information for RBAC resolution.
type MembershipLookup interface {
	GetMembership(ctx context.Context, orgID, userID string) (string, bool, error)
}

// UserLookup retrieves user information for server-admin checks.
type UserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// IdentityInternalService implements IdentityInternalServiceServer for
// peer-to-peer org/tenant resolution and API key validation.
type IdentityInternalService struct {
	pb.UnimplementedIdentityInternalServiceServer
	log           logger.Logger
	orgResolver   org.Resolver
	apiKeyRepo    APIKeyLookup
	membershipSvc MembershipLookup
	userSvc       UserLookup
	eventWriter   grpcerrors.EventWriter
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

// NewIdentityInternalService creates a new IdentityInternalService.
func NewIdentityInternalService(resolver org.Resolver, apiKeyRepo APIKeyLookup, membershipSvc MembershipLookup, userSvc UserLookup, eventWriter grpcerrors.EventWriter) *IdentityInternalService {
	return &IdentityInternalService{
		log:           logger.Get().WithField("component", "identity-internal-service"),
		orgResolver:   resolver,
		apiKeyRepo:    apiKeyRepo,
		membershipSvc: membershipSvc,
		userSvc:       userSvc,
		eventWriter:   eventWriter,
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
		s.log.ErrorContext(ctx, "org resolution failed", "org_id", req.OrgId, "error", err)
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
		s.log.ErrorContext(ctx, "default org lookup failed", "tenant_id", req.TenantId, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound))
	}

	return &pb.GetDefaultOrgForTenantResponse{OrgId: orgUUID.String()}, nil
}

// GetUserMembership resolves a user's role within an organization for RBAC.
func (s *IdentityInternalService) GetUserMembership(ctx context.Context, req *pb.GetUserMembershipRequest) (*pb.GetUserMembershipResponse, error) {
	if req.OrgId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired))
	}
	if req.UserId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserIDRequired), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserIDRequired))
	}

	if _, err := uuid.Parse(req.OrgId); err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat))
	}
	if _, err := uuid.Parse(req.UserId); err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
	}

	if s.membershipSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	role, isActive, err := s.membershipSvc.GetMembership(ctx, req.OrgId, req.UserId)
	if err != nil {
		s.log.ErrorContext(ctx, "membership lookup failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMembershipNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMembershipNotFound))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetMemberFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetMemberFailed))
	}

	return &pb.GetUserMembershipResponse{
		Role:     role,
		IsActive: isActive,
	}, nil
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
		s.log.ErrorContext(ctx, "server admin check failed", "user_id", req.UserId, "error", err)
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserNotFound))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	return &pb.CheckServerAdminResponse{IsAdmin: user.IsAdmin}, nil
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
		s.log.Error("failed to record platform event", "error", err, "eventType", req.EventType)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	return &pb.RecordPlatformEventResponse{}, nil
}
