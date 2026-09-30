package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func mustBaseStationProto(t *testing.T, baseStation *models.BaseStation) *pb.BaseStation {
	t.Helper()
	converted, err := baseStationToProto(baseStation)
	require.NoError(t, err)
	return converted
}

func TestBaseStationToProto_StoredTagsRoundTrip(t *testing.T) {
	stored, err := encodeBaseStationTags(map[string]string{"env": "test"})
	require.NoError(t, err)

	converted := mustBaseStationProto(t, &models.BaseStation{TenantID: 1, Tags: stored})

	assert.Equal(t, map[string]string{"env": "test"}, converted.Tags)
}

func TestBaseStationToProto_CorruptStoredTagsFail(t *testing.T) {
	corrupt := `{"env":`

	_, err := baseStationToProto(&models.BaseStation{TenantID: 1, Tags: &corrupt})

	require.Error(t, err, "tags that do not decode must not be reported as an empty tag set")
}

func TestEncodeBaseStationTags_NoTagsStoresNone(t *testing.T) {
	stored, err := encodeBaseStationTags(nil)

	require.NoError(t, err)
	assert.Nil(t, stored)
}
