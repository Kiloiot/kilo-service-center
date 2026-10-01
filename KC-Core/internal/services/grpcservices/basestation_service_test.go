package grpcservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
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
	svc := NewBaseStationService(BaseStationServiceDeps{Store: store, Protocol: &config.ProtocolConfig{BSCIHost: unreachableBSSCIHost, BSCIPort: config.DefaultProtocolBSCIPort}})

	_, err := svc.Create(testutil.TestContext(), &models.BaseStation{})

	require.NoError(t, err)
	assert.Nil(t, store.created.ServiceCenterURL)
}

// deletedStations removes the one base station it holds.
type deletedStations struct {
	BaseStationStore
	station *models.BaseStation
	steps   *[]string
}

func (s *deletedStations) DeleteByEUI(context.Context, int64, []byte) (*models.BaseStation, error) {
	if s.station == nil {
		return nil, storage.ErrNotFound
	}
	*s.steps = append(*s.steps, stepDeleted)
	return s.station, nil
}

// retiredStation records, in order, the closing of a station's session and
// the settling of the downlinks it held.
type retiredStation struct {
	steps    *[]string
	stations []uint64
}

func (r *retiredStation) CloseSessionByEUI(_ context.Context, eui uint64) bool {
	*r.steps = append(*r.steps, stepSessionClosed)
	r.stations = append(r.stations, eui)
	return true
}

func (r *retiredStation) ReleaseDeletedStation(_ context.Context, bsEUI uint64) {
	*r.steps = append(*r.steps, stepDownlinksSettled)
	r.stations = append(r.stations, bsEUI)
}

const (
	stepDeleted          = "deleted"
	stepSessionClosed    = "session closed"
	stepDownlinksSettled = "downlinks settled"
	deletedStationEUI    = "70B3D59CD000FF03"
)

// A deleted base station can never transmit what it held: its live session
// is closed before its downlinks are settled; a station that does not exist
// closes and settles nothing.
func TestDelete_ClosesTheSessionBeforeSettlingTheDownlinks(t *testing.T) {
	var steps []string
	eui := models.EUIFromString(deletedStationEUI)
	retired := &retiredStation{steps: &steps}
	svc := NewBaseStationService(BaseStationServiceDeps{
		Store: &deletedStations{station: &models.BaseStation{EUI: eui}, steps: &steps}, Sessions: retired, Downlinks: retired,
	})

	removed, err := svc.Delete(testutil.TestContext(), eui[:], 1)

	require.NoError(t, err)
	assert.Equal(t, eui, removed.EUI)
	assert.Equal(t, []string{stepDeleted, stepSessionClosed, stepDownlinksSettled}, steps)
	assert.Equal(t, []uint64{eui.ToUint64(), eui.ToUint64()}, retired.stations)

	steps = nil
	missing := NewBaseStationService(BaseStationServiceDeps{Store: &deletedStations{steps: &steps}, Sessions: retired, Downlinks: retired})
	_, err = missing.Delete(testutil.TestContext(), eui[:], 1)
	require.ErrorIs(t, err, storage.ErrNotFound)
	assert.Empty(t, steps)
}
