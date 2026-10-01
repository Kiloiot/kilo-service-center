package adapter

import "time"

// Adapter timing defaults.
const (
	// EventRecordTimeout bounds the platform-event RPC so event recording
	// never blocks request processing.
	EventRecordTimeout = 5 * time.Second
	// OrgCacheDefaultTTL is the fallback organization-cache lifetime when the
	// configuration provides none.
	OrgCacheDefaultTTL = 5 * time.Minute
)

// OrgCacheDefaultMaxEntries is the fallback organization-cache capacity when
// the configuration provides none or a non-positive value.
const OrgCacheDefaultMaxEntries = 1000
