package postgres

import (
	"bytes"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// selectRawEndpointKeys reads the stored nwk_key / app_key bytes for an endpoint
// straight from the column, bypassing the repository decryption path.
func selectRawEndpointKeys(t *testing.T, db *sqlx.DB, eui models.EUI) (nwk, app []byte) {
	t.Helper()
	err := db.QueryRow("SELECT nwk_key, app_key FROM endpoints WHERE ep_eui = $1", eui[:]).Scan(&nwk, &app)
	require.NoError(t, err, "read raw endpoint key columns")
	return nwk, app
}

// TestEndpointKeys_EncryptedAtRest verifies that Create and Update write the
// nwk_key and app_key columns as keycrypto envelopes (never plaintext) and that
// the repository read path decrypts them back to the original key material.
func TestEndpointKeys_EncryptedAtRest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 700, "TestTenant700")
	cleanupEndpointTestData(t, db, "KeyEnc%")
	defer cleanupEndpointTestData(t, db, "KeyEnc%")

	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x01}
	nwkKey := []byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8,
		0xA9, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xAF, 0xB0}
	appKey := []byte{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8,
		0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF, 0xD0}

	endpoint := &models.EndPoint{
		EUI:        eui,
		Name:       "KeyEnc-EP1",
		TenantID:   700,
		EPClass:    "A",
		NwkSnKey:   append([]byte(nil), nwkKey...),
		AppKey:     append([]byte(nil), appKey...),
		CryptoMode: 0,
		Tags:       make(map[string]string),
	}

	require.NoError(t, repo.Create(ctx, endpoint))

	// Raw column bytes must be envelopes and must NOT contain the plaintext key.
	rawNwk, rawApp := selectRawEndpointKeys(t, db, eui)
	assert.True(t, keycrypto.IsEnvelope(rawNwk), "stored nwk_key must be a keycrypto envelope")
	assert.True(t, keycrypto.IsEnvelope(rawApp), "stored app_key must be a keycrypto envelope")
	assert.False(t, bytes.Contains(rawNwk, nwkKey), "plaintext nwk_key must not appear in stored bytes")
	assert.False(t, bytes.Contains(rawApp, appKey), "plaintext app_key must not appear in stored bytes")

	// Read path must decrypt back to the original key material.
	got, err := repo.Get(ctx, eui)
	require.NoError(t, err)
	assert.Equal(t, nwkKey, got.NwkSnKey, "decrypted nwk_key must equal original")
	assert.Equal(t, appKey, got.AppKey, "decrypted app_key must equal original")

	// Update rotates the network key; the new value must also be encrypted.
	// Reuse the original object (which carries the full field set) so the
	// update only changes the key.
	newNwkKey := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00}
	endpoint.NwkSnKey = append([]byte(nil), newNwkKey...)
	require.NoError(t, repo.Update(ctx, endpoint))

	rawNwk2, _ := selectRawEndpointKeys(t, db, eui)
	assert.True(t, keycrypto.IsEnvelope(rawNwk2), "updated nwk_key must be a keycrypto envelope")
	assert.False(t, bytes.Contains(rawNwk2, newNwkKey), "plaintext updated nwk_key must not appear in stored bytes")

	reread, err := repo.Get(ctx, eui)
	require.NoError(t, err)
	assert.Equal(t, newNwkKey, reread.NwkSnKey, "decrypted updated nwk_key must equal new value")
}

// TestEndpointRegistrationUpdate_EncryptsNwkKeyAtRest verifies that the SCACI
// registration path (§3.6) stores nwk_key as a keycrypto envelope and never as
// plaintext, and that the repository read path decrypts it back to the
// registered key material.
func TestEndpointRegistrationUpdate_EncryptsNwkKeyAtRest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 710, "TestTenant710")
	cleanupEndpointTestData(t, db, "RegEnc%")
	defer cleanupEndpointTestData(t, db, "RegEnc%")

	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x10}
	endpoint := &models.EndPoint{
		EUI:      eui,
		Name:     "RegEnc-EP1",
		TenantID: 710,
		EPClass:  "A",
		Tags:     make(map[string]string),
	}
	require.NoError(t, repo.Create(ctx, endpoint))

	created, err := repo.GetByEUI(ctx, 710, eui[:])
	require.NoError(t, err)

	nwkKey := []byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30}

	require.NoError(t, repo.EndpointRegistrationUpdate(ctx, 710, created.ID, models.EndpointRegistrationParams{
		NwkKey:    nwkKey,
		Bidi:      true,
		ShAddr:    0x1234,
		AttachCnt: 5,
		PacketCnt: 6,
	}))

	// Raw column bytes must be an envelope and must NOT contain the plaintext key.
	rawNwk, _ := selectRawEndpointKeys(t, db, eui)
	require.True(t, keycrypto.IsEnvelope(rawNwk), "SCACI-registered nwk_key must be a keycrypto envelope")
	assert.False(t, bytes.Contains(rawNwk, nwkKey), "plaintext nwk_key must not appear in stored bytes")

	// Read path must decrypt back to the registered key material.
	got, err := repo.GetByEUI(ctx, 710, eui[:])
	require.NoError(t, err)
	assert.Equal(t, nwkKey, got.NwkSnKey, "decrypted nwk_key must equal the registered key")
}

