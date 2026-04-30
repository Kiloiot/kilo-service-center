package postgres

import (
	"testing"
	"time"

	"github.com/kilocenter/KC-Core/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDurationToInterval verifies time.Duration → PostgreSQL interval conversion
func TestDurationToInterval(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "24 hours",
			duration: 24 * time.Hour,
			expected: "86400.000000 seconds",
		},
		{
			name:     "10 minutes",
			duration: 10 * time.Minute,
			expected: "600.000000 seconds",
		},
		{
			name:     "1 hour 30 minutes",
			duration: 90 * time.Minute,
			expected: "5400.000000 seconds",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "0.000000 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := durationToInterval(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetOperationEventsByID_InvalidOperationType verifies error for invalid operation type
func TestGetOperationEventsByID_InvalidOperationType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Invalid operation type should return error
	_, err := repo.GetOperationEventsByID(
		ctx,
		"12345",
		[]string{testEventCategoryBSSCI},
		24*time.Hour,
		"invalid_type", // Invalid
		10,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid operation type")
}

// TestGetOperationEventsByID_AttachFilter verifies attach-only filtering
func TestGetOperationEventsByID_AttachFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Insert test events
	testOpID := "test-attach-op-12345"
	CleanupTestData(t, db, "system_events", "data::text", "%"+testOpID+"%")
	defer CleanupTestData(t, db, "system_events", "data::text", "%"+testOpID+"%")

	_, err := db.Exec(`
		INSERT INTO system_events (event_category, event_type, title, occurred_at, data)
		VALUES
			($1, 'attach_propagate', 'Attach started', NOW() - interval '1 hour', $2),
			($1, 'endpoint_attached', 'Endpoint attached', NOW() - interval '30 minutes', $2),
			($1, 'detach_propagate', 'Detach started', NOW() - interval '15 minutes', $2)
	`, testEventCategoryBSSCI, `{"operation_id": "`+testOpID+`"}`)
	require.NoError(t, err)

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Attach filter should return only attach events
	events, err := repo.GetOperationEventsByID(
		ctx,
		testOpID,
		[]string{testEventCategoryBSSCI},
		24*time.Hour,
		"attach",
		10,
	)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 2, "Should find at least 2 attach events")

	// Verify no detach events returned
	for _, event := range events {
		assert.NotContains(t, event.EventType, "detach")
	}
}

// TestGetOperationEventsByID_DetachFilter verifies detach-only filtering
func TestGetOperationEventsByID_DetachFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Insert test events
	testOpID := "test-detach-op-67890"
	CleanupTestData(t, db, "system_events", "data::text", "%"+testOpID+"%")
	defer CleanupTestData(t, db, "system_events", "data::text", "%"+testOpID+"%")

	_, err := db.Exec(`
		INSERT INTO system_events (event_category, event_type, title, occurred_at, data)
		VALUES
			($1, 'attach_propagate', 'Attach started', NOW() - interval '1 hour', $2),
			($1, 'detach_propagate', 'Detach started', NOW() - interval '30 minutes', $2),
			($1, 'endpoint_detached', 'Endpoint detached', NOW() - interval '15 minutes', $2)
	`, testEventCategoryBSSCI, `{"operation_id": "`+testOpID+`"}`)
	require.NoError(t, err)

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Detach filter should return only detach events
	events, err := repo.GetOperationEventsByID(
		ctx,
		testOpID,
		[]string{testEventCategoryBSSCI},
		24*time.Hour,
		"detach",
		10,
	)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 2, "Should find at least 2 detach events")

	// Verify no attach events returned (except detach ones)
	for _, event := range events {
		assert.NotEqual(t, "attach_propagate", event.EventType)
		assert.NotEqual(t, "endpoint_attached", event.EventType)
	}
}

// TestGetOperationEventsByID_AllOperations verifies empty filter returns all events
func TestGetOperationEventsByID_AllOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Insert test events
	testOpID := "test-all-ops-11111"
	CleanupTestData(t, db, "system_events", "data::text", "%"+testOpID+"%")
	defer CleanupTestData(t, db, "system_events", "data::text", "%"+testOpID+"%")

	_, err := db.Exec(`
		INSERT INTO system_events (event_category, event_type, title, occurred_at, data)
		VALUES
			($1, 'attach_propagate', 'Attach started', NOW() - interval '1 hour', $2),
			($1, 'detach_propagate', 'Detach started', NOW() - interval '30 minutes', $2),
			($1, 'endpoint_attached', 'Endpoint attached', NOW() - interval '20 minutes', $2),
			($1, 'endpoint_detached', 'Endpoint detached', NOW() - interval '10 minutes', $2)
	`, testEventCategoryBSSCI, `{"operation_id": "`+testOpID+`"}`)
	require.NoError(t, err)

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Empty filter should return all attach/detach events
	events, err := repo.GetOperationEventsByID(
		ctx,
		testOpID,
		[]string{testEventCategoryBSSCI},
		24*time.Hour,
		"", // All operations
		10,
	)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 4, "Should find all 4 events")
}

