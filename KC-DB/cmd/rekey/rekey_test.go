package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/migrations"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	testLegacyPassphrase = "rekey-test-legacy-passphrase"
	testMasterKeyHex     = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	// archiveKeyTypeApplication is the application key type in the key_type
	// vocabulary endpoint_keys_archive kept from before migration 015.
	archiveKeyTypeApplication = "app"
)

func TestMain(m *testing.M) {
	os.Exit(testsupport.Main(m))
}

// legacyEncrypt reproduces the retired KC-Core/pkg/crypto write path
// (nonce||GCM ciphertext under SHA-256 of a passphrase) for fixtures.
func legacyEncrypt(t *testing.T, passphrase string, plaintext []byte) []byte {
	t.Helper()
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)
	return gcm.Seal(nonce, nonce, plaintext, nil)
}

func testCipher(t *testing.T) keycrypto.Cipher {
	t.Helper()
	c, err := keycrypto.NewCipherFromMasterKey(testMasterKeyHex)
	require.NoError(t, err)
	return c
}

func openTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn, drop := testsupport.NewMigratedDatabase(t)
	t.Cleanup(drop)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.Ping())
	return db, dsn
}

func newTestRekeyer(t *testing.T, db *sql.DB, withLegacy, apply bool) *rekeyer {
	t.Helper()
	masterKey, err := keycrypto.ParseMasterKey(testMasterKeyHex)
	require.NoError(t, err)
	r := &rekeyer{
		db:      db,
		cipher:  testCipher(t),
		hmacKey: masterKey,
		apply:   apply,
	}
	if withLegacy {
		r.legacy = newLegacyCipher(testLegacyPassphrase)
	}
	return r
}

func seedTenant(t *testing.T, db *sql.DB, id int64) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, 'rekey test tenant', 'active', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`, id, "RekeyTenant"+strconv.FormatInt(id, 10))
	require.NoError(t, err)
}

func seedEndpoint(t *testing.T, db *sql.DB, tenantID int64, eui byte, nwkKey, appKey []byte) int64 {
	t.Helper()
	epEUI := []byte{0x70, 0xB3, 0xD5, 0x9C, 0x00, 0x00, 0x00, eui}
	var id int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO endpoints (ep_eui, name, tenant_id, owner_tenant_id, nwk_key, app_key)
		VALUES ($1, $2, $3, $3, $4, $5)
		RETURNING id`,
		epEUI, "rekey-ep-"+strconv.Itoa(int(eui)), tenantID,
		nullableBytes(nwkKey), nullableBytes(appKey)).Scan(&id))
	return id
}

func nullableBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

func fixtureKey(seed byte) []byte {
	key := make([]byte, wireKeyLen)
	for i := range key {
		key[i] = seed + byte(i)
	}
	return key
}

