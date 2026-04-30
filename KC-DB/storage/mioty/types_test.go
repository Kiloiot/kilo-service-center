package mioty

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// ============================================================================
// Numeric4 Tests
// ============================================================================

func TestNumeric4_JSONSerializesAsArray(t *testing.T) {
	n := Numeric4{1, 2, 3, 4}
	data, err := json.Marshal(n)

	require.NoError(t, err)
	assert.Equal(t, "[1,2,3,4]", string(data)) // Not base64
}

func TestNumeric4_JSONRoundTrip(t *testing.T) {
	original := Numeric4{10, 20, 30, 40}

	// Marshal
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var decoded Numeric4
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestNumeric4_JSONRejectsWrongLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"too short", "[1,2,3]"},
		{"too long", "[1,2,3,4,5]"},
		{"empty", "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Numeric4
			err := json.Unmarshal([]byte(tt.input), &n)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "4-element")
		})
	}
}

func TestNumeric4_MessagePackSerializesAsArray(t *testing.T) {
	n := Numeric4{1, 2, 3, 4}
	data, err := msgpack.Marshal(n)
	require.NoError(t, err)

	// Verify it's an array (first byte = 0x94 for fixarray of length 4)
	assert.Equal(t, byte(0x94), data[0]) // MessagePack fixarray[4]
}

func TestNumeric4_MessagePackRoundTrip(t *testing.T) {
	original := Numeric4{100, 150, 200, 255}

	// Encode
	data, err := msgpack.Marshal(original)
	require.NoError(t, err)

	// Decode
	var decoded Numeric4
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestNumeric4_MessagePackRejectsWrongLength(t *testing.T) {
	// Manually craft MessagePack array with 3 elements
	data := []byte{0x93, 1, 2, 3} // fixarray[3]

	var n Numeric4
	err := msgpack.Unmarshal(data, &n)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "4 elements")
}

func TestConnectResponseMarshaling(t *testing.T) {
	for _, encoding := range []string{"json", "msgpack"} {
		encoding := encoding
		t.Run(encoding, func(t *testing.T) {
			t.Parallel()

			var uuid SessionUUID
			copy(uuid[:], []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})

			resp := ConnectResponse{
				BaseMessage: BaseMessage{
					CommandType: CmdConnectResponse,
					OpId:        0,
				},
				Version:  "1.0.0",
				SnScUuid: uuid,
			}

			var (
				data []byte
				err  error
			)

			if encoding == "json" {
				data, err = json.Marshal(resp)
			} else {
				data, err = msgpack.Marshal(resp)
			}
			require.NoError(t, err)

			var decoded map[string]interface{}
			if encoding == "json" {
				err = json.Unmarshal(data, &decoded)
			} else {
				err = msgpack.Unmarshal(data, &decoded)
			}
			require.NoError(t, err)

			assert.Equal(t, "1.0.0", decoded["version"])

			uuidValue, ok := decoded["snScUuid"].([]interface{})
			require.True(t, ok, "snScUuid must decode to []interface{}")
			require.Len(t, uuidValue, 16)
			var firstByte int64
			switch v := uuidValue[0].(type) {
			case uint8:
				firstByte = int64(v)
			case int8:
				firstByte = int64(v)
			case int:
				firstByte = int64(v)
			case int64:
				firstByte = v
			case float64:
				firstByte = int64(v)
			default:
				t.Fatalf("unexpected type for UUID byte: %T (value: %v)", uuidValue[0], uuidValue[0])
			}
			assert.Equal(t, int64(1), firstByte)
		})
	}
}

func TestErrorMessageMarshaling(t *testing.T) {
	for _, encoding := range []string{"json", "msgpack"} {
		encoding := encoding
		t.Run(encoding, func(t *testing.T) {
			t.Parallel()

			errMsg := Error{
				BaseMessage: BaseMessage{
					CommandType: CmdError,
					OpId:        -1,
				},
				Code:    95,
				Message: "test error message",
			}

			var (
				data []byte
				err  error
			)

			if encoding == "json" {
				data, err = json.Marshal(errMsg)
			} else {
				data, err = msgpack.Marshal(errMsg)
			}
			require.NoError(t, err)

			var decoded map[string]interface{}
			if encoding == "json" {
				err = json.Unmarshal(data, &decoded)
			} else {
				err = msgpack.Unmarshal(data, &decoded)
			}
			require.NoError(t, err)

			assert.Equal(t, "error", decoded["command"])
			assert.Equal(t, "test error message", decoded["message"])

			opID := decoded["opId"]
			var opIDVal int64
			switch v := opID.(type) {
			case uint8:
				opIDVal = int64(v)
			case int8:
				opIDVal = int64(v)
			case int:
				opIDVal = int64(v)
			case int64:
				opIDVal = v
			case float64:
				opIDVal = int64(v)
			default:
				t.Fatalf("unexpected type for opId: %T (value: %v)", opID, opID)
			}
			assert.Equal(t, int64(-1), opIDVal)

			code := decoded["code"]
			var codeVal int64
			switch v := code.(type) {
			case uint8:
				codeVal = int64(v)
			case int8:
				codeVal = int64(v)
			case int:
				codeVal = int64(v)
			case int64:
				codeVal = v
			case float64:
				codeVal = int64(v)
			default:
				t.Fatalf("unexpected type for code: %T (value: %v)", code, code)
			}
			assert.Equal(t, int64(95), codeVal)
		})
	}
}