// TestGetOperationEventsByID_NoResults verifies empty slice for no matches
func TestGetOperationEventsByID_NoResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Non-existent operation ID should return empty slice, not error
	events, err := repo.GetOperationEventsByID(
		ctx,
		"nonexistent-op-99999",
		[]string{testEventCategoryBSSCI},
		24*time.Hour,
		"",
		10,
	)

	require.NoError(t, err)
	assert.Empty(t, events)
}

// TestGetEndpointOperationsByID_FindsEvents verifies endpoint operation search by EUI
func TestGetEndpointOperationsByID_FindsEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Insert test endpoint with known EUI
	testEndpointID := int64(12345)
	CleanupTestData(t, db, "endpoints", "id", "12345")
	defer CleanupTestData(t, db, "endpoints", "id", "12345")

	insertEndpoint(t, db, EndpointInsertParams{
		ID:       12345,
		EpEUI:    0x0000000012345678,
		Name:     "Test EP 12345",
		TenantID: 1,
	})

	// Insert test events with EUI hex string
	CleanupTestData(t, db, "system_events", "data::text", "%0000000012345678%")
	defer CleanupTestData(t, db, "system_events", "data::text", "%0000000012345678%")

	_, err := db.Exec(`
		INSERT INTO system_events (event_category, event_type, title, description, occurred_at, data)
		VALUES
			($1, 'endpoint_attached', 'Endpoint attached', 'Successful attach', NOW() - interval '1 hour', '{"eui": "0000000012345678"}'),
			($1, 'downlink_queued', 'Downlink queued', 'DL queued', NOW() - interval '30 minutes', '{"eui": "0000000012345678"}')
	`, testEventCategoryBSSCI)
	require.NoError(t, err)

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Search by endpoint ID (with tenant) should find events by EUI
	events, err := repo.GetEndpointOperationsByID(
		ctx,
		testEndpointID,
		1, // tenant ID
		[]string{testEventCategoryBSSCI},
		24*time.Hour,
		10,
		0, // offset
	)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 2, "Should find at least 2 events for endpoint")
}

// TestGetEndpointOperationsByID_NoResults verifies empty result handling
func TestGetEndpointOperationsByID_NoResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Non-existent endpoint should return ErrNotFound (not found for tenant)
	_, err := repo.GetEndpointOperationsByID(
		ctx,
		99999, // Non-existent
		1,     // tenant ID
		[]string{testEventCategoryBSSCI},
		24*time.Hour,
		10,
		0, // offset
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetBSSCIStatusSummary_ReturnsStatuses verifies base station status query
func TestGetBSSCIStatusSummary_ReturnsStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Clean up and insert test base station
	CleanupTestData(t, db, "basestations", "name", "TestBSSCIStatus%")
	defer CleanupTestData(t, db, "basestations", "name", "TestBSSCIStatus%")

	// Insert base station using decode to convert hex to bytea
	_, err := db.Exec(`
		INSERT INTO basestations (bs_eui, name, is_online, last_seen_at, tenant_id, connection_type, service_center_url, created_at, updated_at)
		VALUES (decode('0000000000000001', 'hex'), 'TestBSSCIStatus-BS1', true, NOW(), 1, 'bssci', 'tcp://localhost:5000', NOW(), NOW())
	`)
	require.NoError(t, err)

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Query base station status
	statuses, err := repo.GetBSSCIStatusSummary(
		ctx,
		1, // tenant_id
		[]string{testEventCategoryBSSCI},
		24*time.Hour,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, statuses, "Should find at least one base station")

	// Verify structure
	for _, status := range statuses {
		assert.NotEmpty(t, status.BsEUI)
		assert.NotEmpty(t, status.Name)
	}
}

// TestGetBSSCIStatusSummary_NoResults verifies empty result handling
func TestGetBSSCIStatusSummary_NoResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Clean up all base stations temporarily
	_, _ = db.Exec("UPDATE basestations SET is_online = false, last_seen_at = NOW() - interval '1 year'")
	defer func() {
		_, _ = db.Exec("UPDATE basestations SET is_online = true, last_seen_at = NOW()")
	}()

	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Test: Query with very short window should return empty slice
	statuses, err := repo.GetBSSCIStatusSummary(
		ctx,
		1, // tenant_id
		[]string{testEventCategoryBSSCI},
		1*time.Second, // Very short event window
	)

	require.NoError(t, err)
	assert.Empty(t, statuses) // Expect empty slice, not error
}

