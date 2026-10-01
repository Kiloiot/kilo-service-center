package postgres

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	uplinkStoreTenantA   = int64(551)
	uplinkStoreTenantB   = int64(552)
	uplinkStoreEpEui     = uint64(0x70B3D56770111551)
	uplinkStoreEpEuiB    = uint64(0x70B3D56770111552)
	uplinkStoreBsEuiOne  = uint64(0x70B3D59CD0000551)
	uplinkStoreBsEuiTwo  = uint64(0x70B3D59CD0000552)
	uplinkStoreWindow    = 5 * time.Minute
	uplinkStoreTestLimit = 10
	// uplinkStoreLateArrival exceeds the window: the service center hears the
	// telegram again this long after the first reception arrived.
	uplinkStoreLateArrival = 10 * time.Minute
	// uplinkStoreRadioSkew is a reception-time spread between base stations
	// that hear the same telegram.
	uplinkStoreRadioSkew = 3 * time.Millisecond
	// uplinkStoreFrameOpID is the opId a base station gave its ulData frame.
	uplinkStoreFrameOpID = int64(4711)
)

type uplinkStoreFixture struct {
	db    *sqlx.DB
	clock *testutil.FakeClock
	store *UplinkStore
}

func newUplinkStoreFixture(t *testing.T) *uplinkStoreFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, db, uplinkStoreTenantA, "UplinkStoreTenantA")
	createTestTenant(t, db, uplinkStoreTenantB, "UplinkStoreTenantB")
	seedUplinkEndpoint(t, db, uplinkStoreTenantA, uplinkStoreEpEui)
	seedUplinkEndpoint(t, db, uplinkStoreTenantB, uplinkStoreEpEuiB)
	clk := testutil.NewFakeClock(time.Now())
	return &uplinkStoreFixture{db: db, clock: clk, store: NewUplinkStore(db, clk, logger.Get())}
}

func seedUplinkEndpoint(t *testing.T, db *sqlx.DB, tenantID int64, epEui uint64) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name) VALUES ($1, $1, $2, $3)`,
		tenantID, mioty.EUI64Bytes(epEui), "uplink-store-"+mioty.FormatEUI64(epEui))
	require.NoError(t, err)
}

func uplinkRequest(tenantID int64, epEui, bsEui uint64, packetCnt uint32, userData []byte, channels ...models.DeliveryChannel) models.UplinkPersistRequest {
	rxTime := time.Now().UnixNano()
	return models.UplinkPersistRequest{
		Message: &mioty.ULDataMessage{
			ID:          uuid.New().String(),
			CommandType: mioty.CmdULData,
			EpEui:       epEui,
			BsEui:       bsEui,
			TenantID:    tenantID,
			RxTime:      rxTime,
			PacketCnt:   packetCnt,
			SNR:         12.5,
			RSSI:        -80,
			UserData:    userData,
			BaseStations: []mioty.BaseStationReception{{
				BsEui: bsEui, RxTime: rxTime, Snr: 12.5, Rssi: -80,
			}},
		},
		Window:   uplinkStoreWindow,
		Channels: channels,
	}
}

// withReceptionTime sets the radio reception time the request carries.
func withReceptionTime(req models.UplinkPersistRequest, rxTime int64) {
	req.Message.RxTime = rxTime
	req.Message.BaseStations[0].RxTime = rxTime
}

func (f *uplinkStoreFixture) count(t *testing.T, query string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.QueryRow(query, args...).Scan(&n))
	return n
}

func TestUplinkStore_FirstReceptionCreatesMessageAndOutbox(t *testing.T) {
	f := newUplinkStoreFixture(t)
	req := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 1, []byte{0x01}, models.DeliveryChannelSCACI, models.DeliveryChannelMQTT)
	req.Message.OpId = uplinkStoreFrameOpID

	outcome, err := f.store.Persist(testutil.TestContext(), req)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkNew, outcome.Classification)
	assert.Equal(t, req.Message.ID, outcome.MessageID)
	assert.Len(t, outcome.BaseStations, 1)

	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE id = $1 AND owner_tenant_id = $2 AND duplicate = false AND op_id = $3`,
		req.Message.ID, uplinkStoreTenantA, uplinkStoreFrameOpID), "the message keeps the base station's ulData opId")
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM mioty_message_deduplication WHERE first_message_id = $1 AND duplicate_count = 0`, req.Message.ID))
	assert.Equal(t, int64(2), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1 AND status = 'pending' AND owner_tenant_id = $2`, req.Message.ID, uplinkStoreTenantA))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM endpoints WHERE ep_eui = $1 AND last_packet_cnt = 1 AND last_seen_at IS NOT NULL`, mioty.EUI64Bytes(uplinkStoreEpEui)))
}

func TestUplinkStore_DuplicateReceptionsMergeIdempotently(t *testing.T) {
	f := newUplinkStoreFixture(t)
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 7, []byte{0x07}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(testutil.TestContext(), first)
	require.NoError(t, err)

	sameStation := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 7, []byte{0x07}, models.DeliveryChannelSCACI)
	outcome, err := f.store.Persist(testutil.TestContext(), sameStation)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkDuplicate, outcome.Classification)
	assert.Equal(t, first.Message.ID, outcome.MessageID)
	assert.Len(t, outcome.BaseStations, 1, "the same base station replaces its own entry")
	assert.Equal(t, 1, outcome.DuplicateCount)

	otherStation := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, 7, []byte{0x07}, models.DeliveryChannelSCACI)
	outcome, err = f.store.Persist(testutil.TestContext(), otherStation)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkDuplicate, outcome.Classification)
	assert.Len(t, outcome.BaseStations, 2, "a second base station is appended")
	assert.Equal(t, 2, outcome.DuplicateCount)

	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE ep_eui = $1 AND packet_cnt = 7`, mioty.EUI64Bytes(uplinkStoreEpEui)))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE id = $1 AND duplicate = true AND jsonb_array_length(base_stations) = 2`, first.Message.ID))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1`, first.Message.ID), "duplicates do not queue delivery again")
	assert.Equal(t, int64(0), f.count(t, `SELECT count(*) FROM messages WHERE id IN ($1, $2)`, sameStation.Message.ID, otherStation.Message.ID))
}

