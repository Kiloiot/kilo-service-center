package postgres

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test lazy migration from base64-encoded GCM to raw GCM format in GetActive()
func TestGetActive_LazyMigration(t *testing.T) {
	// Skip if no test database available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Initialize logger
	logger.Initialize("error", "json")
	log := logger.Get()

	// Create key encryptor
	keyEncryptor, err := crypto.NewKeyEncryptor()
	require.NoError(t, err, "Failed to create key encryptor")

	// Create test endpoint and session with legacy base64-encoded key
	tenantID := int64(1001)
	endpointID := int64(2001)
	sessionID := uuid.NewString()

	// Create a test session key (16 bytes for testing)
	cleartextKey := []byte("test-session-key")

	// Encrypt in legacy base64 format (60 bytes)
	legacyEncrypted, err := keyEncryptor.EncryptKey(cleartextKey)
	require.NoError(t, err, "Failed to encrypt key in legacy format")
	require.Equal(t, 60, len(legacyEncrypted), "Legacy format key should be 60 bytes (base64 encoded)")

	// Verify it's base64 encoded
	_, err = base64.StdEncoding.DecodeString(string(legacyEncrypted))
	require.NoError(t, err, "Legacy format key should be valid base64")

	// Insert endpoint
	insertEndpointWithConn(ctx, t, db.conn, EndpointInsertParams{
		ID:       endpointID,
		EpEUI:    0x0102030405060708,
		Name:     "LazyMigrationEP",
		TenantID: tenantID,
		ShAddr:   0x1234,
		Bidi:     true,
	})

	// Insert session with legacy format key
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO endpoint_sessions (
			endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, endpointID, tenantID, sessionID, legacyEncrypted, 1, "active", time.Now(), time.Now())
	require.NoError(t, err, "Failed to insert test session with legacy format key")

	// Create repository WITH encryption support
	repo := NewEndPointSessionRepositoryWithEncryption(db, keyEncryptor, log)

	// Call GetActive() - should trigger lazy migration
	session, err := repo.GetActive(ctx, fmt.Sprintf("%d", endpointID))
	require.NoError(t, err, "GetActive should not fail")
	require.NotNil(t, session, "Session should be found")

	// Verify session data
	assert.Equal(t, sessionID, session.SessionID, "Session ID should match")
	assert.Equal(t, tenantID, session.TenantID, "Tenant ID should match")

	// Verify key was migrated (should now be 44 bytes in DB)
	var storedKey []byte
	err = db.conn.QueryRowContext(ctx, `
		SELECT session_key FROM endpoint_sessions
		WHERE tenant_id = $1 AND session_id = $2
	`, tenantID, sessionID).Scan(&storedKey)
	require.NoError(t, err, "Failed to query stored key")

	// Key should now be raw GCM (44 bytes)
	assert.Equal(t, 44, len(storedKey), "Stored key should be 44 bytes (raw GCM) after migration")

	// Verify it's no longer base64 encoded
	_, err = base64.StdEncoding.DecodeString(string(storedKey))
	assert.Error(t, err, "Migrated key should NOT be valid base64")

	// Verify the migrated key can be decrypted to the original cleartext
	decrypted, wasLegacy, err := keyEncryptor.DecryptKeyWithMigration(storedKey)
	require.NoError(t, err, "Migrated key should decrypt successfully")
	assert.False(t, wasLegacy, "Migrated key should be detected as raw GCM")
	assert.Equal(t, cleartextKey, decrypted, "Decrypted key should match original")
}

// Test that already-migrated keys are not re-migrated
func TestGetActive_NoMigrationWhenAlreadyRaw(t *testing.T) {
	// Skip if no test database available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Initialize logger
	logger.Initialize("error", "json")
	log := logger.Get()

	// Create key encryptor
	keyEncryptor, err := crypto.NewKeyEncryptor()
	require.NoError(t, err, "Failed to create key encryptor")

	// Create test endpoint and session with RAW GCM key
	tenantID := int64(1002)
	endpointID := int64(2002)
	sessionID := uuid.NewString()

	// Create a test session key
	cleartextKey := []byte("test-session-key")

	// Encrypt in RAW GCM format (44 bytes)
	rawEncrypted, err := keyEncryptor.EncryptKeyRaw(cleartextKey)
	require.NoError(t, err, "Failed to encrypt key in raw format")
	require.Equal(t, 44, len(rawEncrypted), "Raw key should be 44 bytes")

	// Insert endpoint
	insertEndpointWithConn(ctx, t, db.conn, EndpointInsertParams{
		ID:       endpointID,
		EpEUI:    0x0102030405060709,
		Name:     "NoMigrationEP",
		TenantID: tenantID,
		ShAddr:   0x1235,
		Bidi:     true,
	})

	// Insert session with raw key
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO endpoint_sessions (
			endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, endpointID, tenantID, sessionID, rawEncrypted, 1, "active", time.Now(), time.Now())
	require.NoError(t, err, "Failed to insert test session with raw key")

	// Store original key bytes for comparison
	originalKey := make([]byte, len(rawEncrypted))
	copy(originalKey, rawEncrypted)

	// Create repository WITH encryption support
	repo := NewEndPointSessionRepositoryWithEncryption(db, keyEncryptor, log)

	// Call GetActive() - should NOT trigger migration
	session, err := repo.GetActive(ctx, fmt.Sprintf("%d", endpointID))
	require.NoError(t, err, "GetActive should not fail")
	require.NotNil(t, session, "Session should be found")

	// Verify key was NOT modified (still 44 bytes, same content)
	var storedKey []byte
	err = db.conn.QueryRowContext(ctx, `
		SELECT session_key FROM endpoint_sessions
		WHERE tenant_id = $1 AND session_id = $2
	`, tenantID, sessionID).Scan(&storedKey)
	require.NoError(t, err, "Failed to query stored key")

	assert.Equal(t, 44, len(storedKey), "Stored key should still be 44 bytes")
	assert.Equal(t, originalKey, storedKey, "Key should not be modified when already raw")
}

