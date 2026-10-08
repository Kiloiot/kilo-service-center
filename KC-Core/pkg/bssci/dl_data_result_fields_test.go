package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// TestDLDataResultCarriesOnlySpecFields pins BSSCI §3.14.1 (rev1 §5.14.1):
// dlDataRes has no bsEui, the session names the reporting station, so a
// bsEui a station adds is an unknown field dropped under §2.4.
func TestDLDataResultCarriesOnlySpecFields(t *testing.T) {
	normalized, err := CallNormalizePayload(testutil.TestContext(), logger.NewNop(), "dlDataRes", map[string]interface{}{
		"command":   "dlDataRes",
		"opId":      int64(1),
		"epEui":     TestEpEui01,
		"queId":     uint64(42),
		"result":    DLDataResultSent,
		"txTime":    int64(1234567890),
		"packetCnt": uint32(7),
		"bsEui":     TestBsEui02,
	})
	require.NoError(t, err)
	_, hasStation := normalized["bsEui"]
	assert.False(t, hasStation, "dlDataRes defines no bsEui; the session identifies the station")
	for _, field := range []string{"epEui", "queId", "result", "txTime", "packetCnt"} {
		assert.Contains(t, normalized, field)
	}
}
