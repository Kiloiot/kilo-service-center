package blueprint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// stubBlueprintRepo overrides only the methods a test exercises; any other call panics.
type stubBlueprintRepo struct {
	interfaces.BlueprintRepository
	getDefaultForModelFn func(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.Blueprint, error)
}

func (s *stubBlueprintRepo) GetDefaultForModel(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.Blueprint, error) {
	return s.getDefaultForModelFn(ctx, tenantID, modelID)
}

// TestResolveBlueprintForEndpoint_SnapshotFirst: valid snapshot decodes without catalog query; malformed snapshot falls through to catalog.
func TestResolveBlueprintForEndpoint_SnapshotFirst(t *testing.T) {
	snapshotSource := uuid.New()
	validSnap, err := json.Marshal(models.BlueprintSnapshot{
		SourceBlueprintID: snapshotSource.String(),
		Version:           "1.0.0",
		TypeEUI:           "70b3d59cd0000094",
		SpecJSON:          json.RawMessage(`{"format":1}`),
	})
	require.NoError(t, err)
	badSourceSnap, err := json.Marshal(models.BlueprintSnapshot{SourceBlueprintID: "not-a-uuid", SpecJSON: json.RawMessage(`{}`)})
	require.NoError(t, err)

	catalogHit := &models.Blueprint{ID: uuid.New(), Version: "catalog-default"}

	tests := []struct {
		name         string
		snapshot     []byte
		wantID       uuid.UUID
		wantFallThru bool
	}{
		{"valid snapshot decodes from snapshot", validSnap, snapshotSource, false},
		{"malformed json falls through to catalog", []byte(`{not json`), catalogHit.ID, true},
		{"invalid source id falls through to catalog", badSourceSnap, catalogHit.ID, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var consulted bool
			repo := &stubBlueprintRepo{
				getDefaultForModelFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
					consulted = true
					return catalogHit, nil
				},
			}
			modelID := uuid.New()
			ep := &models.EndPoint{
				EUI:               models.EUIFromString("1122334455667788"),
				DeviceModelID:     &modelID,
				BlueprintSnapshot: tt.snapshot,
			}
			svc := NewResolverService(logger.NewNop(), repo, nil, nil)

			bp, err := svc.ResolveBlueprintForEndpoint(testutil.TestContext(), 1, ep, nil)
			require.NoError(t, err)
			require.NotNil(t, bp)
			assert.Equal(t, tt.wantID, bp.ID)
			assert.Equal(t, tt.wantFallThru, consulted, "catalog fall-through consulted?")
		})
	}
}
