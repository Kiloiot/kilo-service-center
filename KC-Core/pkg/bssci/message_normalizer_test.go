package bssci

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// TestDetachEqSnrDefaulting verifies BSSCI §5.7.1 eqSnr defaulting behavior
// When eqSnr is absent, it should default to snr value (special case for detach)
func TestDetachEqSnrDefaulting(t *testing.T) {
	testLogger := logger.NewNop()
	ctx := context.Background()

	tests := []struct {
		name          string
		inputData     map[string]interface{}
		expectedEqSnr interface{}
		description   string
	}{
		{
			name: "eqSnr absent - should default to snr value",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(1),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
				// eqSnr absent
			},
			expectedEqSnr: float64(10.5), // Should default to snr value
			description:   "Per BSSCI §5.7.1, eqSnr defaults to snr when absent",
		},
		{
			name: "eqSnr present - should use provided value",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(2),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
				"eqSnr":     float64(11.2), // Explicit eqSnr value
			},
			expectedEqSnr: float64(11.2), // Should use provided value, not snr
			description:   "Explicit eqSnr value should override default",
		},
		{
			name: "eqSnr=0 present - should preserve zero value",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(3),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
				"eqSnr":     float64(0), // Explicit zero
			},
			expectedEqSnr: float64(0), // Zero is a valid value, not "absent"
			description:   "Zero eqSnr should be preserved, not treated as absent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := CallNormalizePayload(ctx, testLogger, "det", tt.inputData)
			require.NoError(t, err, "Normalization should succeed for valid detach message")

			actualEqSnr, exists := normalized["eqSnr"]
			require.True(t, exists, "eqSnr should be present in normalized payload")
			assert.Equal(t, tt.expectedEqSnr, actualEqSnr, tt.description)
		})
	}
}

// TestDetachRxDurationZeroPreservation verifies that rxDuration=0 is preserved at the normalization layer
// rxDuration=0 must be preserved as explicit value, not treated as absent
func TestDetachRxDurationZeroPreservation(t *testing.T) {
	testLogger := logger.NewNop()
	ctx := context.Background()

	tests := []struct {
		name             string
		inputData        map[string]interface{}
		expectRxDuration bool
		expectedValue    interface{}
		description      string
	}{
		{
			name: "rxDuration=0 should be preserved (not treated as absent)",
			inputData: map[string]interface{}{
				"command":    "det",
				"opId":       int64(1),
				"epEui":      TestEpEui01,
				"rxTime":     int64(1234567890),
				"packetCnt":  uint32(42),
				"snr":        float64(10.5),
				"rssi":       float64(-80.0),
				"sign":       []byte{0xAA, 0xBB, 0xCC, 0xDD},
				"rxDuration": int64(0), // Explicit zero
			},
			expectRxDuration: true,
			expectedValue:    int64(0),
			description:      "Zero rxDuration is semantically valid and must be stored",
		},
		{
			name: "rxDuration=123 should be preserved",
			inputData: map[string]interface{}{
				"command":    "det",
				"opId":       int64(2),
				"epEui":      TestEpEui01,
				"rxTime":     int64(1234567890),
				"packetCnt":  uint32(42),
				"snr":        float64(10.5),
				"rssi":       float64(-80.0),
				"sign":       []byte{0xAA, 0xBB, 0xCC, 0xDD},
				"rxDuration": int64(123), // Non-zero value
			},
			expectRxDuration: true,
			expectedValue:    int64(123),
			description:      "Non-zero rxDuration should be preserved",
		},
		{
			name: "rxDuration absent - should get default (nil for optional pointer field)",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(3),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
				// rxDuration absent
			},
			expectRxDuration: true, // Key exists in normalized map
			expectedValue:    nil,  // Default value for optional field
			description:      "Absent rxDuration should get nil default per message spec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := CallNormalizePayload(ctx, testLogger, "det", tt.inputData)
			require.NoError(t, err, "Normalization should succeed")

			if tt.expectRxDuration {
				actualValue, exists := normalized["rxDuration"]
				require.True(t, exists, "rxDuration key should exist in normalized map")
				assert.Equal(t, tt.expectedValue, actualValue, tt.description)
			}
		})
	}
}

