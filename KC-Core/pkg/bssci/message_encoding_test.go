package bssci

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// TestMessageOpIdFieldCasing verifies BSSCI §2.5 wire format:
// The Message struct's OpId field must serialize to "opId" (lowercase 'd')
// in both JSON and MessagePack formats.
func TestMessageOpIdFieldCasing(t *testing.T) {
	t.Run("JSONSerializationUsesLowercaseD", func(t *testing.T) {
		msg := &Message{
			Command: "con",
			OpId:    -123,
			Data: map[string]interface{}{
				"version": "1.0.0",
			},
		}

		jsonData, err := json.Marshal(msg)
		require.NoError(t, err, "Message should marshal to JSON")

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, `"opId":`, "JSON must contain 'opId' with lowercase 'd'")
		assert.NotContains(t, jsonStr, `"opID":`, "JSON must NOT contain 'opID' with uppercase 'D'")
		assert.NotContains(t, jsonStr, `"OpId":`, "JSON must NOT contain 'OpId' capitalized")

		// Verify roundtrip
		var decoded Message
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err, "JSON should unmarshal back to Message")
		assert.Equal(t, int64(-123), decoded.OpId, "OpId should roundtrip correctly")
	})

	t.Run("MessagePackSerializationUsesLowercaseD", func(t *testing.T) {
		msg := &Message{
			Command: "attach",
			OpId:    42,
			Data: map[string]interface{}{
				"epEui": TestEuiEncoding,
			},
		}

		msgpackData, err := msgpack.Marshal(msg)
		require.NoError(t, err, "Message should marshal to MessagePack")

		// Verify roundtrip
		var decoded Message
		err = msgpack.Unmarshal(msgpackData, &decoded)
		require.NoError(t, err, "MessagePack should unmarshal back to Message")
		assert.Equal(t, int64(42), decoded.OpId, "OpId should roundtrip correctly via MessagePack")
		assert.Equal(t, "attach", decoded.Command)
	})

	t.Run("StructTagsMatchBSSCISpec", func(t *testing.T) {
		// Use reflection to verify the struct tags are correct
		msgType := reflect.TypeOf(Message{})
		opIdField, found := msgType.FieldByName("OpId") //nolint:revive // BSSCI §2.5 requires lowercase 'd'
		require.True(t, found, "Message struct must have OpId field")

		jsonTag := opIdField.Tag.Get("json")
		msgpackTag := opIdField.Tag.Get("msgpack")

		// Both tags must specify lowercase 'd'
		assert.Equal(t, "opId", jsonTag, "JSON tag must be 'opId' per BSSCI §2.5")
		assert.Equal(t, "opId", msgpackTag, "MessagePack tag must be 'opId' per BSSCI §2.5")
	})
}

