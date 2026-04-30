package postgres

import (
	"testing"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransactionalCreate_DefaultsOwnerTenantID verifies that when OwnerTenantID
// is zero during transactional Create, it defaults to TenantID (roaming support).
func TestTransactionalCreate_DefaultsOwnerTenantID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Create test tenant
	createTestTenant(t, db, 100, "TestTenant100")

	// Create endpoint with OwnerTenantID = 0 (unset)
	endpoint := &models.EndPoint{
		Name:          "Test-DefaultOwner",
		Description:   "Test endpoint for owner defaulting",
		TenantID:      100,
		OwnerTenantID: 0, // Should default to TenantID
		NwkSnKey:      make([]byte, 16),
		AppKey:        make([]byte, 16),
		CryptoMode:    0,
	}
	// Set a unique EUI
	copy(endpoint.EUI[:], []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})

	// Initialize logger for test
	logger.Initialize("error", "json")
	log := logger.Get()

	// Create DB wrapper
	storage := &DB{
		conn: db.DB,
		log:  log,
	}

	// Begin transaction
	tx, err := storage.BeginTx(t.Context())
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback() }()

	// Create via transaction
	err = tx.EndPoints().Create(t.Context(), endpoint)
	require.NoError(t, err, "Failed to create endpoint in transaction")

	// Commit to persist
	err = tx.Commit()
	require.NoError(t, err, "Failed to commit transaction")

	// Verify: OwnerTenantID should now equal TenantID
	assert.Equal(t, int64(100), endpoint.OwnerTenantID,
		"OwnerTenantID should default to TenantID when not specified")

	// Also verify in DB directly
	var storedOwnerTenantID int64
	err = db.Get(&storedOwnerTenantID,
		`SELECT owner_tenant_id FROM endpoints WHERE id = $1`, endpoint.ID)
	require.NoError(t, err, "Failed to query owner_tenant_id from DB")
	assert.Equal(t, int64(100), storedOwnerTenantID,
		"DB owner_tenant_id should match TenantID")
}

// TestTransactionalCreate_PreservesExplicitOwnerTenantID verifies that when
// OwnerTenantID is explicitly set, it is preserved (roaming scenario).
func TestTransactionalCreate_PreservesExplicitOwnerTenantID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Create test tenants (current and owner)
	createTestTenant(t, db, 200, "TestTenant200")
	createTestTenant(t, db, 201, "TestTenant201-Owner")

	// Create endpoint with explicit OwnerTenantID (roaming case)
	endpoint := &models.EndPoint{
		Name:          "Test-ExplicitOwner",
		Description:   "Test endpoint with explicit owner",
		TenantID:      200,
		OwnerTenantID: 201, // Explicitly set to different tenant
		NwkSnKey:      make([]byte, 16),
		AppKey:        make([]byte, 16),
		CryptoMode:    0,
	}
	// Set a unique EUI
	copy(endpoint.EUI[:], []byte{0x02, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})

	// Initialize logger for test
	logger.Initialize("error", "json")
	log := logger.Get()

	// Create DB wrapper
	storage := &DB{
		conn: db.DB,
		log:  log,
	}

	// Begin transaction
	tx, err := storage.BeginTx(t.Context())
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback() }()

	// Create via transaction
	err = tx.EndPoints().Create(t.Context(), endpoint)
	require.NoError(t, err, "Failed to create endpoint in transaction")

	// Commit to persist
	err = tx.Commit()
	require.NoError(t, err, "Failed to commit transaction")

	// Verify: OwnerTenantID should remain 201 (not overwritten to TenantID)
	assert.Equal(t, int64(201), endpoint.OwnerTenantID,
		"OwnerTenantID should be preserved when explicitly set")

	// Also verify in DB directly
	var storedOwnerTenantID int64
	err = db.Get(&storedOwnerTenantID,
		`SELECT owner_tenant_id FROM endpoints WHERE id = $1`, endpoint.ID)
	require.NoError(t, err, "Failed to query owner_tenant_id from DB")
	assert.Equal(t, int64(201), storedOwnerTenantID,
		"DB owner_tenant_id should preserve explicit value")
}

// TestTransactionalGet_ReturnsOwnerTenantID verifies that the Get method
// correctly returns the owner_tenant_id field for roaming enforcement.
func TestTransactionalGet_ReturnsOwnerTenantID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Create test tenants
	createTestTenant(t, db, 300, "TestTenant300")
	createTestTenant(t, db, 301, "TestTenant301-Owner")

	// Insert endpoint using helper (validates owner_tenant_id is preserved)
	var eui models.EUI
	copy(eui[:], []byte{0x03, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})

	_ = insertEndpoint(t, db, EndpointInsertParams{
		EpEUI:         0x0302030405060708,
		Name:          "Test-GetOwner",
		Description:   "Test endpoint for Get",
		TenantID:      300,
		OwnerTenantID: 301, // Explicit roaming scenario
	})

	// Initialize logger for test
	logger.Initialize("error", "json")
	log := logger.Get()

	// Create DB wrapper
	storage := &DB{
		conn: db.DB,
		log:  log,
	}

	// Begin transaction and use Get
	tx, err := storage.BeginTx(t.Context())
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = tx.Rollback() }()

	// Get endpoint via transaction
	ep, err := tx.EndPoints().Get(t.Context(), eui)
	require.NoError(t, err, "Failed to get endpoint")
	require.NotNil(t, ep, "Endpoint should be found")

	// Verify OwnerTenantID is returned correctly
	assert.Equal(t, int64(300), ep.TenantID, "TenantID should be 300")
	assert.Equal(t, int64(301), ep.OwnerTenantID, "OwnerTenantID should be 301")
}
