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
	// tenantIDKey stores the tenant ID as a string in request context
	tenantIDKey contextKey = "tenant_id"

	// tenantIDIntKey stores the tenant ID as int64 in request context
	tenantIDIntKey contextKey = "tenant_id_int"

	// organizationIDKey stores the organization UUID in request context
	organizationIDKey contextKey = "organization_id"

	// userIDKey stores the user ID string in request context
	userIDKey contextKey = "user_id"

	// serviceAccountIDKey stores the API key UUID of a caller that authenticated
	// with an organization service-account key, which has no user
	serviceAccountIDKey contextKey = "service_account_id"
)

// Errors
var (
	// errNoTenantInContext is returned when tenant ID is not found in context
	errNoTenantInContext = errors.New("no tenant ID in context")

	// errNoOrganizationInContext is returned when organization ID is not found in context
	errNoOrganizationInContext = errors.New("no organization ID in context")

	// errNoUserInContext is returned when user ID is not found in context
	errNoUserInContext = errors.New("no user ID in context")

	// errNoServiceAccountInContext is returned when no service-account key is in context
	errNoServiceAccountInContext = errors.New("no service account in context")

	// errInvalidTenantIDFormat is returned when the string tenant ID in
	// context cannot be parsed as an integer
	errInvalidTenantIDFormat = errors.New("invalid tenant ID format")
)

// GetTenantID extracts tenant ID as int64 from context.
// It first checks for the int64 value, then falls back to parsing the string value.
//
// Returns errNoTenantInContext if neither value is present.
func GetTenantID(ctx context.Context) (int64, error) {
	// Try int64 first (most common case)
	if tenantID, ok := ctx.Value(tenantIDIntKey).(int64); ok {
		return tenantID, nil
	}

	// Fall back to parsing string value
	if tenantStr, ok := ctx.Value(tenantIDKey).(string); ok && tenantStr != "" {
		tenantID, err := strconv.ParseInt(tenantStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %w", errInvalidTenantIDFormat, err)
		}
		return tenantID, nil
	}

	return 0, errNoTenantInContext
}

// GetOrganizationID extracts organization UUID from context.
// Returns errNoOrganizationInContext if no organization context is present.
//
// In community/self-hosted mode without organization context, this will return an error.
// Callers should handle this gracefully and fall back to tenant-only isolation.
func GetOrganizationID(ctx context.Context) (uuid.UUID, error) {
	orgID, ok := ctx.Value(organizationIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, errNoOrganizationInContext
	}
	return orgID, nil
}

// RequireOrganizationID is GetOrganizationID for callers that must not run
// without an organization: a missing or zero organization is an error.
func RequireOrganizationID(ctx context.Context) (uuid.UUID, error) {
	orgID, err := GetOrganizationID(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if orgID == uuid.Nil {
		return uuid.Nil, errNoOrganizationInContext
	}
	return orgID, nil
}

// GetUserID extracts user ID from context.
// Returns errNoUserInContext if no user context is present.
func GetUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok || userID == "" {
		return "", errNoUserInContext
	}
	return userID, nil
}

// WithTenantID adds tenant ID (as int64) to context.
// This also sets the string representation for compatibility.
func WithTenantID(ctx context.Context, tenantID int64) context.Context {
	ctx = context.WithValue(ctx, tenantIDIntKey, tenantID)
	ctx = context.WithValue(ctx, tenantIDKey, strconv.FormatInt(tenantID, 10))
	return ctx
}

// WithOrganizationID adds organization UUID to context.
func WithOrganizationID(ctx context.Context, organizationID uuid.UUID) context.Context {
	return context.WithValue(ctx, organizationIDKey, organizationID)
}

// WithUserID adds user ID to context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithServiceAccountID records that the caller is the service-account API key keyID.
func WithServiceAccountID(ctx context.Context, keyID uuid.UUID) context.Context {
	return context.WithValue(ctx, serviceAccountIDKey, keyID)
}

// GetServiceAccountID returns the service-account API key the caller authenticated with.
func GetServiceAccountID(ctx context.Context) (uuid.UUID, error) {
	keyID, ok := ctx.Value(serviceAccountIDKey).(uuid.UUID)
	if !ok || keyID == uuid.Nil {
		return uuid.Nil, errNoServiceAccountInContext
	}
	return keyID, nil
}
