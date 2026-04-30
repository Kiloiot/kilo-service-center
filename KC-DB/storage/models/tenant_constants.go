package models

// Tenant status constants - NO inline literals allowed.
const (
	TenantStatusActive    = "active"
	TenantStatusInactive  = "inactive"  // API-initiated deactivation
	TenantStatusSuspended = "suspended" // Administrative suspension
	TenantStatusArchived  = "archived"
)
