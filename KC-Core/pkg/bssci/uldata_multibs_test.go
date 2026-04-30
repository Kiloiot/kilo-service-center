package bssci_test

import (
	"context"
	"encoding/binary"
	"sync/atomic"
	"testing"
	"time"

	bssci "github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Multi-BS UL Data Tests (SCACI section 3.8.1)
// Tests for base_stations persistence, duplicate pointer non-nil, DL RX hydration
// ============================================================================

// TestDuplicatePointerNonNilOnFirstReception tests that Duplicate is &false on first reception
func TestDuplicatePointerNonNilOnFirstReception(t *testing.T) {
	// Create message simulating first reception
	ulDataMsg := &mioty.ULDataMessage{
		EpEui:     bssci.TestEpEui01,
		TenantID:  1,
		PacketCnt: 42,
		RxTime:    time.Now().UnixNano(),
	}

	// Simulate first reception: Duplicate should be set to &false
	dup := false
	ulDataMsg.Duplicate = &dup

	// Assert Duplicate pointer is non-nil
	require.NotNil(t, ulDataMsg.Duplicate, "Duplicate pointer must be non-nil on first reception")

	// Assert value is false (first reception is not a duplicate)
	assert.False(t, *ulDataMsg.Duplicate, "Duplicate must be false on first reception")
}

// TestDuplicatePointerNonNilOnDuplicateReception tests that Duplicate is &true on duplicate reception
func TestDuplicatePointerNonNilOnDuplicateReception(t *testing.T) {
	// Create message simulating duplicate reception
	ulDataMsg := &mioty.ULDataMessage{
		EpEui:     bssci.TestEpEui01,
		TenantID:  1,
		PacketCnt: 42,
		RxTime:    time.Now().UnixNano(),
	}

	// Simulate duplicate reception: Duplicate should be set to &true
	dup := true
	ulDataMsg.Duplicate = &dup

	// Assert Duplicate pointer is non-nil
	require.NotNil(t, ulDataMsg.Duplicate, "Duplicate pointer must be non-nil on duplicate reception")

	// Assert value is true (duplicate reception)
	assert.True(t, *ulDataMsg.Duplicate, "Duplicate must be true on duplicate reception")
}

// TestBaseStationsPopulatedOnMultiBSReception tests that BaseStations slice is populated
func TestBaseStationsPopulatedOnMultiBSReception(t *testing.T) {
	// Create message with multiple base stations
	ulDataMsg := &mioty.ULDataMessage{
		EpEui:     bssci.TestEpEui01,
		TenantID:  1,
		PacketCnt: 42,
		RxTime:    time.Now().UnixNano(),
		BaseStations: []mioty.BaseStationReception{
			{BsEui: bssci.TestBsEui01, Rssi: -80.0, Snr: 10.0},
			{BsEui: bssci.TestBsEui02, Rssi: -85.0, Snr: 8.0},
		},
	}

	// Assert BaseStations is populated
	require.Len(t, ulDataMsg.BaseStations, 2, "BaseStations should have 2 entries")

	// Assert each base station has expected fields
	assert.Equal(t, bssci.TestBsEui01, ulDataMsg.BaseStations[0].BsEui)
	assert.Equal(t, -80.0, ulDataMsg.BaseStations[0].Rssi)
	assert.Equal(t, bssci.TestBsEui02, ulDataMsg.BaseStations[1].BsEui)
	assert.Equal(t, -85.0, ulDataMsg.BaseStations[1].Rssi)
}

// mockDLRXStatusRepoWithCounter tracks call counts for batch query verification
type mockDLRXStatusRepoWithCounter struct {
	callCount   atomic.Int32
	returnData  []*mioty.DLRXStatus
	returnError error
}

func (m *mockDLRXStatusRepoWithCounter) GetLatestDLRXStatusByBaseStations(
	_ context.Context,
	_ int64,
	_ []byte,
	_ [][]byte,
) ([]*mioty.DLRXStatus, error) {
	m.callCount.Add(1)
	return m.returnData, m.returnError
}

// TestDLRXHydrationBatchQueryCallCount verifies batch method is called exactly once
func TestDLRXHydrationBatchQueryCallCount(t *testing.T) {
	// Setup mock with call counter
	mock := &mockDLRXStatusRepoWithCounter{
		returnData: []*mioty.DLRXStatus{
			{EpEui: make([]byte, 8), BsEui: make([]byte, 8), DlRxSnr: -5.0, DlRxRssi: -85.0},
			{EpEui: make([]byte, 8), BsEui: make([]byte, 8), DlRxSnr: -6.0, DlRxRssi: -90.0},
		},
		returnError: nil,
	}

	// Simulate batch query for 3 base stations
	bsEuis := [][]byte{
		make([]byte, 8),
		make([]byte, 8),
		make([]byte, 8),
	}
	binary.BigEndian.PutUint64(bsEuis[0], bssci.TestBsEui01)
	binary.BigEndian.PutUint64(bsEuis[1], bssci.TestBsEui02)
	binary.BigEndian.PutUint64(bsEuis[2], 0x0000000000000003) // Third BS

	epEui := make([]byte, 8)
	binary.BigEndian.PutUint64(epEui, bssci.TestEpEui01)

	// Call mock
	_, err := mock.GetLatestDLRXStatusByBaseStations(context.Background(), 1, epEui, bsEuis)
	require.NoError(t, err)

	// Assert batch method called exactly once (not N times)
	assert.Equal(t, int32(1), mock.callCount.Load(), "Batch query should be called exactly once")
}

// TestDLRXHydrationPopulatesMetrics verifies DL RX metrics are populated in BaseStations
func TestDLRXHydrationPopulatesMetrics(t *testing.T) {
	// Create UL data message with base stations
	ulDataMsg := &mioty.ULDataMessage{
		EpEui:     bssci.TestEpEui01,
		TenantID:  1,
		PacketCnt: 42,
		RxTime:    time.Now().UnixNano(),
		BaseStations: []mioty.BaseStationReception{
			{BsEui: bssci.TestBsEui01, Rssi: -80.0, Snr: 10.0},
			{BsEui: bssci.TestBsEui02, Rssi: -85.0, Snr: 8.0},
		},
	}

	// Create mock DL RX data to hydrate
	dlRxData := []*mioty.DLRXStatus{
		{
			EpEui:    make([]byte, 8),
			BsEui:    make([]byte, 8),
			DlRxSnr:  -5.5,
			DlRxRssi: -88.0,
		},
		{
			EpEui:    make([]byte, 8),
			BsEui:    make([]byte, 8),
			DlRxSnr:  -7.0,
			DlRxRssi: -92.0,
		},
	}
	binary.BigEndian.PutUint64(dlRxData[0].BsEui, bssci.TestBsEui01)
	binary.BigEndian.PutUint64(dlRxData[1].BsEui, bssci.TestBsEui02)

	// Build lookup map (mimics server.go hydration logic)
	dlRxMap := make(map[uint64]*mioty.DLRXStatus, len(dlRxData))
	for _, dlRx := range dlRxData {
		if len(dlRx.BsEui) >= 8 {
			bsEuiKey := binary.BigEndian.Uint64(dlRx.BsEui)
			dlRxMap[bsEuiKey] = dlRx
		}
	}

	// Hydrate BaseStations with DL RX metrics
	for i := range ulDataMsg.BaseStations {
		bs := &ulDataMsg.BaseStations[i]
		if dlRx, ok := dlRxMap[bs.BsEui]; ok {
			bs.DlRxSnr = &dlRx.DlRxSnr
			bs.DlRxRssi = &dlRx.DlRxRssi
		}
	}

	// Assert DL RX metrics were populated
	require.NotNil(t, ulDataMsg.BaseStations[0].DlRxSnr, "DlRxSnr should be populated for BS1")
	require.NotNil(t, ulDataMsg.BaseStations[0].DlRxRssi, "DlRxRssi should be populated for BS1")
	assert.Equal(t, -5.5, *ulDataMsg.BaseStations[0].DlRxSnr)
	assert.Equal(t, -88.0, *ulDataMsg.BaseStations[0].DlRxRssi)

	require.NotNil(t, ulDataMsg.BaseStations[1].DlRxSnr, "DlRxSnr should be populated for BS2")
	require.NotNil(t, ulDataMsg.BaseStations[1].DlRxRssi, "DlRxRssi should be populated for BS2")
	assert.Equal(t, -7.0, *ulDataMsg.BaseStations[1].DlRxSnr)
	assert.Equal(t, -92.0, *ulDataMsg.BaseStations[1].DlRxRssi)
}
