package mioty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	receptionPrimaryStation = uint64(0x70b3d59cd00009e6)
	receptionSecondStation  = uint64(0x70b3d59cd00009e2)
	receptionAbsentStation  = uint64(0x70b3d59cd00009bb)
)

func TestReceptionAt_FindsTheStationsReception(t *testing.T) {
	eqSnr := 4.5
	msg := &ULDataMessage{BaseStations: []BaseStationReception{
		{BsEui: receptionPrimaryStation, Rssi: -90, Snr: 7},
		{BsEui: receptionSecondStation, Rssi: -95, Snr: 3, EqSnr: &eqSnr},
	}}

	reception, ok := msg.ReceptionAt(receptionSecondStation)

	require.True(t, ok)
	assert.Equal(t, receptionSecondStation, reception.BsEui)
	assert.Equal(t, -95.0, reception.Rssi)
	assert.Equal(t, &eqSnr, reception.EqSnr)
}

func TestReceptionAt_StationThatDidNotReceiveIt(t *testing.T) {
	msg := &ULDataMessage{BaseStations: []BaseStationReception{{BsEui: receptionPrimaryStation}}}

	_, ok := msg.ReceptionAt(receptionAbsentStation)
	assert.False(t, ok)

	_, ok = (&ULDataMessage{}).ReceptionAt(receptionPrimaryStation)
	assert.False(t, ok, "an uplink stored without receptions has none")
}

func TestQueueStatusForResult_MapsEveryResult(t *testing.T) {
	for result, want := range map[string]DLQueueStatus{
		DLDataResultSent:    DLQueueStatusTransmitted,
		DLDataResultInvalid: DLQueueStatusFailed,
		DLDataResultExpired: DLQueueStatusExpired,
		DLDataResultRevoked: DLQueueStatusRevoked,
	} {
		got, ok := QueueStatusForResult(result)
		require.True(t, ok, result)
		assert.Equal(t, want, got, result)
		assert.True(t, got.Terminal(), "a result ends the downlink: %s", result)

		back, ok := ResultForQueueStatus(got)
		require.True(t, ok, result)
		assert.Equal(t, result, back, "the mapping reads both ways")
	}
}

func TestQueueStatusForResult_UnknownNames(t *testing.T) {
	_, ok := QueueStatusForResult(string(DLQueueStatusPending))
	assert.False(t, ok)
	_, ok = ResultForQueueStatus(DLQueueStatusAcked)
	assert.False(t, ok, "no result is stored as acked")
	_, ok = ResultForQueueStatus(DLQueueStatusPending)
	assert.False(t, ok)
}

func TestEUI64Bytes_RoundTrip(t *testing.T) {
	b := EUI64Bytes(receptionPrimaryStation)
	assert.Equal(t, []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x09, 0xe6}, b)
	assert.Equal(t, receptionPrimaryStation, EUI64FromBytes(b))
}

func TestEUI64FromBytes_WrongLengthIsZero(t *testing.T) {
	assert.Zero(t, EUI64FromBytes(nil))
	assert.Zero(t, EUI64FromBytes([]byte{0x01, 0x02}))
	assert.Zero(t, EUI64FromBytes(make([]byte, 9)))
}
