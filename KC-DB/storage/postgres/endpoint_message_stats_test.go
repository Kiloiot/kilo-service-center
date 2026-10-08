package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	endpointStatsEarlierPacketCnt = uint32(1)
	endpointStatsCurrentPacketCnt = uint32(2)
	endpointStatsEarlierAge       = 2 * time.Hour
)

func TestGetMessageStatsByEndpointSince_ReRegisteredEUICountsOnlyTheCurrentRegistration(t *testing.T) {
	f := newUplinkStoreFixture(t)
	ctx := testutil.TestContext()
	eui := mioty.EUI64Bytes(uplinkStoreEpEui)
	endpoints := NewRepositories(&DB{clock: f.store.clock, sqlxDB: f.db}).Endpoints

	earlier := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, endpointStatsEarlierPacketCnt, []byte{0x01})
	withReceptionTime(earlier, time.Now().Add(-endpointStatsEarlierAge).UnixNano())
	_, err := f.store.Persist(ctx, earlier)
	require.NoError(t, err)

	_, err = endpoints.DeleteByTenant(ctx, uplinkStoreTenantA, eui)
	require.NoError(t, err)
	seedUplinkEndpoint(t, f.db, uplinkStoreTenantA, uplinkStoreEpEui)
	var registeredAt time.Time
	require.NoError(t, f.db.GetContext(ctx, &registeredAt, `SELECT created_at FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2`, uplinkStoreTenantA, eui))

	_, err = f.store.Persist(ctx, uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, endpointStatsCurrentPacketCnt, []byte{0x02}))
	require.NoError(t, err)

	analytics := NewMessageAnalyticsRepository(f.db)
	current, err := analytics.GetMessageStatsByEndpointSince(ctx, uplinkStoreEpEui, uplinkStoreTenantA, registeredAt)
	require.NoError(t, err)
	assert.Equal(t, int64(1), current.TotalCount, "the previous registration's uplink is not counted")

	all, err := analytics.GetMessageStatsByEndpoint(ctx, uplinkStoreEpEui, uplinkStoreTenantA)
	require.NoError(t, err)
	assert.Equal(t, int64(2), all.TotalCount, "the unscoped query still counts every stored uplink of the EUI")
}
