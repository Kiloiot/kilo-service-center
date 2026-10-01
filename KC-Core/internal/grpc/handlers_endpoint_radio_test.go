package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const radioServingStation = uint64(0x70B3D59CD00009E6)

// servingStationAt answers one serving station per tenant, or none.
type servingStationAt struct {
	tenant  int64
	station uint64
	err     error
}

func (s servingStationAt) ServingStation(_ context.Context, tenantID int64, _ uint64) (uint64, bool, error) {
	if s.err != nil {
		return 0, false, s.err
	}
	if tenantID != s.tenant || s.station == 0 {
		return 0, false, nil
	}
	return s.station, true, nil
}

func radioEndpointService(rssi, snr, eqSnr *float64) *mockEndpointSvcIsolation {
	return &mockEndpointSvcIsolation{
		getByEUIFunc: func(_ context.Context, _ []byte, tenantID int64) (*models.EndPoint, error) {
			return &models.EndPoint{TenantID: tenantID, LastRSSI: rssi, LastSNR: snr, LastEqSNR: eqSnr}, nil
		},
	}
}

func getRadioEndpoint(t *testing.T, endpoints *mockEndpointSvcIsolation, serving servingStationAt) *pb.EndPoint {
	t.Helper()
	svc := testCoreService(coreFields{endpointSvc: endpoints, log: logger.NewNop()})
	svc.servingStations = serving
	resp, err := svc.GetEndPoint(contextForTenant(testOwnerTenant), &pb.GetEndPointRequest{EpEui: testActiveEpEui})
	require.NoError(t, err)
	return resp
}

// An endpoint's detail carries its latest reception metrics and the station
// a downlink queued now would go to.
func TestGetEndPoint_ReportsTheLatestReceptionAndTheServingStation(t *testing.T) {
	rssi, snr, eqSnr := -97.5, 12.25, 14.0
	resp := getRadioEndpoint(t, radioEndpointService(&rssi, &snr, &eqSnr),
		servingStationAt{tenant: testOwnerTenant, station: radioServingStation})

	require.NotNil(t, resp.LastRssi)
	assert.InDelta(t, rssi, resp.LastRssi.GetValue(), 0)
	require.NotNil(t, resp.LastSnr)
	assert.InDelta(t, snr, resp.LastSnr.GetValue(), 0)
	require.NotNil(t, resp.LastEqSnr)
	assert.InDelta(t, eqSnr, resp.LastEqSnr.GetValue(), 0)
	assert.Equal(t, "70B3D59CD00009E6", resp.ServingBsEui)
}

// An endpoint never heard and served by no station carries neither.
func TestGetEndPoint_OfAnEndpointNeverHeardHasNoReceptionOrStation(t *testing.T) {
	resp := getRadioEndpoint(t, radioEndpointService(nil, nil, nil), servingStationAt{tenant: testOwnerTenant})

	assert.Nil(t, resp.LastRssi)
	assert.Nil(t, resp.LastSnr)
	assert.Nil(t, resp.LastEqSnr)
	assert.Empty(t, resp.ServingBsEui)
}

// A failed serving-station lookup leaves the station out; the endpoint is still read.
func TestGetEndPoint_ReadsTheEndpointWhenTheServingStationLookupFails(t *testing.T) {
	resp := getRadioEndpoint(t, radioEndpointService(nil, nil, nil),
		servingStationAt{tenant: testOwnerTenant, err: errors.New("location unavailable")})

	assert.Empty(t, resp.ServingBsEui)
}