// TestRekey_ConvertsEverySurface seeds every legacy encoding on every surface,
// applies, and proves the stored values are envelopes decrypting to the
// original bytes, after which verify reports the database clean.
func TestRekey_ConvertsEverySurface(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, _ := openTestDB(t)
	ctx := testutil.TestContext()
	seedTenant(t, db, 700)
	cipher := testCipher(t)

	// endpoints: plaintext nwk_key, legacy raw-GCM app_key.
	nwkClear := fixtureKey(0x10)
	appClear := fixtureKey(0x20)
	epID := seedEndpoint(t, db, 700, 0x01, nwkClear, legacyEncrypt(t, testLegacyPassphrase, appClear))

	// endpoint_sessions: base64 text form of legacy GCM in the BYTEA column.
	sessionClear := fixtureKey(0x30)
	sessionLegacy := []byte(base64.StdEncoding.EncodeToString(legacyEncrypt(t, testLegacyPassphrase, sessionClear)))
	var sessionRowID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO endpoint_sessions (endpoint_id, tenant_id, session_id, attach_cnt, status, session_key)
		VALUES ($1, 700, gen_random_uuid(), 1, 'active', $2)
		RETURNING id`, epID, sessionLegacy).Scan(&sessionRowID))

	// messages: plaintext nwk_sn_key.
	msgClear := fixtureKey(0x40)
	epEUI := []byte{0x70, 0xB3, 0xD5, 0x9C, 0x00, 0x00, 0x00, 0x01}
	bsEUI := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x00, 0x01}
	var msgID string
	require.NoError(t, db.QueryRow(`
		INSERT INTO messages (tenant_id, op_id, ep_eui, bs_eui, rx_time, packet_cnt, snr, rssi, user_data, received_at, nwk_sn_key)
		VALUES (700, 1, $1, $2, 1, 1, 10.0, -80.0, $3, NOW(), $4)
		RETURNING id`, epEUI, bsEUI, []byte{0x42}, msgClear).Scan(&msgID))

	// basestations: one legacy tls_key (retired KeyEncryptor base64 form) and
	// one cleartext PEM.
	pemClear := []byte("-----BEGIN EC PRIVATE KEY-----\nrekey-test-material\n-----END EC PRIVATE KEY-----\n")
	legacyTLS := base64.StdEncoding.EncodeToString(legacyEncrypt(t, testLegacyPassphrase, pemClear))
	var bsLegacyID, bsPlainID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url, tls_key)
		VALUES (700, $1, 'rekey-bs-legacy', 'bssci', 'bssci://test', $2)
		RETURNING id`, bsEUI, legacyTLS).Scan(&bsLegacyID))
	bsEUI2 := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x00, 0x02}
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url, tls_key)
		VALUES (700, $1, 'rekey-bs-plain', 'bssci', 'bssci://test', $2)
		RETURNING id`, bsEUI2, string(pemClear)).Scan(&bsPlainID))

	// bssci_pending_operations: base64(binary envelope) form (readable with
	// the current master key) and legacy base64 GCM form.
	var bsSessionID int64
	bsUUID := make([]byte, 16)
	scUUID := make([]byte, 16)
	bsUUID[15], scUUID[15] = 1, 2
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestation_sessions (basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid, sn_bs_op_id, sn_sc_op_id, status)
		VALUES ($1, 700, $2, $3, 0, 0, 'active')
		RETURNING id`, bsLegacyID, bsUUID, scUUID).Scan(&bsSessionID))

	pendingClear := fixtureKey(0x50)
	binaryEnvelope, err := cipher.Encrypt(pendingClear)
	require.NoError(t, err)
	insertPendingOp := func(opID int64, encKey string) {
		_, err := db.Exec(`
			INSERT INTO bssci_pending_operations (basestation_session_id, operation_id, operation_type, operation_data, metadata)
			VALUES ($1, $2, 'attPrp', '{}'::jsonb, jsonb_build_object('encryptedKey', $3::text))`,
			bsSessionID, opID, encKey)
		require.NoError(t, err)
	}
	insertPendingOp(-1, base64.StdEncoding.EncodeToString(binaryEnvelope))
	pendingClear2 := fixtureKey(0x60)
	insertPendingOp(-2, base64.StdEncoding.EncodeToString(legacyEncrypt(t, testLegacyPassphrase, pendingClear2)))

	// Dry run first: nothing may change.
	dry := newTestRekeyer(t, db, true, false)
	rep, err := dry.run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, rep.surface("endpoints.nwk_key").Plaintext)
	assert.Equal(t, 1, rep.surface("endpoints.app_key").Legacy)
	assert.Equal(t, 1, rep.surface("endpoint_sessions.session_key").Legacy)
	assert.Equal(t, 1, rep.surface("messages.nwk_sn_key").Plaintext)
	assert.Equal(t, 1, rep.surface("basestations.tls_key").Legacy)
	assert.Equal(t, 1, rep.surface("basestations.tls_key").Plaintext)
	assert.Equal(t, 2, rep.surface("bssci_pending_operations.metadata.encryptedKey").Legacy)
	var stillPlain []byte
	require.NoError(t, db.QueryRow(`SELECT nwk_key FROM endpoints WHERE id = $1`, epID).Scan(&stillPlain))
	assert.Equal(t, nwkClear, stillPlain, "dry run must not modify anything")

	// Apply converts everything.
	applyRun := newTestRekeyer(t, db, true, true)
	rep, err = applyRun.run(ctx)
	require.NoError(t, err)
	require.True(t, rep.applyComplete(), "apply must convert every row: %+v", rep.surfaces)

	assertEnvelope := func(query string, id interface{}, want []byte) {
		t.Helper()
		var stored []byte
		require.NoError(t, db.QueryRow(query, id).Scan(&stored))
		require.True(t, keycrypto.IsEnvelope(stored), "stored value must be an envelope")
		decrypted, err := cipher.Decrypt(stored)
		require.NoError(t, err)
		assert.Equal(t, want, decrypted)
	}
	assertEnvelope(`SELECT nwk_key FROM endpoints WHERE id = $1`, epID, nwkClear)
	assertEnvelope(`SELECT app_key FROM endpoints WHERE id = $1`, epID, appClear)
	assertEnvelope(`SELECT session_key FROM endpoint_sessions WHERE id = $1`, sessionRowID, sessionClear)
	assertEnvelope(`SELECT nwk_sn_key FROM messages WHERE id = $1`, msgID, msgClear)

	assertTextEnvelope := func(query string, id interface{}, want []byte) {
		t.Helper()
		var stored string
		require.NoError(t, db.QueryRow(query, id).Scan(&stored))
		require.True(t, keycrypto.IsTextEnvelope(stored), "stored value must be a text envelope")
		decrypted, err := cipher.DecryptString(stored)
		require.NoError(t, err)
		assert.Equal(t, want, decrypted)
	}
	assertTextEnvelope(`SELECT tls_key FROM basestations WHERE id = $1`, bsLegacyID, pemClear)
	assertTextEnvelope(`SELECT tls_key FROM basestations WHERE id = $1`, bsPlainID, pemClear)
	assertTextEnvelope(`SELECT metadata ->> 'encryptedKey' FROM bssci_pending_operations WHERE operation_id = $1`, int64(-1), pendingClear)
	assertTextEnvelope(`SELECT metadata ->> 'encryptedKey' FROM bssci_pending_operations WHERE operation_id = $1`, int64(-2), pendingClear2)

	// Verify reports clean, and a second apply is an idempotent no-op.
	verify := newTestRekeyer(t, db, true, false)
	rep, err = verify.run(ctx)
	require.NoError(t, err)
	require.NoError(t, verify.countEndpointKeys(ctx, rep))
	assert.True(t, rep.clean(), "post-apply verify must be clean: %+v", rep.surfaces)

	again := newTestRekeyer(t, db, true, true)
	rep, err = again.run(ctx)
	require.NoError(t, err)
	for name, c := range rep.surfaces {
		assert.Zero(t, c.Converted, "second apply must convert nothing on %s", name)
	}
}

