package postgres

// Test constants for KC-DB test suite
// These constants are only available during testing (excluded from production builds)

import "time"

const (
	// testEventCategoryBSSCI is the event category used for BSSCI-related system events
	// This aligns with the event_category filter used in operation_status_repository.go
	// Uses "system" category as BSSCI events are system-level protocol events
	testEventCategoryBSSCI = "system"

	// testContextTimeout bounds per-test database operations.
	testContextTimeout = 5 * time.Second
	// testEventWindow is the lookback window passed to operation/event queries.
	testEventWindow = 24 * time.Hour
	// testTokenTTL is the refresh token lifetime used by repository tests.
	testTokenTTL = 24 * time.Hour
	// testLockWaitWindow is how long a write must stay blocked behind a user row lock held by another transaction.
	testLockWaitWindow = 300 * time.Millisecond
	// testVerifiedPasswordHash is the stored password hash a sign-in verified.
	testVerifiedPasswordHash = "pbkdf2-verified-hash"
	// testNewerPasswordHash is a password hash stored after a sign-in verified the previous one.
	testNewerPasswordHash = "pbkdf2-newer-hash"
	// testAggregationWindow is the summary aggregation window used by event tests.
	testAggregationWindow = 5 * time.Minute
	// testEventQueryLimit is the default page size for event query tests.
	testEventQueryLimit = 10
	// testEventLimitFilterValue exercises the limit filter with fewer rows than exist.
	testEventLimitFilterValue = 3
	// testSessionListLimit is the page size for session listing tests.
	testSessionListLimit = 100
	// testDefaultPostgresPort is the fallback when the container port cannot be parsed.
	testDefaultPostgresPort = 5432
	// testCreatedAtSpacing separates inserts that must not share a created_at value.
	testCreatedAtSpacing = 10 * time.Millisecond
	// testEventTypeRoundtrip is the synthetic event type used by roundtrip tests.
	testEventTypeRoundtrip = "test.roundtrip"
)
