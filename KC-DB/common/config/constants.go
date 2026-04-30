// Package config provides centralized configuration constants for KiloCenter
// These constants should be used instead of hardcoding values throughout the codebase
package config

import (
	"time"
)

// Service Ports - Default ports for each service
const (
	// Database
	DefaultPostgresPort     = 5432
	DevelopmentPostgresPort = 5433

	// API Service
	DefaultAPIPort  = 8080
	ExternalAPIPort = 8090 // Docker mapped port

	// BSSCI Protocol
	DefaultBSSCIPort = 5000

	// Web Frontend
	DefaultWebPort = 5173 // Vite dev server

	// MQTT
	DefaultMQTTPort    = 1883
	DefaultMQTTTLSPort = 8883

	// gRPC
	DefaultGRPCPort = 50051

	// Health Check
	DefaultHealthPort = 8086
)

// Database Configuration
const (
	// Connection pools
	DefaultMaxOpenConns    = 25
	DefaultMaxIdleConns    = 5
	DefaultConnMaxLifetime = 5 * time.Minute
	DefaultConnMaxIdleTime = 5 * time.Minute

	// Query timeouts
	DefaultQueryTimeout = 30 * time.Second
	DefaultTxTimeout    = 60 * time.Second

	// Retry configuration
	DefaultRetryAttempts = 3
	DefaultRetryDelay    = 100 * time.Millisecond
)

// Test Database Configuration
// Used by: KC-DB/storage/postgres/test_helpers.go
// These defaults match docker-compose.dev.yml host port mapping
// CI/CD environments override via TEST_DB_* environment variables
const (
	// TestDBDefaultHost is the default database host for integration tests
	TestDBDefaultHost = "localhost"

	// TestDBDefaultPort is the default database port for integration tests
	// Matches docker-compose.dev.yml host port mapping (5433→5432)
	TestDBDefaultPort = "5433"

	// TestDBDefaultUser is the default database user for integration tests
	TestDBDefaultUser = "kilocenter"

	// TestDBDefaultPassword is the default database password for integration tests
	TestDBDefaultPassword = "changeme"

	// TestDBDefaultName is the default database name for integration tests
	TestDBDefaultName = "kilocenter_test"

	// TestDBDefaultSSLMode is the default SSL mode for integration tests
	TestDBDefaultSSLMode = "disable"
)

// MIOTY Protocol Constants
const (
	// Note: BSSCI Protocol Version moved to KC-DB/storage/mioty/types.go (MIOTYProtocolVersion)

	// Message deduplication window per MIOTY specification
	DeduplicationWindow = 5 * time.Minute

	// Operation timeout
	OperationTimeout = 30 * time.Second

	// Ping mechanism
	PingInterval = 60 * time.Second
	PingTimeout  = 10 * time.Second

	// BSSCI Session Management (Section 3.3)
	// Maximum age for session resumption eligibility
	// After this duration, Base Stations must initiate new sessions
	MaxSessionResumptionAge = 24 * time.Hour

	// Max message sizes
	MaxMessageSize = 1024 * 1024 // 1MB
	MaxPayloadSize = 256 * 1024  // 256KB

	// EUI sizes
	EUISize = 8 // 8 bytes for EUI64

	// MQTT configuration defaults
	DefaultMQTTPingTimeout = 10 * time.Second
	DefaultMQTTTopicPrefix = "mioty"
)

// API Configuration
const (
	// Pagination
	DefaultPageSize = 20
	MaxPageSize     = 100
	DefaultPage     = 1

	// Rate limiting
	DefaultRateLimit = 100 // requests per minute
	DefaultBurstSize = 20

	// Timeouts
	DefaultReadTimeout  = 15 * time.Second
	DefaultWriteTimeout = 15 * time.Second
	DefaultIdleTimeout  = 60 * time.Second

	// CORS
	DefaultCORSMaxAge = 86400 // 24 hours in seconds
)

// File Storage - Paths are configurable via environment
const (
	// File permissions
	DefaultDirPerm  = 0755
	DefaultFilePerm = 0644
	SecureFilePerm  = 0600 // For certificates and keys
)

// Archival Configuration
const (
	// Retention periods
	DefaultMessageRetention = 90 * 24 * time.Hour  // 90 days
	DefaultEventRetention   = 180 * 24 * time.Hour // 180 days

	// Batch sizes
	DefaultArchivalBatchSize = 1000
	DefaultArchivalInterval  = 24 * time.Hour
)

// Environment Variables
const (
	// Database
	EnvDBHost       = "DB_HOST"
	EnvDBPort       = "DB_PORT"
	EnvDBName       = "DB_NAME"
	EnvDBUser       = "DB_USER"
	EnvDBCredential = "DB_PASSWORD" // Using credential instead
	EnvDBSSLMode    = "DB_SSLMODE"

	// API
	EnvAPIAddress = "SERVER_ADDRESS"
	EnvAPIURL     = "API_URL"

	// BSSCI
	EnvBSSCIAddress  = "BSSCI_ADDRESS"
	EnvBSSCICertFile = "BSSCI_CERT_FILE"
	EnvBSSCIKeyFile  = "BSSCI_KEY_FILE"

	// General
	EnvEnvironment = "ENVIRONMENT"
	EnvLogLevel    = "LOG_LEVEL"
	EnvConfigPath  = "CONFIG_PATH"

	// Paths
	EnvCertDir = "CERT_DIR"
	EnvDataDir = "DATA_DIR"
	EnvLogDir  = "LOG_DIR"
)

// Environments
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
	EnvTest        = "test"
)

// Log Levels
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)