func TestUplinkStore_OwnerTenantsAreIndependent(t *testing.T) {
	f := newUplinkStoreFixture(t)
	a := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 9, []byte{0x09})
	b := uplinkRequest(uplinkStoreTenantB, uplinkStoreEpEuiB, uplinkStoreBsEuiOne, 9, []byte{0x09})
	outA, err := f.store.Persist(testutil.TestContext(), a)
	require.NoError(t, err)
	outB, err := f.store.Persist(testutil.TestContext(), b)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkNew, outA.Classification)
	assert.Equal(t, models.UplinkNew, outB.Classification)

	repo := NewMessageRepository(f.db, clock.SystemClock{}, logger.Get())
	_, err = repo.GetULDataMessage(testutil.TestContext(), a.Message.ID, uplinkStoreTenantB)
	assert.ErrorIs(t, err, storage.ErrNotFound, "tenant B must not read tenant A's message")
	_, err = repo.GetULDataMessage(testutil.TestContext(), a.Message.ID, uplinkStoreTenantA)
	assert.NoError(t, err)
}

func TestUplinkStore_CollisionInsideWindowIsRejectedUnchanged(t *testing.T) {
	f := newUplinkStoreFixture(t)
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 11, []byte{0x01}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(testutil.TestContext(), first)
	require.NoError(t, err)

	collision := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, 11, []byte{0xFF}, models.DeliveryChannelSCACI)
	_, err = f.store.Persist(testutil.TestContext(), collision)
	require.ErrorIs(t, err, storage.ErrPacketCounterCollision)

	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE ep_eui = $1 AND packet_cnt = 11 AND duplicate = false`, mioty.EUI64Bytes(uplinkStoreEpEui)))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM mioty_message_deduplication WHERE first_message_id = $1 AND duplicate_count = 0`, first.Message.ID))
	assert.Equal(t, int64(0), f.count(t, `SELECT count(*) FROM messages WHERE id = $1`, collision.Message.ID))
}

