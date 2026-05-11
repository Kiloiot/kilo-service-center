// Package postgres provides PostgreSQL database connectivity with retry logic
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	_ "github.com/lib/pq" // Register PostgreSQL driver
)

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

// GetDSN returns the PostgreSQL connection string
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
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
	maxRetries := 5
	retryDelay := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		cm.logger.Info("Attempting database connection",
			"attempt", attempt,
			"max_retries", maxRetries)

		err := cm.tryConnect(ctx)
		if err == nil {
			cm.logger.Info("Successfully connected to database")
			return cm.configureConnection()
		}

		if attempt < maxRetries {
			cm.logger.Warn("Database connection failed, retrying",
				"error", err,
				"retry_delay", retryDelay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				// Exponential backoff with jitter
				retryDelay = time.Duration(float64(retryDelay) * 1.5)
				if retryDelay > 30*time.Second {
					retryDelay = 30 * time.Second
				}
			}
		} else {
			return fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
		}
	}

	return nil
}

// tryConnect attempts a single database connection
func (cm *ConnectionManager) tryConnect(ctx context.Context) error {
	dsn := cm.config.GetDSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close() // Best effort close
		return fmt.Errorf("failed to ping database: %w", err)
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

	cm.logger.Info("Database connection pool configured",
		"max_open_conns", cm.config.MaxOpenConns,
		"max_idle_conns", cm.config.MaxIdleConns,
		"conn_max_lifetime", cm.config.ConnMaxLifetime,
		"conn_max_idle_time", cm.config.ConnMaxIdleTime)

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
		return fmt.Errorf("database connection not established")
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := cm.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Verify we can execute a simple query
	var result int
	err := cm.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("test query failed: %w", err)
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
			return fmt.Errorf("timeout waiting for database connection: %w", ctx.Err())
		case <-ticker.C:
			if err := cm.HealthCheck(ctx); err == nil {
				return nil
			}
		}
	}
}

// ExecuteWithRetry executes a function with retry logic
func (cm *ConnectionManager) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		if attempt < maxRetries {
			cm.logger.Debug("Retrying database operation",
				"attempt", attempt,
				"error", err,
				"retry_delay", retryDelay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				retryDelay *= 2
			}
		} else {
			return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
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
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"deadlock detected",
		"could not serialize",
		"too many connections",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
