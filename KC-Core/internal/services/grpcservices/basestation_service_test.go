package grpcservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const unreachableBSSCIHost = "0.0.0.0"

type createdStations struct {
	BaseStationStore
	created *models.BaseStation
}

func (s *createdStations) Create(_ context.Context, bs *models.BaseStation) error {
	s.created = bs
	return nil
}

// A base station registered while the service center knows no URL a station
// can reach stores no URL (NULL), not an empty one.
func TestCreate_StoresNoServiceCenterURLWhenNoneIsKnown(t *testing.T) {
	store := &createdStations{}
	svc := NewBaseStationService(store, &config.ProtocolConfig{BSCIHost: unreachableBSSCIHost, BSCIPort: config.DefaultProtocolBSCIPort})

	_, err := svc.Create(testutil.TestContext(), &models.BaseStation{})

	require.NoError(t, err)
	assert.Nil(t, store.created.ServiceCenterURL)
}
