package orgresolver

// Log messages for organization resolution and cache maintenance.
const (
	LogOrgCacheFull        = "Org cache full, clearing all entries"
	LogOrgCacheHit         = "Org cache hit"
	LogOrgCacheMiss        = "Org cache miss, querying database"
	LogOrgCached           = "Org cached"
	LogOrgCertResolved     = "Resolved cert to org and tenant"
	LogOrgDefaultResolved  = "Resolved default org for tenant"
	LogOrgExternalMissing  = "External org ID not found"
	LogOrgExternalResolved = "Resolved external org ID"
)
