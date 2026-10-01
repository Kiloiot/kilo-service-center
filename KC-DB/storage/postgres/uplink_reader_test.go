package postgres

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	readerPacketCnt     = uint32(31)
	readerReusedPayload = byte(0x31)
	readerSearchTerm    = "31"
	readerAbsentStation = uint64(0x70b3d59cd00009bb)
	// readerTiedUplinks is how many uplinks share one reception time.
	readerTiedUplinks = 6
	// readerTiedIDFmt names tied uplinks so the insertion order is the ascending id order.
	readerTiedIDFmt = "00000000-0000-4000-8000-%012d"
	// readerTiePage pages one uplink at a time, so every page boundary falls inside the tie.
	readerTiePage = 1
)

// TestListULDataMessages_CarriesTheReusedCounterFlag: a listed uplink says
// whether it reused a packet counter, as the single-uplink read does.
func TestListULDataMessages_CarriesTheReusedCounterFlag(t *testing.T) {
	f := newUplinkStoreFixture(t)
	repo := NewMessageRepository(f.db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, readerPacketCnt, []byte{readerReusedPayload}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(ctx, first)
	require.NoError(t, err)
	reused := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, readerPacketCnt, []byte{readerReusedPayload}, models.DeliveryChannelSCACI)
	withReceptionTime(reused, first.Message.RxTime+uplinkStoreLateArrival.Nanoseconds())
	_, err = f.store.Persist(ctx, reused)
	require.NoError(t, err)

	listed, _, err := repo.ListULDataMessages(ctx, mioty.ULDataMessageFilter{TenantID: uplinkStoreTenantA, Limit: uplinkStoreTestLimit})
	require.NoError(t, err)
	byID := make(map[string]*mioty.ULDataMessage, len(listed))
	for _, msg := range listed {
		byID[msg.ID] = msg
	}
	require.Contains(t, byID, reused.Message.ID)
	require.Contains(t, byID, first.Message.ID)
	assert.True(t, byID[reused.Message.ID].PacketCntReused, "the listing reports the reused counter")
	assert.False(t, byID[first.Message.ID].PacketCntReused)

	got, err := repo.GetULDataMessage(ctx, reused.Message.ID, uplinkStoreTenantA)
	require.NoError(t, err)
	assert.Equal(t, got, byID[reused.Message.ID], "listing and single read return the same uplink")
}

// TestListULDataMessages_StationFilterMatchesEveryReceiver: a base station
// filter lists an uplink the station received as the primary receiver or as
// a merged reception (SCACI §3.8.1 baseStations); a search term matches the
// payload in hex.
func TestListULDataMessages_StationFilterMatchesEveryReceiver(t *testing.T) {
	f := newUplinkStoreFixture(t)
	repo := NewMessageRepository(f.db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, readerPacketCnt, []byte{readerReusedPayload}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(ctx, first)
	require.NoError(t, err)
	merged := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, readerPacketCnt, []byte{readerReusedPayload}, models.DeliveryChannelSCACI)
	withReceptionTime(merged, first.Message.RxTime)
	_, err = f.store.Persist(ctx, merged)
	require.NoError(t, err)

	for station, want := range map[uint64]int64{uplinkStoreBsEuiOne: 1, uplinkStoreBsEuiTwo: 1, readerAbsentStation: 0} {
		bs := station
		_, total, err := repo.ListULDataMessages(ctx, mioty.ULDataMessageFilter{TenantID: uplinkStoreTenantA, BsEui: &bs, Limit: uplinkStoreTestLimit})
		require.NoError(t, err)
		assert.Equal(t, want, total, mioty.FormatEUI64(station))
	}

	term := readerSearchTerm
	found, total, err := repo.ListULDataMessages(ctx, mioty.ULDataMessageFilter{TenantID: uplinkStoreTenantA, SearchTerm: &term, Limit: uplinkStoreTestLimit})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, found, 1)
	assert.Len(t, found[0].BaseStations, 2, "the merged receptions decode")
}

// TestListULDataMessages_PagesUplinksSharingAReceptionTimeByID: uplinks with
// one reception time list newest first and then by descending id, so a page
// boundary inside the tie neither repeats nor skips one.
func TestListULDataMessages_PagesUplinksSharingAReceptionTimeByID(t *testing.T) {
	f := newUplinkStoreFixture(t)
	repo := NewMessageRepository(f.db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	var rxTime int64
	want := make([]string, readerTiedUplinks)
	for i := range readerTiedUplinks {
		req := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, readerPacketCnt+uint32(i), []byte{byte(i)}, models.DeliveryChannelSCACI)
		req.Message.ID = fmt.Sprintf(readerTiedIDFmt, i)
		if i == 0 {
			rxTime = req.Message.RxTime
		}
		withReceptionTime(req, rxTime)
		_, err := f.store.Persist(ctx, req)
		require.NoError(t, err)
		want[readerTiedUplinks-1-i] = req.Message.ID
	}

	got := make([]string, 0, readerTiedUplinks)
	for offset := range readerTiedUplinks {
		page, _, err := repo.ListULDataMessages(ctx, mioty.ULDataMessageFilter{TenantID: uplinkStoreTenantA, Limit: readerTiePage, Offset: offset})
		require.NoError(t, err)
		require.Len(t, page, 1)
		got = append(got, page[0].ID)
	}
	assert.Equal(t, want, got)
}
