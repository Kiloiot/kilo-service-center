package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/stretchr/testify/require"
)

// SetupTestDB creates a test database using testcontainers.
// For environment-driven database connections, use SetupEnvDBOrSkip instead.
func SetupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// CleanupTestData removes test data matching the given pattern from a table
// Use this in defer statements to ensure test isolation
//
// Example:
//
//	defer CleanupTestData(t, db, "endpoints", "name", "TestCreate%")
func CleanupTestData(t *testing.T, db *sqlx.DB, table, column, pattern string) {
	t.Helper()

	// Use LIKE for patterns with wildcards, = for exact matches
	// This avoids "operator does not exist: bigint ~~ unknown" warnings on integer columns
	var query string
	var operator string
	if containsWildcards(pattern) {
		operator = "LIKE"
	} else {
		operator = "="
	}

	query = fmt.Sprintf("DELETE FROM %s WHERE %s %s $1", table, column, operator)
	_, err := db.Exec(query, pattern)
	if err != nil {
		t.Logf("Warning: cleanup failed for %s.%s %s %s: %v", table, column, operator, pattern, err)
	}
}

// SetupEnvDBOrSkip connects to a running database via environment variables.
// When requireDB is true, the test fails if the DB is unreachable (CI behavior).
// When requireDB is false, the test skips if the DB is unreachable (local behavior).
// Environment variables: TEST_DB_HOST, TEST_DB_PORT, TEST_DB_USER, TEST_DB_PASSWORD,
// TEST_DB_NAME, TEST_DB_SSLMODE (falls back to DB_HOST, DB_PORT, etc.).
func SetupEnvDBOrSkip(t *testing.T, requireDB bool) *sqlx.DB {
	t.Helper()

	host := envOrDefault("TEST_DB_HOST", os.Getenv("DB_HOST"))
	if host == "" {
		host = "localhost"
	}
	port := envOrDefault("TEST_DB_PORT", envOrDefault("DB_PORT", "5433"))
	user := envOrDefault("TEST_DB_USER", envOrDefault("DB_USER", "kilocenter"))
	pass := envOrDefault("TEST_DB_PASSWORD", envOrDefault("DB_PASSWORD", "changeme"))
	dbname := envOrDefault("TEST_DB_NAME", envOrDefault("DB_NAME", "kilocenter"))
	sslmode := envOrDefault("TEST_DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, dbname, sslmode)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		if requireDB {
			t.Fatalf("Database required but unreachable: %v", err)
		}
		t.Skipf("Skipping: database not reachable (%v)", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// containsWildcards checks if a pattern contains SQL wildcard characters
func containsWildcards(pattern string) bool {
	for _, c := range pattern {
		if c == '%' || c == '_' {
			return true
		}
	}
	return false
}

// createTestTenant creates a tenant for testing purposes
// This is a shared helper to avoid duplicating tenant creation SQL across test files
// Use this in test setup functions to ensure required tenants exist
//
// Example:
//
//	createTestTenant(t, db, 100, "Test Tenant")
func createTestTenant(t *testing.T, db *sqlx.DB, id int64, name string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, id, name, "Test tenant for "+name)
	require.NoError(t, err)
}

// EndpointInsertParams contains parameters for inserting a test endpoint.
// Use with insertEndpoint() or insertEndpointWithConn() for consistent test data.
type EndpointInsertParams struct {
	// Required fields
	EpEUI    uint64
	Name     string
	TenantID int64

	// Optional fields (with sensible defaults)
	ID            int64      // Explicit ID (0 = auto-generate)
	Description   string     // Empty string if not specified
	OwnerTenantID int64      // If 0, defaults to TenantID (roaming support)
	NwkKey        []byte     // If nil, defaults to 16 zero bytes
	AppKey        []byte     // If nil, defaults to 16 zero bytes
	CryptoMode    int        // Defaults to 0
	CreatedAt     *time.Time // If nil, uses NOW(); otherwise uses provided timestamp

	// Extended fields for specialized tests
	ShAddr    uint32 // Short address
	Bidi      bool   // Bidirectional flag
	Sign      []byte // Signature key (detach tests)
	Preshared []byte // Preshared key (detach tests)
}

// ptrTime returns a pointer to the given time value.
// Used for setting CreatedAt in EndpointInsertParams.
func ptrTime(t time.Time) *time.Time {
	return &t
}

// applyEndpointDefaults normalizes EndpointInsertParams with sensible defaults.
// This ensures owner_tenant_id defaults to tenant_id and keys default to 16 zero bytes.
func applyEndpointDefaults(p *EndpointInsertParams) {
	if p.OwnerTenantID == 0 {
		p.OwnerTenantID = p.TenantID
	}
	if p.NwkKey == nil {
		p.NwkKey = make([]byte, 16)
	}
	if p.AppKey == nil {
		p.AppKey = make([]byte, 16)
	}
}

// insertEndpoint inserts a test endpoint using *sqlx.DB (for testcontainer-based tests).
// This helper prevents test fixtures from omitting the required owner_tenant_id column.
//
// If OwnerTenantID is 0, it defaults to TenantID (standard non-roaming case).
// If ID is non-zero, uses explicit ID; otherwise auto-generates.
// Returns the inserted endpoint's ID.
//
// Example:
//
//	id := insertEndpoint(t, db, EndpointInsertParams{
//	    EpEUI:    0x0102030405060708,
//	    Name:     "TestEndpoint",
//	    TenantID: 100,
//	})
func insertEndpoint(t *testing.T, db *sqlx.DB, p EndpointInsertParams) int64 {
	t.Helper()
	applyEndpointDefaults(&p)

	// Use provided CreatedAt or current time
	createdAt := time.Now()
	if p.CreatedAt != nil {
		createdAt = *p.CreatedAt
	}

	if p.ID != 0 {
		// Explicit ID provided
		_, err := db.Exec(`
			INSERT INTO endpoints (
				id, ep_eui, name, description, tenant_id, owner_tenant_id,
				nwk_key, app_key, crypto_mode, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, p.ID, euiToBytes(p.EpEUI), p.Name, p.Description, p.TenantID, p.OwnerTenantID,
			p.NwkKey, p.AppKey, p.CryptoMode, createdAt, createdAt)
		require.NoError(t, err, "Failed to insert test endpoint with explicit ID")
		return p.ID
	}

	// Auto-generate ID
	var id int64
	err := db.QueryRow(`
		INSERT INTO endpoints (
			ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, euiToBytes(p.EpEUI), p.Name, p.Description, p.TenantID, p.OwnerTenantID,
		p.NwkKey, p.AppKey, p.CryptoMode, createdAt, createdAt).Scan(&id)
	require.NoError(t, err, "Failed to insert test endpoint")

	return id
}

// insertEndpointWithConn inserts a test endpoint using *sql.DB (for endpoint_session_migration tests).
// Use this when the test uses db.conn.ExecContext pattern instead of *sqlx.DB.
//
// If p.ID is non-zero, uses explicit ID; otherwise auto-generates.
// Returns the endpoint ID (either explicit or auto-generated).
//
// Example:
//
//	id := insertEndpointWithConn(ctx, t, conn, EndpointInsertParams{
//	    ID:       2001,
//	    EpEUI:    0x0102030405060708,
//	    Name:     "TestEndpoint",
//	    TenantID: 1001,
//	    ShAddr:   0x1234,
//	    Bidi:     true,
//	})
func insertEndpointWithConn(ctx context.Context, t *testing.T, conn *sql.DB, p EndpointInsertParams) int64 {
	t.Helper()
	applyEndpointDefaults(&p)

	now := time.Now()

	if p.ID != 0 {
		// Explicit ID provided - use INSERT without RETURNING
		_, err := conn.ExecContext(ctx, `
			INSERT INTO endpoints (
				id, ep_eui, name, description, tenant_id, owner_tenant_id,
				sh_addr, bidi,
				nwk_key, app_key, crypto_mode, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, p.ID, euiToBytes(p.EpEUI), p.Name, p.Description, p.TenantID, p.OwnerTenantID,
			p.ShAddr, p.Bidi,
			p.NwkKey, p.AppKey, p.CryptoMode, now, now)
		require.NoError(t, err, "Failed to insert test endpoint with explicit ID")
		return p.ID
	}

	// Auto-generate ID
	var id int64
	err := conn.QueryRowContext(ctx, `
		INSERT INTO endpoints (
			ep_eui, name, description, tenant_id, owner_tenant_id,
			sh_addr, bidi,
			nwk_key, app_key, crypto_mode, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, euiToBytes(p.EpEUI), p.Name, p.Description, p.TenantID, p.OwnerTenantID,
		p.ShAddr, p.Bidi,
		p.NwkKey, p.AppKey, p.CryptoMode, now, now).Scan(&id)
	require.NoError(t, err, "Failed to insert test endpoint")

	return id
}

// insertEndpointWithDetachKeys inserts a test endpoint with detach-related keys (sign, preshared).
// Used by tests that validate detach validation logic.
func insertEndpointWithDetachKeys(t *testing.T, db *sqlx.DB, p EndpointInsertParams) int64 {
	t.Helper()
	applyEndpointDefaults(&p)

	// Default detach keys to zeros if not provided.
	// sign must be exactly 4 bytes per migration 000048 length constraint.
	// preshared_key must be exactly 16 bytes per migration 000133 length constraint.
	sign := p.Sign
	if sign == nil {
		sign = make([]byte, 4)
	}
	preshared := p.Preshared
	if preshared == nil {
		preshared = make([]byte, 16)
	}

	var id int64
	err := db.QueryRow(`
		INSERT INTO endpoints (
			ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, sign, preshared_key, crypto_mode,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id
	`, euiToBytes(p.EpEUI), p.Name, p.Description, p.TenantID, p.OwnerTenantID,
		p.NwkKey, p.AppKey, sign, preshared, p.CryptoMode).Scan(&id)
	require.NoError(t, err, "Failed to insert test endpoint with detach keys")

	return id
}