// TestMessageIntegrationWithBothEncodings verifies that the Message struct
// correctly handles both JSON and MessagePack encoding as required by BSSCI Section 1.
func TestMessageIntegrationWithBothEncodings(t *testing.T) {
	testCases := []struct {
		name string
		msg  *Message
	}{
		{
			name: "ConnectMessage",
			msg: &Message{
				Command: "con",
				OpId:    0,
				Data: map[string]interface{}{
					"version": "1.0.0",
					"bsEui":   TestBsEui02,
				},
			},
		},
		{
			name: "AttachMessage",
			msg: &Message{
				Command: "att",
				OpId:    1,
				Data: map[string]interface{}{
					"epEui":  TestEuiEncoding,
					"shAddr": uint32(0x1234),
					"bidi":   true,
				},
			},
		},
		{
			name: "SCInitiatedOperation",
			msg: &Message{
				Command: "dlDataQue",
				OpId:    -1,
				Data: map[string]interface{}{
					"epEui":   TestEuiEncoding,
					"queId":   uint64(42),
					"payload": []byte{0x01, 0x02, 0x03},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_JSON", func(t *testing.T) {
			// Encode to JSON
			jsonData, err := json.Marshal(tc.msg)
			require.NoError(t, err)

			// Verify JSON structure
			var rawMap map[string]interface{}
			err = json.Unmarshal(jsonData, &rawMap)
			require.NoError(t, err)

			// Check that opId key exists (lowercase d)
			opIdValue, exists := rawMap["opId"] //nolint:revive // BSSCI §2.5 requires lowercase 'd'
			assert.True(t, exists, "JSON must contain 'opId' key")
			assert.NotNil(t, opIdValue, "opId value must not be nil")

			// Decode back to Message
			var decoded Message
			err = json.Unmarshal(jsonData, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tc.msg.Command, decoded.Command)
			assert.Equal(t, tc.msg.OpId, decoded.OpId)
		})

		t.Run(tc.name+"_MessagePack", func(t *testing.T) {
			// Encode to MessagePack
			msgpackData, err := msgpack.Marshal(tc.msg)
			require.NoError(t, err)

			// Decode back to Message
			var decoded Message
			err = msgpack.Unmarshal(msgpackData, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tc.msg.Command, decoded.Command)
			assert.Equal(t, tc.msg.OpId, decoded.OpId)
		})
	}
}

// TestBSSCIAntiDriftGuard is a reflection-based test that prevents unintended
// changes to the Message struct that could break wire protocol compatibility.
//
// This test will FAIL if:
// - OpId field is renamed
// - OpId field tags are changed
// - OpId field type is changed from int64
//
// If you need to modify the Message struct, update this test AND verify
// BSSCI specification conformance before committing.
func TestBSSCIAntiDriftGuard(t *testing.T) {
	t.Run("MessageStructHasExactlyExpectedFields", func(t *testing.T) {
		msgType := reflect.TypeOf(Message{})

		// Message struct must have exactly these fields (no more, no less)
		expectedFields := map[string]struct {
			Type string
			JSON string
			MP   string
		}{
			"Command":    {Type: "string", JSON: "command", MP: "command"},
			"OpId":       {Type: "int64", JSON: "opId", MP: "opId"},
			"Data":       {Type: "interface {}", JSON: "-", MP: "-"},
			"RawPayload": {Type: "[]uint8", JSON: "-", MP: "-"}, // BSSCI-5.6-RAW
		}

		assert.Equal(t, len(expectedFields), msgType.NumField(),
			"Message struct must have exactly %d fields", len(expectedFields))

		for i := 0; i < msgType.NumField(); i++ {
			field := msgType.Field(i)
			expected, exists := expectedFields[field.Name]
			require.True(t, exists,
				"Unexpected field %s in Message struct. Update this test if intentional.", field.Name)

			assert.Equal(t, expected.Type, field.Type.String(),
				"Field %s has wrong type", field.Name)
			assert.Equal(t, expected.JSON, field.Tag.Get("json"),
				"Field %s has wrong JSON tag", field.Name)
			assert.Equal(t, expected.MP, field.Tag.Get("msgpack"),
				"Field %s has wrong msgpack tag", field.Name)
		}
	})

	t.Run("OpIdFieldHasCorrectLowercaseTags", func(t *testing.T) {
		msgType := reflect.TypeOf(Message{})
		field, found := msgType.FieldByName("OpId")
		require.True(t, found, "Message.OpId field must exist")

		// Critical: tags must use lowercase 'd' per BSSCI §2.5
		jsonTag := field.Tag.Get("json")
		msgpackTag := field.Tag.Get("msgpack")

		// These assertions protect against accidental changes
		assert.Equal(t, "opId", jsonTag,
			"BSSCI §2.5 violation: OpId JSON tag must be 'opId' (lowercase 'd'), not '%s'", jsonTag)
		assert.Equal(t, "opId", msgpackTag,
			"BSSCI §2.5 violation: OpId msgpack tag must be 'opId' (lowercase 'd'), not '%s'", msgpackTag)

		// Verify tags don't contain uppercase variations
		assert.False(t, strings.Contains(jsonTag, "opID"), "JSON tag must not use 'opID'")
		assert.False(t, strings.Contains(jsonTag, "OpId"), "JSON tag must not use 'OpId'")
		assert.False(t, strings.Contains(msgpackTag, "opID"), "MessagePack tag must not use 'opID'")
		assert.False(t, strings.Contains(msgpackTag, "OpId"), "MessagePack tag must not use 'OpId'")
	})

	t.Run("OpIdFieldIsInt64", func(t *testing.T) {
		msgType := reflect.TypeOf(Message{})
		field, _ := msgType.FieldByName("OpId")

		assert.Equal(t, reflect.Int64, field.Type.Kind(),
			"OpId must be int64 for BSSCI signed operation ID semantics")
	})
}