func TestUplinkStore_SameCounterOutsideWindowStartsNewMessage(t *testing.T) {
	f := newUplinkStoreFixture(t)
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 13, []byte{0x13}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(testutil.TestContext(), first)
	require.NoError(t, err)

	later := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 13, []byte{0x13}, models.DeliveryChannelSCACI)
	withReceptionTime(later, first.Message.RxTime+uplinkStoreLateArrival.Nanoseconds())
	outcome, err := f.store.Persist(testutil.TestContext(), later)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkNew, outcome.Classification, "a telegram received outside the window is new data")
	assert.Equal(t, later.Message.ID, outcome.MessageID)
	assert.Equal(t, int64(2), f.count(t, `SELECT count(*) FROM messages WHERE ep_eui = $1 AND packet_cnt = 13`, mioty.EUI64Bytes(uplinkStoreEpEui)))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM mioty_message_deduplication WHERE first_message_id = $1 AND duplicate_count = 0 AND first_rx_time = $2`,
		later.Message.ID, later.Message.RxTime))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1`, later.Message.ID))
}

func TestUplinkStore_CounterReusedOutsideTheWindowIsFlagged(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
	}{
		{"same payload", []byte{0x15}},
		{"different payload", []byte{0xEA}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newUplinkStoreFixture(t)
			repo := NewMessageRepository(f.db, clock.SystemClock{}, logger.Get())
			ctx := testutil.TestContext()
			first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 21, []byte{0x15}, models.DeliveryChannelSCACI)
			_, err := f.store.Persist(ctx, first)
			require.NoError(t, err)
			merged := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, 21, []byte{0x15}, models.DeliveryChannelSCACI)
			withReceptionTime(merged, first.Message.RxTime)
			_, err = f.store.Persist(ctx, merged)
			require.NoError(t, err)

			stored, err := repo.GetULDataMessage(ctx, first.Message.ID, uplinkStoreTenantA)
			require.NoError(t, err)
			assert.False(t, stored.PacketCntReused, "a counter's first telegram, merged receptions included, is novel data")

			reused := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 21, tc.payload, models.DeliveryChannelSCACI)
			withReceptionTime(reused, first.Message.RxTime+uplinkStoreLateArrival.Nanoseconds())
			outcome, err := f.store.Persist(ctx, reused)
			require.NoError(t, err)
			assert.Equal(t, models.UplinkNew, outcome.Classification, "a reused counter is still stored and delivered")

			stored, err = repo.GetULDataMessage(ctx, reused.Message.ID, uplinkStoreTenantA)
			require.NoError(t, err)
			assert.True(t, stored.PacketCntReused, "the telegram reuses a counter sent outside the window")
		})
	}
}

func TestUplinkStore_ReceptionTimeDecidesTheWindowNotArrival(t *testing.T) {
	f := newUplinkStoreFixture(t)
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 14, []byte{0x14}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(testutil.TestContext(), first)
	require.NoError(t, err)

	f.clock.Advance(uplinkStoreLateArrival)
	reissued := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 14, []byte{0x14}, models.DeliveryChannelSCACI)
	withReceptionTime(reissued, first.Message.RxTime)
	outcome, err := f.store.Persist(testutil.TestContext(), reissued)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkDuplicate, outcome.Classification, "a ulData reissued after a resume carries the same reception time and merges")
	assert.Equal(t, first.Message.ID, outcome.MessageID)

	delayed := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, 14, []byte{0x14}, models.DeliveryChannelSCACI)
	withReceptionTime(delayed, first.Message.RxTime-uplinkStoreRadioSkew.Nanoseconds())
	outcome, err = f.store.Persist(testutil.TestContext(), delayed)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkDuplicate, outcome.Classification, "a delayed reception of the same telegram merges")
	assert.Len(t, outcome.BaseStations, 2)

	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE ep_eui = $1 AND packet_cnt = 14`, mioty.EUI64Bytes(uplinkStoreEpEui)))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1`, first.Message.ID),
		"the telegram is delivered as new data once")
}

