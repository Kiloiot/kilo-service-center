package testsupport

import "errors"

// Error-wrap contexts and error format strings, grouped by the file that
// uses them; the shared group serves multiple files.
const (
	// harness.go
	errWrapCloseTemplateDatabase         = "close template database"
	errWrapConnectMaintenanceDatabase    = "connect to maintenance database"
	errWrapConnectTESTDBOverride         = "connect to TEST_DB override"
	errWrapConnectTemplateDatabase       = "connect to template database"
	errWrapCreateMigrateInstance         = "failed to create migrate instance"
	errWrapCreatePostgresDriver          = "failed to create postgres driver"
	errWrapCreateTemplateDatabase        = "create template database"
	errWrapDropStaleTemplateDatabase     = "drop stale template database"
	errWrapMigrateTemplateDatabase       = "migrate template database"
	errWrapMigration                     = "migration failed"
	errWrapParseDSN                      = "failed to parse DSN"
	errWrapReadContainerConnectionString = "read container connection string"
	errWrapStartPostgresContainer        = "start postgres container"

	// Teardown warning formats; failures here never fail a test.
	warnFmtTerminateContainer   = "warning: failed to terminate shared postgres container: %v\n"
	warnFmtCloseSharedAdminDB   = "warning: failed to close shared admin database: %v\n"
	warnFmtDropTemplateDatabase = "warning: failed to drop the template database: %v\n"
	warnFmtDropTestDatabase     = "Warning: failed to drop test database %s: %v"
	warnFmtCloseDBConnection    = "Warning: failed to close database connection: %v"
	warnFmtCloseMigrateInstance = "warning: failed to close migrate instance: source=%v db=%v\n"

	// defaultPostgresPort is the TEST_DB_PORT fallback for the explicit
	// developer override path.
	defaultPostgresPort = "5432"
)

// Sentinel errors migrated from in-function errors.New literals.
var (
	// harness.go
	errTextGetCurrentFilePath = errors.New("failed to get current file path")
)
