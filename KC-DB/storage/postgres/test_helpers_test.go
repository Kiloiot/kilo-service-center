package postgres

import (
	"fmt"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
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
// Use with insertEndpoint() for consistent test data.
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
	ShAddr uint32 // Short address
	Bidi   bool   // Bidirectional flag
	Sign   []byte // Signature key (detach tests)
}

// ptrTime returns a pointer to the given time value.
// Used for setting CreatedAt in EndpointInsertParams.
func ptrTime(t time.Time) *time.Time {
	return &t
}

// applyEndpointDefaults normalizes EndpointInsertParams with sensible defaults.
// This ensures owner_tenant_id defaults to tenant_id and keys default to 16 zero
// bytes. Keys are stored as keycrypto envelopes, mirroring the production write
// path: the strict repository readers accept nothing else.
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
	p.NwkKey = envelopeForTest(p.NwkKey)
	p.AppKey = envelopeForTest(p.AppKey)
}

// envelopeForTest wraps raw fixture key bytes in a test-cipher envelope;
// values that are already envelopes pass through so helpers can be nested.
func envelopeForTest(key []byte) []byte {
	if len(key) == 0 || keycrypto.IsEnvelope(key) {
		return key
	}
	envelope, err := testsupport.TestCipher().Encrypt(key)
	if err != nil {
		panic(err)
	}
	return envelope
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
		`, p.ID, mioty.EUI64Bytes(p.EpEUI), p.Name, p.Description, p.TenantID, p.OwnerTenantID,
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
	`, mioty.EUI64Bytes(p.EpEUI), p.Name, p.Description, p.TenantID, p.OwnerTenantID,
		p.NwkKey, p.AppKey, p.CryptoMode, createdAt, createdAt).Scan(&id)
	require.NoError(t, err, "Failed to insert test endpoint")

	return id
}

// insertEndpointWithDetachKeys inserts a test endpoint with detach-related key material (sign).
// Used by tests that validate detach validation logic.
func insertEndpointWithDetachKeys(t *testing.T, db *sqlx.DB, p EndpointInsertParams) int64 {
	t.Helper()
	applyEndpointDefaults(&p)

	// Default detach keys to zeros if not provided.
	// sign must be exactly 4 bytes per migration 000048 length constraint.
	sign := p.Sign
	if sign == nil {
		sign = make([]byte, 4)
	}

	var id int64
	err := db.QueryRow(`
		INSERT INTO endpoints (
			ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, sign, crypto_mode,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id
	`, mioty.EUI64Bytes(p.EpEUI), p.Name, p.Description, p.TenantID, p.OwnerTenantID,
		p.NwkKey, p.AppKey, sign, p.CryptoMode).Scan(&id)
	require.NoError(t, err, "Failed to insert test endpoint with detach keys")

	return id
}
