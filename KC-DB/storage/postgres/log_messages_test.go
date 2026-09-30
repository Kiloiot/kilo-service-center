package postgres

import "errors"

// Error-wrap contexts and error format strings used only by tests, grouped
// by the file that uses them; the shared group serves multiple files.
const (
	// downlink_result_test.go
	errFmtQueueBelongsTenant100ButUpdate          = "queue %d belongs to tenant 100 but update failed: %w"
	errFmtQueueBelongsTenant200ButTenant100Update = "queue %d belongs to tenant 200 but tenant 100 update succeeded"

	// endpoint_repository_test.go
	errFmtQueryMissingNwkKeyColumnFound            = "query missing 'nwk_key' column. Found: %s"
	errFmtREGRESSIONQueryContainsNwkSnKeyWhichDoes = "REGRESSION: query contains 'nwk_sn_key' which does not exist in endpoints table. Found: %s"
	errWrapGoroutine1Failed                        = "goroutine 1 failed"
	errWrapGoroutine2Failed                        = "goroutine 2 failed"

	// migrations_test.go
	errFmtMigrateVersion         = "migrate to version %d: %w"
	errWrapCreateMigrateInstance = "create migrate instance"
	errWrapCreatePostgresDriver  = "create postgres driver"

	// schema_snapshot_test.go
	errFmtGetColumnsForTable  = "failed to get columns for table %s: %w"
	errFmtGetIndexesForTable  = "failed to get indexes for table %s: %w"
	errFmtGetTriggersForTable = "failed to get triggers for table %s: %w"
	errWrapGetTables          = "failed to get tables"
)

// Sentinel errors migrated from in-function errors.New literals.
var (
	// errors_test.go
	errTextGenericError = errors.New("generic error")
	errTextSomeError    = errors.New("some error")
)
