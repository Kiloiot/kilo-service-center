package federation

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgfederation "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRaceIterations is the per-goroutine iteration count for the concurrency
// race lock; high enough for the race detector to interleave the paths.
const testRaceIterations = 200

// dispositionEndpointRepo answers Get from a fixed EUI set and counts lookups.
type dispositionEndpointRepo struct {
	interfaces.EndpointRepository
	mu      sync.Mutex
	known   map[uint64]struct{}
	lookups int
}

func (r *dispositionEndpointRepo) Get(_ context.Context, eui models.EUI) (*models.EndPoint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lookups++
	if _, ok := r.known[eui.ToUint64()]; ok {
		return &models.EndPoint{EUI: eui}, nil
	}
	return nil, storage.ErrNotFound
}

func (r *dispositionEndpointRepo) lookupCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lookups
}

// fixedEnumerator returns a fixed EUI list or an error.
type fixedEnumerator struct {
	euis []models.EUI
	err  error
}

func (e *fixedEnumerator) ListAllEUIs(context.Context) ([]models.EUI, error) {
	return e.euis, e.err
}

func euiOf(v uint64) models.EUI {
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], v)
	return eui
}

func TestLoadFromDB_PrewarmsIndex(t *testing.T) {
	repo := &dispositionEndpointRepo{known: map[uint64]struct{}{}}
	r := NewDispositionResolver(repo, &fixedEnumerator{euis: []models.EUI{euiOf(0x11), euiOf(0x22)}}, false)
	require.NoError(t, r.LoadFromDB(testutil.TestContext()))

	d, err := r.Resolve(testutil.TestContext(), 0x11)
	require.NoError(t, err)
	assert.Equal(t, bssci.DispositionLocal, d)
	assert.Zero(t, repo.lookupCount(), "a pre-warmed EUI must resolve without a repository lookup")
}

func TestLoadFromDB_FailureKeepsExistingIndex(t *testing.T) {
	repo := &dispositionEndpointRepo{known: map[uint64]struct{}{}}
	r := NewDispositionResolver(repo, &fixedEnumerator{err: errDBDown}, false)
	r.Add(testutil.TestContext(), euiOf(0x33))

	err := r.LoadFromDB(testutil.TestContext())
	assert.ErrorIs(t, err, pkgfederation.ErrEndpointIndexPrewarm)

	d, resolveErr := r.Resolve(testutil.TestContext(), 0x33)
	require.NoError(t, resolveErr)
	assert.Equal(t, bssci.DispositionLocal, d, "a failed pre-warm must not clear the existing index")
}

func TestLoadFromDB_NilEnumeratorIsLazy(t *testing.T) {
	repo := &dispositionEndpointRepo{known: map[uint64]struct{}{0x44: {}}}
	r := NewDispositionResolver(repo, nil, false)
	require.NoError(t, r.LoadFromDB(testutil.TestContext()))

	d, err := r.Resolve(testutil.TestContext(), 0x44)
	require.NoError(t, err)
	assert.Equal(t, bssci.DispositionLocal, d)
	assert.Equal(t, 1, repo.lookupCount(), "lazy warming confirms via the repository")

	// Second resolve is served from the warmed index.
	_, err = r.Resolve(testutil.TestContext(), 0x44)
	require.NoError(t, err)
	assert.Equal(t, 1, repo.lookupCount())
}

func TestResolve_UnknownEndpointRelayGate(t *testing.T) {
	repo := &dispositionEndpointRepo{known: map[uint64]struct{}{}}
	r := NewDispositionResolver(repo, nil, false)

	d, err := r.Resolve(testutil.TestContext(), 0x55)
	require.NoError(t, err)
	assert.Equal(t, bssci.DispositionDrop, d, "relay disabled drops unknown endpoints")

	r.SetRelayEnabled(true)
	d, err = r.Resolve(testutil.TestContext(), 0x55)
	require.NoError(t, err)
	assert.Equal(t, bssci.DispositionRelay, d)
}

func TestAddRemove_SyncTheIndex(t *testing.T) {
	repo := &dispositionEndpointRepo{known: map[uint64]struct{}{}}
	r := NewDispositionResolver(repo, nil, true)
	ctx := testutil.TestContext()

	r.Add(ctx, euiOf(0x66))
	d, err := r.Resolve(ctx, 0x66)
	require.NoError(t, err)
	assert.Equal(t, bssci.DispositionLocal, d)

	r.Remove(ctx, euiOf(0x66))
	d, err = r.Resolve(ctx, 0x66)
	require.NoError(t, err)
	assert.Equal(t, bssci.DispositionRelay, d, "a removed endpoint is unknown again")
}

// TestResolve_ConcurrentWithGateAndCRUD is the race lock: Resolve races
// SetRelayEnabled and Add/Remove without data races (run with -race).
func TestResolve_ConcurrentWithGateAndCRUD(_ *testing.T) {
	repo := &dispositionEndpointRepo{known: map[uint64]struct{}{}}
	r := NewDispositionResolver(repo, nil, false)
	ctx := testutil.TestContext()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(3)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < testRaceIterations; j++ {
				_, _ = r.Resolve(ctx, uint64(j%7))
			}
		}(i)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < testRaceIterations; j++ {
				r.SetRelayEnabled(j%2 == 0)
			}
		}(i)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < testRaceIterations; j++ {
				r.Add(ctx, euiOf(uint64(j%7)))
				r.Remove(ctx, euiOf(uint64(j%7)))
			}
		}(i)
	}
	wg.Wait()
}

// TestLoadFromDB_RestartPrewarmFromStore proves the pre-warm path over the
// real store: endpoints persisted before a restart resolve locally without a
// repository lookup afterwards, across multiple tenants.
func TestLoadFromDB_RestartPrewarmFromStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()

	seed := []struct {
		eui    uint64
		tenant int64
	}{
		{0x70B3D56770111505, 1},
		{0x70B3D56770111506, 2},
	}
	for _, e := range seed {
		_, err := db.Query(ctx, `
			INSERT INTO tenants (id, name, description, status, created_at, updated_at)
			VALUES ($1, 'Prewarm Tenant', 'prewarm test', 'active', NOW(), NOW())
			ON CONFLICT (id) DO NOTHING`, e.tenant)
		require.NoError(t, err)
		err = postgres.NewRepositories(db).Endpoints.Create(ctx, &models.EndPoint{
			Name:          "prewarm-test",
			EUI:           euiOf(e.eui),
			TenantID:      e.tenant,
			OwnerTenantID: e.tenant,
			NwkSnKey:      make([]byte, 16),
			EPClass:       "A",
		})
		require.NoError(t, err)
	}

	countingRepo := &dispositionEndpointRepo{known: map[uint64]struct{}{}}
	r := NewDispositionResolver(countingRepo, postgres.NewRepositories(db).Endpoints, false)
	require.NoError(t, r.LoadFromDB(ctx))

	for _, e := range seed {
		d, err := r.Resolve(ctx, e.eui)
		require.NoError(t, err)
		assert.Equal(t, bssci.DispositionLocal, d)
	}
	assert.Zero(t, countingRepo.lookupCount(), "restart pre-warm must serve every persisted EUI from the index")
}