func TestUplinkStore_EndpointCounterNeverMovesBackwards(t *testing.T) {
	f := newUplinkStoreFixture(t)
	_, err := f.store.Persist(testutil.TestContext(), uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 20, []byte{0x20}))
	require.NoError(t, err)
	_, err = f.store.Persist(testutil.TestContext(), uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 10, []byte{0x10}))
	require.NoError(t, err)

	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM endpoints WHERE ep_eui = $1 AND packet_cnt = 20 AND last_packet_cnt = 20`, mioty.EUI64Bytes(uplinkStoreEpEui)),
		"only an over-the-air attach resets the packet counter")
}

func TestUplinkStore_OwnerKeysEveryRowOfTheReception(t *testing.T) {
	f := newUplinkStoreFixture(t)
	const registeredTenant, ownerTenant = int64(553), int64(554)
	epEui := uint64(0x70B3D56770111553)
	createTestTenant(t, f.db, registeredTenant, "UplinkStoreRegisteredTenant")
	createTestTenant(t, f.db, ownerTenant, "UplinkStoreOwnerTenant")
	_, err := f.db.Exec(`INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name) VALUES ($1, $2, $3, $4)`,
		registeredTenant, ownerTenant, mioty.EUI64Bytes(epEui), "uplink-store-owner")
	require.NoError(t, err)

	first := uplinkRequest(ownerTenant, epEui, uplinkStoreBsEuiOne, 3, []byte{0x03}, models.DeliveryChannelSCACI)
	outcome, err := f.store.Persist(testutil.TestContext(), first)
	require.NoError(t, err, "the owner the ingest resolves must be able to persist")
	assert.Equal(t, models.UplinkNew, outcome.Classification)

	second := uplinkRequest(ownerTenant, epEui, uplinkStoreBsEuiTwo, 3, []byte{0x03}, models.DeliveryChannelSCACI)
	outcome, err = f.store.Persist(testutil.TestContext(), second)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkDuplicate, outcome.Classification, "the merge finds the first message under the same owner")

	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE ep_eui = $1 AND tenant_id = $2 AND owner_tenant_id = $2`, mioty.EUI64Bytes(epEui), ownerTenant))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM endpoints WHERE ep_eui = $1 AND last_packet_cnt = 3`, mioty.EUI64Bytes(epEui)))
}

func TestUplinkStore_ArchivedFirstMessageStillMerges(t *testing.T) {
	f := newUplinkStoreFixture(t)
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 16, []byte{0x16}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(testutil.TestContext(), first)
	require.NoError(t, err)
	_, err = f.db.Exec(`UPDATE messages SET archived = true, archived_at = NOW() WHERE id = $1`, first.Message.ID)
	require.NoError(t, err)

	second := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, 16, []byte{0x16}, models.DeliveryChannelSCACI)
	outcome, err := f.store.Persist(testutil.TestContext(), second)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkDuplicate, outcome.Classification, "archival only flags the row, the first message is still there to merge into")
	assert.Equal(t, first.Message.ID, outcome.MessageID)
}

func TestUplinkStore_PurgedFirstMessageTakesItsClassifierRow(t *testing.T) {
	f := newUplinkStoreFixture(t)
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 17, []byte{0x17}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(testutil.TestContext(), first)
	require.NoError(t, err)
	_, err = f.db.Exec(`DELETE FROM messages WHERE id = $1`, first.Message.ID)
	require.NoError(t, err)
	assert.Zero(t, f.count(t, `SELECT count(*) FROM mioty_message_deduplication WHERE ep_eui = $1 AND packet_cnt = 17`, mioty.EUI64Bytes(uplinkStoreEpEui)),
		"purging the first message cascades to its classifier row")

	again := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, 17, []byte{0x17}, models.DeliveryChannelSCACI)
	outcome, err := f.store.Persist(testutil.TestContext(), again)
	require.NoError(t, err)
	assert.Equal(t, models.UplinkNew, outcome.Classification)
	assert.Equal(t, again.Message.ID, outcome.MessageID)
}

// attachOverTheAir runs the endpoint side of an over-the-air attach in one
// transaction, the way the attach persistence drives it.
func (f *uplinkStoreFixture) attachOverTheAir(t *testing.T, tenantID int64, epEui uint64) {
	t.Helper()
	ctx := testutil.TestContext()
	var endpointID int64
	require.NoError(t, f.db.Get(&endpointID, `SELECT id FROM endpoints WHERE ep_eui = $1`, mioty.EUI64Bytes(epEui)))
	tx, err := (&DB{sqlxDB: f.db, log: logger.Get(), clock: clock.SystemClock{}}).BeginTx(ctx)
	require.NoError(t, err)
	attachCnt := int64(1)
	require.NoError(t, tx.EndPoints().EndpointAttachmentStateUpdate(ctx, tenantID, endpointID, models.EndpointAttachmentStateParams{AttachCnt: &attachCnt}))
	require.NoError(t, tx.EndPoints().RestartPacketCounter(ctx, tenantID, endpointID))
	require.NoError(t, tx.UplinkClassifier().ForgetEndpoint(ctx, tenantID, endpointID))
	require.NoError(t, tx.Commit())
}

func TestUplinkStore_OverTheAirAttachRestartsTheCounter(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
	}{
		{"same payload", []byte{0x01}},
		{"different payload", []byte{0xFE}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newUplinkStoreFixture(t)
			before := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 1, []byte{0x01}, models.DeliveryChannelSCACI)
			_, err := f.store.Persist(testutil.TestContext(), before)
			require.NoError(t, err)

			f.attachOverTheAir(t, uplinkStoreTenantA, uplinkStoreEpEui)

			after := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 1, tc.payload, models.DeliveryChannelSCACI)
			outcome, err := f.store.Persist(testutil.TestContext(), after)
			require.NoError(t, err, "the attach restarted the counter, so the reused value is not a collision")
			assert.Equal(t, models.UplinkNew, outcome.Classification, "the attach restarted the counter, so the reused value is new data")
			assert.Equal(t, after.Message.ID, outcome.MessageID)
			assert.Equal(t, int64(2), f.count(t, `SELECT count(*) FROM messages WHERE ep_eui = $1 AND packet_cnt = 1`, mioty.EUI64Bytes(uplinkStoreEpEui)))
			assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1`, after.Message.ID))
		})
	}
}

