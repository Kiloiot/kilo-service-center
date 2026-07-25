package postgres

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// euiHex converts an EUI64 to the lowercase hex form used by the results API.
func euiHex(eui uint64) string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, eui)
	return hex.EncodeToString(b)
}

// setTransmittedAt backfills transmitted_at for a queued test downlink so
// time-range filters have deterministic data.
func setTransmittedAt(t *testing.T, db *DB, queID int64, at time.Time) {
	t.Helper()
	_, err := db.sqlxDB.Exec(
		"UPDATE downlink_queue SET transmitted_at = $1 WHERE que_id = $2", at, queID)
	require.NoError(t, err)
}

// TestDownlinkResultsFiltering verifies GetDownlinkResults endpoint, status,
// and time-range filters against the current downlink_queue schema.
func TestDownlinkResultsFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epA := uint64(0x1234567890ABCDEF)
	epB := uint64(0xFEDCBA0987654321)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epA, Name: "TestEndpoint-Results-A", TenantID: 100})
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epB, Name: "TestEndpoint-Results-B", TenantID: 100})

	now := time.Now()
	seed := []struct {
		epEUI         uint64
		status        string
		transmittedAt *time.Time
	}{
		{epA, bssci.DLQueueStatusTransmitted, ptrTime(now.Add(-1 * time.Hour))},
		{epA, bssci.DLQueueStatusExpired, ptrTime(now.Add(-45 * time.Minute))},
		{epA, bssci.DLQueueStatusFailed, ptrTime(now.Add(-15 * time.Minute))},
		{epB, bssci.DLQueueStatusTransmitted, ptrTime(now.Add(-1 * time.Hour))},
		{epB, bssci.DLQueueStatusPending, nil},
	}
	for _, s := range seed {
		queID := insertDownlink(t, db, DownlinkInsertParams{
			EpEUI:    s.epEUI,
			TenantID: 100,
			Status:   s.status,
		})
		if s.transmittedAt != nil {
			setTransmittedAt(t, db, queID, *s.transmittedAt)
		}
	}

	tests := []struct {
		name             string
		epEUI            string
		statusFilter     string
		timeFrom         *time.Time
		timeTo           *time.Time
		expectedCount    int
		expectedStatuses map[string]int
	}{
		{
			name:          "no filters returns only result statuses",
			expectedCount: 4,
			expectedStatuses: map[string]int{
				bssci.DLQueueStatusTransmitted: 2,
				bssci.DLQueueStatusExpired:     1,
				bssci.DLQueueStatusFailed:      1,
			},
		},
		{
			name:          "filter by endpoint",
			epEUI:         euiHex(epA),
			expectedCount: 3,
			expectedStatuses: map[string]int{
				bssci.DLQueueStatusTransmitted: 1,
				bssci.DLQueueStatusExpired:     1,
				bssci.DLQueueStatusFailed:      1,
			},
		},
		{
			name:          "filter by status transmitted",
			statusFilter:  bssci.DLQueueStatusTransmitted,
			expectedCount: 2,
			expectedStatuses: map[string]int{
				bssci.DLQueueStatusTransmitted: 2,
			},
		},
		{
			name:          "filter by endpoint and status",
			epEUI:         euiHex(epA),
			statusFilter:  bssci.DLQueueStatusTransmitted,
			expectedCount: 1,
			expectedStatuses: map[string]int{
				bssci.DLQueueStatusTransmitted: 1,
			},
		},
		{
			name:          "filter by time range",
			timeFrom:      ptrTime(now.Add(-2 * time.Hour)),
			timeTo:        ptrTime(now.Add(-30 * time.Minute)),
			expectedCount: 3,
			expectedStatuses: map[string]int{
				bssci.DLQueueStatusTransmitted: 2,
				bssci.DLQueueStatusExpired:     1,
			},
		},
		{
			name:          "all filters combined",
			epEUI:         euiHex(epA),
			statusFilter:  bssci.DLQueueStatusTransmitted,
			timeFrom:      ptrTime(now.Add(-2 * time.Hour)),
			timeTo:        ptrTime(now),
			expectedCount: 1,
			expectedStatuses: map[string]int{
				bssci.DLQueueStatusTransmitted: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, totalCount, err := db.GetDownlinkResults(
				t.Context(), tt.epEUI, "100", nil, tt.statusFilter, tt.timeFrom, tt.timeTo, 10, 0)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, totalCount)
			assert.Len(t, results, tt.expectedCount)

			gotStatuses := make(map[string]int, len(results))
			for _, r := range results {
				gotStatuses[r.Status]++
				if tt.epEUI != "" {
					assert.Equal(t, tt.epEUI, r.EPEUI, "results must match the endpoint filter")
				}
			}
			assert.Equal(t, tt.expectedStatuses, gotStatuses)
		})
	}

	t.Run("in-flight status filter is rejected", func(t *testing.T) {
		_, _, err := db.GetDownlinkResults(
			t.Context(), "", "100", nil, bssci.DLQueueStatusQueued, nil, nil, 10, 0)
		require.Error(t, err, "non-terminal statuses are not valid result filters")
	})
}

