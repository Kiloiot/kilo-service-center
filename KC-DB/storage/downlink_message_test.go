package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// TestDownlinkMessage_Outcome: a revoked downlink reports revoked whatever
// its result column holds; any other downlink reports the station's result.
func TestDownlinkMessage_Outcome(t *testing.T) {
	for name, tc := range map[string]struct {
		message DownlinkMessage
		want    string
	}{
		"revoked":           {DownlinkMessage{Status: mioty.DLQueueStatusRevoked}, mioty.DLDataResultRevoked},
		"revoked after one": {DownlinkMessage{Status: mioty.DLQueueStatusRevoked, Result: mioty.ResultExpired}, mioty.DLDataResultRevoked},
		"sent":              {DownlinkMessage{Status: mioty.DLQueueStatusTransmitted, Result: mioty.ResultSent}, mioty.ResultSent},
		"in flight":         {DownlinkMessage{Status: mioty.DLQueueStatusQueued}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.message.Outcome())
		})
	}
}