// TestRekey_LegacyWithoutKeyIsLockedNotClean proves legacy ciphertext without
// the legacy key is reported, never silently treated as clean or converted.
func TestRekey_LegacyWithoutKeyIsLockedNotClean(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, _ := openTestDB(t)
	ctx := testutil.TestContext()
	seedTenant(t, db, 701)
	seedEndpoint(t, db, 701, 0x02, legacyEncrypt(t, testLegacyPassphrase, fixtureKey(0x70)), nil)

	r := newTestRekeyer(t, db, false, true)
	rep, err := r.run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, rep.surface("endpoints.nwk_key").LegacyLocked)
	assert.Zero(t, rep.surface("endpoints.nwk_key").Converted)
	assert.False(t, rep.clean())
	assert.False(t, rep.applyComplete())
}

// seedBaseStationSession creates a base station and a BSSCI session for
// pending-operation fixtures and returns the session id.
func seedBaseStationSession(t *testing.T, db *sql.DB, tenantID int64, eui byte) int64 {
	t.Helper()
	bsEUI := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x01, eui}
	var bsID, sessionID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url)
		VALUES ($1, $2, $3, 'bssci', 'bssci://test')
		RETURNING id`, tenantID, bsEUI, "rekey-bs-"+strconv.Itoa(int(eui))).Scan(&bsID))
	bsUUID := make([]byte, 16)
	scUUID := make([]byte, 16)
	bsUUID[15], scUUID[15] = eui, eui+1
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestation_sessions (basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid, sn_bs_op_id, sn_sc_op_id, status)
		VALUES ($1, $2, $3, $4, 0, 0, 'active')
		RETURNING id`, bsID, tenantID, bsUUID, scUUID).Scan(&sessionID))
	return sessionID
}

