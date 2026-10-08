// Package testutil re-exports the shared test-context helpers so existing
// KC-Core import paths keep working. The canonical implementation now lives in
// github.com/Kiloiot/kilo-service-center/pkg/testutil.
package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"

	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"

	pkgtestutil "github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// TestContext returns a background context for use in tests.
func TestContext() context.Context { return pkgtestutil.TestContext() }

// TestContextWithTenant returns a context that carries the provided tenant ID.
func TestContextWithTenant(tenantID int64) context.Context {
	return pkgtestutil.TestContextWithTenant(tenantID)
}

// TestContextWithTenantAndOrg returns a context that carries the provided
// tenant and organization, the shape every organization-scoped handler sees.
func TestContextWithTenantAndOrg(tenantID int64, orgID uuid.UUID) context.Context {
	return pkgcontext.WithOrganizationID(pkgtestutil.TestContextWithTenant(tenantID), orgID)
}

// TestContextWithTimeout returns a test context that expires after d.
// The caller must call the cancel function.
func TestContextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return pkgtestutil.TestContextWithTimeout(d)
}

// TestContextWithCancel returns a cancellable test context.
// The caller must call the cancel function.
func TestContextWithCancel() (context.Context, context.CancelFunc) {
	return pkgtestutil.TestContextWithCancel()
}

// NewFakeClock starts a manually driven clock at the given instant.
func NewFakeClock(now time.Time) *pkgtestutil.FakeClock { return pkgtestutil.NewFakeClock(now) }
