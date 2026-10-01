package builders

import (
	"errors"
	"time"
)

// Sentinel errors for infrastructure bring-up failures.
var (
	// ErrStorageInit reports a failed storage-layer initialization.
	ErrStorageInit = errors.New("failed to initialize storage")

	// ErrCipherInit reports a missing or malformed key-material master key.
	ErrCipherInit = errors.New("failed to build key-material cipher from KILOCENTER_MASTER_KEY")

	// ErrDatabaseConnectTimeout reports that the database did not become reachable in time.
	ErrDatabaseConnectTimeout = errors.New("database connection timeout")

	// ErrMigrationsFailed reports a failed database migration run.
	ErrMigrationsFailed = errors.New("failed to run database migrations")

	// ErrCEDefaultOrgRequired reports a CE deployment whose configured tenant has no default org.
	ErrCEDefaultOrgRequired = errors.New("CE requires default org for tenant")

	// ErrOrgResolverInit reports an organization resolver that could not be built.
	ErrOrgResolverInit = errors.New("failed to build organization resolver")
)

// databaseConnectTimeout bounds the startup wait for database reachability.
const databaseConnectTimeout = 30 * time.Second