// TestGetEndpointOperationsByID_TenantIsolation verifies tenant isolation
func TestGetEndpointOperationsByID_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() { _ = db.Close() }()
	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Create tenants first
	_, err := db.Exec(`
		INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES
			(1, 'Tenant1', 'Test tenant 1', 'active', NOW(), NOW()),
			(2, 'Tenant2', 'Test tenant 2', 'active', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`)
	require.NoError(t, err)
	defer CleanupTestData(t, db, "tenants", "name", "Tenant1")
	defer CleanupTestData(t, db, "tenants", "name", "Tenant2")

	// Insert endpoint for tenant 1
	insertEndpoint(t, db, EndpointInsertParams{
		ID:       999,
		EpEUI:    0x0000000000000999,
		Name:     "Test EP",
		TenantID: 1,
	})
	defer CleanupTestData(t, db, "endpoints", "id", "999")

	// Insert event for this endpoint
	_, err = db.Exec(`
		INSERT INTO system_events (event_category, event_type, title, description, occurred_at, data)
		VALUES ($1, 'attach', 'Test', 'Test event for tenant isolation', NOW(), '{"eui": "0000000000000999"}')
	`, testEventCategoryBSSCI)
	require.NoError(t, err)
	defer CleanupTestData(t, db, "system_events", "title", "Test")

	// Test: Tenant 1 should see events
	events, err := repo.GetEndpointOperationsByID(ctx, 999, 1,
		[]string{testEventCategoryBSSCI}, 24*time.Hour, 10, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, events)

	// Test: Tenant 2 should get ErrNotFound (cross-tenant access blocked)
	_, err = repo.GetEndpointOperationsByID(ctx, 999, 2,
		[]string{testEventCategoryBSSCI}, 24*time.Hour, 10, 0)
	require.Error(t, err)
	// Note: Using strings.Contains because errors.ErrNotFound may be wrapped
	assert.Contains(t, err.Error(), "not found")
}

// TestGetBSSCIStatusSummary_NullLastSeenAt verifies NULL timestamp handling
func TestGetBSSCIStatusSummary_NullLastSeenAt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() { _ = db.Close() }()
	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Insert base station with NULL last_seen_at
	_, err := db.Exec(`
		INSERT INTO basestations (bs_eui, name, is_online, last_seen_at, tenant_id, connection_type, service_center_url, created_at, updated_at)
		VALUES (decode('0000000000000AAA', 'hex'), 'NullTimeBS', false, NULL, 1, 'bssci', 'tcp://localhost:5000', NOW(), NOW())
	`)
	require.NoError(t, err)
	defer CleanupTestData(t, db, "basestations", "name", "NullTimeBS")

	// Test: Should not panic on NULL
	statuses, err := repo.GetBSSCIStatusSummary(ctx,
		1, // tenant_id
		[]string{testEventCategoryBSSCI}, 24*time.Hour)
	require.NoError(t, err)

	for _, status := range statuses {
		if status.Name == "NullTimeBS" {
			assert.Nil(t, status.LastSeenAt, "NULL timestamp should be nil pointer")
			return
		}
	}
}

// TestGetEventSummary_Aggregation verifies event summary aggregation
func TestGetEventSummary_Aggregation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() { _ = db.Close() }()
	repo := NewOperationStatusRepository(db)
	ctx := testutil.TestContext()

	// Insert multiple events of different types
	testData := []struct{ eventType, title string }{
		{"attach", "Attach1"},
		{"attach", "Attach2"},
		{"detach", "Detach1"},
	}

	for _, td := range testData {
		_, err := db.Exec(`
			INSERT INTO system_events (event_category, event_type, title, occurred_at, data)
			VALUES ($1, $2, $3, NOW(), '{}')
		`, testEventCategoryBSSCI, td.eventType, td.title)
		require.NoError(t, err)
		defer CleanupTestData(t, db, "system_events", "title", td.title)
	}

	// Test: 5-minute window aggregation
	summary, err := repo.GetEventSummary(ctx, []string{testEventCategoryBSSCI}, 5*time.Minute)
	require.NoError(t, err)

	// Verify counts
	assert.Equal(t, 2, summary["attach"], "Should count 2 attach events")
	assert.Equal(t, 1, summary["detach"], "Should count 1 detach event")
}