// insertPendingOperation stores a pending operation with the given JSON
// operation record and metadata.
func insertPendingOperation(t *testing.T, db *sql.DB, sessionID, opID int64, operationData, metadata string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO bssci_pending_operations (basestation_session_id, operation_id, operation_type, operation_data, metadata)
		VALUES ($1, $2, 'attPrp', $3::jsonb, $4::jsonb)`,
		sessionID, opID, operationData, metadata)
	require.NoError(t, err)
}

// numericKeyJSON renders a key as the BSSCI Numeric[16] JSON array.
func numericKeyJSON(key []byte) string {
	parts := make([]string, len(key))
	for i, b := range key {
		parts[i] = strconv.Itoa(int(b))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// TestRekey_PendingKeyStoredUnencryptedIsPlaintext covers the recovery
// records written while the key encryptor was unavailable: base64 of the
// 16-byte cleartext key with isEncrypted=false must convert without any
// legacy key, and verify must then pass.
func TestRekey_PendingKeyStoredUnencryptedIsPlaintext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, _ := openTestDB(t)
	ctx := testutil.TestContext()
	seedTenant(t, db, 707)
	sessionID := seedBaseStationSession(t, db, 707, 0x07)

	clearKey := fixtureKey(0x70)
	insertPendingOperation(t, db, sessionID, -1, `{"command":"ulDataTx"}`,
		`{"encryptedKey":"`+base64.StdEncoding.EncodeToString(clearKey)+`","isEncrypted":false}`)

	r := newTestRekeyer(t, db, false, true)
	rep, err := r.run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, rep.surface("bssci_pending_operations.metadata.encryptedKey").Plaintext,
		"a 16-byte key stored with isEncrypted=false is cleartext")
	require.True(t, rep.applyComplete(), "apply must convert the cleartext key: %+v", rep.surfaces)

	var stored string
	var hasMarker bool
	require.NoError(t, db.QueryRow(`
		SELECT metadata ->> 'encryptedKey', metadata ? 'isEncrypted'
		FROM bssci_pending_operations WHERE operation_id = -1`).Scan(&stored, &hasMarker))
	decrypted, err := testCipher(t).DecryptString(stored)
	require.NoError(t, err)
	assert.Equal(t, clearKey, decrypted)
	assert.False(t, hasMarker, "the obsolete isEncrypted marker must go with the conversion")

	verify := newTestRekeyer(t, db, false, false)
	rep, err = verify.run(ctx)
	require.NoError(t, err)
	require.NoError(t, verify.countEndpointKeys(ctx, rep))
	assert.True(t, rep.clean(), "verify must pass after the conversion: %+v", rep.surfaces)
}

// TestRekey_ConvertsArchivedMessageKeys proves archived message copies are a
// key surface like the live table.
func TestRekey_ConvertsArchivedMessageKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, _ := openTestDB(t)
	ctx := testutil.TestContext()
	seedTenant(t, db, 708)

	clearKey := fixtureKey(0x80)
	var archivedID string
	require.NoError(t, db.QueryRow(`
		INSERT INTO messages_archive (tenant_id, op_id, ep_eui, bs_eui, rx_time, packet_cnt, snr, rssi, nwk_sn_key)
		VALUES (708, 1, $1, $2, 1, 1, 10.0, -80.0, $3)
		RETURNING id`,
		[]byte{0x70, 0xB3, 0xD5, 0x9C, 0x00, 0x00, 0x08, 0x01},
		[]byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x08, 0x01}, clearKey).Scan(&archivedID))

	dry := newTestRekeyer(t, db, false, false)
	rep, err := dry.run(ctx)
	require.NoError(t, err)
	require.NoError(t, dry.countEndpointKeys(ctx, rep))
	assert.Equal(t, 1, rep.surface("messages_archive.nwk_sn_key").Plaintext)
	assert.False(t, rep.clean(), "verify must not report an archived cleartext key as clean")

	applyRun := newTestRekeyer(t, db, false, true)
	rep, err = applyRun.run(ctx)
	require.NoError(t, err)
	require.True(t, rep.applyComplete(), "%+v", rep.surfaces)

	var stored []byte
	require.NoError(t, db.QueryRow(`SELECT nwk_sn_key FROM messages_archive WHERE id = $1`, archivedID).Scan(&stored))
	decrypted, err := testCipher(t).Decrypt(stored)
	require.NoError(t, err)
	assert.Equal(t, clearKey, decrypted)
}

// TestRekey_MovesOperationRecordKeyIntoEncryptedMetadata covers attach
// propagate recovery records persisted before key sanitization: the cleartext
// nwkSnKey inside operation_data moves into the encrypted metadata form resume
// reads, and a record carrying both forms is reported instead of guessed at.
func TestRekey_MovesOperationRecordKeyIntoEncryptedMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, _ := openTestDB(t)
	ctx := testutil.TestContext()
	seedTenant(t, db, 709)
	sessionID := seedBaseStationSession(t, db, 709, 0x09)
	cipher := testCipher(t)

	clearKey := fixtureKey(0x90)
	insertPendingOperation(t, db, sessionID, -1,
		`{"command":"attPrp","opId":-1,"nwkSnKey":`+numericKeyJSON(clearKey)+`}`,
		`{"epEui":"70-B3-D5-9C-00-00-09-01"}`)

	envelope, err := cipher.EncryptString(fixtureKey(0x91))
	require.NoError(t, err)
	insertPendingOperation(t, db, sessionID, -2,
		`{"command":"attPrp","opId":-2,"nwkSnKey":`+numericKeyJSON(fixtureKey(0x92))+`}`,
		`{"encryptedKey":"`+envelope+`"}`)

	const surface = "bssci_pending_operations.operation_data.nwkSnKey"
	dry := newTestRekeyer(t, db, false, false)
	rep, err := dry.run(ctx)
	require.NoError(t, err)
	require.NoError(t, dry.countEndpointKeys(ctx, rep))
	assert.Equal(t, 1, rep.surface(surface).Plaintext)
	assert.Equal(t, 1, rep.surface(surface).Malformed, "a record with both key forms must be reported")
	assert.False(t, rep.clean())

	applyRun := newTestRekeyer(t, db, false, true)
	rep, err = applyRun.run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, rep.surface(surface).Converted)
	assert.False(t, rep.applyComplete(), "the conflicting record must keep apply incomplete")

	var stored, command string
	var keyStillInRecord bool
	require.NoError(t, db.QueryRow(`
		SELECT metadata ->> 'encryptedKey', operation_data ->> 'command', operation_data ? 'nwkSnKey'
		FROM bssci_pending_operations WHERE operation_id = -1`).Scan(&stored, &command, &keyStillInRecord))
	assert.False(t, keyStillInRecord, "the cleartext key must leave the operation record")
	assert.Equal(t, "attPrp", command, "the rest of the operation record must be preserved")
	decrypted, err := cipher.DecryptString(stored)
	require.NoError(t, err)
	assert.Equal(t, clearKey, decrypted)

	var conflictKept bool
	require.NoError(t, db.QueryRow(`
		SELECT operation_data ? 'nwkSnKey' FROM bssci_pending_operations WHERE operation_id = -2`).Scan(&conflictKept))
	assert.True(t, conflictKept, "a conflicting record must be left for the operator")
}

// TestRekey_PerRowStatementsUseThePrimaryKeyIndex proves the per-row
// compare-and-swap and read-back reach their row through the primary-key
// index instead of scanning the whole table once per converted row.
func TestRekey_PerRowStatementsUseThePrimaryKeyIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, _ := openTestDB(t)
	ctx := testutil.TestContext()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = conn.ExecContext(ctx, `SET enable_seqscan = off`)
	require.NoError(t, err)

	for _, col := range postgres.KeyMaterialColumns() {
		column := byteaColumn{col}
		for _, statement := range []string{column.swapQuery(), column.readBackQuery()} {
			rows, err := conn.QueryContext(ctx, `EXPLAIN (GENERIC_PLAN) `+statement)
			require.NoError(t, err)
			var plan []string
			for rows.Next() {
				var line string
				require.NoError(t, rows.Scan(&line))
				plan = append(plan, line)
			}
			require.NoError(t, rows.Err())
			require.NoError(t, rows.Close())
			assert.NotContains(t, strings.Join(plan, "\n"), "Seq Scan",
				"%s must use the %s primary-key index:\n%s", statement, col.Table, strings.Join(plan, "\n"))
		}
	}
}

// migrationRunnerFor builds a runner over the per-test database.
// rollBackTo steps the schema down from its head to target, counting the
// migration versions in between so gaps in the numbering are not stepped
// over, and returns the head it started from.
func rollBackTo(t *testing.T, runner *postgres.MigrationRunner, target uint) (head uint) {
	t.Helper()
	head, _, err := runner.Version(testutil.TestContext())
	require.NoError(t, err)
	names, err := fs.Glob(migrations.MigrationsFS, "*.up.sql")
	require.NoError(t, err)
	steps := 0
	for _, name := range names {
		version, parseErr := strconv.ParseUint(strings.SplitN(name, "_", 2)[0], 10, 32)
		require.NoError(t, parseErr, name)
		if uint(version) > target && uint(version) <= head {
			steps++
		}
	}
	require.NoError(t, runner.Steps(testutil.TestContext(), -steps))
	return head
}

func migrationRunnerFor(t *testing.T, dsn string) *postgres.MigrationRunner {
	t.Helper()
	cfg, err := testsupport.ParseDSN(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(cfg.Port)
	require.NoError(t, err)
	runner, err := postgres.NewMigrationRunner(&postgres.Config{
		Host:     cfg.Host,
		Port:     port,
		Database: cfg.Database,
		Username: cfg.User,
		Password: cfg.Password,
		SSLMode:  "disable",
	}, postgres.UpgradeGates())
	require.NoError(t, err)
	return runner
}

// TestMigration145_RequiresEmptyTablesAndRekeyResolves exercises the guard
// end-to-end on a populated legacy database: any remaining live or archive
// row aborts the migration; the rekey reconciliation resolves redundant and
// migratable rows, exports the archive and the conflicts, and after the
// operator resolves the reported conflicts the migration applies.
func TestMigration145_RequiresEmptyTablesAndRekeyResolves(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, dsn := openTestDB(t)
	ctx := testutil.TestContext()
	runner := migrationRunnerFor(t, dsn)

	// Roll back to 000144 to restore the endpoint_keys subsystem.
	rollBackTo(t, runner, 144)

	seedTenant(t, db, 702)
	cipher := testCipher(t)

	matchingKey := fixtureKey(0x80)
	matchingEnvelope, err := cipher.Encrypt(matchingKey)
	require.NoError(t, err)
	epMatching := seedEndpoint(t, db, 702, 0x03, matchingEnvelope, nil)

	mismatchKey := fixtureKey(0x90)
	otherEnvelope, err := cipher.Encrypt(fixtureKey(0xA0))
	require.NoError(t, err)
	epMismatch := seedEndpoint(t, db, 702, 0x04, otherEnvelope, nil)

	uncoveredKey := fixtureKey(0xB0)
	epUncovered := seedEndpoint(t, db, 702, 0x05, nil, nil)

	// Wrong tenant on the row: the endpoint lookup is tenant-scoped, so the
	// row cannot be reconciled.
	seedTenant(t, db, 703)
	epWrongTenant := seedEndpoint(t, db, 702, 0x06, nil, nil)

	insertKeyRow := func(endpointID, tenantID int64, keyType string, value []byte, active bool) {
		_, err := db.Exec(`
			INSERT INTO endpoint_keys (endpoint_id, tenant_id, key_type, key_version, key_value, is_active)
			VALUES ($1, $2, $3, 1, $4, $5)`, endpointID, tenantID, keyType, value, active)
		require.NoError(t, err)
	}
	insertKeyRow(epMatching, 702, keyTypeNetwork, matchingKey, true)
	insertKeyRow(epMismatch, 702, keyTypeNetwork, mismatchKey, true)
	insertKeyRow(epUncovered, 702, keyTypeNetwork, uncoveredKey, false)
	insertKeyRow(epMatching, 702, "join", fixtureKey(0xC0), true)
	insertKeyRow(epWrongTenant, 703, keyTypeNetwork, fixtureKey(0xD0), true)

	// One archived row must also block the drop.
	_, err = db.Exec(`
		INSERT INTO endpoint_keys_archive (endpoint_id, tenant_id, key_type, key_version, key_value, is_active)
		VALUES ($1, 702, $2, 1, $3, false)`, epMatching, archiveKeyTypeApplication, fixtureKey(0xE0))
	require.NoError(t, err)

	// The populated tables must abort the migration.
	require.Error(t, runner.Steps(ctx, 2), "migration 000145 must refuse populated tables")
	require.NoError(t, runner.Force(ctx, 144))

	// Reconcile: matching row deleted; inactive uncovered row migrated into
	// its endpoint; mismatch, join, and wrong-tenant rows exported and
	// reported; archive exported and emptied.
	exportPath := filepath.Join(t.TempDir(), "endpoint-keys-export.enc")
	r := newTestRekeyer(t, db, false, true)
	rep := newReport()
	require.NoError(t, r.reconcileEndpointKeys(ctx, rep, exportPath))
	assert.Equal(t, 2, rep.epKeysResolved, "matching and uncovered rows must resolve")
	assert.Len(t, rep.epKeysConflicts, 3, "mismatch, join, and wrong-tenant rows must be reported: %v", rep.epKeysConflicts)
	assert.Equal(t, 4, rep.epKeysExported, "three conflicts plus one archive row must be exported")
	assert.Zero(t, rep.epKeysArchiveRows, "the exported archive must be emptied")

	// The uncovered endpoint now holds the migrated key as an envelope.
	var migrated []byte
	require.NoError(t, db.QueryRow(`SELECT nwk_key FROM endpoints WHERE id = $1`, epUncovered).Scan(&migrated))
	require.True(t, keycrypto.IsEnvelope(migrated))
	decrypted, err := cipher.Decrypt(migrated)
	require.NoError(t, err)
	assert.Equal(t, uncoveredKey, decrypted)

	// The export decrypts under the master key and holds the four rows.
	exported, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	payload, err := cipher.Decrypt(exported)
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"keyType":"join"`)

	// Conflicted rows still block the migration until the operator resolves
	// them (here: deleting them after reviewing the export).
	require.Error(t, runner.Steps(ctx, 2), "conflicted rows must still block the drop")
	require.NoError(t, runner.Force(ctx, 144))
	_, err = db.Exec(`DELETE FROM endpoint_keys`)
	require.NoError(t, err)

	_, err = runner.Run(ctx)
	require.NoError(t, err, "empty tables must let 000145 and every later migration apply")
}

