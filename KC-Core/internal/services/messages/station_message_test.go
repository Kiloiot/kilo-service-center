package messages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	stationMessageID     = "uplink-received-twice"
	stationForeignTenant = testTenant + 1
	stationPrimaryEUI    = uint64(0x70B3D59CD00009E6)
	stationSecondaryEUI  = uint64(0x70B3D59CD00009E2)
	stationUnrelatedEUI  = uint64(0x70B3D59CD00009BB)
	stationSecondaryRssi = -97.5
	stationSecondarySnr  = 3.25
	stationPrimaryRssi   = -80.0
	stationPrimarySnr    = 12.5
)

// tenantMessages reads one tenant's uplinks by id, as the repository does.
type tenantMessages struct {
	MessageStore
	uplink *mioty.ULDataMessage
}

func (s tenantMessages) GetByID(_ context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error) {
	if tenantID != s.uplink.TenantID || messageID != s.uplink.ID {
		return nil, storage.ErrNotFound
	}
	return s.uplink, nil
}

func receivedTwice() *Service {
	uplink := &mioty.ULDataMessage{
		ID: stationMessageID, TenantID: testTenant, BsEui: stationPrimaryEUI, RSSI: stationPrimaryRssi, SNR: stationPrimarySnr,
		BaseStations: []mioty.BaseStationReception{
			{BsEui: stationPrimaryEUI, Rssi: stationPrimaryRssi, Snr: stationPrimarySnr},
			{BsEui: stationSecondaryEUI, Rssi: stationSecondaryRssi, Snr: stationSecondarySnr},
		},
	}
	return New(tenantMessages{uplink: uplink}, nil, testPollInterval, testOverlap, streamwake.NewSignal(), testBatchSize, logger.NewNop())
}

// Every station that received the uplink, primary or secondary, retrieves it.
func TestGetBaseStationMessage_EveryReceiverRetrievesTheUplink(t *testing.T) {
	svc := receivedTwice()

	for _, station := range []uint64{stationPrimaryEUI, stationSecondaryEUI} {
		got, err := svc.GetBaseStationMessage(testutil.TestContext(), testTenant, mioty.EUI64Bytes(station), stationMessageID)
		require.NoError(t, err, mioty.FormatEUI64(station))
		assert.Equal(t, stationMessageID, got.ID)
		reception, ok := got.ReceptionAt(station)
		require.True(t, ok)
		assert.Equal(t, station, reception.BsEui)
	}
}

// A station that did not receive the uplink is refused, and another tenant
// does not find it.
func TestGetBaseStationMessage_RefusesAStationThatDidNotReceiveItAndAnotherTenant(t *testing.T) {
	svc := receivedTwice()

	_, err := svc.GetBaseStationMessage(testutil.TestContext(), testTenant, mioty.EUI64Bytes(stationUnrelatedEUI), stationMessageID)
	require.ErrorIs(t, err, ErrMessageNotOwnedByBaseStation)

	_, err = svc.GetBaseStationMessage(testutil.TestContext(), stationForeignTenant, mioty.EUI64Bytes(stationSecondaryEUI), stationMessageID)
	require.ErrorIs(t, err, storage.ErrNotFound)
	assert.ErrorIs(t, err, ErrGetMessage)
}