// TestEndpointKeys_NilAppKeyStaysNull verifies that a nil app key is stored as
// SQL NULL rather than an all-zero placeholder envelope.
func TestEndpointKeys_NilAppKeyStaysNull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 701, "TestTenant701")
	cleanupEndpointTestData(t, db, "KeyNull%")
	defer cleanupEndpointTestData(t, db, "KeyNull%")

	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x02}
	nwkKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	endpoint := &models.EndPoint{
		EUI:      eui,
		Name:     "KeyNull-EP1",
		TenantID: 701,
		EPClass:  "A",
		NwkSnKey: append([]byte(nil), nwkKey...),
		AppKey:   nil,
		Tags:     make(map[string]string),
	}
	require.NoError(t, repo.Create(ctx, endpoint))

	var appIsNull bool
	require.NoError(t, db.Get(&appIsNull,
		"SELECT app_key IS NULL FROM endpoints WHERE ep_eui = $1", eui[:]))
	assert.True(t, appIsNull, "app_key must be SQL NULL when no application key is provided")

	got, err := repo.Get(ctx, eui)
	require.NoError(t, err)
	assert.Empty(t, got.AppKey, "read-back app_key must be empty for a NULL column")
	assert.Equal(t, nwkKey, got.NwkSnKey)
}

// TestEndpointKeys_MagicPrefixedPlaintextIsEncrypted pins the write-path
// invariant: a key whose bytes begin with the envelope magic is still key
// material and must be encrypted like any other - the writer performs no
// format detection a caller could exploit to store cleartext.
func TestEndpointKeys_MagicPrefixedPlaintextIsEncrypted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 702, "TestTenant702")
	cleanupEndpointTestData(t, db, "KeyEncMagic%")
	defer cleanupEndpointTestData(t, db, "KeyEncMagic%")

	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x03}
	// A 16-byte key that begins with the envelope magic "KCE1".
	trickKey := append([]byte("KCE1"), []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}...)
	require.Len(t, trickKey, 16)

	endpoint := &models.EndPoint{
		EUI:      eui,
		Name:     "KeyEncMagic-EP1",
		TenantID: 702,
		EPClass:  "A",
		NwkSnKey: append([]byte(nil), trickKey...),
		Tags:     make(map[string]string),
	}
	require.NoError(t, repo.Create(ctx, endpoint))

	rawNwk, _ := selectRawEndpointKeys(t, db, eui)
	assert.False(t, bytes.Equal(rawNwk, trickKey),
		"a magic-prefixed key must be encrypted, never stored verbatim")

	loaded, err := repo.GetByEUI(ctx, 702, eui[:])
	require.NoError(t, err)
	assert.Equal(t, trickKey, loaded.NwkSnKey, "the key must round-trip intact")
}

// TestEndpointKeys_NonEnvelopeAndTamperedReadsFailClosed pins the strict read
// path: raw pre-envelope bytes and a tampered envelope are read errors, never
// silently returned as key material.
func TestEndpointKeys_NonEnvelopeAndTamperedReadsFailClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 703, "TestTenant703")
	cleanupEndpointTestData(t, db, "KeyEncStrict%")
	defer cleanupEndpointTestData(t, db, "KeyEncStrict%")

	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	// Raw 16-byte pre-envelope key written straight into the column.
	euiRaw := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x04}
	rawKey := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	_, err := db.Exec(`
		INSERT INTO endpoints (ep_eui, name, description, tenant_id, owner_tenant_id, nwk_key)
		VALUES ($1, 'KeyEncStrict-Raw', '', 703, 703, $2)`, euiRaw[:], rawKey)
	require.NoError(t, err)

	_, err = repo.GetByEUI(ctx, 703, euiRaw[:])
	require.ErrorIs(t, err, ErrKeyMaterialNotEnvelope,
		"a non-envelope value must fail the read, not pass through as cleartext")

	// A valid envelope with one flipped ciphertext byte must fail authentication.
	envelope, err := testsupport.TestCipher().Encrypt(rawKey)
	require.NoError(t, err)
	tampered := append([]byte(nil), envelope...)
	tampered[len(tampered)-1] ^= 0x01
	euiTampered := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x05}
	_, err = db.Exec(`
		INSERT INTO endpoints (ep_eui, name, description, tenant_id, owner_tenant_id, nwk_key)
		VALUES ($1, 'KeyEncStrict-Tampered', '', 703, 703, $2)`, euiTampered[:], tampered)
	require.NoError(t, err)

	_, err = repo.GetByEUI(ctx, 703, euiTampered[:])
	require.Error(t, err, "a tampered envelope must fail authentication")
	require.NotErrorIs(t, err, ErrKeyMaterialNotEnvelope)
}
