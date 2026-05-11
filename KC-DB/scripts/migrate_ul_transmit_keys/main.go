// Package main provides a UL transmit key migration script for BSSCI §5.11 security.
package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// UL transmit key migration script for BSSCI §5.11 security
//
// Migrates bssci_pending_operations.operation_data (network session keys) from
// plaintext to encrypted format using KILOCENTER_MASTER_KEY.
//
// IMPORTANT: This script works with existing JSONB schema:
//   - operation_data: JSONB containing the operation message
//   - metadata: JSONB with isEncrypted flag
//
// Expected operation_data structure for ulDataTx operations:
//   {
//     "nwkSnKey": "base64-encoded-plaintext-key",  // or "nwkSnKeyEncrypted"
//     ... other fields ...
//   }
//
// Expected metadata structure:
//   {
//     "isEncrypted": "false",  // string boolean
//     ... other fields ...
//   }
//
// Usage:
//   KILOCENTER_MASTER_KEY=<key> go run migrate_ul_transmit_keys.go [flags]
//
// Flags:
//   -db-host string        Database host (default "localhost")
//   -db-port int           Database port (default 5432)
//   -db-name string        Database name (default "kilocenter")
//   -db-user string        Database user (default "kilocenter")
//   -db-password string    Database password
//   -dry-run               Print migration plan without modifying data (default false)
//   -batch-size int        Number of rows to process per batch (default 500)
//   -max-retries int       Maximum retry attempts per operation (default 3)
//   -key-field string      JSONB field name for network key (default "nwkSnKey")