// TestUpgradeFromSchema142_MigratesTo143BeforeRekey reproduces the upgrade of
// a v1.3.0 database: at 000142 the 16-byte key checks reject envelopes, so
// rekey cannot complete; migrating to 000143 first lets rekey convert every
// key and reconcile endpoint_keys, after which the remaining migrations apply.
func TestUpgradeFromSchema142_MigratesTo143BeforeRekey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, dsn := openTestDB(t)
	ctx := testutil.TestContext()
	runner := migrationRunnerFor(t, dsn)

	head := rollBackTo(t, runner, 142)

	seedTenant(t, db, 704)
	nwkKey, appKey := fixtureKey(0x10), fixtureKey(0x20)
	epID := seedEndpoint(t, db, 704, 0x07, nwkKey, appKey)
	_, err := db.Exec(`
		INSERT INTO endpoint_keys (endpoint_id, tenant_id, key_type, key_version, key_value, is_active)
		VALUES ($1, 704, $2, 1, $3, true)`, epID, keyTypeNetwork, nwkKey)
	require.NoError(t, err)

	early := newTestRekeyer(t, db, false, true)
	rep, err := early.run(ctx)
	require.NoError(t, err)
	assert.False(t, rep.applyComplete(), "the 16-byte key checks at 000142 must reject envelopes")

	version, err := runner.RunTo(ctx, 143)
	require.NoError(t, err)
	require.Equal(t, uint(143), version)

	r := newTestRekeyer(t, db, false, true)
	rep, err = r.run(ctx)
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpointKeys(ctx, rep, filepath.Join(t.TempDir(), "endpoint-keys-export.enc")))
	require.True(t, rep.applyComplete(), "rekey at 000143 must convert every key and resolve endpoint_keys: %v", rep.epKeysConflicts)

	verify := newTestRekeyer(t, db, false, false)
	rep, err = verify.run(ctx)
	require.NoError(t, err)
	require.NoError(t, verify.countEndpointKeys(ctx, rep))
	require.True(t, rep.clean(), "verify must report the database clean after apply")

	final, err := runner.Run(ctx)
	require.NoError(t, err, "every migration after 000143 must apply once rekey has run")
	assert.Equal(t, head, final)

	cipher := testCipher(t)
	var storedNwk, storedApp []byte
	require.NoError(t, db.QueryRow(`SELECT nwk_key, app_key FROM endpoints WHERE id = $1`, epID).Scan(&storedNwk, &storedApp))
	decryptedNwk, err := cipher.Decrypt(storedNwk)
	require.NoError(t, err)
	assert.Equal(t, nwkKey, decryptedNwk)
	decryptedApp, err := cipher.Decrypt(storedApp)
	require.NoError(t, err)
	assert.Equal(t, appKey, decryptedApp)

	unchanged, err := runner.RunTo(ctx, 143)
	require.NoError(t, err)
	assert.Equal(t, head, unchanged, "RunTo must never migrate down")
}

