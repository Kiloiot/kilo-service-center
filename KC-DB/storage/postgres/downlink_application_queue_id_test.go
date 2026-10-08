package postgres

import (
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// fixtureEndpointEUI is the endpoint every applicationDownlink addresses.
const fixtureEndpointEUI uint64 = 0x70B3D59CD0000321

func applicationQueueIDFixture(t *testing.T) (*MIOTYDownlinkRepository, *sqlx.DB, map[int64]uuid.UUID) {
	t.Helper()
	repos, db, orgs := downlinkRepositoriesFixture(t)
	return repos.Downlinks, db, orgs
}

// downlinkRepositoriesFixture builds the repositories over a fresh database
// with two tenants, 321 and 322, each with one organization.
func downlinkRepositoriesFixture(t *testing.T) (*Repositories, *sqlx.DB, map[int64]uuid.UUID) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	orgs := map[int64]uuid.UUID{}
	for _, tenantID := range []int64{321, 322} {
		createTestTenant(t, sqlxDB, tenantID, "ApplicationQueueIDTenant"+strconv.FormatInt(tenantID, 10))
		orgID := uuid.New()
		_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`,
			orgID, tenantID, "ac-queue-org-"+orgID.String())
		require.NoError(t, err)
		orgs[tenantID] = orgID
	}

	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	return NewRepositories(db), sqlxDB, orgs
}

func applicationDownlink(tenantID int64, orgID uuid.UUID, queID int64, acQueID *uint64) *storage.DownlinkMessage {
	return &storage.DownlinkMessage{
		EPEUI:          "70B3D59CD0000321",
		TenantID:       strconv.FormatInt(tenantID, 10),
		OrganizationID: &orgID,
		Payload:        []byte{0x01},
		Status:         mioty.DLQueueStatusPending,
		QueID:          queID,
		ACQueID:        acQueID,
	}
}

// TestEnqueueDownlink_ApplicationQueueIDIsTenantScoped pins that an
// Application Center queue id (SCACI §3.10.1) never crosses tenants: the same
// id works independently in two tenants, and an in-flight duplicate inside
// one organization is storage.ErrDuplicateKey - never the service center
// collision the allocator retries.
func TestEnqueueDownlink_ApplicationQueueIDIsTenantScoped(t *testing.T) {
	downlinks, _, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	acID := uint64(42)

	_, err := downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 810001, &acID), testDownlinkLifetime)
	require.NoError(t, err)
	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(322, orgs[322], 810002, &acID), testDownlinkLifetime)
	require.NoError(t, err, "another tenant's Application Center may use the same queue id")

	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 810003, &acID), testDownlinkLifetime)
	require.ErrorIs(t, err, storage.ErrDuplicateKey)
	require.NotErrorIs(t, err, storage.ErrDownlinkQueueIDTaken)

	otherID := uint64(43)
	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 810001, &otherID), testDownlinkLifetime)
	require.ErrorIs(t, err, storage.ErrDownlinkQueueIDTaken)
}

// TestEnqueueDownlink_ApplicationQueueIDIsScopedToTheOrganization pins that
// an Application Center names its downlinks within its organization (SCACI
// §1, §3.10.1): an id another organization of the tenant has in flight does
// not block it, while two in-flight downlinks of one organization never
// share an id.
func TestEnqueueDownlink_ApplicationQueueIDIsScopedToTheOrganization(t *testing.T) {
	downlinks, sqlxDB, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	other := uuid.New()
	_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, 321, $2)`, other, "ac-queue-org-"+other.String())
	require.NoError(t, err)
	acID := uint64(42)

	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, other, 860001, &acID), testDownlinkLifetime)
	require.NoError(t, err)
	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 860002, &acID), testDownlinkLifetime)
	require.NoError(t, err, "another organization's downlink in flight does not block the id")

	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 860003, &acID), testDownlinkLifetime)
	require.ErrorIs(t, err, storage.ErrDuplicateKey, "two in-flight downlinks of one organization never share the id")
}

// TestEnqueueDownlink_ApplicationQueueIDReusedAfterItsDownlinkEnded pins that
// the id names a downlink until its result (SCACI §3.12.1): once the downlink
// ended the Application Center may queue another under the same id, and each
// base station result reaches the downlink its service center queue id names.
func TestEnqueueDownlink_ApplicationQueueIDReusedAfterItsDownlinkEnded(t *testing.T) {
	downlinks, _, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	acID := uint64(77)
	txTime := int64(1_700_000_000_000_000_000)
	packetCnt := uint32(9)
	sent := func(queID uint64) *mioty.DLDataResult {
		return &mioty.DLDataResult{EpEui: fixtureEndpointEUI, QueId: queID, Result: mioty.DLDataResultSent, TxTime: &txTime, PacketCnt: &packetCnt}
	}
	reserve := func(queID uint64) {
		_, err := downlinks.ReservePendingDownlinkByQueueID(ctx, 321, orgs[321], queID, mioty.EUI64Bytes(fixtureEndpointEUI), releaseStation)
		require.NoError(t, err)
	}

	_, err := downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 870001, &acID), testDownlinkLifetime)
	require.NoError(t, err)
	reserve(870001)
	first, err := downlinks.UpdateDownlinkResult(ctx, 321, releaseStation, sent(870001))
	require.NoError(t, err)

	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 870002, &acID), testDownlinkLifetime)
	require.NoError(t, err, "the id is free again once its downlink ended")
	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 870003, &acID), testDownlinkLifetime)
	require.ErrorIs(t, err, storage.ErrDuplicateKey, "the id is taken again while the new downlink is in flight")

	_, err = downlinks.UpdateDownlinkResult(ctx, 321, releaseStation, sent(870001))
	require.ErrorIs(t, err, storage.ErrDownlinkFinished, "a repeated result of the ended downlink changes nothing")
	reserve(870002)
	second, err := downlinks.UpdateDownlinkResult(ctx, 321, releaseStation, sent(870002))
	require.NoError(t, err)
	assert.Equal(t, int64(870002), second.QueID, "the result reaches the downlink it names")
	assert.NotEqual(t, first.ID, second.ID)
	require.NotNil(t, second.ACQueID)
	assert.Equal(t, acID, *second.ACQueID, "and is reported to the Application Center under its id")
}

