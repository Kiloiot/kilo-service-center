package grpc

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

func orgModelToProto(o *models.Organization) *pb.Organization {
	if o == nil {
		return nil
	}
	pbOrg := &pb.Organization{
		Id:        o.OrgID.String(),
		TenantId:  o.TenantID,
		Name:      o.Name,
		State:     o.State,
		CreatedAt: timestamppb.New(o.CreatedAt),
		UpdatedAt: timestamppb.New(o.UpdatedAt),
	}
	if o.Description != nil {
		pbOrg.Description = *o.Description
	}
	if o.ExternalID != nil {
		pbOrg.ExternalId = *o.ExternalID
	}
	if o.Tags != nil {
		pbOrg.Tags = map[string]string(o.Tags)
	}
	return pbOrg
}

func apiKeyModelToProto(k *models.APIKey) *pb.ApiKey {
	if k == nil {
		return nil
	}
	pbKey := &pb.ApiKey{
		Id:        k.ID.String(),
		OrgId:     k.OrgID.String(),
		Name:      k.Name,
		KeyPrefix: k.KeyPrefix,
		KeyType:   k.KeyType,
		IsActive:  k.IsActive,
		CreatedAt: timestamppb.New(k.CreatedAt),
	}
	if k.UserID != nil {
		pbKey.UserId = k.UserID.String()
	}
	if k.ExpiresAt != nil {
		pbKey.ExpiresAt = timestamppb.New(*k.ExpiresAt)
	}
	if k.LastUsedAt != nil {
		pbKey.LastUsedAt = timestamppb.New(*k.LastUsedAt)
	}
	return pbKey
}

// orgMemberToProto converts a grpcservices.OrganizationMember to proto OrganizationUser.
// Maps all membership fields including admin flags.
func orgMemberToProto(m *grpcservices.OrganizationMember) *pb.OrganizationUser {
	if m == nil {
		return nil
	}
	return &pb.OrganizationUser{
		OrgId:              m.OrgID.String(),
		UserId:             m.UserID.String(),
		Email:              m.UserEmail,
		Role:               m.Role,
		Status:             m.Status,
		IsOrgAdmin:         m.IsOrgAdmin,
		IsBaseStationAdmin: m.IsBaseStationAdmin,
		IsEndpointAdmin:    m.IsEndpointAdmin,
		CreatedAt:          timestamppb.New(m.JoinedAt),
		UpdatedAt:          timestamppb.New(m.UpdatedAt),
	}
}

// userToProto converts a User model to proto.
func userToProto(u *models.User) *pb.User {
	if u == nil {
		return nil
	}
	pbUser := &pb.User{
		Id:                   u.ID.String(),
		Email:                u.Email,
		IsAdmin:              u.IsAdmin,
		IsActive:             u.IsActive,
		IsTenantManager:      u.IsTenantManager,
		IsBaseStationManager: u.IsBaseStationManager,
		IsEndpointManager:    u.IsEndpointManager,
		CreatedAt:            timestamppb.New(u.CreatedAt),
		UpdatedAt:            timestamppb.New(u.UpdatedAt),
	}
	if u.Note != nil {
		pbUser.Note = *u.Note
	}
	if u.FirstName != nil {
		pbUser.FirstName = *u.FirstName
	}
	if u.LastName != nil {
		pbUser.LastName = *u.LastName
	}
	if u.CompanyName != nil {
		pbUser.CompanyName = *u.CompanyName
	}
	return pbUser
}

// rolesToProto maps resolved roles onto the wire message.
func rolesToProto(r authz.Roles) *pb.UserRoles {
	return &pb.UserRoles{
		Admin:              r.Admin,
		TenantManager:      r.TenantManager,
		BaseStationManager: r.BaseStationManager,
		EndpointManager:    r.EndpointManager,
	}
}
