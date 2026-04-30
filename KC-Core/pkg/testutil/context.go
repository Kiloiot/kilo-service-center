// Package testutil provides utility functions for test context management and tenant isolation testing.
package testutil

import (
	"context"

	pkgcontext "github.com/kilocenter/pkg/context"
)

// TestContext returns a background context for use in tests.
func TestContext() context.Context {
	return context.Background()
}

// TestContextWithTenant returns a context that carries the provided tenant ID.
// Uses the shared pkgcontext package to ensure production code can extract the tenant.
func TestContextWithTenant(tenantID int64) context.Context {
	return pkgcontext.WithTenantID(TestContext(), tenantID)
}

// TenantIDFromContext extracts the tenant ID from context.
// Returns 0 when no tenant information is present.
// Uses the shared pkgcontext package for extraction.
func TenantIDFromContext(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	tenantID, err := pkgcontext.GetTenantID(ctx)
	if err != nil {
		return 0
	}
	return tenantID
}
