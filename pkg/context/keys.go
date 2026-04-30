// Package context provides shared context key definitions and utilities
// for tenant, organization, and user identification across KiloCenter modules.
//
// This package eliminates duplicate context key definitions across KC-API and KC-Core,
// providing a single source of truth for request-scoped values.
package context

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

// contextKey is a private type for context keys to prevent collisions
type contextKey string

// Context Key Constants
// These constants define all context keys used to store request-scoped values.
const (
	// TenantIDKey stores the tenant ID as a string in request context
	TenantIDKey contextKey = "tenant_id"

	// TenantIDIntKey stores the tenant ID as int64 in request context
	TenantIDIntKey contextKey = "tenant_id_int"

	// OrganizationIDKey stores the organization UUID in request context
	OrganizationIDKey contextKey = "organization_id"

	// UserIDKey stores the user ID string in request context
	UserIDKey contextKey = "user_id"
)

// Errors
var (
	// ErrNoTenantInContext is returned when tenant ID is not found in context
	ErrNoTenantInContext = errors.New("no tenant ID in context")

	// ErrNoOrganizationInContext is returned when organization ID is not found in context
	ErrNoOrganizationInContext = errors.New("no organization ID in context")

	// ErrNoUserInContext is returned when user ID is not found in context
	ErrNoUserInContext = errors.New("no user ID in context")
)

// GetTenantID extracts tenant ID as int64 from context.
// It first checks for the int64 value, then falls back to parsing the string value.
//
// Returns ErrNoTenantInContext if neither value is present.
func GetTenantID(ctx context.Context) (int64, error) {
	// Try int64 first (most common case)
	if tenantID, ok := ctx.Value(TenantIDIntKey).(int64); ok {
		return tenantID, nil
	}

	// Fall back to parsing string value
	if tenantStr, ok := ctx.Value(TenantIDKey).(string); ok && tenantStr != "" {
		tenantID, err := strconv.ParseInt(tenantStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid tenant ID format: %w", err)
		}
		return tenantID, nil
	}

	return 0, ErrNoTenantInContext
}

// GetOrganizationID extracts organization UUID from context.
// Returns ErrNoOrganizationInContext if no organization context is present.
//
// In community/self-hosted mode without organization context, this will return an error.
// Callers should handle this gracefully and fall back to tenant-only isolation.
func GetOrganizationID(ctx context.Context) (uuid.UUID, error) {
	orgID, ok := ctx.Value(OrganizationIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrNoOrganizationInContext
	}
	return orgID, nil
}

// GetUserID extracts user ID from context.
// Returns ErrNoUserInContext if no user context is present.
func GetUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		return "", ErrNoUserInContext
	}
	return userID, nil
}

// WithTenantID adds tenant ID (as int64) to context.
// This also sets the string representation for compatibility.
func WithTenantID(ctx context.Context, tenantID int64) context.Context {
	ctx = context.WithValue(ctx, TenantIDIntKey, tenantID)
	ctx = context.WithValue(ctx, TenantIDKey, strconv.FormatInt(tenantID, 10))
	return ctx
}

// WithOrganizationID adds organization UUID to context.
func WithOrganizationID(ctx context.Context, organizationID uuid.UUID) context.Context {
	return context.WithValue(ctx, OrganizationIDKey, organizationID)
}

// WithUserID adds user ID to context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}
