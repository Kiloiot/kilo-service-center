// Package postgres provides PostgreSQL database connectivity with retry logic
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	_ "github.com/lib/pq" // Register PostgreSQL driver
)

// Options carries the database connection settings New needs to open a pool.
// Callers translate their application configuration into Options at the call
// site so the persistence layer stays independent of any app config package.
type Options struct {
	// Clock stamps rows; nil means the system clock.
	Clock           clock.Clock
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DSN is the connection string of the database the options name.
func (o Options) DSN() string {
	return o.config().GetDSN()
}

// config is the connection configuration the options describe.
func (o Options) config() *Config {
	return &Config{
		Host:            o.Host,
		Port:            o.Port,
		Database:        o.Database,
		Username:        o.Username,
		Password:        o.Password,
		SSLMode:         o.SSLMode,
		MaxOpenConns:    o.MaxOpenConns,
		MaxIdleConns:    o.MaxIdleConns,
		ConnMaxLifetime: o.ConnMaxLifetime,
		ConnMaxIdleTime: o.ConnMaxIdleTime,
	}
}

// Config holds PostgreSQL connection configuration
type Config struct {
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

const dsnFormat = "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s"

// connectRetryBackoffFactor grows the connect retry delay up to its cap.
const connectRetryBackoffFactor = 1.5

// retryablePatterns are the PostgreSQL error text fragments treated as
// transient.
var retryablePatterns = []string{
	"connection refused",
	"connection reset",
	"broken pipe",
	"deadlock detected",
	"could not serialize",
	"too many connections",
}

// GetDSN returns the PostgreSQL connection string
// dsnFormat renders the PostgreSQL connection string.
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		dsnFormat,
		c.Host,
		c.Port,
		c.Username,
		c.Password,
		c.Database,
		c.SSLMode,
	)
}

// ConnectionManager manages database connections with retry logic
type ConnectionManager struct {
	config *Config
	db     *sql.DB
	logger logger.Logger
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(config *Config, logger logger.Logger) *ConnectionManager {
	return &ConnectionManager{
		config: config,
		logger: logger,
	}
}

// Connect establishes a database connection with retry logic
func (cm *ConnectionManager) Connect(ctx context.Context) error {
	maxRetries := connectMaxRetries
	retryDelay := time.Second

	for attempt := initialRetryAttempt; attempt <= maxRetries; attempt++ {
		cm.logger.Info(logMsgAttemptingDatabaseConnection,
			logger.FieldAttempt, attempt,
			logger.FieldMaxRetries, maxRetries)

		err := cm.tryConnect(ctx)
		if err == nil {
			cm.logger.Info(logMsgSuccessfullyConnectedDatabase)
			return cm.configureConnection()
		}

		if attempt < maxRetries {
			cm.logger.Warn(logMsgDatabaseConnectionRetrying,
				logger.FieldError, err,
				logger.FieldRetryDelay, retryDelay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				// Exponential backoff with jitter
				retryDelay = time.Duration(float64(retryDelay) * connectRetryBackoffFactor)
				if retryDelay > connectRetryDelayMax {
					retryDelay = connectRetryDelayMax
				}
			}
		} else {
			return fmt.Errorf(errFmtConnectAfterAttempts, maxRetries, err)
		}
	}

	return nil
}

// tryConnect attempts a single database connection
func (cm *ConnectionManager) tryConnect(ctx context.Context) error {
	dsn := cm.config.GetDSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapOpenDatabase, err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(ctx, connectPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return joinCloseError(fmt.Errorf("%s: %w", errWrapPingDatabase, err), errWrapCloseDatabase, db.Close())
	}

	cm.db = db
	return nil
}

// configureConnection sets up connection pool parameters
func (cm *ConnectionManager) configureConnection() error {
	// Set connection pool settings
	cm.db.SetMaxOpenConns(cm.config.MaxOpenConns)
	cm.db.SetMaxIdleConns(cm.config.MaxIdleConns)
	cm.db.SetConnMaxLifetime(cm.config.ConnMaxLifetime)
	cm.db.SetConnMaxIdleTime(cm.config.ConnMaxIdleTime)

	cm.logger.Info(logMsgDatabaseConnectionPoolConfigured,
		logger.FieldMaxOpenConns, cm.config.MaxOpenConns,
		logger.FieldMaxIdleConns, cm.config.MaxIdleConns,
		logger.FieldConnMaxLifetime, cm.config.ConnMaxLifetime,
		logger.FieldConnMaxIdleTime, cm.config.ConnMaxIdleTime)

	return nil
}

// GetDB returns the database connection
func (cm *ConnectionManager) GetDB() *sql.DB {
	return cm.db
}

// Close closes the database connection
func (cm *ConnectionManager) Close() error {
	if cm.db != nil {
		return cm.db.Close()
	}
	return nil
}

// HealthCheck performs a health check on the database connection
func (cm *ConnectionManager) HealthCheck(ctx context.Context) error {
	if cm.db == nil {
		return errTextDatabaseConnectionNotEstablished
	}

	ctx, cancel := context.WithTimeout(ctx, healthCheckPingTimeout)
	defer cancel()

	if err := cm.db.PingContext(ctx); err != nil {
		return fmt.Errorf("%s: %w", errWrapDatabasePing, err)
	}

	// Verify we can execute a simple query
	var result int
	err := cm.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapTestQuery, err)
	}

	return nil
}

// Stats returns database connection pool statistics
func (cm *ConnectionManager) Stats() sql.DBStats {
	if cm.db == nil {
		return sql.DBStats{}
	}
	return cm.db.Stats()
}

// WaitForConnection waits for the database to become available
func (cm *ConnectionManager) WaitForConnection(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w", errWrapTimeoutWaitingForDatabaseConnection, ctx.Err())
		case <-ticker.C:
			if err := cm.HealthCheck(ctx); err == nil {
				return nil
			}
		}
	}
}

// ExecuteWithRetry executes a function with retry logic
func (cm *ConnectionManager) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	maxRetries := executeMaxRetries
	retryDelay := executeInitialRetryDelay

	for attempt := initialRetryAttempt; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		if attempt < maxRetries {
			cm.logger.Debug(logMsgRetryingDatabaseOperation,
				logger.FieldAttempt, attempt,
				logger.FieldError, err,
				logger.FieldRetryDelay, retryDelay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				retryDelay *= retryBackoffMultiplier
			}
		} else {
			return fmt.Errorf(errFmtOperationAfterRetries, maxRetries, err)
		}
	}

	return nil
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific PostgreSQL error codes that are retryable.
	// This is a simplified version; in production check specific error codes.
	errStr := strings.ToLower(err.Error())
	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
