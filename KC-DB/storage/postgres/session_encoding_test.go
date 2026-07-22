package postgres

import (
	"encoding/binary"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

// checkDockerAvailable skips the test if Docker daemon is unavailable
func checkDockerAvailable(t *testing.T) {
	t.Helper()

	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		t.Skipf("Skipping integration test - Docker unavailable: %v", err)
	}
	_ = provider.Close()
}

// setupSessionEncodingTestDB connects to the test database using testcontainers
func setupSessionEncodingTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// bsEUIToBytes converts a uint64 BS EUI to an 8-byte big-endian slice for BYTEA columns.
// The pq driver cannot serialize uint64 values with the high bit set directly to BYTEA.
func bsEUIToBytes(eui uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, eui)
	return b
}

// createTestBaseStation creates a base station for testing
func createTestBaseStation(t *testing.T, db *sqlx.DB, id int64, eui uint64, tenantID int64, name string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO basestations (id, bs_eui, name, tenant_id, is_online, connection_type, service_center_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, false, 'bssci', 'tls://localhost:5000', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, id, bsEUIToBytes(eui), name, tenantID)
	require.NoError(t, err)
}

// createTestOrganization creates an organization for testing
func createTestOrganization(t *testing.T, db *sqlx.DB, orgID uuid.UUID, tenantID int64, name string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())
		ON CONFLICT (org_id) DO NOTHING
	`, orgID, tenantID, name)
	require.NoError(t, err)
}

// cleanupSessionTestData removes test sessions by connection_id pattern
func cleanupSessionTestData(t *testing.T, db *sqlx.DB, connectionIDPattern string) {
	_, err := db.Exec("DELETE FROM basestation_sessions WHERE connection_id LIKE $1", connectionIDPattern)
	if err != nil {
		t.Logf("Warning: cleanup failed: %v", err)
	}
}

// stringPtr returns a pointer to the given string (helper for test setup)
func stringPtr(s string) *string {
	return &s
}

// uuidPtr returns a pointer to the given UUID (helper for test setup)
func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

// TestCreateSession_WithJSONEncoding verifies JSON encoding is persisted
func TestCreateSession_WithJSONEncoding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create prerequisites
	orgID := uuid.New()
	createTestTenant(t, db, 100, "TestTenant100")
	createTestOrganization(t, db, orgID, 100, "TestOrg")
	createTestBaseStation(t, db, 1, 0x0102030405060708, 100, "TestBS-JSON")

	cleanupSessionTestData(t, db, "TestJSON%")
	defer cleanupSessionTestData(t, db, "TestJSON%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	// Create session with explicit JSON encoding
	snBsUUID := uuid.New()
	snScUUID := uuid.New()

	var snBsBytes, snScBytes [16]byte
	copy(snBsBytes[:], snBsUUID[:])
	copy(snScBytes[:], snScUUID[:])

	req := &models.BaseStationSessionCreateRequest{
		BaseStationID:  1,
		TenantID:       100,
		SnBsUuid:       snBsBytes,
		SnScUuid:       snScBytes,
		ConnectionId:   stringPtr("TestJSON-Conn-001"),
		RemoteAddr:     stringPtr("192.168.1.100"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingJSON,
	}

	session, err := repo.CreateSession(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Verify encoding was persisted
	assert.Equal(t, bssci.EncodingJSON, session.Encoding, "Session should have JSON encoding")

	// Verify by re-reading from database
	retrieved, err := repo.GetSessionByID(ctx, 100, session.ID)
	require.NoError(t, err)
	assert.Equal(t, bssci.EncodingJSON, retrieved.Encoding, "Retrieved session should preserve JSON encoding")
}

// TestCreateSession_WithMessagePackEncoding verifies MessagePack encoding is persisted
func TestCreateSession_WithMessagePackEncoding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create prerequisites
	orgID := uuid.New()
	createTestTenant(t, db, 101, "TestTenant101")
	createTestOrganization(t, db, orgID, 101, "TestOrg")
	createTestBaseStation(t, db, 2, 0x0102030405060709, 101, "TestBS-MsgPack")

	cleanupSessionTestData(t, db, "TestMsgPack%")
	defer cleanupSessionTestData(t, db, "TestMsgPack%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	// Create session with explicit MessagePack encoding
	snBsUUID := uuid.New()
	snScUUID := uuid.New()

	var snBsBytes, snScBytes [16]byte
	copy(snBsBytes[:], snBsUUID[:])
	copy(snScBytes[:], snScUUID[:])

	req := &models.BaseStationSessionCreateRequest{
		BaseStationID:  2,
		TenantID:       101,
		SnBsUuid:       snBsBytes,
		SnScUuid:       snScBytes,
		ConnectionId:   stringPtr("TestMsgPack-Conn-002"),
		RemoteAddr:     stringPtr("192.168.1.101"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingMessagePack,
	}

	session, err := repo.CreateSession(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Verify encoding was persisted
	assert.Equal(t, bssci.EncodingMessagePack, session.Encoding, "Session should have MessagePack encoding")

	// Verify by re-reading from database
	retrieved, err := repo.GetSessionByID(ctx, 101, session.ID)
	require.NoError(t, err)
	assert.Equal(t, bssci.EncodingMessagePack, retrieved.Encoding, "Retrieved session should preserve MessagePack encoding")
}

// TestCreateSession_WithEmptyEncoding verifies default MessagePack encoding
func TestCreateSession_WithEmptyEncoding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create prerequisites
	orgID := uuid.New()
	createTestTenant(t, db, 102, "TestTenant102")
	createTestOrganization(t, db, orgID, 102, "TestOrg")
	createTestBaseStation(t, db, 3, 0x010203040506070A, 102, "TestBS-Default")

	cleanupSessionTestData(t, db, "TestDefault%")
	defer cleanupSessionTestData(t, db, "TestDefault%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	// Create session with empty encoding (should default to msgpack)
	snBsUUID := uuid.New()
	snScUUID := uuid.New()

	var snBsBytes, snScBytes [16]byte
	copy(snBsBytes[:], snBsUUID[:])
	copy(snScBytes[:], snScUUID[:])

	req := &models.BaseStationSessionCreateRequest{
		BaseStationID:  3,
		TenantID:       102,
		SnBsUuid:       snBsBytes,
		SnScUuid:       snScBytes,
		ConnectionId:   stringPtr("TestDefault-Conn-003"),
		RemoteAddr:     stringPtr("192.168.1.102"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       "", // Empty - should default to msgpack
	}

	session, err := repo.CreateSession(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Verify defaults to MessagePack per BSSCI Section 1
	assert.Equal(t, bssci.EncodingMessagePack, session.Encoding, "Empty encoding should default to MessagePack")

	// Verify by re-reading from database
	retrieved, err := repo.GetSessionByID(ctx, 102, session.ID)
	require.NoError(t, err)
	assert.Equal(t, bssci.EncodingMessagePack, retrieved.Encoding, "Retrieved session should have default MessagePack encoding")
}

// TestUpdateEncoding_PersistsChange verifies encoding updates are persisted
func TestUpdateEncoding_PersistsChange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create prerequisites
	orgID := uuid.New()
	createTestTenant(t, db, 103, "TestTenant103")
	createTestOrganization(t, db, orgID, 103, "TestOrg")
	createTestBaseStation(t, db, 4, 0x010203040506070B, 103, "TestBS-Update")

	cleanupSessionTestData(t, db, "TestUpdate%")
	defer cleanupSessionTestData(t, db, "TestUpdate%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	// Create session with MessagePack encoding
	snBsUUID := uuid.New()
	snScUUID := uuid.New()

	var snBsBytes, snScBytes [16]byte
	copy(snBsBytes[:], snBsUUID[:])
	copy(snScBytes[:], snScUUID[:])

	req := &models.BaseStationSessionCreateRequest{
		BaseStationID:  4,
		TenantID:       103,
		SnBsUuid:       snBsBytes,
		SnScUuid:       snScBytes,
		ConnectionId:   stringPtr("TestUpdate-Conn-004"),
		RemoteAddr:     stringPtr("192.168.1.103"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingMessagePack,
	}

	session, err := repo.CreateSession(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, bssci.EncodingMessagePack, session.Encoding)

	// Update encoding to JSON
	err = repo.UpdateEncoding(ctx, session.TenantID, session.ID, bssci.EncodingJSON)
	require.NoError(t, err)

	// Verify update was persisted
	retrieved, err := repo.GetSessionByID(ctx, 103, session.ID)
	require.NoError(t, err)
	assert.Equal(t, bssci.EncodingJSON, retrieved.Encoding, "Encoding update should be persisted")
}

// TestUpdateSession_WithEncodingField verifies UpdateSession handles encoding changes
func TestUpdateSession_WithEncodingField(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create prerequisites
	orgID := uuid.New()
	createTestTenant(t, db, 104, "TestTenant104")
	createTestOrganization(t, db, orgID, 104, "TestOrg")
	createTestBaseStation(t, db, 5, 0x010203040506070C, 104, "TestBS-UpdateSession")

	cleanupSessionTestData(t, db, "TestUpdateSess%")
	defer cleanupSessionTestData(t, db, "TestUpdateSess%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	// Create session with JSON encoding
	snBsUUID := uuid.New()
	snScUUID := uuid.New()

	var snBsBytes, snScBytes [16]byte
	copy(snBsBytes[:], snBsUUID[:])
	copy(snScBytes[:], snScUUID[:])

	req := &models.BaseStationSessionCreateRequest{
		BaseStationID:  5,
		TenantID:       104,
		SnBsUuid:       snBsBytes,
		SnScUuid:       snScBytes,
		ConnectionId:   stringPtr("TestUpdateSess-Conn-005"),
		RemoteAddr:     stringPtr("192.168.1.104"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingJSON,
	}

	session, err := repo.CreateSession(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, bssci.EncodingJSON, session.Encoding)

	// Update session with encoding change
	newEncoding := bssci.EncodingMessagePack
	updateReq := &models.BaseStationSessionUpdateRequest{
		Encoding: &newEncoding,
	}

	err = repo.UpdateSession(ctx, 104, session.ID, updateReq)
	require.NoError(t, err)

	// Verify encoding was updated
	retrieved, err := repo.GetSessionByID(ctx, 104, session.ID)
	require.NoError(t, err)
	assert.Equal(t, bssci.EncodingMessagePack, retrieved.Encoding, "UpdateSession should persist encoding change")
}

// TestListSessions_IncludesEncoding verifies ListSessions returns encoding field
func TestListSessions_IncludesEncoding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create prerequisites
	orgID := uuid.New()
	createTestTenant(t, db, 105, "TestTenant105")
	createTestOrganization(t, db, orgID, 105, "TestOrg")
	// Create two different base stations for the two sessions
	// (unique constraint allows only one active session per base station)
	createTestBaseStation(t, db, 6, 0x010203040506070D, 105, "TestBS-List-JSON")
	createTestBaseStation(t, db, 7, 0x010203040506070E, 105, "TestBS-List-MsgPack")

	cleanupSessionTestData(t, db, "TestList%")
	defer cleanupSessionTestData(t, db, "TestList%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	// Create two sessions with different encodings

	snBsUUID1 := uuid.New()
	snScUUID1 := uuid.New()
	var snBsBytes1, snScBytes1 [16]byte
	copy(snBsBytes1[:], snBsUUID1[:])
	copy(snScBytes1[:], snScUUID1[:])

	req1 := &models.BaseStationSessionCreateRequest{
		BaseStationID:  6,
		TenantID:       105,
		SnBsUuid:       snBsBytes1,
		SnScUuid:       snScBytes1,
		ConnectionId:   stringPtr("TestList-Conn-JSON"),
		RemoteAddr:     stringPtr("192.168.1.105"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingJSON,
	}

	snBsUUID2 := uuid.New()
	snScUUID2 := uuid.New()
	var snBsBytes2, snScBytes2 [16]byte
	copy(snBsBytes2[:], snBsUUID2[:])
	copy(snScBytes2[:], snScUUID2[:])

	req2 := &models.BaseStationSessionCreateRequest{
		BaseStationID:  7, // Use different base station to satisfy unique active session constraint
		TenantID:       105,
		SnBsUuid:       snBsBytes2,
		SnScUuid:       snScBytes2,
		ConnectionId:   stringPtr("TestList-Conn-MsgPack"),
		RemoteAddr:     stringPtr("192.168.1.106"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingMessagePack,
	}

	session1, err := repo.CreateSession(ctx, req1)
	require.NoError(t, err)

	session2, err := repo.CreateSession(ctx, req2)
	require.NoError(t, err)

	// List sessions and verify encoding is included
	filter := &models.BaseStationSessionFilter{
		TenantID:   105,
		Status:     []models.BaseStationSessionStatus{models.SessionStatusActive},
		ActiveOnly: true,
		Limit:      100,
	}

	sessions, total, err := repo.ListSessions(ctx, filter)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(2), "Should have at least 2 sessions")
	require.GreaterOrEqual(t, len(sessions), 2, "Should have at least 2 sessions")

	// Find our test sessions and verify encodings
	var foundJSON, foundMsgPack bool
	for _, s := range sessions {
		if s.ID == session1.ID {
			assert.Equal(t, bssci.EncodingJSON, s.Encoding, "JSON session should preserve encoding in list")
			foundJSON = true
		}
		if s.ID == session2.ID {
			assert.Equal(t, bssci.EncodingMessagePack, s.Encoding, "MessagePack session should preserve encoding in list")
			foundMsgPack = true
		}
	}

	assert.True(t, foundJSON, "Should find JSON-encoded session in list")
	assert.True(t, foundMsgPack, "Should find MessagePack-encoded session in list")
}

// TestGetActiveSessionByBaseStation_IncludesEncoding verifies encoding in active session lookup
func TestGetActiveSessionByBaseStation_IncludesEncoding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create prerequisites
	orgID := uuid.New()
	createTestTenant(t, db, 106, "TestTenant106")
	createTestOrganization(t, db, orgID, 106, "TestOrg")
	createTestBaseStation(t, db, 7, 0x010203040506070E, 106, "TestBS-Active")

	cleanupSessionTestData(t, db, "TestActive%")
	defer cleanupSessionTestData(t, db, "TestActive%")

	repo := NewBaseStationSessionRepository(db)
	ctx := testutil.TestContext()

	// Create active session with JSON encoding
	snBsUUID := uuid.New()
	snScUUID := uuid.New()

	var snBsBytes, snScBytes [16]byte
	copy(snBsBytes[:], snBsUUID[:])
	copy(snScBytes[:], snScUUID[:])

	req := &models.BaseStationSessionCreateRequest{
		BaseStationID:  7,
		TenantID:       106,
		SnBsUuid:       snBsBytes,
		SnScUuid:       snScBytes,
		ConnectionId:   stringPtr("TestActive-Conn-007"),
		RemoteAddr:     stringPtr("192.168.1.107"),
		CanResume:      true,
		OrganizationID: uuidPtr(orgID),
		Encoding:       bssci.EncodingJSON,
	}

	created, err := repo.CreateSession(ctx, req)
	require.NoError(t, err)

	// Retrieve active session by base station (tenantID, baseStationID)
	session, err := repo.GetActiveSessionByBaseStation(ctx, 106, 7)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Verify encoding is included
	assert.Equal(t, created.ID, session.ID)
	assert.Equal(t, bssci.EncodingJSON, session.Encoding, "Active session lookup should include encoding")
}
