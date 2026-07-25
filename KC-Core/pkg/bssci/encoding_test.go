package bssci

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// TestDetectEncoding verifies that the detectEncoding helper correctly
// identifies JSON and MessagePack messages based on first byte inspection.
// Per BSSCI Section 1, messages can be either JSON or MessagePack.
func TestDetectEncoding(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		expected string
	}{
		{
			name:     "JSONObjectStartingWithBrace",
			payload:  []byte(`{"command":"con"}`),
			expected: EncodingJSON,
		},
		{
			name:     "JSONArrayStartingWithBracket",
			payload:  []byte(`["value1","value2"]`),
			expected: EncodingJSON,
		},
		{
			name: "MessagePackFixmap1Element",
			// 0x81 = fixmap with 1 element
			payload:  []byte{0x81, 0xa7, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0xa3, 0x63, 0x6f, 0x6e},
			expected: EncodingMessagePack,
		},
		{
			name: "MessagePackFixmap8Elements",
			// 0x88 = fixmap with 8 elements (within fixmap range 0x80-0x8f)
			payload:  []byte{0x88, 0xa7},
			expected: EncodingMessagePack,
		},
		{
			name: "MessagePackMap16",
			// 0xde = map 16 marker
			payload:  []byte{0xde, 0x00, 0x01},
			expected: EncodingMessagePack,
		},
		{
			name: "MessagePackMap32",
			// 0xdf = map 32 marker
			payload:  []byte{0xdf, 0x00, 0x00, 0x00, 0x01},
			expected: EncodingMessagePack,
		},
		{
			name:     "EmptyPayloadDefaultsToMsgpack",
			payload:  []byte{},
			expected: EncodingMessagePack,
		},
		{
			name: "AmbiguousPayloadDefaultsToMsgpack",
			// 0x50 is not a JSON or recognized MessagePack marker
			payload:  []byte{0x50, 0x41, 0x42},
			expected: EncodingMessagePack,
		},
		{
			name: "JSONWithWhitespace",
			// JSON objects with leading whitespace are correctly detected as JSON after trimming
			payload:  []byte(`  {"command":"con"}`),
			expected: EncodingJSON, // trimJSONWhitespace removes leading whitespace per RFC 8259
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := detectEncoding(tt.payload, EncodingMessagePack)
			if len(tt.payload) > 0 {
				assert.Equal(t, tt.expected, detected,
					"Expected encoding %s but got %s for payload starting with byte 0x%02X",
					tt.expected, detected, tt.payload[0])
			} else {
				assert.Equal(t, tt.expected, detected,
					"Expected encoding %s but got %s for empty payload",
					tt.expected, detected)
			}
		})
	}
}