// TestEnqueueDownlink_ApplicationQueueIDZero pins SCACI §3.10.1 l.509: the
// Application Center may assign the queue id zero, which is stored and read
// back as its id, never as a downlink without one.
func TestEnqueueDownlink_ApplicationQueueIDZero(t *testing.T) {
	downlinks, _, orgs := applicationQueueIDFixture(t)
	zero := uint64(0)

	_, err := downlinks.EnqueueDownlink(t.Context(), applicationDownlink(321, orgs[321], 815001, &zero), testDownlinkLifetime)
	require.NoError(t, err, "an Application Center queue id of zero is stored")

	stored, err := downlinks.GetDownlinkByQueueID(t.Context(), 815001, "321")
	require.NoError(t, err)
	require.NotNil(t, stored.ACQueID)
	assert.Zero(t, *stored.ACQueID)
}

// TestDownlinkLookups_ReturnApplicationQueueID pins that the lookups feeding
// Application Center messages carry the Application Center id, and that a
// downlink enqueued without one carries none.
func TestDownlinkLookups_ReturnApplicationQueueID(t *testing.T) {
	downlinks, _, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	acID := uint64(7)

	scheduled := applicationDownlink(321, orgs[321], 820001, &acID)
	scheduled.CntDepend = true
	scheduled.PacketCntArray = []int64{1}
	scheduled.UserData = [][]byte{{0x01}}
	_, err := downlinks.EnqueueDownlink(ctx, scheduled, testDownlinkLifetime)
	require.NoError(t, err)
	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(322, orgs[322], 820002, nil), testDownlinkLifetime)
	require.NoError(t, err)

	byQueue, err := downlinks.GetDownlinkByQueueID(ctx, 820001, "321")
	require.NoError(t, err)
	require.NotNil(t, byQueue.ACQueID)
	assert.Equal(t, acID, *byQueue.ACQueID)

	byPacketCnt, err := downlinks.GetDownlinksByPacketCnt(ctx, "321", "70B3D59CD0000321", 1)
	require.NoError(t, err)
	require.Len(t, byPacketCnt, 1)
	require.NotNil(t, byPacketCnt[0].ACQueID)
	assert.Equal(t, acID, *byPacketCnt[0].ACQueID)

	internalOnly, err := downlinks.GetDownlinkByQueueID(ctx, 820002, "322")
	require.NoError(t, err)
	assert.Nil(t, internalOnly.ACQueID)
}

// TestApplicationQueueID_KeepsTheFullUnsignedRange pins that an Application
// Center queue id at or above 2^63 (SCACI §3.10.1: a 64-bit numeric) is stored
// and read back exactly by every lookup that names the downlink to an
// Application Center, stays unique among its organization's downlinks in
// flight, and may repeat in another tenant.
func TestApplicationQueueID_KeepsTheFullUnsignedRange(t *testing.T) {
	downlinks, _, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	acID := uint64(1)<<63 + 5
	largest := ^uint64(0)

	scheduled := applicationDownlink(321, orgs[321], 840001, &acID)
	scheduled.CntDepend = true
	scheduled.PacketCntArray = []int64{3}
	scheduled.UserData = [][]byte{{0x01}}
	_, err := downlinks.EnqueueDownlink(ctx, scheduled, testDownlinkLifetime)
	require.NoError(t, err)
	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 840002, &largest), testDownlinkLifetime)
	require.NoError(t, err)
	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(322, orgs[322], 840003, &acID), testDownlinkLifetime)
	require.NoError(t, err, "another tenant's Application Center may use the same queue id")
	_, err = downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], 840004, &acID), testDownlinkLifetime)
	require.ErrorIs(t, err, storage.ErrDuplicateKey)

	byQueue, err := downlinks.GetDownlinkByQueueID(ctx, 840001, "321")
	require.NoError(t, err)
	require.NotNil(t, byQueue.ACQueID)
	assert.Equal(t, acID, *byQueue.ACQueID)

	byPacketCnt, err := downlinks.GetDownlinksByPacketCnt(ctx, "321", "70B3D59CD0000321", 3)
	require.NoError(t, err)
	require.Len(t, byPacketCnt, 1)
	require.NotNil(t, byPacketCnt[0].ACQueID)
	assert.Equal(t, acID, *byPacketCnt[0].ACQueID)

	top, err := downlinks.GetDownlinkByQueueID(ctx, 840002, "321")
	require.NoError(t, err)
	require.NotNil(t, top.ACQueID)
	assert.Equal(t, largest, *top.ACQueID)
}
