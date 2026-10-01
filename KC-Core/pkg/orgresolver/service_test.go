package orgresolver

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	testTenant      = int64(7)
	testOtherTenant = int64(8)
	testCacheTTL    = 5 * time.Minute
	testMaxEntries  = 10
	testOneEntry    = 1
	testRepeats     = 3
)

var (
	errTestDirectory = errors.New("directory unavailable")
	testStart        = time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
)

// steppingClock is a clock the test moves forward by hand.
type steppingClock struct{ now time.Time }

func (c *steppingClock) Now() time.Time { return c.now }

// countingDirectory answers every lookup and counts the queries it served.
type countingDirectory struct {
	tenantQueries  int
	defaultQueries int
	defaultOrg     uuid.UUID
	defaultErr     error
	tenantErr      error
}

func (d *countingDirectory) GetTenantByOrgID(context.Context, uuid.UUID) (int64, error) {
	d.tenantQueries++
	if d.tenantErr != nil {
		return 0, d.tenantErr
	}
	return testTenant, nil
}

func (d *countingDirectory) GetOrgByTenantID(context.Context, int64) (*models.Organization, error) {
	d.defaultQueries++
	if d.defaultErr != nil {
		return nil, d.defaultErr
	}
	return &models.Organization{OrgID: d.defaultOrg}, nil
}

func (d *countingDirectory) GetOrgByExternalID(context.Context, string) (*models.Organization, error) {
	return nil, storage.ErrNotFound
}

// newTestResolver builds a resolver over dir with the test cache TTL.
func newTestResolver(t *testing.T, dir OrgDirectory, maxEntries int, clk clock.Clock) org.Resolver {
	t.Helper()
	resolver, err := New(dir, logger.NewNop(), testCacheTTL, maxEntries, clk)
	require.NoError(t, err)
	return resolver
}

func TestNew_RefusesMissingDependencies(t *testing.T) {
	clk := &steppingClock{now: testStart}
	for name, build := range map[string]func() (org.Resolver, error){
		"directory": func() (org.Resolver, error) { return New(nil, logger.NewNop(), testCacheTTL, testMaxEntries, clk) },
		"logger":    func() (org.Resolver, error) { return New(&countingDirectory{}, nil, testCacheTTL, testMaxEntries, clk) },
		"clock": func() (org.Resolver, error) {
			return New(&countingDirectory{}, logger.NewNop(), testCacheTTL, testMaxEntries, nil)
		},
	} {
		_, err := build()
		assert.ErrorIs(t, err, ErrMissingDependency, name)
	}
}

func TestGetDefaultOrgForTenant_ResolvesOncePerTenant(t *testing.T) {
	dir := &countingDirectory{defaultOrg: uuid.New()}
	resolver := newTestResolver(t, dir, testMaxEntries, &steppingClock{now: testStart})

	for range testRepeats {
		org, err := resolver.GetDefaultOrgForTenant(testutil.TestContext(), testTenant)
		require.NoError(t, err)
		assert.Equal(t, dir.defaultOrg, org)
	}
	assert.Equal(t, 1, dir.defaultQueries, "the default organization of a tenant is resolved once, not per request")
}

func TestGetDefaultOrgForTenant_ExpiresAfterTheCacheTTL(t *testing.T) {
	clk := &steppingClock{now: testStart}
	dir := &countingDirectory{defaultOrg: uuid.New()}
	resolver := newTestResolver(t, dir, testMaxEntries, clk)

	_, err := resolver.GetDefaultOrgForTenant(testutil.TestContext(), testTenant)
	require.NoError(t, err)
	clk.now = clk.now.Add(testCacheTTL)
	_, err = resolver.GetDefaultOrgForTenant(testutil.TestContext(), testTenant)
	require.NoError(t, err)
	assert.Equal(t, 2, dir.defaultQueries, "an entry older than the TTL is resolved again")
}

func TestGetDefaultOrgForTenant_FailuresAreNotCached(t *testing.T) {
	dir := &countingDirectory{defaultOrg: uuid.New(), defaultErr: errTestDirectory}
	resolver := newTestResolver(t, dir, testMaxEntries, &steppingClock{now: testStart})

	_, err := resolver.GetDefaultOrgForTenant(testutil.TestContext(), testTenant)
	require.ErrorIs(t, err, ErrNoDefaultOrg)
	require.ErrorIs(t, err, errTestDirectory)

	dir.defaultErr = nil
	org, err := resolver.GetDefaultOrgForTenant(testutil.TestContext(), testTenant)
	require.NoError(t, err)
	assert.Equal(t, dir.defaultOrg, org)
	assert.Equal(t, 2, dir.defaultQueries)
}

func TestGetDefaultOrgForTenant_KeepsTenantsApart(t *testing.T) {
	dir := &countingDirectory{defaultOrg: uuid.New()}
	resolver := newTestResolver(t, dir, testMaxEntries, &steppingClock{now: testStart})

	_, err := resolver.GetDefaultOrgForTenant(testutil.TestContext(), testTenant)
	require.NoError(t, err)
	_, err = resolver.GetDefaultOrgForTenant(testutil.TestContext(), testOtherTenant)
	require.NoError(t, err)
	assert.Equal(t, 2, dir.defaultQueries, "each tenant resolves its own default organization")
}

func TestLookupTenant_FullCacheIsClearedBeforeTheNextEntry(t *testing.T) {
	dir := &countingDirectory{}
	resolver := newTestResolver(t, dir, testOneEntry, &steppingClock{now: testStart})
	first, second := uuid.New(), uuid.New()

	for _, orgID := range []uuid.UUID{first, first, second, first} {
		tenantID, err := resolver.LookupTenant(testutil.TestContext(), orgID)
		require.NoError(t, err)
		assert.Equal(t, testTenant, tenantID)
	}
	assert.Equal(t, 3, dir.tenantQueries, "storing the second organization dropped the first")
}

// Only an organization the directory does not know is "not found"; a
// directory that cannot answer must never read as a missing organization.
func TestLookupTenant_TellsMissingOrganizationFromLookupFailure(t *testing.T) {
	tests := map[string]struct {
		directoryErr error
		want         error
		notWant      error
		missing      bool
	}{
		"unknown organization": {directoryErr: fmt.Errorf("organization: %w", storage.ErrNotFound), want: ErrOrgNotFound, notWant: ErrOrgLookupFailed, missing: true},
		"directory down":       {directoryErr: errTestDirectory, want: ErrOrgLookupFailed, notWant: ErrOrgNotFound},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			resolver := newTestResolver(t, &countingDirectory{tenantErr: tt.directoryErr}, testMaxEntries, &steppingClock{now: testStart})

			_, err := resolver.LookupTenant(testutil.TestContext(), uuid.New())

			require.ErrorIs(t, err, tt.want)
			assert.NotErrorIs(t, err, tt.notWant)
			assert.Equal(t, tt.missing, errors.Is(err, storage.ErrNotFound))
		})
	}
}