func main() {
	// Parse flags
	dbHost := flag.String("db-host", "localhost", "Database host")
	dbPort := flag.Int("db-port", 5432, "Database port")
	dbName := flag.String("db-name", "kilocenter", "Database name")
	dbUser := flag.String("db-user", "kilocenter", "Database user")
	dbPassword := flag.String("db-password", "", "Database password")
	dryRun := flag.Bool("dry-run", false, "Print migration plan without modifying data")
	batchSize := flag.Int("batch-size", 500, "Number of rows to process per batch")
	maxRetries := flag.Int("max-retries", 3, "Maximum retry attempts per operation")
	keyField := flag.String("key-field", "nwkSnKey", "JSONB field name for network session key")
	flag.Parse()

	// Initialize logger
	logger.Initialize("info", "json")
	log := logger.Get()

	if *dryRun {
		log.Info("Running in DRY-RUN mode - no data will be modified")
	}

	log.Info("Migration configuration",
		"key_field", *keyField,
		"batch_size", *batchSize,
		"max_retries", *maxRetries)

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
		log.Error("Failed to initialize key encryptor - ensure KILOCENTER_MASTER_KEY is set", "error", err)
		os.Exit(1)
	}

	// Verification query: count operations with plaintext keys
	verifyQuery := `
		SELECT COUNT(*)
		FROM bssci_pending_operations
		WHERE operation_type = 'ulDataTx'
		  AND (metadata->>'isEncrypted' = 'false' OR metadata->>'isEncrypted' IS NULL)
	`
	var plaintextCount int
	if err := db.QueryRowContext(ctx, verifyQuery).Scan(&plaintextCount); err != nil {
		log.Error("Failed to run verification query", "error", err)
		os.Exit(1)
	}
	log.Info("Verification: operations with plaintext keys", "count", plaintextCount)

	if plaintextCount == 0 {
		log.Info("No plaintext keys found - migration not needed")
		return
	}

	// Query for UL transmit operations with plaintext keys
	// Using id instead of operation_id for primary key
	query := `
		SELECT id, operation_data, metadata
		FROM bssci_pending_operations
		WHERE operation_type = 'ulDataTx'
		  AND (metadata->>'isEncrypted' = 'false' OR metadata->>'isEncrypted' IS NULL)
		ORDER BY id
		LIMIT $1
	`

	// Statistics
	stats := struct {
		total       int
		migrated    int
		skipped     int
		failed      int
		retryFailed int
		startTime   time.Time
	}{
		startTime: time.Now(),
	}

	// Process in batches with offset-based pagination
	offset := 0
	for {
		log.Info("Processing batch...", "offset", offset, "batch_size", *batchSize)

		rows, err := db.QueryContext(ctx, query, *batchSize)
		if err != nil {
			log.Error("Failed to query operations", "error", err, "offset", offset)
			os.Exit(1)
		}

		batch := make([]operationToMigrate, 0, *batchSize)
		for rows.Next() {
			var op operationToMigrate
			var operationDataJSON []byte
			var metadataJSON []byte

			if err := rows.Scan(&op.id, &operationDataJSON, &metadataJSON); err != nil {
				log.Error("Failed to scan row", "error", err)
				stats.failed++
				continue
			}

			// Parse JSONB columns
			if err := json.Unmarshal(operationDataJSON, &op.operationData); err != nil {
				log.Error("Failed to parse operation_data JSONB", "id", op.id, "error", err)
				stats.failed++
				continue
			}
			if err := json.Unmarshal(metadataJSON, &op.metadata); err != nil {
				log.Warn("Failed to parse metadata JSONB - using empty object", "id", op.id, "error", err)
				op.metadata = make(map[string]interface{})
			}

			batch = append(batch, op)
			stats.total++
		}
		if err := rows.Close(); err != nil {
			log.Warn("Failed to close rows", "error", err)
		}

		if len(batch) == 0 {
			// No more rows to process
			break
		}

		// Process batch with retry logic
		processBatchWithRetry(ctx, db, keyEncryptor, batch, *keyField, *dryRun, *maxRetries, &stats, log)

		offset += len(batch)
	}

	// Final verification query
	if !*dryRun {
		var remainingCount int
		if err := db.QueryRowContext(ctx, verifyQuery).Scan(&remainingCount); err != nil {
			log.Error("Failed to run final verification query", "error", err)
		} else {
			log.Info("Final verification: remaining plaintext keys", "count", remainingCount)
		}
	}

	// Print summary
	duration := time.Since(stats.startTime)
	log.Info("Migration complete",
		"total_operations", stats.total,
		"migrated", stats.migrated,
		"skipped", stats.skipped,
		"failed", stats.failed,
		"retry_failed", stats.retryFailed,
		"duration", duration,
		"dry_run", *dryRun,
	)

	if stats.failed > 0 || stats.retryFailed > 0 {
		os.Exit(1)
	}
}

type operationToMigrate struct {
	id            int64
	operationData map[string]interface{}
	metadata      map[string]interface{}
}

func processBatchWithRetry(
	ctx context.Context,
	db *sql.DB,
	keyEncryptor *crypto.KeyEncryptor,
	batch []operationToMigrate,
	keyField string,
	dryRun bool,
	maxRetries int,
	stats *struct {
		total       int
		migrated    int
		skipped     int
		failed      int
		retryFailed int
		startTime   time.Time
	},
	log logger.Logger,
) {
	for _, op := range batch {
		// Try migration with retries
		migrated := false
		var lastErr error

		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := migrateOperation(ctx, db, keyEncryptor, op, keyField, dryRun, log)
			if err == nil {
				stats.migrated++
				migrated = true
				break
			}

			// Check if this is a skip condition (not a failure)
			if err.Error() == "skipped" {
				stats.skipped++
				migrated = true
				break
			}

			lastErr = err
			log.Warn("Migration attempt failed",
				"id", op.id,
				"attempt", attempt,
				"max_retries", maxRetries,
				"error", err,
			)

			// Exponential backoff
			if attempt < maxRetries {
				backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
				time.Sleep(backoff)
			}
		}

		if !migrated {
			log.Error("Migration failed after retries",
				"id", op.id,
				"max_retries", maxRetries,
				"error", lastErr,
			)
			stats.retryFailed++
		}
	}
}

