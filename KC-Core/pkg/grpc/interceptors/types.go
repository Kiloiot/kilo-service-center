// Package interceptors provides exported gRPC interceptors for authentication,
// organization resolution, and internal trust validation.
// These interceptors are shared between KC-Core and KC-Gateway.
package interceptors

import (
	"context"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/google/uuid"
)

// APIKeyRecord is a decoupled representation of an API key for interceptor use.
// Adapters convert concrete DB models to this struct to avoid coupling interceptors to KC-DB.
type APIKeyRecord struct {
	ID             uuid.UUID
	TenantID       int64
	OrganizationID uuid.UUID
	UserID         *uuid.UUID // nil for service accounts
	IsActive       bool
	IsExpired      bool
}

// APIKeyAuthenticator is the narrow contract for API key bearer auth lookup.
// Adapters implement this to bridge between interceptors and concrete DB types.
type APIKeyAuthenticator interface {
	LookupByHash(ctx context.Context, hash string) (*APIKeyRecord, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// OrganizationResolver maps tenant IDs to organization UUIDs.
// Used by the auth interceptor to enrich context after JWT tenant extraction.
type OrganizationResolver interface {
	ResolveOrganization(ctx context.Context, tenantID int64) (uuid.UUID, error)
}

// TenantResolver maps organization UUIDs to tenant IDs.
// Used by the auth interceptor when JWT claims contain org UUID instead of tenant ID.
type TenantResolver interface {
	LookupTenant(ctx context.Context, orgID uuid.UUID) (int64, error)
}

// AuthConfig holds authentication configuration for the interceptor.
type AuthConfig struct {
	Enabled          bool
	JWKSEndpoint     string
	Issuer           string
	Audience         string
	TenantClaim      string
	Algorithm        string
	UserClaim        string
	HMACSecret       string
	EventWriter      grpcconst.EventWriter // Persists security events (nil = no persistence)
	PlatformTenantID int64                 // Fallback tenant for pre-auth events
}
