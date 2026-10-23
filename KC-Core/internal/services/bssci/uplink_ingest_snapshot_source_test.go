package bssciservices

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci/blueprint"
	pkgblueprint "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ownerEndpointRepo answers the owner-scoped endpoint read the decode step makes.
type ownerEndpointRepo struct {
	*uplinkIngestEndpointRepo
}

func (r *ownerEndpointRepo) GetByEUI(_ context.Context, _ int64, _ []byte) (*models.EndPoint, error) {
	return r.endpoints[uplinkIngestTestEpEUI], nil
}

// snapshotOnlyOff answers every tenant with the switch off.
type snapshotOnlyOff struct{}

func (snapshotOnlyOff) IsSnapshotOnly(context.Context, int64) (bool, error) { return false, nil }

// TC-CM-109
func TestUplinkIngest_SnapshotWithoutSourceWritesNoBlueprintReference(t *testing.T) {
	snapshot, err := json.Marshal(models.BlueprintSnapshot{
		SpecJSON: json.RawMessage(`{"version":"2.1","typeEui":"70B3D59CD00000A1",
			"uplink":[{"id":0,"payload":[{"name":"t","size":8,"type":"uint"}]}]}`),
		Version:   "2.1",
		TypeEUI:   "70B3D59CD00000A1",
		AdoptedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	repo := &ownerEndpointRepo{uplinkIngestEndpointRepo: &uplinkIngestEndpointRepo{endpoints: map[uint64]*models.EndPoint{
		uplinkIngestTestEpEUI: {ID: 1, TenantID: uplinkIngestTestTenantID, OwnerTenantID: uplinkIngestTestTenantID,
			BlueprintSnapshot: snapshot},
	}}}
	owners, err := NewEndpointOwnerResolver(repo)
	require.NoError(t, err)
	store := &fakeUplinkStore{}
	resolver, err := blueprint.NewResolverService(logger.NewNop(), nil, snapshotOnlyOff{})
	require.NoError(t, err)
	svc, err := NewUplinkIngestService(store, testUplinkWindows, allChannels(), nil,
		&fakeOrgResolver{defaultOrgByTenant: map[int64]uuid.UUID{}}, nil, repo, owners,
		resolver, blueprint.NewDecoderService(logger.NewNop()),
		&recordingDownlinkAcks{}, logger.NewNop(), uplinkIngestTestTenantID, 0)
	require.NoError(t, err)
	payload := buildUplinkPayload()
	format := uint8(0)
	payload.Format = &format

	_, err = svc.Ingest(testutil.TestContextWithTenant(uplinkIngestTestTenantID), payload,
		bssci.UplinkIngestOptions{Source: bssci.UplinkSourceBSSCI})
	require.NoError(t, err)

	require.Len(t, store.requests, 1)
	msg := store.requests[0].Message
	assert.Equal(t, pkgblueprint.DecodeStatusSuccess, msg.DecodeStatus, "the snapshot decodes")
	assert.Nil(t, msg.BlueprintVersionID, "no catalog blueprint to reference: blueprint_version_id stays NULL")
}
