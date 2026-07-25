package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// setupSystemEventsTestDB creates a test database with testcontainers
func setupSystemEventsTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// createSystemEventsTestTenant inserts a tenant for system events tests
func createSystemEventsTestTenant(t *testing.T, db *sqlx.DB, id int64, name string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO tenants (id, name, created_at, updated_at)
                       VALUES ($1, $2, NOW(), NOW()) ON CONFLICT (id) DO NOTHING`, id, name)
	require.NoError(t, err, "Failed to create test tenant")
}

// insertTestEvent inserts a test system event directly into the database
func insertTestEvent(t *testing.T, db *sqlx.DB, tenantID int64, eventType, category, title string, occurredAt time.Time, data map[string]interface{}) {
	t.Helper()

	dataJSON, err := json.Marshal(data)
	require.NoError(t, err, "Failed to marshal event data")

	_, err = db.Exec(`
		INSERT INTO system_events (
			tenant_id, event_type, event_category, severity,
			source_type, source_name, title, description,
			data, status, occurred_at, recorded_at
		) VALUES ($1, $2, $3, 'info', 'service_center', 'test', $4, 'Test event', $5, 'new', $6, NOW())
	`, tenantID, eventType, category, title, dataJSON, occurredAt)
	require.NoError(t, err, "Failed to insert test event")
}

// TestGetEvents_TenantIsolation verifies that events are filtered by tenant_id
func TestGetEvents_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantA = int64(100)
	const tenantB = int64(200)

	createSystemEventsTestTenant(t, db, tenantA, "TenantA")
	createSystemEventsTestTenant(t, db, tenantB, "TenantB")

	now := time.Now()

	// Insert events for both tenants
	insertTestEvent(t, db, tenantA, "endpoint.attached", "endpoint", "EP Attached A1", now, nil)
	insertTestEvent(t, db, tenantA, "endpoint.attached", "endpoint", "EP Attached A2", now.Add(-1*time.Second), nil)
	insertTestEvent(t, db, tenantB, "endpoint.attached", "endpoint", "EP Attached B1", now, nil)

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	// Query for tenant A only
	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "100",
		Limit:    10,
	})
	require.NoError(t, err)
	assert.Len(t, events, 2, "Should return exactly 2 events for tenant A")

	// Verify all returned events belong to tenant A
	for _, e := range events {
		assert.Equal(t, "100", e.TenantID, "All events should belong to tenant A")
	}

	// Query for tenant B only
	eventsB, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "200",
		Limit:    10,
	})
	require.NoError(t, err)
	assert.Len(t, eventsB, 1, "Should return exactly 1 event for tenant B")
	assert.Equal(t, "200", eventsB[0].TenantID, "Event should belong to tenant B")
}

// TestGetEvents_SinceFilter verifies that the Since filter works correctly
func TestGetEvents_SinceFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantID = int64(300)
	createSystemEventsTestTenant(t, db, tenantID, "SinceFilterTest")

	now := time.Now()

	// Insert events at different times
	insertTestEvent(t, db, tenantID, "event.old", "system", "Old Event", now.Add(-60*time.Second), nil)
	insertTestEvent(t, db, tenantID, "event.medium", "system", "Medium Event", now.Add(-30*time.Second), nil)
	insertTestEvent(t, db, tenantID, "event.recent", "system", "Recent Event", now.Add(-10*time.Second), nil)

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	// Query events since 20 seconds ago
	since := now.Add(-20 * time.Second)
	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "300",
		Since:    &since,
		Limit:    10,
	})
	require.NoError(t, err)
	assert.Len(t, events, 1, "Should return only the recent event")
	assert.Equal(t, "Recent Event", events[0].Title)
}

// TestGetEvents_LimitFilter verifies that the Limit filter works correctly
func TestGetEvents_LimitFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantID = int64(400)
	createSystemEventsTestTenant(t, db, tenantID, "LimitFilterTest")

	now := time.Now()

	// Insert 10 events
	for i := 0; i < 10; i++ {
		insertTestEvent(t, db, tenantID, "event.test", "system", "Event", now.Add(-time.Duration(i)*time.Second), nil)
	}

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	// Query with limit of 3
	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "400",
		Limit:    3,
	})
	require.NoError(t, err)
	assert.Len(t, events, 3, "Should return exactly 3 events")
}

// TestGetEvents_CombinedFilters verifies multiple filters work together
func TestGetEvents_CombinedFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantID = int64(500)
	createSystemEventsTestTenant(t, db, tenantID, "CombinedFilterTest")

	now := time.Now()

	// Insert events with different categories
	insertTestEvent(t, db, tenantID, "event.old", "endpoint", "Old Endpoint", now.Add(-60*time.Second), nil)
	insertTestEvent(t, db, tenantID, "event.recent", "endpoint", "Recent Endpoint", now.Add(-10*time.Second), nil)
	insertTestEvent(t, db, tenantID, "event.recent", "system", "Recent System", now.Add(-5*time.Second), nil)

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	// Query: tenant + since + category
	since := now.Add(-20 * time.Second)
	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID:   "500",
		Since:      &since,
		Categories: []string{"endpoint"},
		Limit:      10,
	})
	require.NoError(t, err)
	assert.Len(t, events, 1, "Should return only recent endpoint event")
	assert.Equal(t, "Recent Endpoint", events[0].Title)
}

// TestGetEvents_EmptyResult verifies empty result handling
func TestGetEvents_EmptyResult(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantID = int64(600)
	createSystemEventsTestTenant(t, db, tenantID, "EmptyResultTest")

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	// Query for non-existent tenant
	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "999999",
		Limit:    10,
	})
	require.NoError(t, err, "Should not error on empty result")
	assert.Empty(t, events, "Should return empty slice, not nil")
}

// TestGetEvents_InvalidTenantID verifies error handling for invalid tenant ID
func TestGetEvents_InvalidTenantID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	// Query with invalid tenant ID (non-numeric)
	_, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "not-a-number",
		Limit:    10,
	})
	assert.Error(t, err, "Should error on invalid tenant ID format")
	assert.Contains(t, err.Error(), "invalid tenant ID format")
}

// TestGetEvents_JSONBDataUnmarshal verifies JSONB data is correctly returned
func TestGetEvents_JSONBDataUnmarshal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantID = int64(700)
	createSystemEventsTestTenant(t, db, tenantID, "JSONBTest")

	now := time.Now()

	// Insert event with complex JSONB data
	testData := map[string]interface{}{
		"epEui":     "0102030405060708",
		"shAddr":    uint32(1234),
		"success":   true,
		"timestamp": now.Unix(),
	}
	insertTestEvent(t, db, tenantID, "endpoint.attached", "endpoint", "EP Attached", now, testData)

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "700",
		Limit:    10,
	})
	require.NoError(t, err)
	require.Len(t, events, 1)

	// Verify Details field contains the JSONB data
	assert.NotNil(t, events[0].Details, "Details should be populated")

	var data map[string]interface{}
	err = json.Unmarshal(events[0].Details, &data)
	require.NoError(t, err, "Should unmarshal Details to map")
	assert.Equal(t, "0102030405060708", data["epEui"])
	assert.Equal(t, true, data["success"])
}

// TestGetEvents_OrderByDefault verifies default ordering is occurred_at DESC
func TestGetEvents_OrderByDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantID = int64(800)
	createSystemEventsTestTenant(t, db, tenantID, "OrderByTest")

	now := time.Now()

	// Insert events in reverse chronological order
	insertTestEvent(t, db, tenantID, "event.oldest", "system", "Oldest", now.Add(-3*time.Second), nil)
	insertTestEvent(t, db, tenantID, "event.middle", "system", "Middle", now.Add(-2*time.Second), nil)
	insertTestEvent(t, db, tenantID, "event.newest", "system", "Newest", now.Add(-1*time.Second), nil)

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "800",
		Limit:    10,
	})
	require.NoError(t, err)
	require.Len(t, events, 3)

	// Verify order is DESC (newest first)
	assert.Equal(t, "Newest", events[0].Title)
	assert.Equal(t, "Middle", events[1].Title)
	assert.Equal(t, "Oldest", events[2].Title)
}

// TestGetEvents_CreateEventRoundtrip verifies CreateEvent and GetEvents work together
func TestGetEvents_CreateEventRoundtrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantID = int64(900)
	createSystemEventsTestTenant(t, db, tenantID, "RoundtripTest")

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	// Create event using the store's CreateEvent method
	now := time.Now()
	event := &models.SystemEvent{
		TenantID:    "900",
		EventType:   "test.roundtrip",
		Category:    "system",
		Severity:    "info",
		SourceType:  "service_center",
		SourceName:  "test",
		Title:       "Roundtrip Test Event",
		Description: "Testing CreateEvent and GetEvents together",
		Details:     json.RawMessage(`{"testKey": "testValue"}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := store.CreateEvent(ctx, event)
	require.NoError(t, err, "CreateEvent should succeed")

	// Retrieve the event
	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID: "900",
		Limit:    10,
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Roundtrip Test Event", events[0].Title)
}

