// Package main provides a session key migration script for BSSCI §5.6 compliance.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/kilocenter/KC-Core/pkg/crypto"
	"github.com/kilocenter/KC-Core/pkg/logger"
)

// Session key migration script for BSSCI §5.6 compliance
//
// Migrates endpoint_sessions.session_key from legacy base64-encoded GCM format
// (60 bytes) to raw GCM blob format (44 bytes: 12-byte nonce + 16-byte ciphertext + 16-byte auth tag).
//
// Usage:
//   go run migrate_session_keys.go [flags]
//
// Flags:
//   -db-host string        Database host (default "localhost")
//   -db-port int           Database port (default 5432)
//   -db-name string        Database name (default "kilocenter")
//   -db-user string        Database user (default "kilocenter")
//   -db-password string    Database password
//   -dry-run               Print migration plan without modifying data (default false)
//   -batch-size int        Number of rows to process per batch (default 100)

func main() {
	// Parse flags
	dbHost := flag.String("db-host", "localhost", "Database host")
	dbPort := flag.Int("db-port", 5432, "Database port")
	dbName := flag.String("db-name", "kilocenter", "Database name")
	dbUser := flag.String("db-user", "kilocenter", "Database user")
	dbPassword := flag.String("db-password", "", "Database password")
	dryRun := flag.Bool("dry-run", false, "Print migration plan without modifying data")
	batchSize := flag.Int("batch-size", 100, "Number of rows to process per batch")
	flag.Parse()

	// Initialize logger
	logger.Initialize("info", "json")
	log := logger.Get()

	if *dryRun {
		log.Info("Running in DRY-RUN mode - no data will be modified")
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPassword, *dbName,
	)

	// Connect to database
	log.Info("Connecting to database...", "host", *dbHost, "port", *dbPort, "dbname", *dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Error("Failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Warn("Failed to close database", "error", err)
		}
	}()

	// Test connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	log.Info("Database connection established")

	// Initialize key encryptor
	log.Info("Initializing key encryptor...")
	keyEncryptor, err := crypto.NewKeyEncryptor()
	if err != nil {
		log.Error("Failed to initialize key encryptor", "error", err)
		os.Exit(1)
	}

	// Query for sessions with legacy key format (not 44 bytes)
	query := `
		SELECT session_id, tenant_id, session_key
		FROM endpoint_sessions
		WHERE session_key IS NOT NULL
		  AND LENGTH(session_key) != 44
		ORDER BY session_id
	`

	log.Info("Querying for sessions with legacy key format...")
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Error("Failed to query sessions", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Warn("Failed to close rows", "error", err)
		}
	}()

	// Statistics
	stats := struct {
		total     int
		migrated  int
		skipped   int
		failed    int
		startTime time.Time
	}{
		startTime: time.Now(),
	}

	// Process rows in batches
	batch := make([]sessionToMigrate, 0, *batchSize)

	for rows.Next() {
		var s sessionToMigrate
		if err := rows.Scan(&s.sessionID, &s.tenantID, &s.legacyKey); err != nil {
			log.Error("Failed to scan row", "error", err)
			stats.failed++
			continue
		}

		stats.total++
		batch = append(batch, s)

		// Process batch when full
		if len(batch) >= *batchSize {
			processBatch(ctx, db, keyEncryptor, batch, *dryRun, &stats, log)
			batch = batch[:0] // Reset batch
		}
	}

	// Process remaining rows
	if len(batch) > 0 {
		processBatch(ctx, db, keyEncryptor, batch, *dryRun, &stats, log)
	}

	if err := rows.Err(); err != nil {
		log.Error("Error iterating rows", "error", err)
	}

	// Print summary
	duration := time.Since(stats.startTime)
	log.Info("Migration complete",
		"total_sessions", stats.total,
		"migrated", stats.migrated,
		"skipped", stats.skipped,
		"failed", stats.failed,
		"duration", duration,
		"dry_run", *dryRun,
	)

	if stats.failed > 0 {
		os.Exit(1)
	}
}

type sessionToMigrate struct {
	sessionID int64
	tenantID  int64
	legacyKey []byte
}

func processBatch(
	ctx context.Context,
	db *sql.DB,
	keyEncryptor *crypto.KeyEncryptor,
	batch []sessionToMigrate,
	dryRun bool,
	stats *struct {
		total     int
		migrated  int
		skipped   int
		failed    int
		startTime time.Time
	},
	log logger.Logger,
) {
	for _, s := range batch {
		// Try decrypting as legacy base64 format
		clearKey, err := keyEncryptor.DecryptKey(string(s.legacyKey))
		if err != nil {
			// May already be raw GCM — try raw decrypt
			_, err = keyEncryptor.DecryptKeyRaw(s.legacyKey)
			if err != nil {
				log.Error("Failed to decrypt session key",
					"session_id", s.sessionID,
					"tenant_id", s.tenantID,
					"error", err,
				)
				stats.failed++
				continue
			}
			// Already in raw format
			stats.skipped++
			continue
		}

		// Re-encrypt in raw GCM format
		newEncrypted, err := keyEncryptor.EncryptKeyRaw(clearKey)
		if err != nil {
			log.Error("Failed to re-encrypt session key",
				"session_id", s.sessionID,
				"tenant_id", s.tenantID,
				"error", err,
			)
			stats.failed++
			continue
		}

		if dryRun {
			log.Info("Would migrate session key",
				"session_id", s.sessionID,
				"tenant_id", s.tenantID,
				"old_length", len(s.legacyKey),
				"new_length", len(newEncrypted),
			)
			stats.migrated++
			continue
		}

		// Persist migrated key (tenant-scoped)
		updateQuery := `
			UPDATE endpoint_sessions
			SET session_key = $1
			WHERE tenant_id = $2 AND session_id = $3
		`
		result, err := db.ExecContext(ctx, updateQuery, newEncrypted, s.tenantID, s.sessionID)
		if err != nil {
			log.Error("Failed to update session key",
				"session_id", s.sessionID,
				"tenant_id", s.tenantID,
				"error", err,
			)
			stats.failed++
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected != 1 {
			log.Error("Unexpected rows affected during update",
				"session_id", s.sessionID,
				"tenant_id", s.tenantID,
				"rows_affected", rowsAffected,
			)
			stats.failed++
			continue
		}

		log.Info("Migrated session key",
			"session_id", s.sessionID,
			"tenant_id", s.tenantID,
			"old_length", len(s.legacyKey),
			"new_length", len(newEncrypted),
		)
		stats.migrated++
	}
}
