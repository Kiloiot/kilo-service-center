package blueprint

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var errModeSource = errors.New("tenant switch unreadable")

// fixedMode answers every tenant the same switch.
type fixedMode struct {
	on  bool
	err error
}

func (m fixedMode) IsSnapshotOnly(context.Context, int64) (bool, error) { return m.on, m.err }

// countingModeSource counts the reads that reach the store.
type countingModeSource struct {
	reads   int
	enabled bool
	err     error
}

func (s *countingModeSource) GetSnapshotOnlyDecoding(context.Context, int64) (bool, error) {
	s.reads++
	return s.enabled, s.err
}

// TC-CM-107: a tenant in snapshot-only mode decodes only from a snapshot.
func TestResolveBlueprintForEndpoint_SnapshotOnlyMode(t *testing.T) {
	snapshotSource := uuid.New()
	validSnap, err := json.Marshal(models.BlueprintSnapshot{
		SourceBlueprintID: snapshotSource.String(), Version: "1.0.0", TypeEUI: "70b3d59cd0000094", SpecJSON: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	catalogHit := &models.Blueprint{ID: uuid.New(), Version: "catalog-default"}
	tests := []struct {
		name          string
		mode          fixedMode
		snapshot      []byte
		wantID        *uuid.UUID
		wantConsulted bool
	}{
		{name: "snapshot decodes", mode: fixedMode{on: true}, snapshot: validSnap, wantID: &snapshotSource},
		{name: "no snapshot is not decoded", mode: fixedMode{on: true}},
		{name: "corrupt snapshot is not decoded", mode: fixedMode{on: true}, snapshot: []byte(`{not json`)},
		{name: "unreadable switch counts as on", mode: fixedMode{err: errModeSource}},
		{name: "mode off follows the catalog", mode: fixedMode{}, wantID: &catalogHit.ID, wantConsulted: true},
		{name: "mode off: corrupt snapshot falls through as before", mode: fixedMode{}, snapshot: []byte(`{not json`), wantID: &catalogHit.ID, wantConsulted: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var consulted bool
			repo := &stubBlueprintRepo{getDefaultForModelFn: func(context.Context, int64, uuid.UUID) (*models.Blueprint, error) {
				consulted = true
				return catalogHit, nil
			}}
			modelID := uuid.New()
			ep := &models.EndPoint{EUI: models.EUIFromString("1122334455667788"), DeviceModelID: &modelID, BlueprintSnapshot: tt.snapshot}
			svc := NewResolverService(logger.NewNop(), repo, tt.mode)

			bp, err := svc.ResolveBlueprintForEndpoint(testutil.TestContext(), 1, ep, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.wantConsulted, consulted, "catalog consulted?")
			if tt.wantID == nil {
				assert.Nil(t, bp, "not decoded")
				return
			}
			require.NotNil(t, bp)
			assert.Equal(t, *tt.wantID, bp.ID)
		})
	}
}

func TestCachedSnapshotOnlyModes_ReadsTheStoreAtMostOncePerMinute(t *testing.T) {
	source := &countingModeSource{enabled: true}
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	modes := NewCachedSnapshotOnlyModes(source, func() time.Time { return now })
	ctx := testutil.TestContext()

	on, err := modes.IsSnapshotOnly(ctx, 7)
	require.NoError(t, err)
	assert.True(t, on)
	now = now.Add(SnapshotOnlyModeCacheTTL - time.Second)
	_, err = modes.IsSnapshotOnly(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, 1, source.reads, "a value younger than the TTL is reused")

	_, err = modes.IsSnapshotOnly(ctx, 8)
	require.NoError(t, err)
	assert.Equal(t, 2, source.reads, "each tenant has its own entry")

	now = now.Add(2 * time.Second)
	source.enabled = false
	on, err = modes.IsSnapshotOnly(ctx, 7)
	require.NoError(t, err)
	assert.False(t, on, "an expired value is read again")
	assert.Equal(t, 3, source.reads)
}

func TestCachedSnapshotOnlyModes_DoesNotCacheAFailedRead(t *testing.T) {
	source := &countingModeSource{err: errModeSource}
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	modes := NewCachedSnapshotOnlyModes(source, func() time.Time { return now })
	ctx := testutil.TestContext()

	_, err := modes.IsSnapshotOnly(ctx, 7)
	require.ErrorIs(t, err, errModeSource)
	source.err = nil
	on, err := modes.IsSnapshotOnly(ctx, 7)
	require.NoError(t, err)
	assert.False(t, on)
	assert.Equal(t, 2, source.reads)
}
