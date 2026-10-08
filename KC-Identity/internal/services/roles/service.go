// Package roles resolves the roles a caller - a user or an organization
// service-account key - holds while acting in an organization, the single
// answer KC-Core's authorization and KC-Identity's own membership checks rely on.
package roles

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// UserReader loads a user; an unknown user is reported as storage.ErrRecordNotFound.
type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// MemberReader loads a user's membership in an organization; no membership is
// reported as storage.ErrRecordNotFound.
type MemberReader interface {
	GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error)
}

// APIKeyReader loads an API key by id; an unknown key is reported as storage.ErrRecordNotFound.
type APIKeyReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)
}

var (
	// ErrUserNotFound reports roles asked for a user that does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrAPIKeyNotFound reports roles asked for an API key that does not exist.
	ErrAPIKeyNotFound = errors.New("api key not found")
)

const (
	errCtxLoadUser           = "load user"
	errCtxLoadMembership     = "load membership"
	errCtxLoadServiceAccount = "load service account"
)

// Service resolves effective roles.
type Service struct {
	users   UserReader
	members MemberReader
	keys    APIKeyReader
}

// New creates the role resolver.
func New(users UserReader, members MemberReader, keys APIKeyReader) *Service {
	return &Service{users: users, members: members, keys: keys}
}

// ResolveServiceAccount returns the roles the service-account key holds in the organization.
func (s *Service) ResolveServiceAccount(ctx context.Context, keyID, orgID uuid.UUID) (authz.Roles, error) {
	key, err := s.keys.GetByID(ctx, keyID)
	if errors.Is(err, storage.ErrRecordNotFound) {
		return authz.Roles{}, ErrAPIKeyNotFound
	}
	if err != nil {
		return authz.Roles{}, fmt.Errorf("%s: %w", errCtxLoadServiceAccount, err)
	}
	return authz.ServiceAccount(key, orgID), nil
}

// Resolve returns the roles the user holds in the organization; uuid.Nil as
// the organization resolves the organization-independent roles only.
func (s *Service) Resolve(ctx context.Context, userID, orgID uuid.UUID) (authz.Roles, error) {
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, storage.ErrRecordNotFound) {
		return authz.Roles{}, ErrUserNotFound
	}
	if err != nil {
		return authz.Roles{}, fmt.Errorf("%s: %w", errCtxLoadUser, err)
	}
	if orgID == uuid.Nil {
		return authz.Effective(user, nil), nil
	}

	member, err := s.members.GetMember(ctx, orgID, userID)
	if errors.Is(err, storage.ErrRecordNotFound) {
		return authz.Effective(user, nil), nil
	}
	if err != nil {
		return authz.Roles{}, fmt.Errorf("%s: %w", errCtxLoadMembership, err)
	}
	return authz.Effective(user, &models.OrganizationMember{
		OrgID:              member.OrgID,
		UserID:             member.UserID,
		Role:               member.Role,
		Status:             member.Status,
		IsOrgAdmin:         member.IsOrgAdmin,
		IsBaseStationAdmin: member.IsBaseStationAdmin,
		IsEndpointAdmin:    member.IsEndpointAdmin,
	}), nil
}