// TestEncodeMessage verifies that the encodeMessage helper correctly
// encodes messages using the specified encoding (JSON or MessagePack).
func TestEncodeMessage(t *testing.T) {
	testMsg := map[string]interface{}{
		"command": "con",
		"opId":    int64(0),
		"version": "1.0.0",
	}

	t.Run("EncodeAsJSON", func(t *testing.T) {
		encoded, err := encodeMessage(testMsg, EncodingJSON)
		require.NoError(t, err, "JSON encoding should succeed")

		// Verify it's valid JSON
		var decoded map[string]interface{}
		err = json.Unmarshal(encoded, &decoded)
		require.NoError(t, err, "Encoded data should be valid JSON")
		assert.Equal(t, "con", decoded["command"])
	})

	t.Run("EncodeAsMessagePack", func(t *testing.T) {
		encoded, err := encodeMessage(testMsg, EncodingMessagePack)
		require.NoError(t, err, "MessagePack encoding should succeed")

		// Verify it's valid MessagePack
		var decoded map[string]interface{}
		err = msgpack.Unmarshal(encoded, &decoded)
		require.NoError(t, err, "Encoded data should be valid MessagePack")
		assert.Equal(t, "con", decoded["command"])
	})

	t.Run("DefaultEncodingIsMsgpack", func(t *testing.T) {
		encoded, err := encodeMessage(testMsg, "unknown")
		require.NoError(t, err, "Unknown encoding should default to MessagePack")

		// Verify it defaults to MessagePack
		var decoded map[string]interface{}
		err = msgpack.Unmarshal(encoded, &decoded)
		require.NoError(t, err, "Default encoding should be MessagePack")
	})

	t.Run("EmptyEncodingDefaultsToMsgpack", func(t *testing.T) {
		encoded, err := encodeMessage(testMsg, "")
		require.NoError(t, err, "Empty encoding should default to MessagePack")

		var decoded map[string]interface{}
		err = msgpack.Unmarshal(encoded, &decoded)
		require.NoError(t, err, "Empty encoding should default to MessagePack")
	})

	t.Run("StructEncoding", func(t *testing.T) {
		type TestStruct struct {
			Command string `json:"command" msgpack:"command"`
			OpId    int64  `json:"opId" msgpack:"opId"` //nolint:revive // BSSCI §2.5 requires lowercase 'd'
		}

		testStruct := TestStruct{
			Command: "attach",
			OpId:    123,
		}

		// Test JSON encoding of struct
		jsonEncoded, err := encodeMessage(testStruct, EncodingJSON)
		require.NoError(t, err, "JSON struct encoding should succeed")
		var jsonDecoded map[string]interface{}
		err = json.Unmarshal(jsonEncoded, &jsonDecoded)
		require.NoError(t, err)
		assert.Equal(t, "attach", jsonDecoded["command"])

		// Test MessagePack encoding of struct
		msgpackEncoded, err := encodeMessage(testStruct, EncodingMessagePack)
		require.NoError(t, err, "MessagePack struct encoding should succeed")
		var msgpackDecoded map[string]interface{}
		err = msgpack.Unmarshal(msgpackEncoded, &msgpackDecoded)
		require.NoError(t, err)
		assert.Equal(t, "attach", msgpackDecoded["command"])
	})
}

// TestDecodeMessage verifies that the decodeMessage helper correctly
// decodes messages using the specified encoding (JSON or MessagePack).
func TestDecodeMessage(t *testing.T) {
	t.Run("DecodeJSON", func(t *testing.T) {
		jsonPayload := []byte(`{"command":"con","opId":0,"version":"1.0.0"}`)
		decoded, err := decodeMessage(jsonPayload, EncodingJSON)
		require.NoError(t, err, "JSON decoding should succeed")
		assert.Equal(t, "con", decoded["command"])
		assert.Equal(t, "1.0.0", decoded["version"])
	})

	t.Run("DecodeMessagePack", func(t *testing.T) {
		// Create a MessagePack payload
		testMsg := map[string]interface{}{
			"command": "attach",
			"opId":    int64(1),
		}
		msgpackPayload, err := msgpack.Marshal(testMsg)
		require.NoError(t, err)

		decoded, err := decodeMessage(msgpackPayload, EncodingMessagePack)
		require.NoError(t, err, "MessagePack decoding should succeed")
		assert.Equal(t, "attach", decoded["command"])
	})

	t.Run("InvalidJSONPayload", func(t *testing.T) {
		invalidJSON := []byte(`{"command":invalid}`)
		_, err := decodeMessage(invalidJSON, "json")
		assert.Error(t, err, "Invalid JSON should return error")
	})

	t.Run("InvalidMessagePackPayload", func(t *testing.T) {
		invalidMsgpack := []byte{0xFF, 0xFF, 0xFF}
		_, err := decodeMessage(invalidMsgpack, "msgpack")
		assert.Error(t, err, "Invalid MessagePack should return error")
	})

	t.Run("DefaultEncodingIsMsgpack", func(t *testing.T) {
		// Create MessagePack payload
		testMsg := map[string]interface{}{"command": "ping"}
		msgpackPayload, err := msgpack.Marshal(testMsg)
		require.NoError(t, err)

		decoded, err := decodeMessage(msgpackPayload, "unknown")
		require.NoError(t, err, "Unknown encoding should default to MessagePack")
		assert.Equal(t, "ping", decoded["command"])
	})

	t.Run("EmptyEncodingDefaultsToMsgpack", func(t *testing.T) {
		testMsg := map[string]interface{}{"command": "status"}
		msgpackPayload, err := msgpack.Marshal(testMsg)
		require.NoError(t, err)

		decoded, err := decodeMessage(msgpackPayload, "")
		require.NoError(t, err, "Empty encoding should default to MessagePack")
		assert.Equal(t, "status", decoded["command"])
	})
}

