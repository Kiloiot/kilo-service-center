package postgres

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantRepository_SnapshotOnlyDecodingIsPerTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupTenantTestDB(t)
	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()
	a, err := repo.CreateTenant(ctx, "TestSnapshotOnly-A", nil)
	require.NoError(t, err)
	b, err := repo.CreateTenant(ctx, "TestSnapshotOnly-B", nil)
	require.NoError(t, err)
	assert.False(t, a.IsSnapshotOnlyDecoding, "a new tenant decodes as before")

	require.NoError(t, repo.SetSnapshotOnlyDecoding(ctx, a.ID, true))

	enabled, err := repo.GetSnapshotOnlyDecoding(ctx, a.ID)
	require.NoError(t, err)
	assert.True(t, enabled)
	other, err := repo.GetSnapshotOnlyDecoding(ctx, b.ID)
	require.NoError(t, err)
	assert.False(t, other, "another tenant is not switched")
	loaded, err := repo.GetTenantByID(ctx, a.ID)
	require.NoError(t, err)
	assert.True(t, loaded.IsSnapshotOnlyDecoding)

	require.NoError(t, repo.SetSnapshotOnlyDecoding(ctx, a.ID, false))
	enabled, err = repo.GetSnapshotOnlyDecoding(ctx, a.ID)
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestTenantRepository_SnapshotOnlyDecodingOfAMissingTenantIsNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	repo := NewTenantRepository(setupTenantTestDB(t))
	ctx := testutil.TestContext()

	_, err := repo.GetSnapshotOnlyDecoding(ctx, 987654321)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.ErrorIs(t, repo.SetSnapshotOnlyDecoding(ctx, 987654321, true), storage.ErrNotFound)
}
