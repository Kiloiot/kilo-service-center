// Package testutil provides utility functions for test context management and tenant isolation testing.
package testutil

import (
	"context"
	"time"

	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
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

// TestContextWithTimeout returns a test context that expires after d.
// The caller must call the cancel function.
func TestContextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(TestContext(), d)
}

// TestContextWithCancel returns a cancellable test context.
// The caller must call the cancel function.
func TestContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(TestContext())
}
