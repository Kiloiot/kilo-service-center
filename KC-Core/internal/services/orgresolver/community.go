// Package orgresolver implements the org.Resolver interface with in-memory caching.
package orgresolver

import (
	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/org"
)

// NewCommunityResolver creates a CE resolver that maps all requests to the given
// default tenant and organization. Delegates to org.NewCommunityResolver.
func NewCommunityResolver(defaultTenantID int64, defaultOrgUUID uuid.UUID) *org.CommunityResolver {
	return org.NewCommunityResolver(defaultTenantID, defaultOrgUUID)
}