// rollBackTo142 returns a per-test database to the v1.3.0 schema and reports
// the head version it came from.
func rollBackTo142(t *testing.T, runner *postgres.MigrationRunner) uint {
	t.Helper()
	head, _, err := runner.Version(testutil.TestContext())
	require.NoError(t, err)
	// Migration numbers have gaps, so step down by version rather than by count.
	for version := head; version > 142; {
		require.NoError(t, runner.Steps(testutil.TestContext(), -1))
		version, _, err = runner.Version(testutil.TestContext())
		require.NoError(t, err)
	}
	return head
}

// assertHeldCleanAt143 proves an upgrade stopped at 000143 without leaving
// golang-migrate dirty.
func assertHeldCleanAt143(t *testing.T, runner *postgres.MigrationRunner) {
	t.Helper()
	version, dirty, err := runner.Version(testutil.TestContext())
	require.NoError(t, err)
	assert.Equal(t, uint(143), version, "the upgrade must stop at 000143")
	assert.False(t, dirty, "the schema must stay clean at 000143")
}

// TestMigrationRunner_HoldsPlaintextKeysAt143 upgrades a populated v1.3.0
// database without running rekey: both Run and RunTo stop at 000143 with a
// clean schema and an error naming the rekey command, and the head is reached
// once rekey has converted every key.
func TestMigrationRunner_HoldsPlaintextKeysAt143(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, dsn := openTestDB(t)
	ctx := testutil.TestContext()
	runner := migrationRunnerFor(t, dsn)
	head := rollBackTo142(t, runner)

	seedTenant(t, db, 705)
	seedEndpoint(t, db, 705, 0x08, fixtureKey(0x10), fixtureKey(0x20))

	_, err := runner.Run(ctx)
	require.Error(t, err, "Run must refuse to migrate plaintext keys past 000143")
	assert.ErrorIs(t, err, postgres.ErrUnconvertedKeyMaterial)
	assert.Contains(t, err.Error(), "rekey")
	assert.Contains(t, err.Error(), "endpoints.nwk_key")
	assertHeldCleanAt143(t, runner)

	_, err = runner.RunTo(ctx, head)
	require.Error(t, err, "RunTo must refuse to migrate plaintext keys past 000143")
	assert.ErrorIs(t, err, postgres.ErrUnconvertedKeyMaterial)
	assertHeldCleanAt143(t, runner)

	r := newTestRekeyer(t, db, false, true)
	rep, err := r.run(ctx)
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpointKeys(ctx, rep, ""))
	require.True(t, rep.applyComplete(), "rekey must convert every key: %+v", rep.surfaces)

	final, err := runner.Run(ctx)
	require.NoError(t, err, "a rekeyed database must migrate to the head")
	assert.Equal(t, head, final)
}