func TestUplinkStore_OverTheAirAttachLetsTheCounterStartOver(t *testing.T) {
	f := newUplinkStoreFixture(t)
	_, err := f.store.Persist(testutil.TestContext(), uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 90, []byte{0x5A}))
	require.NoError(t, err)
	_, err = f.store.Persist(testutil.TestContext(), uplinkRequest(uplinkStoreTenantB, uplinkStoreEpEuiB, uplinkStoreBsEuiOne, 90, []byte{0x5A}))
	require.NoError(t, err)

	f.attachOverTheAir(t, uplinkStoreTenantA, uplinkStoreEpEui)
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM endpoints WHERE ep_eui = $1 AND packet_cnt = 0 AND last_packet_cnt = 0`, mioty.EUI64Bytes(uplinkStoreEpEui)),
		"an attach request implies packet counter 0 (radio protocol §3.6.5.3)")

	_, err = f.store.Persist(testutil.TestContext(), uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 2, []byte{0x02}))
	require.NoError(t, err)
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM endpoints WHERE ep_eui = $1 AND last_packet_cnt = 2`, mioty.EUI64Bytes(uplinkStoreEpEui)))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM mioty_message_deduplication WHERE ep_eui = $1 AND owner_tenant_id = $2`,
		mioty.EUI64Bytes(uplinkStoreEpEuiB), uplinkStoreTenantB), "another owner's classifier rows are untouched")
}

func TestUplinkStore_RestartOfAnUnknownEndpointIsNotFound(t *testing.T) {
	f := newUplinkStoreFixture(t)
	err := NewEndPointRepository(f.db, nil, clock.SystemClock{}, logger.Get()).RestartPacketCounter(testutil.TestContext(), uplinkStoreTenantB, -1)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

// TestUplinkStore_StoresEveryUnsigned32BitCounter pins BSSCI §3.10.1 and
// radio §3.6.5.3: packetCnt is a 32-bit counter, so the largest value is stored
// and read back unchanged.
func TestUplinkStore_StoresEveryUnsigned32BitCounter(t *testing.T) {
	f := newUplinkStoreFixture(t)
	for _, counter := range []uint32{math.MaxInt32 + 1, math.MaxUint32} {
		req := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, counter, []byte{0x80})
		_, err := f.store.Persist(testutil.TestContext(), req)
		require.NoError(t, err, "packet counter %d", counter)
		assert.Equal(t, int64(counter), f.count(t, `SELECT packet_cnt FROM messages WHERE id = $1`, req.Message.ID))
	}
}

func TestUplinkStore_UnknownEndpointRollsBackEverything(t *testing.T) {
	f := newUplinkStoreFixture(t)
	req := uplinkRequest(uplinkStoreTenantA, 0x70B3D56770119999, uplinkStoreBsEuiOne, 15, []byte{0x15}, models.DeliveryChannelSCACI)

	_, err := f.store.Persist(testutil.TestContext(), req)
	require.ErrorIs(t, err, storage.ErrNotFound)

	assert.Equal(t, int64(0), f.count(t, `SELECT count(*) FROM messages WHERE id = $1`, req.Message.ID))
	assert.Equal(t, int64(0), f.count(t, `SELECT count(*) FROM mioty_message_deduplication WHERE first_message_id = $1`, req.Message.ID))
	assert.Equal(t, int64(0), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1`, req.Message.ID))
}

