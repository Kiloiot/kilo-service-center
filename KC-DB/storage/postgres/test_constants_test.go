package postgres

// Test constants for KC-DB test suite
// These constants are only available during testing (excluded from production builds)

const (
	// testEventCategoryBSSCI is the event category used for BSSCI-related system events
	// This aligns with the event_category filter used in operation_status_repository.go
	// Uses "system" category as BSSCI events are system-level protocol events
	testEventCategoryBSSCI = "system"
)