// TestMigrationRunner_EndpointKeyRowsHoldCleanAt143 proves a populated
// endpoint_keys table stops the upgrade at 000143 instead of failing the
// 000145 guard mid-run and leaving the schema dirty.
func TestMigrationRunner_EndpointKeyRowsHoldCleanAt143(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, dsn := openTestDB(t)
	runner := migrationRunnerFor(t, dsn)
	rollBackTo142(t, runner)

	seedTenant(t, db, 706)
	epID := seedEndpoint(t, db, 706, 0x09, nil, nil)
	_, err := db.Exec(`
		INSERT INTO endpoint_keys (endpoint_id, tenant_id, key_type, key_version, key_value, is_active)
		VALUES ($1, 706, $2, 1, $3, true)`, epID, keyTypeNetwork, fixtureKey(0x30))
	require.NoError(t, err)

	_, err = runner.Run(testutil.TestContext())
	require.Error(t, err)
	assert.ErrorIs(t, err, postgres.ErrUnconvertedKeyMaterial)
	assert.Contains(t, err.Error(), "endpoint_keys")
	assertHeldCleanAt143(t, runner)
}

// TestMigrationRunner_GateNamesEveryUnconvertedSurface seeds pre-envelope
// material on every surface the rekey command converts: the upgrade must name
// each one while holding at 000143, and must reach the head once rekey has
// converted them all.
func TestMigrationRunner_GateNamesEveryUnconvertedSurface(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, dsn := openTestDB(t)
	ctx := testutil.TestContext()
	runner := migrationRunnerFor(t, dsn)
	head := rollBackTo142(t, runner)
	cipher := testCipher(t)

	seedTenant(t, db, 710)
	nwkKey := fixtureKey(0x11)
	epID := seedEndpoint(t, db, 710, 0x0A, nwkKey, fixtureKey(0x12))
	_, err := db.Exec(`
		INSERT INTO endpoint_sessions (endpoint_id, tenant_id, session_id, attach_cnt, status, session_key)
		VALUES ($1, 710, gen_random_uuid(), 1, 'active', $2)`, epID, fixtureKey(0x13))
	require.NoError(t, err)
	epEUI := []byte{0x70, 0xB3, 0xD5, 0x9C, 0x00, 0x00, 0x00, 0x0A}
	bsEUI := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x00, 0x0A}
	for _, table := range []string{"messages", "messages_archive"} {
		_, err = db.Exec(`
			INSERT INTO `+table+` (tenant_id, op_id, ep_eui, bs_eui, rx_time, packet_cnt, snr, rssi, nwk_sn_key)
			VALUES (710, 1, $1, $2, 1, 1, 10.0, -80.0, $3)`, epEUI, bsEUI, fixtureKey(0x14))
		require.NoError(t, err)
	}
	sessionID := seedBaseStationSession(t, db, 710, 0x0A)
	_, err = db.Exec(`UPDATE basestations SET tls_key = $1 WHERE tenant_id = 710`,
		"-----BEGIN EC PRIVATE KEY-----\ngate-test\n-----END EC PRIVATE KEY-----\n")
	require.NoError(t, err)
	binaryEnvelope, err := cipher.Encrypt(fixtureKey(0x15))
	require.NoError(t, err)
	insertPendingOperation(t, db, sessionID, -1, `{"command":"attPrp"}`,
		`{"encryptedKey":"`+base64.StdEncoding.EncodeToString(binaryEnvelope)+`"}`)
	insertPendingOperation(t, db, sessionID, -2,
		`{"command":"attPrp","nwkSnKey":`+numericKeyJSON(fixtureKey(0x16))+`}`, `{}`)
	_, err = db.Exec(`
		INSERT INTO endpoint_keys (endpoint_id, tenant_id, key_type, key_version, key_value, is_active)
		VALUES ($1, 710, $2, 1, $3, true)`, epID, keyTypeNetwork, nwkKey)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO endpoint_keys_archive (endpoint_id, tenant_id, key_type, key_version, key_value, is_active)
		VALUES ($1, 710, $2, 1, $3, false)`, epID, archiveKeyTypeApplication, fixtureKey(0x17))
	require.NoError(t, err)

	_, err = runner.Run(ctx)
	require.ErrorIs(t, err, postgres.ErrUnconvertedKeyMaterial)
	for _, surface := range []string{
		"endpoints.nwk_key", "endpoints.app_key", "endpoint_sessions.session_key",
		"messages.nwk_sn_key", "messages_archive.nwk_sn_key", "basestations.tls_key",
		"bssci_pending_operations.metadata.encryptedKey", "bssci_pending_operations.operation_data.nwkSnKey",
		"endpoint_keys", "endpoint_keys_archive",
	} {
		assert.Contains(t, err.Error(), surface)
	}
	assertHeldCleanAt143(t, runner)

	r := newTestRekeyer(t, db, false, true)
	rep, err := r.run(ctx)
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpointKeys(ctx, rep, filepath.Join(t.TempDir(), "endpoint-keys-export.enc")))
	require.True(t, rep.applyComplete(), "%+v %v", rep.surfaces, rep.epKeysConflicts)

	final, err := runner.Run(ctx)
	require.NoError(t, err, "every surface rekey converts must satisfy the gate")
	assert.Equal(t, head, final)
}

// TestMigrationRunner_FreshDatabaseReachesHead proves an empty database
// crosses 000143 without any rekey step.
func TestMigrationRunner_FreshDatabaseReachesHead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	_, migratedDSN := openTestDB(t)
	ctx := testutil.TestContext()
	head, _, err := migrationRunnerFor(t, migratedDSN).Version(ctx)
	require.NoError(t, err)

	_, cfg, cleanup := testsupport.SetupPostgresContainerWithoutMigrations(t)
	t.Cleanup(cleanup)
	port, err := strconv.Atoi(cfg.Port)
	require.NoError(t, err)
	runner, err := postgres.NewMigrationRunner(&postgres.Config{
		Host:     cfg.Host,
		Port:     port,
		Database: cfg.Database,
		Username: cfg.User,
		Password: cfg.Password,
		SSLMode:  "disable",
	}, postgres.UpgradeGates())
	require.NoError(t, err)

	version, err := runner.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, head, version)
}