func TestUplinkStore_ConcurrentClaimsAdmitOneMessage(t *testing.T) {
	f := newUplinkStoreFixture(t)
	const receptions = 8
	var wg sync.WaitGroup
	outcomes := make([]models.UplinkPersistOutcome, receptions)
	errs := make([]error, receptions)
	for i := 0; i < receptions; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bsEui := uplinkStoreBsEuiOne + uint64(i)
			outcomes[i], errs[i] = f.store.Persist(testutil.TestContext(),
				uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, bsEui, 21, []byte{0x21}, models.DeliveryChannelSCACI))
		}(i)
	}
	wg.Wait()

	var newCount int
	for i := range outcomes {
		require.NoError(t, errs[i], "reception %d", i)
		if outcomes[i].Classification == models.UplinkNew {
			newCount++
		}
	}
	assert.Equal(t, 1, newCount, "exactly one reception may create the message")
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE ep_eui = $1 AND packet_cnt = 21`, mioty.EUI64Bytes(uplinkStoreEpEui)))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE owner_tenant_id = $1 AND channel = 'scaci'`, uplinkStoreTenantA))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE ep_eui = $1 AND packet_cnt = 21 AND jsonb_array_length(base_stations) = $2`, mioty.EUI64Bytes(uplinkStoreEpEui), receptions))
}

func TestUplinkStore_RejectsMalformedRequests(t *testing.T) {
	f := newUplinkStoreFixture(t)
	valid := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 1, nil)

	noMessage := valid
	noMessage.Message = nil
	_, err := f.store.Persist(testutil.TestContext(), noMessage)
	assert.ErrorIs(t, err, storage.ErrInvalidInput)

	zeroTenant := uplinkRequest(0, uplinkStoreEpEui, uplinkStoreBsEuiOne, 1, nil)
	_, err = f.store.Persist(testutil.TestContext(), zeroTenant)
	assert.ErrorIs(t, err, storage.ErrInvalidTenantID)

	noWindow := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 1, nil)
	noWindow.Window = 0
	_, err = f.store.Persist(testutil.TestContext(), noWindow)
	assert.ErrorIs(t, err, storage.ErrInvalidInput)

	noReceptionTime := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 1, nil)
	withReceptionTime(noReceptionTime, 0)
	_, err = f.store.Persist(testutil.TestContext(), noReceptionTime)
	assert.ErrorIs(t, err, storage.ErrInvalidInput)

	twoReceptions := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 1, nil)
	twoReceptions.Message.BaseStations = append(twoReceptions.Message.BaseStations, twoReceptions.Message.BaseStations[0])
	_, err = f.store.Persist(testutil.TestContext(), twoReceptions)
	assert.ErrorIs(t, err, storage.ErrInvalidInput)
}

func TestMessageDeliveryOutbox_ClaimRetryAndPark(t *testing.T) {
	f := newUplinkStoreFixture(t)
	outbox := NewMessageDeliveryOutboxRepository(f.db, f.clock)
	req := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 31, []byte{0x31}, models.DeliveryChannelSCACI, models.DeliveryChannelMQTT)
	_, err := f.store.Persist(testutil.TestContext(), req)
	require.NoError(t, err)
	messageID := uuid.MustParse(req.Message.ID)

	claimed, err := outbox.ClaimDue(testutil.TestContext(), uplinkStoreTestLimit, time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 2)
	for _, row := range claimed {
		assert.Equal(t, messageID, row.MessageID)
		assert.Equal(t, 1, row.Attempts)
		assert.True(t, row.NextAttemptAt.After(time.Now()), "the claim pushes the next attempt into the future")
	}

	again, err := outbox.ClaimDue(testutil.TestContext(), uplinkStoreTestLimit, time.Second)
	require.NoError(t, err)
	assert.Empty(t, again, "rows are not due again until their lease elapses")

	require.NoError(t, outbox.Reschedule(testutil.TestContext(), messageID, models.DeliveryChannelSCACI, time.Hour, "application center unreachable"))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1 AND channel = 'scaci' AND status = 'pending'
		AND last_error = 'application center unreachable' AND next_attempt_at > NOW() + INTERVAL '59 minutes'`, messageID),
		"a transient failure keeps the row pending for the policy's delay")

	require.NoError(t, outbox.MarkDelivered(testutil.TestContext(), messageID, models.DeliveryChannelSCACI))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1 AND channel = 'scaci' AND status = 'delivered' AND delivered_at IS NOT NULL`, messageID))

	require.NoError(t, outbox.Park(testutil.TestContext(), messageID, models.DeliveryChannelMQTT, "broker unreachable"))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1 AND channel = 'mqtt' AND status = 'parked' AND last_error = 'broker unreachable'`, messageID))

	_, err = f.db.Exec(`UPDATE message_delivery_outbox SET next_attempt_at = NOW() - INTERVAL '1 second'`)
	require.NoError(t, err)
	due, err := outbox.ClaimDue(testutil.TestContext(), uplinkStoreTestLimit, time.Second)
	require.NoError(t, err)
	assert.Empty(t, due, "delivered and parked rows are never claimed again")
	assert.ErrorIs(t, outbox.Reschedule(testutil.TestContext(), messageID, models.DeliveryChannelMQTT, time.Second, "late"), storage.ErrNotFound,
		"a parked row is not put back into the retry loop")

	assert.ErrorIs(t, outbox.MarkDelivered(testutil.TestContext(), uuid.New(), models.DeliveryChannelSCACI), storage.ErrNotFound)
}

