// Command rekey converts legacy at-rest key material to the authenticated
// pkg/keycrypto envelope format and reconciles the retired endpoint_keys
// subsystem, so the runtime readers can require valid envelopes and migration
// 000146+ guards can prove the database clean.
//
// Surfaces:
//   - the BYTEA key columns listed by postgres.KeyMaterialColumns
//     (endpoints, endpoint_sessions, messages, messages_archive)
//   - the key fields listed by postgres.KeyMaterialFields:
//     basestations.tls_key (TEXT), bssci_pending_operations.metadata ->>
//     'encryptedKey' (JSONB text field) and operation_data -> 'nwkSnKey'
//     (cleartext key of a recovery record persisted before key sanitization);
//     the command refuses to start when one of them has no scanner
//   - endpoint_keys / endpoint_keys_archive (reconcile + export + delete)
//
// Modes:
//   - dry-run (default): classify every value, print counts, change nothing.
//   - apply: convert legacy and plaintext values to envelopes row by row, each
//     conversion verified by a decrypt-and-compare read-back inside its own
//     transaction. Then reconcile endpoint_keys.
//   - verify: classify and exit nonzero unless every surface holds only valid
//     envelopes (or NULL) and the endpoint_keys tables are gone or empty.
//
// Legacy formats are decrypted only when their key is explicitly supplied:
// the retired KC-Core/pkg/crypto scheme derived an AES-256-GCM key as
// SHA-256 of a passphrase. A deployment that set KC_ENCRYPTION_KEY passes
// that passphrase via the environment variable named by
// -legacy-passphrase-env. A deployment that never set it - every Helm and
// Docker Compose install up to v1.3.0 - sealed its rows under the published
// development passphrase, and converting them requires
// -allow-published-development-key. Without the matching option, legacy
// ciphertext is counted and reported but never silently skipped as clean.
//
// Output contains only counts and HMAC-SHA256 digests keyed by the master
// key - never key material.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strconv"

	_ "github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

const (
	modeDryRun = "dry-run"
	modeApply  = "apply"
	modeVerify = "verify"

	envMasterKey = keycrypto.EnvMasterKey

	// publishedDevPassphrase is the passphrase KC-Core/pkg/crypto used
	// whenever KC_ENCRYPTION_KEY was unset. It is public in the repository
	// history, so rows sealed under it were never secret; the explicit flag
	// makes converting them a recorded operator decision.
	publishedDevPassphrase = "development-key-do-not-use-in-production-2024"

	// Connection defaults mirror the development docker-compose stack.
	dsnFormat         = "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s"
	defaultDBPort     = "5432"
	defaultDBHost     = "localhost"
	defaultDBName     = "kilocenter"
	defaultDBUser     = "kilocenter"
	defaultDBPassword = "changeme"
	defaultDBSSLMode  = "disable"
)

func main() {
	var (
		mode = flag.String("mode", modeDryRun,
			"dry-run (classify only), apply (convert + reconcile), verify (fail unless clean)")
		legacyPassphraseEnv = flag.String("legacy-passphrase-env", "",
			"name of the environment variable holding the retired KC_ENCRYPTION_KEY passphrase")
		allowPublishedDevKey = flag.Bool("allow-published-development-key", false,
			"decrypt legacy rows sealed under the published development passphrase, which every deployment that never set KC_ENCRYPTION_KEY used; required to convert those rows")
		archiveExport = flag.String("archive-export", "",
			"path for the encrypted endpoint_keys/archive export (required by apply when archive rows exist)")
	)
	flag.Parse()

	if *mode != modeDryRun && *mode != modeApply && *mode != modeVerify {
		fatalf(fatalFmtUnknownMode, *mode)
	}

	cipher, err := keycrypto.NewCipherFromMasterKey(os.Getenv(envMasterKey))
	if err != nil {
		fatalf("%s: %v", envMasterKey, err)
	}
	masterKey, err := keycrypto.ParseMasterKey(os.Getenv(envMasterKey))
	if err != nil {
		fatalf("%s: %v", envMasterKey, err)
	}

	var legacy *legacyCipher
	switch {
	case *legacyPassphraseEnv != "" && *allowPublishedDevKey:
		fatalf(fatalMsgLegacyFlagsConflict)
	case *legacyPassphraseEnv != "":
		passphrase := os.Getenv(*legacyPassphraseEnv)
		if passphrase == "" {
			fatalf(fatalFmtEnvVarEmpty, *legacyPassphraseEnv)
		}
		legacy = newLegacyCipher(passphrase)
	case *allowPublishedDevKey:
		fmt.Println(warnPublishedDevelopmentKey)
		legacy = newLegacyCipher(publishedDevPassphrase)
	}

	dsn, err := dsnFromEnv()
	if err != nil {
		fatalf("%v", err)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fatalf(fatalFmtOpenDatabase, err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, fatalPrefix+warnFmtCloseDatabase+"\n", err)
		}
	}()
	ctx := context.Background() // context-root: process
	if err := db.PingContext(ctx); err != nil {
		fatalf(fatalFmtConnectDatabase, err)
	}

	r := &rekeyer{
		db:      db,
		cipher:  cipher,
		legacy:  legacy,
		hmacKey: masterKey,
		apply:   *mode == modeApply,
	}

	report, err := r.run(ctx)
	if err != nil {
		fatalf("%v", err)
	}

	if *mode == modeApply {
		if err := r.reconcileEndpointKeys(ctx, report, *archiveExport); err != nil {
			fatalf(fatalFmtReconcile, err)
		}
	} else {
		if err := r.countEndpointKeys(ctx, report); err != nil {
			fatalf(fatalFmtInspect, err)
		}
	}

	report.print(*mode)

	if *mode == modeVerify && !report.clean() {
		os.Exit(1)
	}
	if !report.applyComplete() && *mode == modeApply {
		os.Exit(1)
	}
}

func dsnFromEnv() (string, error) {
	port, err := strconv.Atoi(getEnv("DB_PORT", defaultDBPort))
	if err != nil {
		return "", fmt.Errorf(errFmtInvalidDBPort, err)
	}
	return fmt.Sprintf(dsnFormat,
		getEnv("DB_HOST", defaultDBHost), port,
		getEnv("DB_NAME", defaultDBName),
		getEnv("DB_USER", defaultDBUser),
		getEnv("DB_PASSWORD", defaultDBPassword),
		getEnv("DB_SSLMODE", defaultDBSSLMode)), nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fatalPrefix+format+"\n", args...)
	os.Exit(2)
}

// keyedDigest returns a short HMAC-SHA256 digest of value under the master
// key, so reports can correlate values without exposing them.
func keyedDigest(key, value []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(value)
	return hex.EncodeToString(mac.Sum(nil)[:8])
}