// TestDetachProfileEmptyStringPreservation verifies profile="" is valid and preserved
func TestDetachProfileEmptyStringPreservation(t *testing.T) {
	testLogger := logger.NewNop()
	ctx := context.Background()

	tests := []struct {
		name        string
		inputData   map[string]interface{}
		expected    interface{}
		description string
	}{
		{
			name: "profile='' (empty string) should be preserved",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(1),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
				"profile":   "", // Empty string is semantically valid
			},
			expected:    "",
			description: "Empty profile string is valid per MIOTY spec and must be preserved",
		},
		{
			name: "profile='eu1' should be preserved",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(2),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
				"profile":   "eu1", // Valid profile string
			},
			expected:    "eu1",
			description: "Non-empty profile string should be preserved",
		},
		{
			name: "profile absent - should get default (nil)",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(3),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
				// profile absent
			},
			expected:    nil,
			description: "Absent profile should get nil default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := CallNormalizePayload(ctx, testLogger, "det", tt.inputData)
			require.NoError(t, err, "Normalization should succeed")

			actualValue, exists := normalized["profile"]
			require.True(t, exists, "profile key should exist in normalized map")
			assert.Equal(t, tt.expected, actualValue, tt.description)
		})
	}
}

// TestDetachSignatureByteArrayValidation verifies signature field byte array handling
func TestDetachSignatureByteArrayValidation(t *testing.T) {
	testLogger := logger.NewNop()
	ctx := context.Background()

	tests := []struct {
		name        string
		signValue   interface{}
		expectError bool
		description string
	}{
		{
			name:        "signature as []byte - should succeed",
			signValue:   []byte{0xAA, 0xBB, 0xCC, 0xDD},
			expectError: false,
			description: "Native []byte signature should validate",
		},
		{
			name:        "signature as []interface{} numeric array - should convert to []byte",
			signValue:   []interface{}{float64(0xAA), float64(0xBB), float64(0xCC), float64(0xDD)},
			expectError: false,
			description: "MessagePack numeric array should convert to []byte",
		},
		{
			name:        "signature with wrong length - should fail (if spec requires 4 bytes)",
			signValue:   []byte{0xAA, 0xBB, 0xCC}, // Only 3 bytes
			expectError: true,                     // Depends on ByteLength spec
			description: "Signature with incorrect byte count should fail validation",
		},
		{
			name:        "signature with values > 255 in numeric array - should fail",
			signValue:   []interface{}{float64(256), float64(0xBB), float64(0xCC), float64(0xDD)},
			expectError: true,
			description: "Numeric array with out-of-range values should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputData := map[string]interface{}{
				"command":   "det",
				"opId":      int64(1),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      tt.signValue,
			}

			normalized, err := CallNormalizePayload(ctx, testLogger, "det", inputData)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				// Verify signature was normalized to []byte
				sig, exists := normalized["sign"]
				assert.True(t, exists, "Signature should exist in normalized payload")
				_, ok := sig.([]byte)
				assert.True(t, ok, "Signature should be []byte after normalization")
			}
		})
	}
}

// TestDetachUnknownFieldDropping verifies BSSCI §2.4 forward compatibility
// Unknown fields should be logged (WARN) and dropped without failing normalization
func TestDetachUnknownFieldDropping(t *testing.T) {
	testLogger := logger.NewNop()
	ctx := context.Background()

	inputData := map[string]interface{}{
		"command":   "det",
		"opId":      int64(1),
		"epEui":     TestEpEui01,
		"rxTime":    int64(1234567890),
		"packetCnt": uint32(42),
		"snr":       float64(10.5),
		"rssi":      float64(-80.0),
		"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
		// Unknown fields (future BSSCI extensions)
		"futureField1": "unknown_value",
		"futureField2": int64(123),
	}

	normalized, err := CallNormalizePayload(ctx, testLogger, "det", inputData)
	require.NoError(t, err, "Unknown fields should not fail normalization")

	// Verify unknown fields were dropped
	_, exists := normalized["futureField1"]
	assert.False(t, exists, "Unknown field futureField1 should be dropped")

	_, exists = normalized["futureField2"]
	assert.False(t, exists, "Unknown field futureField2 should be dropped")

	// Verify known fields are still present
	_, exists = normalized["epEui"]
	assert.True(t, exists, "Known field epEui should be preserved")

	_, exists = normalized["snr"]
	assert.True(t, exists, "Known field snr should be preserved")
}

