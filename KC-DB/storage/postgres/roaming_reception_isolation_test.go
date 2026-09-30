package postgres

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	roamingReceptionPacketCnt = uint32(1)
	roamingReceptionBucket    = int64(3600)
)

// TestRoamingReception_OwnerSeesTrafficHostSeesLoadOnly persists an uplink of
// tenant A's endpoint received through tenant B's base station: A lists the
// message with B's base station, B lists nothing carrying A's endpoint yet
// counts the reception on its base station.
func TestRoamingReception_OwnerSeesTrafficHostSeesLoadOnly(t *testing.T) {
	f := newUplinkStoreFixture(t)
	ctx := testutil.TestContext()
	var hostBs models.EUI
	binary.BigEndian.PutUint64(hostBs[:], uplinkStoreBsEuiTwo)
	insertTestBaseStation(t, f.db, hostBs, uplinkStoreTenantB, "host-base-station")

	req := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, roamingReceptionPacketCnt, []byte{0x01}, models.DeliveryChannelSCACI)
	outcome, err := f.store.Persist(ctx, req)
	require.NoError(t, err)
	require.Equal(t, models.UplinkNew, outcome.Classification)

	messages := NewMessageRepository(f.db, f.store.clock, logger.Get())
	owner, ownerTotal, err := messages.ListULDataMessages(ctx, mioty.ULDataMessageFilter{TenantID: uplinkStoreTenantA, Limit: uplinkStoreTestLimit})
	require.NoError(t, err)
	assert.Equal(t, int64(1), ownerTotal)
	require.Len(t, owner, 1)
	assert.Equal(t, uplinkStoreEpEui, owner[0].EpEui, "the owner sees its endpoint")
	assert.Equal(t, uplinkStoreBsEuiTwo, owner[0].BsEui, "the owner sees which base station received it")

	host, hostTotal, err := messages.ListULDataMessages(ctx, mioty.ULDataMessageFilter{TenantID: uplinkStoreTenantB, Limit: uplinkStoreTestLimit})
	require.NoError(t, err)
	assert.Zero(t, hostTotal)
	assert.Empty(t, host, "the host tenant lists no foreign traffic")
	hostByBs, _, err := messages.ListULDataMessages(ctx, mioty.ULDataMessageFilter{TenantID: uplinkStoreTenantB, BsEui: &req.Message.BsEui, Limit: uplinkStoreTestLimit})
	require.NoError(t, err)
	assert.Empty(t, hostByBs, "filtering by its own base station does not surface the foreign endpoint")

	assert.Zero(t, f.count(t, `SELECT count(*) FROM messages WHERE tenant_id = $1`, uplinkStoreTenantB))
	assert.Zero(t, f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE owner_tenant_id = $1`, uplinkStoreTenantB), "delivery rows belong to the endpoint owner")
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE owner_tenant_id = $1`, uplinkStoreTenantA))

	metrics := NewRepositories(&DB{clock: f.store.clock, sqlxDB: f.db}).BaseStationMetrics
	received := time.Unix(0, req.Message.RxTime)
	start := received.Add(-time.Hour)
	end := received.Add(time.Hour)
	hostCounts, err := metrics.CountBaseStationMessagesByBucket(ctx, uplinkStoreTenantB, hostBs[:], start, end, roamingReceptionBucket)
	require.NoError(t, err)
	assert.Equal(t, int64(1), hostCounts[received.Unix()/roamingReceptionBucket], "the host counts the reception as load on its base station")

	ownerCounts, err := metrics.CountBaseStationMessagesByBucket(ctx, uplinkStoreTenantA, hostBs[:], start, end, roamingReceptionBucket)
	require.NoError(t, err)
	assert.Empty(t, ownerCounts, "the endpoint owner gets no counters for a base station it does not own")
}
