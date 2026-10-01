package bssci

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A propagate recovery record names its endpoint owner whether it is still
// the live record the send path wrote (int64) or one reloaded from the
// database (JSON number), so its completion never falls back to the tenant of
// the base station that received the propagate.
func TestPropagateRecordOwnerReadsLiveAndReloadedRecords(t *testing.T) {
	const owner = int64(100)
	records := map[string]map[string]interface{}{
		"live record":              {metadataKeyTenantID: owner},
		"reloaded record (float)":  {metadataKeyTenantID: float64(owner)},
		"reloaded record (number)": {metadataKeyTenantID: json.Number("100")},
	}
	for name, metadata := range records {
		t.Run(name, func(t *testing.T) {
			tenantID, found, _ := propagateRecordOwner(metadata)
			assert.True(t, found, "the recovery record names its owner")
			assert.Equal(t, owner, tenantID)
		})
	}

	_, found, _ := propagateRecordOwner(map[string]interface{}{})
	assert.False(t, found, "a record without an owner is reported as such")
}