// TestDetachMandatoryFieldValidation verifies mandatory field enforcement
func TestDetachMandatoryFieldValidation(t *testing.T) {
	testLogger := logger.NewNop()
	ctx := context.Background()

	tests := []struct {
		name        string
		inputData   map[string]interface{}
		expectError bool
		description string
	}{
		{
			name: "missing epEui - should fail",
			inputData: map[string]interface{}{
				"command": "det",
				"opId":    int64(1),
				// epEui missing
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
			},
			expectError: true,
			description: "Missing mandatory epEui field should fail normalization",
		},
		{
			name: "missing rxTime - should fail",
			inputData: map[string]interface{}{
				"command": "det",
				"opId":    int64(1),
				"epEui":   TestEpEui01,
				// rxTime missing
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
			},
			expectError: true,
			description: "Missing mandatory rxTime field should fail normalization",
		},
		{
			name: "missing signature - should fail",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(1),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				// sign missing
			},
			expectError: true,
			description: "Missing mandatory signature field should fail normalization",
		},
		{
			name: "all mandatory fields present - should succeed",
			inputData: map[string]interface{}{
				"command":   "det",
				"opId":      int64(1),
				"epEui":     TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"snr":       float64(10.5),
				"rssi":      float64(-80.0),
				"sign":      []byte{0xAA, 0xBB, 0xCC, 0xDD},
			},
			expectError: false,
			description: "All mandatory fields present should pass normalization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CallNormalizePayload(ctx, testLogger, "det", tt.inputData)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestCoerceUint32_AcceptsUint64 verifies coerceUint32 handles uint64 input from msgpack decoder
func TestCoerceUint32_AcceptsUint64(t *testing.T) {
	result, err := coerceUint32(uint64(5000))
	require.NoError(t, err)
	assert.Equal(t, uint32(5000), result)
}

// TestCoerceUint16_AcceptsUint64 verifies coerceUint16 handles uint64 input from msgpack decoder
func TestCoerceUint16_AcceptsUint64(t *testing.T) {
	result, err := coerceUint16(uint64(1024))
	require.NoError(t, err)
	assert.Equal(t, uint16(1024), result)
}

// TestCoerceUint32_RejectsUint64Overflow verifies coerceUint32 rejects values exceeding uint32 range
func TestCoerceUint32_RejectsUint64Overflow(t *testing.T) {
	_, err := coerceUint32(uint64(math.MaxUint32 + 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// TestCoerceUint16_RejectsUint64Overflow verifies coerceUint16 rejects values exceeding uint16 range
func TestCoerceUint16_RejectsUint64Overflow(t *testing.T) {
	_, err := coerceUint16(uint64(math.MaxUint16 + 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// Signed-integer coercion cases — added after a real-device uplink was rejected
// because msgpack-go encoded packetCnt=1 as int8 and the coerce helpers only
// recognised the unsigned widths + int/int64/float64. These pin the int8/int16/
// int32 paths so a regression silently dropping them gets caught.

func TestCoerceUint64_AcceptsInt8(t *testing.T) {
	got, err := coerceUint64(int8(1))
	require.NoError(t, err)
	assert.Equal(t, uint64(1), got)

	max, err := coerceUint64(int8(math.MaxInt8))
	require.NoError(t, err)
	assert.Equal(t, uint64(math.MaxInt8), max)
}

func TestCoerceUint64_AcceptsInt16(t *testing.T) {
	got, err := coerceUint64(int16(1))
	require.NoError(t, err)
	assert.Equal(t, uint64(1), got)

	max, err := coerceUint64(int16(math.MaxInt16))
	require.NoError(t, err)
	assert.Equal(t, uint64(math.MaxInt16), max)
}

func TestCoerceUint64_AcceptsInt32(t *testing.T) {
	got, err := coerceUint64(int32(1))
	require.NoError(t, err)
	assert.Equal(t, uint64(1), got)

	max, err := coerceUint64(int32(math.MaxInt32))
	require.NoError(t, err)
	assert.Equal(t, uint64(math.MaxInt32), max)
}

func TestCoerceUint64_RejectsNegativeSigned(t *testing.T) {
	for _, v := range []interface{}{int8(-1), int16(-1), int32(-1)} {
		_, err := coerceUint64(v)
		require.Error(t, err, "should reject %T(%v)", v, v)
		assert.Contains(t, err.Error(), "negative value cannot be coerced to uint64")
	}
}

func TestCoerceUint32_AcceptsInt8(t *testing.T) {
	got, err := coerceUint32(int8(1))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), got)

	max, err := coerceUint32(int8(math.MaxInt8))
	require.NoError(t, err)
	assert.Equal(t, uint32(math.MaxInt8), max)
}

func TestCoerceUint32_AcceptsInt16(t *testing.T) {
	got, err := coerceUint32(int16(1))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), got)

	max, err := coerceUint32(int16(math.MaxInt16))
	require.NoError(t, err)
	assert.Equal(t, uint32(math.MaxInt16), max)
}

func TestCoerceUint32_AcceptsInt32(t *testing.T) {
	got, err := coerceUint32(int32(1))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), got)

	max, err := coerceUint32(int32(math.MaxInt32))
	require.NoError(t, err)
	assert.Equal(t, uint32(math.MaxInt32), max)
}

func TestCoerceUint32_RejectsNegativeSigned(t *testing.T) {
	for _, v := range []interface{}{int8(-1), int16(-1), int32(-1)} {
		_, err := coerceUint32(v)
		require.Error(t, err, "should reject %T(%v)", v, v)
		assert.Contains(t, err.Error(), "negative value cannot be coerced to uint32")
	}
}

func TestCoerceUint16_AcceptsInt8(t *testing.T) {
	got, err := coerceUint16(int8(1))
	require.NoError(t, err)
	assert.Equal(t, uint16(1), got)

	max, err := coerceUint16(int8(math.MaxInt8))
	require.NoError(t, err)
	assert.Equal(t, uint16(math.MaxInt8), max)
}

func TestCoerceUint16_AcceptsInt16(t *testing.T) {
	got, err := coerceUint16(int16(1))
	require.NoError(t, err)
	assert.Equal(t, uint16(1), got)

	max, err := coerceUint16(int16(math.MaxInt16))
	require.NoError(t, err)
	assert.Equal(t, uint16(math.MaxInt16), max)
}

func TestCoerceUint16_AcceptsInt32InRange(t *testing.T) {
	got, err := coerceUint16(int32(1))
	require.NoError(t, err)
	assert.Equal(t, uint16(1), got)

	// uint16 fits values up to 65535; assert max boundary.
	maxVal, err := coerceUint16(int32(math.MaxUint16))
	require.NoError(t, err)
	assert.Equal(t, uint16(math.MaxUint16), maxVal)
}

func TestCoerceUint16_RejectsInt32Overflow(t *testing.T) {
	_, err := coerceUint16(int32(math.MaxUint16 + 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCoerceUint16_RejectsNegativeSigned(t *testing.T) {
	for _, v := range []interface{}{int8(-1), int16(-1), int32(-1)} {
		_, err := coerceUint16(v)
		require.Error(t, err, "should reject %T(%v)", v, v)
		// int32 negative goes through the "out of range" branch; int8/int16
		// negatives go through the dedicated "negative value" branch. Either
		// way the error mentions uint16.
		assert.Contains(t, err.Error(), "uint16")
	}
}

// TestMsgpackDecoderProducesUint64ForUnsigned locks the assumption that vmihailenco/msgpack
// decodes unsigned integers exceeding MaxInt64 as Go uint64 when unmarshaling into interface{}.
// Values that fit in int64 are decoded as int64 by the Go library, but C-based BS encoders
// may use unsigned wire formats (uint32/uint64) for any positive value, which the Go decoder
// maps to uint64 when the value exceeds int64 range. This test uses a value > MaxInt64 to
// verify the uint64 path exists. Combined with coercion fixes, this ensures all unsigned
// wire values are handled regardless of magnitude.
func TestMsgpackDecoderProducesUint64ForUnsigned(t *testing.T) {
	// Use a value that exceeds int64 range to force uint64 decoding
	largeUint := uint64(math.MaxInt64) + 1
	input := map[string]interface{}{"val": largeUint}
	encoded, err := msgpack.Marshal(input)
	require.NoError(t, err)

	var result map[string]interface{}
	err = msgpack.Unmarshal(encoded, &result)
	require.NoError(t, err)

	assert.Equal(t, reflect.Uint64, reflect.TypeOf(result["val"]).Kind(),
		"vmihailenco/msgpack should decode large unsigned integers as uint64 in interface{}")
	assert.Equal(t, largeUint, result["val"].(uint64))
}

// TestNormalizeULData_PacketCntAsUint64 reproduces the real BS failure: packetCnt arrives as
// uint64 from the msgpack decoder, and normalization must coerce it to uint32 without error.
func TestNormalizeULData_PacketCntAsUint64(t *testing.T) {
	testLogger := logger.NewNop()
	ctx := context.Background()

	data := map[string]interface{}{
		"command":     "ulData",
		"opId":        int64(1),
		"epEui":       uint64(TestEpEui01),
		"rxTime":      int64(1234567890000000000),
		"packetCnt":   uint64(5000), // msgpack decoder produces uint64
		"userData":    []byte{0x01, 0x02, 0x03},
		"snr":         float64(10.5),
		"rssi":        float64(-80.0),
		"dlOpen":      false,
		"responseExp": false,
		"dlAck":       false,
	}

	normalized, err := CallNormalizePayload(ctx, testLogger, "ulData", data)
	require.NoError(t, err, "ulData normalization must accept uint64 packetCnt from msgpack decoder")
	assert.Equal(t, uint32(5000), normalized["packetCnt"],
		"packetCnt should be coerced from uint64 to uint32")
}
