package orgresolver

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// movingDirectory answers the organization's current tenant, which the test
// changes, or storage.ErrNotFound once the organization is deleted.
type movingDirectory struct {
	countingDirectory
	tenant  int64
	deleted bool
}

func (d *movingDirectory) GetTenantByOrgID(context.Context, uuid.UUID) (int64, error) {
	d.tenantQueries++
	if d.deleted {
		return 0, storage.ErrNotFound
	}
	return d.tenant, nil
}

func orgCertificate(orgID uuid.UUID) *x509.Certificate {
	return &x509.Certificate{Subject: pkix.Name{CommonName: orgCNPrefix + orgID.String()}}
}

// A peer certificate is resolved once per connection, so it is resolved from
// the directory every time: an organization moved to another tenant, or
// deleted, within the cache TTL no longer resolves to its old tenant, even
// when a request cached that tenant for it.
func TestResolveCertReadsTheDirectoryOnEveryConnection(t *testing.T) {
	orgID := uuid.New()
	dir := &movingDirectory{tenant: testTenant}
	resolver := newTestResolver(t, dir, testMaxEntries, &steppingClock{now: testStart})

	tenantID, err := resolver.LookupTenant(testutil.TestContext(), orgID)
	require.NoError(t, err)
	require.Equal(t, testTenant, tenantID)

	dir.tenant = testOtherTenant
	_, tenantID, err = resolver.ResolveCert(testutil.TestContext(), orgCertificate(orgID))
	require.NoError(t, err)
	assert.Equal(t, testOtherTenant, tenantID, "a moved organization resolves to its new tenant")

	dir.deleted = true
	_, _, err = resolver.ResolveCert(testutil.TestContext(), orgCertificate(orgID))
	assert.ErrorIs(t, err, storage.ErrNotFound, "a deleted organization resolves to no tenant")

	tenantID, err = resolver.LookupTenant(testutil.TestContext(), orgID)
	require.NoError(t, err)
	assert.Equal(t, testTenant, tenantID, "per-request resolution keeps its cache")
}