// TestEncodeDecodeRoundtrip verifies that encoding and decoding are symmetric
// for both JSON and MessagePack formats.
func TestEncodeDecodeRoundtrip(t *testing.T) {
	testCases := []struct {
		name string
		msg  map[string]interface{}
	}{
		{
			name: "SimpleMessage",
			msg: map[string]interface{}{
				"command": "con",
				"opId":    int64(0),
			},
		},
		{
			name: "ComplexMessage",
			msg: map[string]interface{}{
				"command":  "attach",
				"opId":     int64(1),
				"epEui":    TestEuiEncoding,
				"shAddr":   uint32(0x1234),
				"bidi":     true,
				"nwkSnKey": []byte{0x01, 0x02, 0x03, 0x04},
			},
		},
	}

	encodings := []string{"json", "msgpack"}

	for _, tc := range testCases {
		for _, encoding := range encodings {
			t.Run(tc.name+"_"+encoding, func(t *testing.T) {
				// Encode
				encoded, err := encodeMessage(tc.msg, encoding)
				require.NoError(t, err, "Encoding should succeed")

				// Decode
				decoded, err := decodeMessage(encoded, encoding)
				require.NoError(t, err, "Decoding should succeed")

				// Verify key fields match
				assert.Equal(t, tc.msg["command"], decoded["command"], "Command should match")
			})
		}
	}
}

// TestEncodingDetectionAndDecode verifies the complete flow of detecting
// encoding from a raw payload and then decoding it correctly.
func TestEncodingDetectionAndDecode(t *testing.T) {
	t.Run("DetectAndDecodeJSON", func(t *testing.T) {
		jsonPayload := []byte(`{"command":"con","opId":0}`)

		// Detect encoding
		encoding := detectEncoding(jsonPayload, EncodingMessagePack)
		assert.Equal(t, "json", encoding, "Should detect JSON encoding")

		// Decode using detected encoding
		decoded, err := decodeMessage(jsonPayload, encoding)
		require.NoError(t, err, "Should decode successfully")
		assert.Equal(t, "con", decoded["command"])
	})

	t.Run("DetectAndDecodeMessagePack", func(t *testing.T) {
		// Create MessagePack payload
		testMsg := map[string]interface{}{
			"command": "attach",
			"opId":    int64(1),
		}
		msgpackPayload, err := msgpack.Marshal(testMsg)
		require.NoError(t, err)

		// Detect encoding
		encoding := detectEncoding(msgpackPayload, EncodingMessagePack)
		assert.Equal(t, "msgpack", encoding, "Should detect MessagePack encoding")

		// Decode using detected encoding
		decoded, err := decodeMessage(msgpackPayload, encoding)
		require.NoError(t, err, "Should decode successfully")
		assert.Equal(t, "attach", decoded["command"])
	})
}

// TestEncodingPersistence verifies that session encoding is correctly
// tracked in Session struct and persists across operations.
func TestEncodingPersistence(t *testing.T) {
	t.Run("SessionEncodingInitiallyEmpty", func(t *testing.T) {
		session := &Session{
			ProtocolSessionState: ProtocolSessionState{
				ID: "test-session",
			},
		}
		assert.Equal(t, "", session.Encoding, "New session should have empty encoding")
	})

	t.Run("SessionEncodingCanBeSet", func(t *testing.T) {
		session := &Session{
			ProtocolSessionState: ProtocolSessionState{
				ID: "test-session",
			},
		}
		session.Encoding = "json"
		assert.Equal(t, "json", session.Encoding, "Session encoding should be settable")

		session.Encoding = "msgpack"
		assert.Equal(t, "msgpack", session.Encoding, "Session encoding should be changeable")
	})
}
