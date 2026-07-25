package bssci

import (
	"encoding/base64"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReconstituteDLDataQueUserDataShape: the rebuilt dlDataQue keeps userData
// a Numeric[m][n] array - one entry per payload when counter-dependent, exactly
// one entry (zero bytes long for an acknowledgement-only downlink) otherwise.
func TestReconstituteDLDataQueUserDataShape(t *testing.T) {
	tests := []struct {
		name      string
		metadata  map[string]interface{}
		pendingOp *PendingOperation
		expected  [][]byte
		wantErr   bool
	}{
		{
			name:     "acknowledgement only from empty payload list",
			metadata: map[string]interface{}{"payloads": []interface{}{}},
			expected: [][]byte{{}},
		},
		{
			name:     "acknowledgement only from zero-length payload",
			metadata: map[string]interface{}{"payloads": []interface{}{""}},
			expected: [][]byte{{}},
		},
		{
			name: "single payload",
			metadata: map[string]interface{}{
				"payloads": []interface{}{base64.StdEncoding.EncodeToString([]byte{0x0A, 0x0B})},
			},
			expected: [][]byte{{0x0A, 0x0B}},
		},
		{
			name:      "acknowledgement only from missing metadata",
			metadata:  map[string]interface{}{},
			pendingOp: &PendingOperation{},
			expected:  [][]byte{{}},
		},
		{
			name: "counter dependent keeps one entry per payload",
			metadata: map[string]interface{}{
				"cntDepend": true,
				"payloads": []interface{}{
					base64.StdEncoding.EncodeToString([]byte{0x01}),
					base64.StdEncoding.EncodeToString(nil),
					base64.StdEncoding.EncodeToString([]byte{0x02, 0x03}),
				},
				"packetCnt": []interface{}{float64(1), float64(2), float64(3)},
			},
			expected: [][]byte{{0x01}, {}, {0x02, 0x03}},
		},
		{
			name: "multiple payloads without counter dependency are rejected",
			metadata: map[string]interface{}{
				"payloads": []interface{}{
					base64.StdEncoding.EncodeToString([]byte{0x01}),
					base64.StdEncoding.EncodeToString([]byte{0x02}),
				},
			},
			wantErr: true,
		},
	}

	server := &Server{config: &Config{}, logger: logger.NewNop()}
	sanitized := map[string]interface{}{
		"command":   mioty.CmdDLDataQueue,
		"opId":      int64(-1),
		"epEui":     TestEpEui01,
		"queId":     int64(4711),
		"cntDepend": false,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := server.reconstitueDLDataQueMessage(sanitized, tt.metadata, tt.pendingOp)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			outer, ok := msg["userData"].([]interface{})
			require.True(t, ok, "userData must be the outer Numeric[m][n] array")
			require.Len(t, outer, len(tt.expected))
			for i, expectedEntry := range tt.expected {
				inner, ok := outer[i].([]interface{})
				require.True(t, ok, "entry %d must be a Numeric[n] array", i)
				assert.Equal(t, expectedEntry, server.normalizeUserDataField(inner))
			}
		})
	}
}