func migrateOperation(
	ctx context.Context,
	db *sql.DB,
	keyEncryptor *crypto.KeyEncryptor,
	op operationToMigrate,
	keyField string,
	dryRun bool,
	log logger.Logger,
) error {
	// Extract network session key from operation_data
	keyValue, ok := op.operationData[keyField]
	if !ok {
		log.Warn("Key field not found in operation_data - skipping",
			"id", op.id,
			"key_field", keyField,
			"available_fields", getKeys(op.operationData),
		)
		return fmt.Errorf("skipped")
	}

	// Extract epEui for logging (if available in operation_data)
	epEui := ""
	if epEuiVal, ok := op.operationData["epEui"]; ok {
		if epEuiStr, ok := epEuiVal.(string); ok {
			epEui = epEuiStr
		}
	}

	// Handle key as string (base64-encoded)
	keyStr, ok := keyValue.(string)
	if !ok {
		log.Warn("Key field is not a string - skipping",
			"id", op.id,
			"ep_eui", epEui,
			"key_field", keyField,
			"type", fmt.Sprintf("%T", keyValue),
		)
		return fmt.Errorf("skipped")
	}

	// 1. Decode base64 from database storage
	rawKeyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		log.Warn("Failed to decode base64 key",
			"id", op.id,
			"ep_eui", epEui,
			"error", err,
		)
		return fmt.Errorf("failed to decode base64 key: %w", err)
	}

	// 2. Validate decoded key is exactly 16 bytes (128-bit MIOTY key)
	if len(rawKeyBytes) != 16 {
		log.Warn("Skipping operation - decoded key is not 16 bytes",
			"id", op.id,
			"ep_eui", epEui,
			"length", len(rawKeyBytes),
			"expected", 16,
		)
		return fmt.Errorf("skipped")
	}

	// 3. Encrypt the actual key bytes (NOT the base64 string)
	encryptedKey, err := keyEncryptor.EncryptKey(rawKeyBytes)
	if err != nil {
		log.Error("Failed to encrypt key",
			"id", op.id,
			"ep_eui", epEui,
			"error", err,
		)
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	// 4. Validate encryption returned non-empty result
	if encryptedKey == "" {
		log.Error("Encrypted key empty - EncryptKey returned empty string",
			"id", op.id,
			"ep_eui", epEui,
		)
		return fmt.Errorf("encrypted key empty")
	}

	// 5. Use encrypted key directly (already base64 encoded by EncryptKey)
	encryptedKeyB64 := encryptedKey

	if dryRun {
		log.Info("Would migrate operation",
			"id", op.id,
			"key_field", keyField,
			"old_length", len(keyStr),
			"new_length", len(encryptedKeyB64),
		)
		return nil
	}

	// Update operation_data with encrypted key
	op.operationData[keyField] = encryptedKeyB64

	// Update metadata to mark as encrypted
	if op.metadata == nil {
		op.metadata = make(map[string]interface{})
	}
	op.metadata["isEncrypted"] = "true"

	// Marshal back to JSONB
	operationDataJSON, err := json.Marshal(op.operationData)
	if err != nil {
		return fmt.Errorf("failed to marshal operation_data: %w", err)
	}

	metadataJSON, err := json.Marshal(op.metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Update operation with encrypted key
	updateQuery := `
		UPDATE bssci_pending_operations
		SET operation_data = $1::jsonb,
		    metadata = $2::jsonb
		WHERE id = $3
	`
	result, err := db.ExecContext(ctx, updateQuery, operationDataJSON, metadataJSON, op.id)
	if err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected != 1 {
		return fmt.Errorf("unexpected rows affected: %d", rowsAffected)
	}

	log.Info("Migrated operation",
		"id", op.id,
		"key_field", keyField,
		"old_length", len(keyStr),
		"new_length", len(encryptedKeyB64),
	)

	return nil
}

// getKeys returns the keys of a map for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