// Test tenant isolation during migration
func TestGetActive_TenantIsolationInMigration(t *testing.T) {
	// Skip if no test database available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Initialize logger
	logger.Initialize("error", "json")
	log := logger.Get()

	// Create key encryptor
	keyEncryptor, err := crypto.NewKeyEncryptor()
	require.NoError(t, err, "Failed to create key encryptor")

	// Create sessions for two different tenants with same endpoint/session IDs
	tenant1ID := int64(1003)
	tenant2ID := int64(1004)
	endpointID := int64(2003)
	sessionID := uuid.NewString()

	// Create keys
	key1 := []byte("tenant1-sess-key")
	key2 := []byte("tenant2-sess-key")

	// Encrypt both in legacy format
	legacy1, err := keyEncryptor.EncryptKey(key1)
	require.NoError(t, err)
	legacy2, err := keyEncryptor.EncryptKey(key2)
	require.NoError(t, err)

	// Insert endpoints for both tenants
	for _, tid := range []int64{tenant1ID, tenant2ID} {
		insertEndpointWithConn(ctx, t, db.conn, EndpointInsertParams{
			ID:       endpointID + tid,
			EpEUI:    uint64(0x0102030405060700 + tid),
			Name:     fmt.Sprintf("TenantIsolationEP_%d", tid),
			TenantID: tid,
			ShAddr:   0x1240,
			Bidi:     true,
		})
	}

	// Insert sessions with legacy format keys
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO endpoint_sessions (
			endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, endpointID+tenant1ID, tenant1ID, sessionID, legacy1, 1, "active", time.Now(), time.Now())
	require.NoError(t, err)

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO endpoint_sessions (
			endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, endpointID+tenant2ID, tenant2ID, sessionID, legacy2, 1, "active", time.Now(), time.Now())
	require.NoError(t, err)

	// Create repository
	repo := NewEndPointSessionRepositoryWithEncryption(db, keyEncryptor, log)

	// Migrate tenant1's session
	session1, err := repo.GetActive(ctx, fmt.Sprintf("%d", endpointID+tenant1ID))
	require.NoError(t, err)
	require.NotNil(t, session1)

	// Verify tenant1's key was migrated
	var key1Stored []byte
	err = db.conn.QueryRowContext(ctx, `
		SELECT session_key FROM endpoint_sessions
		WHERE tenant_id = $1 AND session_id = $2
	`, tenant1ID, sessionID).Scan(&key1Stored)
	require.NoError(t, err)
	assert.Equal(t, 44, len(key1Stored), "Tenant 1 key should be migrated")

	// Verify tenant2's key was NOT affected
	var key2Stored []byte
	err = db.conn.QueryRowContext(ctx, `
		SELECT session_key FROM endpoint_sessions
		WHERE tenant_id = $1 AND session_id = $2
	`, tenant2ID, sessionID).Scan(&key2Stored)
	require.NoError(t, err)
	assert.Equal(t, 60, len(key2Stored), "Tenant 2 key should still be legacy format")

	// Verify decryption produces correct original keys
	decrypted1, _, err := keyEncryptor.DecryptKeyWithMigration(key1Stored)
	require.NoError(t, err)
	assert.Equal(t, key1, decrypted1, "Tenant 1 key should decrypt to original")

	decrypted2, _, err := keyEncryptor.DecryptKeyWithMigration(key2Stored)
	require.NoError(t, err)
	assert.Equal(t, key2, decrypted2, "Tenant 2 key should decrypt to original")
}

// Helper functions for test database setup
func setupTestDB(t *testing.T) *DB {
	t.Helper()

	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	// Seed the tenant rows referenced by the lazy-migration tests
	for id, name := range map[int64]string{
		1001: "MigrationTenant1001",
		1002: "MigrationTenant1002",
		1003: "MigrationTenant1003",
		1004: "MigrationTenant1004",
	} {
		createTestTenant(t, sqlxDB, id, name)
	}

	logger.Initialize("error", "json")
	log := logger.Get()

	return &DB{
		conn:   sqlxDB.DB,
		sqlxDB: sqlxDB,
		log:    log,
	}
}

func cleanupTestDB(t *testing.T, db *DB) {
	t.Helper()

	// Clean up test data
	ctx := context.Background()

	// Delete test sessions
	_, err := db.conn.ExecContext(ctx, `
		DELETE FROM endpoint_sessions
		WHERE tenant_id >= 1000 AND tenant_id < 2000
	`)
	if err != nil {
		t.Logf("Warning: Failed to clean up test sessions: %v", err)
	}

	// Delete test endpoints
	_, err = db.conn.ExecContext(ctx, `
		DELETE FROM endpoints
		WHERE tenant_id >= 1000 AND tenant_id < 2000
	`)
	if err != nil {
		t.Logf("Warning: Failed to clean up test endpoints: %v", err)
	}

	if err := db.conn.Close(); err != nil {
		t.Fatalf("failed to close DB: %v", err)
	}
}
