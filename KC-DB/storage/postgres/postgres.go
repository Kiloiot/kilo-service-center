// Package postgres provides PostgreSQL database implementation for KiloCenter
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// DB represents a PostgreSQL database connection
type DB struct {
	conn    *sql.DB
	sqlxDB  *sqlx.DB
	clock   clock.Clock
	manager *ConnectionManager
	log     logger.Logger
	cipher  keycrypto.Cipher
}

const componentPostgres = "postgres"

// tenantQueueReadLimit bounds the tenant downlink queue listing.
const tenantQueueReadLimit = 1000

// queueIDThreshold separates queue IDs from small legacy row IDs when
// resolving a downlink identifier.
const queueIDThreshold = 1000000

// New creates a new PostgreSQL database connection with retry logic. The cipher
// is mandatory: it encrypts all key material at rest, so a nil cipher is a
// configuration error rather than a silent plaintext fallback.
// componentPostgres labels this component in structured logs.
func New(opts Options, cipher keycrypto.Cipher) (*DB, error) {
	if cipher == nil {
		return nil, errTextNilCipher
	}

	log := logger.Get().WithField(logger.FieldComponent, componentPostgres)

	manager := NewConnectionManager(opts.config(), log)

	// Connect with retry logic
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout) // context-root: process
	defer cancel()

	if err := manager.Connect(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapEstablishDatabaseConnection, err)
	}

	// Get the established connection
	conn := manager.GetDB()

	// Test the connection
	if err := conn.PingContext(ctx); err != nil {
		return nil, joinCloseError(fmt.Errorf("%s: %w", errWrapPingDatabase, err), errWrapCloseDatabase, manager.Close())
	}

	// Create sqlx wrapper for the connection
	sqlxDB := sqlx.NewDb(conn, "postgres")
	clk := opts.Clock
	if clk == nil {
		clk = clock.SystemClock{}
	}

	log.Info(logMsgPostgreSQLConnectionEstablished,
		logger.FieldHost, opts.Host,
		logger.FieldDatabase, opts.Database,
		logger.FieldMaxOpenConns, opts.MaxOpenConns,
		logger.FieldMaxIdleConns, opts.MaxIdleConns)

	db := &DB{
		conn:    conn,
		sqlxDB:  sqlxDB,
		clock:   clk,
		manager: manager,
		log:     log,
		cipher:  cipher,
	}

	// Initialize repositories

	return db, nil
}

// Ping checks the database connection with context
func (db *DB) Ping(ctx context.Context) error {
	return db.manager.HealthCheck(ctx)
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.manager.Close()
}

// Query executes a query that returns rows with retry logic
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	retryErr := db.manager.ExecuteWithRetry(ctx, func() error {
		rows, err = db.conn.QueryContext(ctx, query, args...)
		return err
	})

	if retryErr != nil {
		return nil, retryErr
	}
	return rows, nil
}

// QueryRow executes a query that returns at most one row
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRowContext(ctx, query, args...)
}

// Exec executes a query without returning any rows with retry logic
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error

	retryErr := db.manager.ExecuteWithRetry(ctx, func() error {
		result, err = db.conn.ExecContext(ctx, query, args...)
		return err
	})

	if retryErr != nil {
		return nil, retryErr
	}
	return result, nil
}

// WaitForConnection waits for the database to become available
func (db *DB) WaitForConnection(ctx context.Context, timeout time.Duration) error {
	return db.manager.WaitForConnection(ctx, timeout)
}

// Sqlx returns the facade's sqlx handle for repositories constructed by the
// composition roots.
func (db *DB) Sqlx() *sqlx.DB {
	return db.sqlxDB
}

// GetDB returns the underlying database connection
// This should be used sparingly and only for operations that need direct access
func (db *DB) GetDB() *sql.DB {
	return db.conn
}