// TestDownlinkResultsTenantIsolation verifies GetDownlinkResults never leaks
// rows across tenants.
func TestDownlinkResultsTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")
	createTestTenant(t, sqlxDB, 200, "TestTenant200")
	createTestTenant(t, sqlxDB, 300, "TestTenant300")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x1234567890ABCDEF)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epEUI, Name: "TestEndpoint-TenantIso", TenantID: 100})

	seed := []struct {
		tenantID int64
		status   string
	}{
		{100, bssci.DLQueueStatusTransmitted},
		{100, bssci.DLQueueStatusExpired},
		{100, bssci.DLQueueStatusFailed},
		{200, bssci.DLQueueStatusTransmitted},
		{200, bssci.DLQueueStatusExpired},
	}
	for _, s := range seed {
		insertDownlink(t, db, DownlinkInsertParams{
			EpEUI:    epEUI,
			TenantID: s.tenantID,
			Status:   s.status,
		})
	}

	tests := []struct {
		tenantID      string
		expectedCount int
	}{
		{"100", 3},
		{"200", 2},
		{"300", 0},
	}

	for _, tt := range tests {
		t.Run("tenant "+tt.tenantID, func(t *testing.T) {
			results, totalCount, err := db.GetDownlinkResults(
				t.Context(), "", tt.tenantID, nil, "", nil, nil, 10, 0)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, totalCount)
			assert.Len(t, results, tt.expectedCount)

			for _, r := range results {
				assert.Equal(t, tt.tenantID, r.TenantID, "results must belong to the requesting tenant")
			}
		})
	}
}

// TestDownlinkResultSecurityBoundaries verifies the tenant and endpoint match
// requirements of UpdateDownlinkResult (BSSCI section 3.14 cross-tenant guard).
func TestDownlinkResultSecurityBoundaries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")
	createTestTenant(t, sqlxDB, 200, "TestTenant200")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x1234567890ABCDEF)
	wrongEpEUI := uint64(0xFEDCBA0987654321)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epEUI, Name: "TestEndpoint-Security", TenantID: 100})

	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)
	wrongEpEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(wrongEpEUIBytes, wrongEpEUI)
	bsEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEUIBytes, uint64(0x0123456789ABCDEF))

	queID := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Status:   bssci.DLQueueStatusQueued,
	})

	txTime := time.Now().UnixNano()
	packetCnt := uint32(42)

	resetQueued := func(t *testing.T) {
		t.Helper()
		_, err := sqlxDB.Exec(
			"UPDATE downlink_queue SET status = $1 WHERE que_id = $2",
			bssci.DLQueueStatusQueued, queID)
		require.NoError(t, err)
	}

	requireStatus := func(t *testing.T, want string) {
		t.Helper()
		var status string
		require.NoError(t, sqlxDB.QueryRow(
			"SELECT status FROM downlink_queue WHERE que_id = $1", queID).Scan(&status))
		assert.Equal(t, want, status)
	}

	t.Run("same tenant and endpoint can update", func(t *testing.T) {
		err := db.UpdateDownlinkResult(
			t.Context(), queID, bssci.DLDataResultSent, &txTime, &packetCnt,
			bsEUIBytes, epEUIBytes, "100", nil)
		require.NoError(t, err)
		requireStatus(t, bssci.DLQueueStatusTransmitted)
		resetQueued(t)
	})

	t.Run("cross-tenant update is rejected", func(t *testing.T) {
		err := db.UpdateDownlinkResult(
			t.Context(), queID, bssci.DLDataResultSent, &txTime, &packetCnt,
			bsEUIBytes, epEUIBytes, "200", nil)
		assert.ErrorIs(t, err, storage.ErrDownlinkNotFound)
		requireStatus(t, bssci.DLQueueStatusQueued)
	})

	t.Run("endpoint mismatch is rejected", func(t *testing.T) {
		err := db.UpdateDownlinkResult(
			t.Context(), queID, bssci.DLDataResultSent, &txTime, &packetCnt,
			bsEUIBytes, wrongEpEUIBytes, "100", nil)
		assert.ErrorIs(t, err, storage.ErrDownlinkNotFound)
		requireStatus(t, bssci.DLQueueStatusQueued)
	})

	t.Run("tenant and endpoint both wrong is rejected", func(t *testing.T) {
		err := db.UpdateDownlinkResult(
			t.Context(), queID, bssci.DLDataResultSent, &txTime, &packetCnt,
			bsEUIBytes, wrongEpEUIBytes, "200", nil)
		assert.ErrorIs(t, err, storage.ErrDownlinkNotFound)
		requireStatus(t, bssci.DLQueueStatusQueued)
	})

	t.Run("non-existent queue ID is rejected", func(t *testing.T) {
		err := db.UpdateDownlinkResult(
			t.Context(), queID+9999, bssci.DLDataResultSent, &txTime, &packetCnt,
			bsEUIBytes, epEUIBytes, "100", nil)
		assert.ErrorIs(t, err, storage.ErrDownlinkNotFound)
		requireStatus(t, bssci.DLQueueStatusQueued)
	})

	t.Run("expired result with nil txTime and packetCnt succeeds", func(t *testing.T) {
		err := db.UpdateDownlinkResult(
			t.Context(), queID, bssci.DLDataResultExpired, nil, nil,
			bsEUIBytes, epEUIBytes, "100", nil)
		require.NoError(t, err)
		requireStatus(t, bssci.DLQueueStatusExpired)
	})
}