// TestGetEvents_MultipleCategories_IncludesProtocol verifies that category filtering
// works correctly with protocol categories (scaci, bssci, protocol)
func TestGetEvents_MultipleCategories_IncludesProtocol(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()

	const tenantID = int64(950)
	createSystemEventsTestTenant(t, db, tenantID, "ProtocolCategoryTest")

	now := time.Now()

	// Insert events with different categories including protocol categories
	insertTestEvent(t, db, tenantID, "test.scaci", models.EventCategorySCACI, "SCACI Event", now, nil)
	insertTestEvent(t, db, tenantID, "test.bssci", models.EventCategoryBSSCI, "BSSCI Event", now.Add(-1*time.Second), nil)
	insertTestEvent(t, db, tenantID, "test.protocol", models.EventCategoryProtocol, "Protocol Event", now.Add(-2*time.Second), nil)
	insertTestEvent(t, db, tenantID, "test.message", models.EventCategoryMessage, "Message Event", now.Add(-3*time.Second), nil)

	store := NewSystemEventStore(db.DB)
	ctx, cancel := context.WithTimeout(testutil.TestContext(), 5*time.Second)
	defer cancel()

	// Filter by protocol categories only
	events, err := store.GetEvents(ctx, interfaces.SystemEventFilter{
		TenantID:   "950",
		Categories: []string{models.EventCategorySCACI, models.EventCategoryBSSCI, models.EventCategoryProtocol},
		Limit:      10,
	})
	require.NoError(t, err)
	assert.Len(t, events, 3, "Should return 3 events matching protocol categories")

	// Verify returned categories
	categories := make(map[string]bool)
	for _, e := range events {
		categories[e.Category] = true
	}
	assert.True(t, categories[models.EventCategorySCACI], "Should include scaci event")
	assert.True(t, categories[models.EventCategoryBSSCI], "Should include bssci event")
	assert.True(t, categories[models.EventCategoryProtocol], "Should include protocol event")
	assert.False(t, categories[models.EventCategoryMessage], "Should not include message event")
}
