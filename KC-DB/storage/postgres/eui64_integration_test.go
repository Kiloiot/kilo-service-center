package postgres

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eui64MatrixValues covers the full unsigned EUI-64 range boundaries,
// including values above INT64_MAX that overflow signed storage.
var eui64MatrixValues = []uint64{
	0x0000000000000001,
	0x7FFFFFFFFFFFFFFF,
	0x8000000000000000,
	0xCAFECAFECAFECAFE,
	0xFFFFFFFFFFFFFFFF,
}

func eui64Bytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// TestEUI64BaseStationPersistence verifies base stations with any EUI-64
// value persist and read back bit-exact through the BYTEA storage path,
// both tenant-scoped and via the global connect-time lookup.
func TestEUI64BaseStationPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	const tenantID = int64(400)
	createTestTenant(t, db, tenantID, "TestTenantEUI64")

	repo := NewBaseStationRepository(db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	for _, eui := range eui64MatrixValues {
		var euiArr models.EUI
		copy(euiArr[:], eui64Bytes(eui))

		bs := &models.BaseStation{
			EUI:              euiArr,
			TenantID:         tenantID,
			Name:             "TestEUI64-BS",
			ConnectionType:   models.ConnectionTypeBSSCI,
			ServiceCenterURL: testServiceCenterURLPtr(),
		}
		require.NoError(t, repo.Create(ctx, bs), "EUI %s must persist", mioty.FormatEUI64(eui))

		stored, err := repo.GetByEUI(ctx, tenantID, euiArr[:])
		require.NoError(t, err, "EUI %s must read back tenant-scoped", mioty.FormatEUI64(eui))
		assert.Equal(t, euiArr, stored.EUI, "EUI %s must be bit-exact", mioty.FormatEUI64(eui))

		global, err := repo.GetByEUIGlobal(ctx, euiArr[:])
		require.NoError(t, err, "EUI %s must resolve via connect-time global lookup", mioty.FormatEUI64(eui))
		assert.Equal(t, euiArr, global.EUI)
		assert.Equal(t, tenantID, global.TenantID)

		removed, err := repo.DeleteByEUI(ctx, tenantID, euiArr[:])
		require.NoError(t, err)
		assert.Equal(t, stored.ID, removed.ID, "the delete returns the row it removed")
	}
}

// TestEUI64BaseStationListWithStats verifies the message-statistics join
// matches BYTEA bs_eui values across the full unsigned range (regression:
// the join previously cast basestations.bs_eui to signed BIGINT).
func TestEUI64BaseStationListWithStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	const tenantID = int64(403)
	createTestTenant(t, db, tenantID, "TestTenantEUI64Stats")

	repo := NewBaseStationRepository(db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	const bsEUI = uint64(0xCAFECAFECAFECAFE)
	var euiArr models.EUI
	copy(euiArr[:], eui64Bytes(bsEUI))

	bs := &models.BaseStation{
		EUI:              euiArr,
		TenantID:         tenantID,
		Name:             "TestEUI64-Stats-BS",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}
	require.NoError(t, repo.Create(ctx, bs))

	received := time.Now()
	insertArchivalMessage(t, db, archivalMessageParams{
		TenantID: tenantID, EpEUI: 0x8000000000000001, BsEUI: bsEUI, ReceivedAt: received,
	})
	insertArchivalMessage(t, db, archivalMessageParams{
		TenantID: tenantID, EpEUI: 0xFFFFFFFFFFFFFFFF, BsEUI: bsEUI, ReceivedAt: received,
	})

	stats, total, err := repo.ListWithStats(ctx, tenantID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)

	var found *BaseStationWithStats
	for _, s := range stats {
		if string(s.BsEui) == string(eui64Bytes(bsEUI)) {
			found = s
			break
		}
	}
	require.NotNil(t, found, "high-bit BS must appear in ListWithStats")
	assert.Equal(t, 2, found.MessageCount,
		"messages with high-bit bs_eui must join to the base station")
	assert.Equal(t, 2, found.UniqueEndpoints)
}

// TestEUI64MessagePersistence verifies uplink messages carrying full-range
// endpoint and base station EUIs survive the message store round-trip.
func TestEUI64MessagePersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	const tenantID = int64(401)
	createTestTenant(t, db, tenantID, "TestTenantEUI64Msg")

	for _, eui := range eui64MatrixValues {
		received := time.Now()
		id := insertArchivalMessage(t, db, archivalMessageParams{
			TenantID:   tenantID,
			EpEUI:      eui,
			BsEUI:      eui,
			ReceivedAt: received,
		})

		var epStored, bsStored []byte
		require.NoError(t, db.QueryRow(
			`SELECT ep_eui, bs_eui FROM messages WHERE id = $1`, id,
		).Scan(&epStored, &bsStored))

		assert.Equal(t, eui64Bytes(eui), epStored, "ep_eui %s must be bit-exact", mioty.FormatEUI64(eui))
		assert.Equal(t, eui64Bytes(eui), bsStored, "bs_eui %s must be bit-exact", mioty.FormatEUI64(eui))
		assert.Equal(t, eui, binary.BigEndian.Uint64(epStored),
			"ep_eui %s must recover the exact uint64", mioty.FormatEUI64(eui))
	}
}

// TestEUI64EndpointPersistence verifies endpoints with full-range EUIs
// persist through the endpoint repository BYTEA path.
func TestEUI64EndpointPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	const tenantID = int64(402)
	createTestTenant(t, db, tenantID, "TestTenantEUI64Ep")

	for _, eui := range eui64MatrixValues {
		var stored []byte
		require.NoError(t, db.QueryRow(
			`
			INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, sh_addr, nwk_key, bidi)
			VALUES ($1, $1, $2, $3, $4, $5, true)
			RETURNING ep_eui`,
			tenantID, eui64Bytes(eui), "TestEUI64-EP", 0x1234,
			envelopeForTest(make([]byte, 16)),
		).Scan(&stored), "endpoint EUI %s must persist", mioty.FormatEUI64(eui))

		assert.Equal(t, eui, binary.BigEndian.Uint64(stored),
			"endpoint EUI %s must be bit-exact", mioty.FormatEUI64(eui))
	}
}
