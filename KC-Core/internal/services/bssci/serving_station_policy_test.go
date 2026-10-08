package bssciservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Stations of the serving station policy tests.
const (
	policyStationNear     uint64 = 0x70B3D59CD0000A01
	policyStationFar      uint64 = 0x70B3D59CD0000A02
	policyStationAttached uint64 = 0x70B3D59CD0000A03
	policyEndpoint        uint64 = 0x70B3D59CD0000341
)

// fixedLocation answers every lookup with one location, or with err.
type fixedLocation struct {
	location storage.EndpointLocation
	err      error
}

func (f fixedLocation) GetEndpointLocation(context.Context, int64, uint64) (storage.EndpointLocation, error) {
	return f.location, f.err
}

func servingStationOf(t *testing.T, locator EndpointLocator) (uint64, bool, error) {
	t.Helper()
	policy, err := NewServingStationPolicy(locator)
	require.NoError(t, err)
	return policy.ServingStation(testutil.TestContext(), 3, policyEndpoint)
}

// TestServingStation_TheOrderOfThePolicy pins radio §3.6.1: the best receiver
// of the latest uplink serves, then the station the endpoint was attached
// through, and without either no station is known.
func TestServingStation_TheOrderOfThePolicy(t *testing.T) {
	heard := []mioty.BaseStationReception{{BsEui: policyStationFar, Snr: 2}, {BsEui: policyStationNear, Snr: 9}}
	cases := map[string]struct {
		location storage.EndpointLocation
		station  uint64
		known    bool
	}{
		"best reception wins over the attach":     {storage.EndpointLocation{LatestReceptions: heard, AttachedThrough: policyStationAttached}, policyStationNear, true},
		"first receiver of an unlisted reception": {storage.EndpointLocation{LatestReceptions: []mioty.BaseStationReception{{BsEui: policyStationFar}}}, policyStationFar, true},
		"attached through before it is heard":     {storage.EndpointLocation{AttachedThrough: policyStationAttached}, policyStationAttached, true},
		"neither heard nor attached":              {storage.EndpointLocation{}, 0, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			station, known, err := servingStationOf(t, fixedLocation{location: tc.location})

			require.NoError(t, err)
			assert.Equal(t, tc.known, known)
			assert.Equal(t, tc.station, station)
		})
	}
}

// TestServingStation_AMissingEndpointIsNotAnUnknownLocation: the tenant
// having no such endpoint is reported as such, never as an endpoint whose
// location is unknown.
func TestServingStation_AMissingEndpointIsNotAnUnknownLocation(t *testing.T) {
	_, known, err := servingStationOf(t, fixedLocation{err: storage.ErrNotFound})

	require.ErrorIs(t, err, storage.ErrNotFound)
	assert.False(t, known)
}

func TestNewServingStationPolicy_RequiresTheLocations(t *testing.T) {
	_, err := NewServingStationPolicy(nil)
	require.ErrorIs(t, err, ErrNilEndpointLocator)
}
