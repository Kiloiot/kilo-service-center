package postgres

import (
	"encoding/binary"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarkDLRXStatusReceived_CrossBaseStationIsolation verifies BSSCI §5.16
// correlation: two concurrent dlRxStatQry queries for the same tenant and
// endpoint but different base stations must be satisfied independently - a
// dlRxStat report from one base station only marks that base station's query,
// never the other's.
func TestMarkDLRXStatusReceived_CrossBaseStationIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	const tenantID = int64(600)
	createTestTenant(t, db, tenantID, "TestTenantDLRXCorrelation")

	repo := NewDLRXStatusRepository(db, logger.NewNop())
	ctx := testutil.TestContext()

	epEui := euiBytesFromUint64(0x1122334455667788)
	bsA := euiBytesFromUint64(0xAAAAAAAAAAAAAAAA)
	bsB := euiBytesFromUint64(0xBBBBBBBBBBBBBBBB)

	// Two pending queries for the same endpoint, different base stations.
	require.NoError(t, repo.CreateDLRXStatusQuery(ctx, tenantID, nil, epEui, bsA, -1))
	require.NoError(t, repo.CreateDLRXStatusQuery(ctx, tenantID, nil, epEui, bsB, -2))

	// A report from base station A must mark only A's query.
	matched, err := repo.MarkDLRXStatusReceived(ctx, tenantID, epEui, bsA, 5)
	require.NoError(t, err)
	assert.True(t, matched, "base station A's report must match its own query")

	assertQueryStatus(t, db, tenantID, epEui, bsA, "received")
	assertQueryStatus(t, db, tenantID, epEui, bsB, "pending")

	// A report from base station B then marks B's query.
	matched, err = repo.MarkDLRXStatusReceived(ctx, tenantID, epEui, bsB, 6)
	require.NoError(t, err)
	assert.True(t, matched, "base station B's report must match its own query")
	assertQueryStatus(t, db, tenantID, epEui, bsB, "received")

	// A report for a base station with no pending query matches nothing.
	matched, err = repo.MarkDLRXStatusReceived(ctx, tenantID, epEui, euiBytesFromUint64(0xCCCCCCCCCCCCCCCC), 7)
	require.NoError(t, err)
	assert.False(t, matched, "a report for an un-queried base station must not match")
}

func euiBytesFromUint64(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func assertQueryStatus(t *testing.T, db *sqlx.DB, tenantID int64, epEui, bsEui []byte, want string) {
	t.Helper()
	var status string
	err := db.QueryRow(
		`SELECT status FROM dl_rx_status_queries WHERE tenant_id = $1 AND ep_eui = $2 AND bs_eui = $3`,
		tenantID, epEui, bsEui,
	).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, want, status)
}
