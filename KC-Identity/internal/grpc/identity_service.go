package grpc

import (
	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
)

// IdentityService implements the IdentityService gRPC service for authentication,
// user management, organization management, memberships, and API keys.
type IdentityService struct {
	pb.UnimplementedIdentityServiceServer
	log              logger.Logger
	authSvc          grpcservices.AuthService
	externalAuthSvc  grpcservices.ExternalAuthService
	adminUserSvc     grpcservices.AdminUserService
	orgSvc           grpcservices.OrganizationService
	membershipSvc    grpcservices.MembershipService
	apiKeySvc        grpcservices.APIKeyService
	registrationSvc  grpcservices.RegistrationService
	eventWriter      grpcservices.EventWriter
	platformTenantID int64
}

// NewIdentityService creates a new IdentityService instance.
// All service dependencies are optional and set via With* builders.
func NewIdentityService() *IdentityService {
	return &IdentityService{
		log: logger.Get().WithField("component", "identity-service"),
	}
}

// WithAuthService sets the auth service for authentication operations.
func (s *IdentityService) WithAuthService(svc grpcservices.AuthService) *IdentityService {
	s.authSvc = svc
	return s
}

// WithExternalAuthService sets the external auth service for OIDC/OAuth2 flows.
func (s *IdentityService) WithExternalAuthService(svc grpcservices.ExternalAuthService) *IdentityService {
	s.externalAuthSvc = svc
	return s
}

// WithAdminUserService sets the admin user service for user management.
func (s *IdentityService) WithAdminUserService(svc grpcservices.AdminUserService) *IdentityService {
	s.adminUserSvc = svc
	return s
}

// WithOrganizationService sets the organization service for org management.
func (s *IdentityService) WithOrganizationService(svc grpcservices.OrganizationService) *IdentityService {
	s.orgSvc = svc
	return s
}

// WithMembershipService sets the membership service for org membership management.
func (s *IdentityService) WithMembershipService(svc grpcservices.MembershipService) *IdentityService {
	s.membershipSvc = svc
	return s
}

// WithAPIKeyService sets the API key service for key management.
func (s *IdentityService) WithAPIKeyService(svc grpcservices.APIKeyService) *IdentityService {
	s.apiKeySvc = svc
	return s
}

// WithRegistrationService sets the registration service for self-service signup.
func (s *IdentityService) WithRegistrationService(svc grpcservices.RegistrationService) *IdentityService {
	s.registrationSvc = svc
	return s
}

// WithEventWriter sets the event writer for system events.
func (s *IdentityService) WithEventWriter(w grpcservices.EventWriter) *IdentityService {
	s.eventWriter = w
	return s
}

// WithPlatformTenantID sets the fallback tenant for pre-auth security events.
func (s *IdentityService) WithPlatformTenantID(id int64) *IdentityService {
	s.platformTenantID = id
	return s
}
