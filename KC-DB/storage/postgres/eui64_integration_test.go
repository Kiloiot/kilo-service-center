package postgres

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
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

	repo := NewBaseStationRepository(db)
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
		require.NoError(t, repo.Create(ctx, bs), "EUI %016X must persist", eui)

		stored, err := repo.GetByEUI(ctx, tenantID, euiArr[:])
		require.NoError(t, err, "EUI %016X must read back tenant-scoped", eui)
		assert.Equal(t, euiArr, stored.EUI, "EUI %016X must be bit-exact", eui)

		global, err := repo.GetByEUIGlobal(ctx, euiArr[:])
		require.NoError(t, err, "EUI %016X must resolve via connect-time global lookup", eui)
		assert.Equal(t, euiArr, global.EUI)
		assert.Equal(t, tenantID, global.TenantID)

		require.NoError(t, repo.Delete(ctx, tenantID, stored.ID))
	}
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

		assert.Equal(t, eui64Bytes(eui), epStored, "ep_eui %016X must be bit-exact", eui)
		assert.Equal(t, eui64Bytes(eui), bsStored, "bs_eui %016X must be bit-exact", eui)
		assert.Equal(t, eui, binary.BigEndian.Uint64(epStored),
			"ep_eui %016X must recover the exact uint64", eui)
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
		require.NoError(t, db.QueryRow(`
			INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, sh_addr, nwk_sn_key, bidi)
			VALUES ($1, $1, $2, $3, $4, $5, true)
			RETURNING ep_eui`,
			tenantID, eui64Bytes(eui), "TestEUI64-EP", 0x1234,
			make([]byte, 16),
		).Scan(&stored), "endpoint EUI %016X must persist", eui)

		assert.Equal(t, eui, binary.BigEndian.Uint64(stored),
			"endpoint EUI %016X must be bit-exact", eui)
	}
}
