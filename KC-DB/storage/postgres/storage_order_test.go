package postgres

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// storageOrderLateBy is how far the late row's own time lies behind the row stored before it.
const storageOrderLateBy = time.Hour

func uplinkIDs(uplinks []*mioty.ULDataMessage) []string {
	ids := make([]string, len(uplinks))
	for i, uplink := range uplinks {
		ids[i] = uplink.ID
	}
	return ids
}

// An uplink stored after another but received earlier lists behind it by
// reception time and ahead of it in storage order, and a storage bound keeps
// exactly the uplinks stored at or after it.
func TestListStoredULData_ListsInTheOrderTheDatabaseStored(t *testing.T) {
	f := newUplinkStoreFixture(t)
	repo := NewMessageRepository(f.db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, readerPacketCnt, []byte{readerReusedPayload}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(ctx, first)
	require.NoError(t, err)
	late := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, readerPacketCnt+1, []byte{readerReusedPayload}, models.DeliveryChannelSCACI)
	withReceptionTime(late, first.Message.RxTime-storageOrderLateBy.Nanoseconds())
	_, err = f.store.Persist(ctx, late)
	require.NoError(t, err)
	filter := mioty.ULDataMessageFilter{TenantID: uplinkStoreTenantA, Limit: uplinkStoreTestLimit}

	byReception, _, err := repo.ListULDataMessages(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, []string{first.Message.ID, late.Message.ID}, uplinkIDs(byReception), "the listing keeps reception order")
	stored, err := repo.ListStoredULData(ctx, filter)
	require.NoError(t, err)
	require.Equal(t, []string{late.Message.ID, first.Message.ID}, uplinkIDs(stored))
	assert.True(t, stored[0].StoredAt.After(stored[1].StoredAt))

	filter.StoredSince = &stored[0].StoredAt
	since, err := repo.ListStoredULData(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, []string{late.Message.ID}, uplinkIDs(since))
}

// An event stored after another but recorded as having occurred earlier lists
// ahead of it in storage order, and a storage bound keeps exactly the events
// stored at or after it.
func TestGetEvents_ListsInTheOrderTheDatabaseStored(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSystemEventsTestDB(t)
	const tenantID = int64(830)
	createSystemEventsTestTenant(t, db, tenantID, "StorageOrderTest")
	store := NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	now := time.Now()
	first := &models.SystemEvent{TenantID: strconv.FormatInt(tenantID, 10), EventType: models.EventTypeBSUpdated, Category: models.EventCategoryBaseStation,
		Severity: models.EventSeverityInfo, SourceType: models.SourceTypeServiceCenter, Title: t.Name(), CreatedAt: now}
	require.NoError(t, store.CreateEvent(ctx, first))
	late := *first
	late.ID, late.CreatedAt, late.UpdatedAt = "", now.Add(-storageOrderLateBy), time.Time{}
	require.NoError(t, store.CreateEvent(ctx, &late))
	filter := models.SystemEventFilter{TenantID: strconv.FormatInt(tenantID, 10), OrderBy: models.EventOrderByStored, Limit: testEventQueryLimit}

	stored, err := store.GetEvents(ctx, filter)
	require.NoError(t, err)
	require.Len(t, stored, 2)
	assert.Equal(t, []string{late.ID, first.ID}, []string{stored[0].ID, stored[1].ID})
	assert.True(t, stored[0].StoredAt.After(stored[1].StoredAt))

	filter.StoredSince = &stored[0].StoredAt
	since, err := store.GetEvents(ctx, filter)
	require.NoError(t, err)
	require.Len(t, since, 1)
	assert.Equal(t, late.ID, since[0].ID)
	count, err := store.CountEvents(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