func TestMessageDeliveryOutbox_ConcurrentClaimsDoNotOverlap(t *testing.T) {
	f := newUplinkStoreFixture(t)
	outbox := NewMessageDeliveryOutboxRepository(f.db, f.clock)
	const messages = 6
	for i := 0; i < messages; i++ {
		_, err := f.store.Persist(testutil.TestContext(),
			uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, uint32(100+i), []byte{byte(i)}, models.DeliveryChannelSCACI))
		require.NoError(t, err)
	}

	const workers = 4
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := map[uuid.UUID]int{}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := outbox.ClaimDue(testutil.TestContext(), 2, time.Minute)
			require.NoError(t, err)
			mu.Lock()
			for _, row := range rows {
				seen[row.MessageID]++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	assert.Len(t, seen, messages, "every row is claimed once across the workers")
	for id, n := range seen {
		assert.Equal(t, 1, n, "row %s claimed by more than one worker", id)
	}
}

// outboxClockOffset moves the injected clock far from the database clock, so
// a statement that reads NOW() instead of the bound time is caught.
const outboxClockOffset = -24 * time.Hour

// TestMessageDeliveryOutbox_ReadsTheInjectedClock pins every outbox time to
// the injected clock: the queued row, its lease, a reschedule and delivery.
func TestMessageDeliveryOutbox_ReadsTheInjectedClock(t *testing.T) {
	f := newUplinkStoreFixture(t)
	now := time.Now().Add(outboxClockOffset).Truncate(time.Microsecond).UTC()
	f.clock.Set(now)
	outbox := NewMessageDeliveryOutboxRepository(f.db, f.clock)
	req := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, 41, []byte{0x41}, models.DeliveryChannelSCACI)
	_, err := f.store.Persist(testutil.TestContext(), req)
	require.NoError(t, err)
	messageID := uuid.MustParse(req.Message.ID)

	claimed, err := outbox.ClaimDue(testutil.TestContext(), uplinkStoreTestLimit, time.Minute)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.True(t, claimed[0].NextAttemptAt.Equal(now.Add(time.Minute)), "lease from the clock: got %v, want %v", claimed[0].NextAttemptAt, now.Add(time.Minute))
	assert.True(t, claimed[0].CreatedAt.Equal(now), "queued at the clock: got %v, want %v", claimed[0].CreatedAt, now)

	require.NoError(t, outbox.Reschedule(testutil.TestContext(), messageID, models.DeliveryChannelSCACI, time.Hour, "application center unreachable"))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1 AND next_attempt_at = $2`, messageID, now.Add(time.Hour)),
		"reschedule from the clock")

	require.NoError(t, outbox.MarkDelivered(testutil.TestContext(), messageID, models.DeliveryChannelSCACI))
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM message_delivery_outbox WHERE message_id = $1 AND delivered_at = $2`, messageID, now),
		"delivered at the clock")
}
