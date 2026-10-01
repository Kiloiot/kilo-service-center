package postgres

import (
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lockAttachCounterWaitProbe is how long the second lock is watched for not returning early.
const lockAttachCounterWaitProbe = 300 * time.Millisecond

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
		EPClass:       "A",
	}
	// Set a unique EUI
	copy(endpoint.EUI[:], []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})

	// Initialize logger for test
	logger.Initialize("error", "json")
	log := logger.Get()

	// Create DB wrapper
	storage := &DB{
		clock:  clock.SystemClock{},
		conn:   db.DB,
		sqlxDB: db,
		log:    log,
		cipher: testsupport.TestCipher(),
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
		EPClass:       "A",
	}
	// Set a unique EUI
	copy(endpoint.EUI[:], []byte{0x02, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})

	// Initialize logger for test
	logger.Initialize("error", "json")
	log := logger.Get()

	// Create DB wrapper
	storage := &DB{
		clock:  clock.SystemClock{},
		conn:   db.DB,
		sqlxDB: db,
		log:    log,
		cipher: testsupport.TestCipher(),
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
		clock:  clock.SystemClock{},
		conn:   db.DB,
		sqlxDB: db,
		log:    log,
		cipher: testsupport.TestCipher(),
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

// LockAttachCounter holds the endpoint row: a second transaction's lock waits
// for the first to commit and then reads the counter it wrote.
func TestLockAttachCounterSerializesTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	const tenantID = int64(920)
	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	createTestTenant(t, db, tenantID, "LockAttachCounterTenant")
	endpointID := insertEndpoint(t, db, EndpointInsertParams{EpEUI: 0x70B3D5677011150C, Name: "lock-attach-counter", TenantID: tenantID})
	store := &DB{clock: clock.SystemClock{}, conn: db.DB, sqlxDB: db, log: logger.Get(), cipher: testsupport.TestCipher()}
	ctx := t.Context()

	first, err := store.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = first.Rollback() }()
	_, err = first.EndPoints().LockAttachCounter(ctx, tenantID, endpointID)
	require.NoError(t, err)
	attachCnt := int64(7)
	overTheAirAttach := models.EndpointAttachmentStateParams{AttachCnt: &attachCnt, Nonce: []byte{1, 2, 3, 4}, Sign: []byte{5, 6, 7, 8}}
	require.NoError(t, first.EndPoints().EndpointAttachmentStateUpdate(ctx, tenantID, endpointID, overTheAirAttach))

	secondRead := make(chan *uint32, 1)
	go func() {
		second, beginErr := store.BeginTx(ctx)
		if beginErr != nil {
			secondRead <- nil
			return
		}
		defer func() { _ = second.Rollback() }()
		stored, lockErr := second.EndPoints().LockAttachCounter(ctx, tenantID, endpointID)
		if lockErr != nil {
			stored = nil
		}
		secondRead <- stored
	}()

	select {
	case <-secondRead:
		t.Fatal("the second lock must wait while the first transaction holds the row")
	case <-time.After(lockAttachCounterWaitProbe):
	}
	require.NoError(t, first.Commit())

	stored := <-secondRead
	require.NotNil(t, stored)
	assert.Equal(t, uint32(attachCnt), *stored, "the second transaction reads the committed counter")
}

func TestLockAttachCounterOfUnknownEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	store := &DB{clock: clock.SystemClock{}, conn: db.DB, sqlxDB: db, log: logger.Get(), cipher: testsupport.TestCipher()}
	tx, err := store.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	_, err = tx.EndPoints().LockAttachCounter(t.Context(), 1, 999999)
	require.ErrorIs(t, err, storage.ErrNotFound)
}

// ep_status values the endpoints CHECK constraint accepts (migration 003).
const (
	epStatusAttachedForTest = "attached"
	epStatusDetachedForTest = "detached"
)

// TransitionEndpointStatus reports a change only to the call that made it, so
// one attach or detach announcement follows however many stations confirm it.
func TestTransitionEndpointStatusReportsOnlyTheChangingCall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	const tenantID = int64(921)
	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	createTestTenant(t, db, tenantID, "TransitionEndpointStatusTenant")
	endpointID := insertEndpoint(t, db, EndpointInsertParams{EpEUI: 0x70B3D5677011150E, Name: "transition-status", TenantID: tenantID})
	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := t.Context()

	for _, step := range []struct {
		status  string
		changed bool
	}{
		{epStatusAttachedForTest, true},
		{epStatusAttachedForTest, false},
		{epStatusDetachedForTest, true},
		{epStatusDetachedForTest, false},
	} {
		changed, err := repo.TransitionEndpointStatus(ctx, tenantID, endpointID, step.status)
		require.NoError(t, err)
		assert.Equal(t, step.changed, changed, "transition to %s", step.status)
	}

	changed, err := repo.TransitionEndpointStatus(ctx, tenantID+1, endpointID, epStatusAttachedForTest)
	require.NoError(t, err)
	assert.False(t, changed, "another tenant cannot move the endpoint")
}
