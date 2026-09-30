package grpc

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/google/uuid"
)

// fixedRoles resolves the same roles in every organization.
type fixedRoles authz.Roles

func (r fixedRoles) Resolve(context.Context, uuid.UUID, uuid.UUID) (authz.Roles, error) {
	return authz.Roles(r), nil
}

// orgRoles resolves the caller's roles per organization; any other organization grants none.
type orgRoles map[uuid.UUID]authz.Roles

func (r orgRoles) Resolve(_ context.Context, _ uuid.UUID, orgID uuid.UUID) (authz.Roles, error) {
	return r[orgID], nil
}