// TestDownlinkResultConcurrentTenantIsolation verifies concurrent
// UpdateDownlinkResult calls cannot cross tenant boundaries.
func TestDownlinkResultConcurrentTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")
	createTestTenant(t, sqlxDB, 200, "TestTenant200")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x1234567890ABCDEF)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epEUI, Name: "TestEndpoint-Concurrent", TenantID: 100})

	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)
	bsEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEUIBytes, uint64(0x0123456789ABCDEF))

	// Ten downlinks alternating between tenants: even indices belong to
	// tenant 200, odd indices to tenant 100.
	const total = 10
	queIDs := make([]int64, total)
	for i := 0; i < total; i++ {
		tenantID := int64(100)
		if i%2 == 0 {
			tenantID = 200
		}
		queIDs[i] = insertDownlink(t, db, DownlinkInsertParams{
			EpEUI:    epEUI,
			TenantID: tenantID,
			Status:   bssci.DLQueueStatusQueued,
		})
	}

	// All goroutines act as tenant 100; updates against tenant 200 rows must
	// fail with the not-found sentinel and leave those rows untouched.
	var wg sync.WaitGroup
	errCh := make(chan error, total)
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			txTime := time.Now().UnixNano()
			packetCnt := uint32(index)
			err := db.UpdateDownlinkResult(
				t.Context(), queIDs[index], bssci.DLDataResultSent, &txTime, &packetCnt,
				bsEUIBytes, epEUIBytes, "100", nil)

			if index%2 == 0 {
				if err == nil {
					errCh <- fmt.Errorf("queue %d belongs to tenant 200 but tenant 100 update succeeded", queIDs[index])
				}
			} else if err != nil {
				errCh <- fmt.Errorf("queue %d belongs to tenant 100 but update failed: %w", queIDs[index], err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}

	var tenant100Transmitted, tenant200Queued int
	require.NoError(t, sqlxDB.QueryRow(
		"SELECT COUNT(*) FROM downlink_queue WHERE tenant_id = 100 AND status = $1",
		bssci.DLQueueStatusTransmitted).Scan(&tenant100Transmitted))
	require.NoError(t, sqlxDB.QueryRow(
		"SELECT COUNT(*) FROM downlink_queue WHERE tenant_id = 200 AND status = $1",
		bssci.DLQueueStatusQueued).Scan(&tenant200Queued))

	assert.Equal(t, total/2, tenant100Transmitted, "all tenant 100 rows must be transmitted")
	assert.Equal(t, total/2, tenant200Queued, "all tenant 200 rows must remain queued")
}
