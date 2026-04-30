package grpc

import (
	"context"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
)

// KiloCenterServiceCompatIdentity implements the 29 identity RPCs of
// KiloCenterServiceServer by delegating to the local IdentityService.
// Core RPCs fall through to UnimplementedKiloCenterServiceServer.
type KiloCenterServiceCompatIdentity struct {
	pb.UnimplementedKiloCenterServiceServer
	identity *IdentityService
}

// NewKiloCenterServiceCompatIdentity creates the identity compat shim.
func NewKiloCenterServiceCompatIdentity(identity *IdentityService) *KiloCenterServiceCompatIdentity {
	return &KiloCenterServiceCompatIdentity{identity: identity}
}

//revive:disable:exported Delegation methods are self-documenting one-liners.

// Auth (8) + Registration (1)

func (s *KiloCenterServiceCompatIdentity) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return s.identity.Login(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) RegisterAccount(ctx context.Context, req *pb.RegisterAccountRequest) (*pb.LoginResponse, error) {
	return s.identity.RegisterAccount(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) RefreshTokens(ctx context.Context, req *pb.RefreshTokensRequest) (*pb.RefreshTokensResponse, error) {
	return s.identity.RefreshTokens(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return s.identity.GetProfile(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) GetAuthSettings(ctx context.Context, req *pb.GetAuthSettingsRequest) (*pb.GetAuthSettingsResponse, error) {
	return s.identity.GetAuthSettings(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return s.identity.Logout(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	return s.identity.ChangePassword(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) ExchangeOIDC(ctx context.Context, req *pb.ExchangeOIDCRequest) (*pb.LoginResponse, error) {
	return s.identity.ExchangeOIDC(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) ExchangeOAuth2(ctx context.Context, req *pb.ExchangeOAuth2Request) (*pb.LoginResponse, error) {
	return s.identity.ExchangeOAuth2(ctx, req)
}

// Users (6)

func (s *KiloCenterServiceCompatIdentity) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	return s.identity.CreateUser(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	return s.identity.GetUser(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	return s.identity.UpdateUser(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	return s.identity.DeleteUser(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	return s.identity.ListUsers(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) UpdateUserPassword(ctx context.Context, req *pb.UpdateUserPasswordRequest) (*pb.UpdateUserPasswordResponse, error) {
	return s.identity.UpdateUserPassword(ctx, req)
}

// Organizations (5)

func (s *KiloCenterServiceCompatIdentity) CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	return s.identity.CreateOrganization(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) GetOrganization(ctx context.Context, req *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error) {
	return s.identity.GetOrganization(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	return s.identity.UpdateOrganization(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	return s.identity.DeleteOrganization(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) ListOrganizations(ctx context.Context, req *pb.ListOrganizationsRequest) (*pb.ListOrganizationsResponse, error) {
	return s.identity.ListOrganizations(ctx, req)
}

// Memberships (6)

func (s *KiloCenterServiceCompatIdentity) AddOrganizationUser(ctx context.Context, req *pb.AddOrganizationUserRequest) (*pb.AddOrganizationUserResponse, error) {
	return s.identity.AddOrganizationUser(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) GetOrganizationUser(ctx context.Context, req *pb.GetOrganizationUserRequest) (*pb.GetOrganizationUserResponse, error) {
	return s.identity.GetOrganizationUser(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) UpdateOrganizationUser(ctx context.Context, req *pb.UpdateOrganizationUserRequest) (*pb.UpdateOrganizationUserResponse, error) {
	return s.identity.UpdateOrganizationUser(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) RemoveOrganizationUser(ctx context.Context, req *pb.RemoveOrganizationUserRequest) (*pb.RemoveOrganizationUserResponse, error) {
	return s.identity.RemoveOrganizationUser(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) ListOrganizationUsers(ctx context.Context, req *pb.ListOrganizationUsersRequest) (*pb.ListOrganizationUsersResponse, error) {
	return s.identity.ListOrganizationUsers(ctx, req)
}

func (s *KiloCenterServiceCompatIdentity) ListUserOrganizations(ctx context.Context, req *pb.ListUserOrganizationsRequest) (*pb.ListUserOrganizationsResponse, error) {
	return s.identity.ListUserOrganizations(ctx, req)
}

// API Keys (4)

//revive:disable-next-line:var-naming
func (s *KiloCenterServiceCompatIdentity) CreateApiKey(ctx context.Context, req *pb.CreateApiKeyRequest) (*pb.CreateApiKeyResponse, error) {
	return s.identity.CreateApiKey(ctx, req)
}

//revive:disable-next-line:var-naming
func (s *KiloCenterServiceCompatIdentity) GetApiKey(ctx context.Context, req *pb.GetApiKeyRequest) (*pb.GetApiKeyResponse, error) {
	return s.identity.GetApiKey(ctx, req)
}

//revive:disable-next-line:var-naming
func (s *KiloCenterServiceCompatIdentity) DeleteApiKey(ctx context.Context, req *pb.DeleteApiKeyRequest) (*pb.DeleteApiKeyResponse, error) {
	return s.identity.DeleteApiKey(ctx, req)
}

//revive:disable-next-line:var-naming
func (s *KiloCenterServiceCompatIdentity) ListApiKeys(ctx context.Context, req *pb.ListApiKeysRequest) (*pb.ListApiKeysResponse, error) {
	return s.identity.ListApiKeys(ctx, req)
}

//revive:enable:exported
