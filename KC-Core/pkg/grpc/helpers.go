// Package grpc provides gRPC service constants and utilities for the KiloCenter API.
// Shared helper functions for context extraction and field mask operations,
// importable by KC-Identity and other services outside KC-Core/internal/.
package grpc

import (
	"context"

	"github.com/google/uuid"
	pkgcontext "github.com/kilocenter/pkg/context"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// GetTenantFromContext extracts tenant ID from gRPC context.
func GetTenantFromContext(ctx context.Context) (int64, error) {
	tenantID, err := pkgcontext.GetTenantID(ctx)
	if err != nil {
		return 0, status.Error(GetGRPCCode(ErrTokenMissingTenantCtx),
			ResolveErrorMessage(ErrTokenMissingTenantCtx))
	}
	return tenantID, nil
}

// GetUserFromContext extracts user ID from gRPC context as UUID.
func GetUserFromContext(ctx context.Context) (uuid.UUID, error) {
	userIDStr, err := pkgcontext.GetUserID(ctx)
	if err != nil {
		return uuid.Nil, status.Error(GetGRPCCode(ErrTokenMissingUserCtx),
			ResolveErrorMessage(ErrTokenMissingUserCtx))
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, status.Error(GetGRPCCode(ErrTokenInvalidUserIDFormat),
			ResolveErrorMessage(ErrTokenInvalidUserIDFormat))
	}
	return userID, nil
}

// FieldInMask checks if a field path is in the update mask.
// If mask is nil or empty, returns true (update all provided fields).
func FieldInMask(mask *fieldmaskpb.FieldMask, field string) bool {
	if mask == nil || len(mask.GetPaths()) == 0 {
		return true
	}
	for _, path := range mask.GetPaths() {
		if path == field {
			return true
		}
	}
	return false
}
