package postgres

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// euiHex converts an EUI64 to the lowercase hex form used by the results API.
func euiHex(eui uint64) string {
	return mioty.FormatEUI64(eui)
}

// setTransmittedAt backfills transmitted_at for a queued test downlink so
// time-range filters have deterministic data.
func setTransmittedAt(t *testing.T, db *DB, queID int64, at time.Time) {
	t.Helper()
	_, err := db.sqlxDB.Exec(
		"UPDATE downlink_queue SET transmitted_at = $1 WHERE que_id = $2", at, queID,
	)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epA := uint64(0x1234567890ABCDEF)
	epB := uint64(0xFEDCBA0987654321)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epA, Name: "TestEndpoint-Results-A", TenantID: 100})
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epB, Name: "TestEndpoint-Results-B", TenantID: 100})

	now := time.Now()
	seed := []struct {
		epEUI         uint64
		status        mioty.DLQueueStatus
		transmittedAt *time.Time
	}{
		{epA, mioty.DLQueueStatusTransmitted, ptrTime(now.Add(-1 * time.Hour))},
		{epA, mioty.DLQueueStatusExpired, ptrTime(now.Add(-45 * time.Minute))},
		{epA, mioty.DLQueueStatusFailed, ptrTime(now.Add(-15 * time.Minute))},
		{epB, mioty.DLQueueStatusTransmitted, ptrTime(now.Add(-1 * time.Hour))},
		{epB, mioty.DLQueueStatusPending, nil},
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
		expectedRows     int
		expectedStatuses map[mioty.DLQueueStatus]int
	}{
		{
			name:         "no filters returns only result statuses",
			expectedRows: 4,
			expectedStatuses: map[mioty.DLQueueStatus]int{
				mioty.DLQueueStatusTransmitted: 2,
				mioty.DLQueueStatusExpired:     1,
				mioty.DLQueueStatusFailed:      1,
			},
		},
		{
			name:         "filter by endpoint",
			epEUI:        euiHex(epA),
			expectedRows: 3,
			expectedStatuses: map[mioty.DLQueueStatus]int{
				mioty.DLQueueStatusTransmitted: 1,
				mioty.DLQueueStatusExpired:     1,
				mioty.DLQueueStatusFailed:      1,
			},
		},
		{
			name:         "filter by status transmitted",
			statusFilter: string(mioty.DLQueueStatusTransmitted),
			expectedRows: 2,
			expectedStatuses: map[mioty.DLQueueStatus]int{
				mioty.DLQueueStatusTransmitted: 2,
			},
		},
		{
			name:         "filter by endpoint and status",
			epEUI:        euiHex(epA),
			statusFilter: string(mioty.DLQueueStatusTransmitted),
			expectedRows: 1,
			expectedStatuses: map[mioty.DLQueueStatus]int{
				mioty.DLQueueStatusTransmitted: 1,
			},
		},
		{
			name:         "filter by time range",
			timeFrom:     ptrTime(now.Add(-2 * time.Hour)),
			timeTo:       ptrTime(now.Add(-30 * time.Minute)),
			expectedRows: 3,
			expectedStatuses: map[mioty.DLQueueStatus]int{
				mioty.DLQueueStatusTransmitted: 2,
				mioty.DLQueueStatusExpired:     1,
			},
		},
		{
			name:         "all filters combined",
			epEUI:        euiHex(epA),
			statusFilter: string(mioty.DLQueueStatusTransmitted),
			timeFrom:     ptrTime(now.Add(-2 * time.Hour)),
			timeTo:       ptrTime(now),
			expectedRows: 1,
			expectedStatuses: map[mioty.DLQueueStatus]int{
				mioty.DLQueueStatusTransmitted: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, totalCount, err := NewRepositories(db).Downlinks.GetDownlinkResults(
				t.Context(), 100, nil, storage.DownlinkResultFilter{EpEUI: hexEUIOrNil(t, tt.epEUI), Status: tt.statusFilter, From: tt.timeFrom, To: tt.timeTo}, 10, 0,
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedRows, totalCount)
			assert.Len(t, results, tt.expectedRows)

			gotStatuses := make(map[mioty.DLQueueStatus]int, len(results))
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
		_, _, err := NewRepositories(db).Downlinks.GetDownlinkResults(
			t.Context(), 100, nil, storage.DownlinkResultFilter{Status: string(mioty.DLQueueStatusQueued)}, 10, 0,
		)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x1234567890ABCDEF)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epEUI, Name: "TestEndpoint-TenantIso", TenantID: 100})

	seed := []struct {
		tenantID int64
		status   mioty.DLQueueStatus
	}{
		{100, mioty.DLQueueStatusTransmitted},
		{100, mioty.DLQueueStatusExpired},
		{100, mioty.DLQueueStatusFailed},
		{200, mioty.DLQueueStatusTransmitted},
		{200, mioty.DLQueueStatusExpired},
	}
	for _, s := range seed {
		insertDownlink(t, db, DownlinkInsertParams{
			EpEUI:    epEUI,
			TenantID: s.tenantID,
			Status:   s.status,
		})
	}

	tests := []struct {
		tenantID     string
		expectedRows int
	}{
		{"100", 3},
		{"200", 2},
		{"300", 0},
	}

	for _, tt := range tests {
		t.Run("tenant "+tt.tenantID, func(t *testing.T) {
			tenantID, err := strconv.ParseInt(tt.tenantID, 10, 64)
			require.NoError(t, err)
			results, totalCount, err := NewRepositories(db).Downlinks.GetDownlinkResults(
				t.Context(), tenantID, nil, storage.DownlinkResultFilter{}, 10, 0,
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedRows, totalCount)
			assert.Len(t, results, tt.expectedRows)

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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x1234567890ABCDEF)
	wrongEpEUI := uint64(0xFEDCBA0987654321)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epEUI, Name: "TestEndpoint-Security", TenantID: 100})

	bsEUI := uint64(0x0123456789ABCDEF)

	queID := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Status:   mioty.DLQueueStatusQueued,
		BsEUI:    bsEUI,
	})

	txTime := time.Now().UnixNano()
	packetCnt := uint32(42)
	sent := func(queID int64, epEUI uint64) *mioty.DLDataResult {
		return &mioty.DLDataResult{EpEui: epEUI, QueId: uint64(queID), Result: mioty.DLDataResultSent, TxTime: &txTime, PacketCnt: &packetCnt}
	}

	resetQueued := func(t *testing.T) {
		t.Helper()
		_, err := sqlxDB.Exec(
			"UPDATE downlink_queue SET status = $1 WHERE que_id = $2",
			mioty.DLQueueStatusQueued, queID,
		)
		require.NoError(t, err)
	}

	requireStatus := func(t *testing.T, want mioty.DLQueueStatus) {
		t.Helper()
		var status mioty.DLQueueStatus
		require.NoError(t, sqlxDB.QueryRow(
			"SELECT status FROM downlink_queue WHERE que_id = $1", queID,
		).Scan(&status))
		assert.Equal(t, want, status)
	}

	t.Run("same tenant and endpoint can update", func(t *testing.T) {
		row, err := NewRepositories(db).Downlinks.UpdateDownlinkResult(t.Context(), 100, bsEUI, sent(queID, epEUI))
		require.NoError(t, err)
		assert.Equal(t, queID, row.QueID)
		assert.Equal(t, "100", row.TenantID)
		requireStatus(t, mioty.DLQueueStatusTransmitted)
		resetQueued(t)
	})

	t.Run("cross-tenant update is rejected", func(t *testing.T) {
		_, err := NewRepositories(db).Downlinks.UpdateDownlinkResult(t.Context(), 200, bsEUI, sent(queID, epEUI))
		assert.ErrorIs(t, err, storage.ErrDownlinkNotFound)
		requireStatus(t, mioty.DLQueueStatusQueued)
	})

	t.Run("endpoint mismatch is rejected", func(t *testing.T) {
		_, err := NewRepositories(db).Downlinks.UpdateDownlinkResult(t.Context(), 100, bsEUI, sent(queID, wrongEpEUI))
		assert.ErrorIs(t, err, storage.ErrDownlinkNotFound)
		requireStatus(t, mioty.DLQueueStatusQueued)
	})

	t.Run("tenant and endpoint both wrong is rejected", func(t *testing.T) {
		_, err := NewRepositories(db).Downlinks.UpdateDownlinkResult(t.Context(), 200, bsEUI, sent(queID, wrongEpEUI))
		assert.ErrorIs(t, err, storage.ErrDownlinkNotFound)
		requireStatus(t, mioty.DLQueueStatusQueued)
	})

	t.Run("non-existent queue ID is rejected", func(t *testing.T) {
		_, err := NewRepositories(db).Downlinks.UpdateDownlinkResult(t.Context(), 100, bsEUI, sent(queID+9999, epEUI))
		assert.ErrorIs(t, err, storage.ErrDownlinkNotFound)
		requireStatus(t, mioty.DLQueueStatusQueued)
	})

	t.Run("expired result with nil txTime and packetCnt succeeds", func(t *testing.T) {
		_, err := NewRepositories(db).Downlinks.UpdateDownlinkResult(t.Context(), 100, bsEUI,
			&mioty.DLDataResult{EpEui: epEUI, QueId: uint64(queID), Result: mioty.DLDataResultExpired})
		require.NoError(t, err)
		requireStatus(t, mioty.DLQueueStatusExpired)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x1234567890ABCDEF)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: epEUI, Name: "TestEndpoint-Concurrent", TenantID: 100})

	bsEUI := uint64(0x0123456789ABCDEF)

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
			Status:   mioty.DLQueueStatusQueued,
			BsEUI:    bsEUI,
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
			_, err := NewRepositories(db).Downlinks.UpdateDownlinkResult(t.Context(), 100, bsEUI, &mioty.DLDataResult{
				EpEui: epEUI, QueId: uint64(queIDs[index]), Result: mioty.DLDataResultSent, TxTime: &txTime, PacketCnt: &packetCnt,
			})

			if index%2 == 0 {
				if err == nil {
					errCh <- fmt.Errorf(errFmtQueueBelongsTenant200ButTenant100Update, queIDs[index])
				}
			} else if err != nil {
				errCh <- fmt.Errorf(errFmtQueueBelongsTenant100ButUpdate, queIDs[index], err)
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
		mioty.DLQueueStatusTransmitted,
	).Scan(&tenant100Transmitted))
	require.NoError(t, sqlxDB.QueryRow(
		"SELECT COUNT(*) FROM downlink_queue WHERE tenant_id = 200 AND status = $1",
		mioty.DLQueueStatusQueued,
	).Scan(&tenant200Queued))

	assert.Equal(t, total/2, tenant100Transmitted, "all tenant 100 rows must be transmitted")
	assert.Equal(t, total/2, tenant200Queued, "all tenant 200 rows must remain queued")
}

// hexEUIOrNil decodes a hex EUI for a filter, or leaves the filter open.
func hexEUIOrNil(t *testing.T, eui string) []byte {
	t.Helper()
	if eui == "" {
		return nil
	}
	b, err := hex.DecodeString(eui)
	require.NoError(t, err)
	return b
}
