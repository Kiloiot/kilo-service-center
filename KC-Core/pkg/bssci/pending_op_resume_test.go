package bssci

import (
	"encoding/base64"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resumeTestKey is the 16-byte network session key the reconstitution tests
// round-trip through the recovery formats.
var resumeTestKey = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

func assertRestoredKey(t *testing.T, msg map[string]interface{}) {
	t.Helper()
	arr, ok := msg[wireFieldNwkSnKey].([]interface{})
	require.True(t, ok, "the wire message must carry the Numeric[16] key array")
	require.Len(t, arr, 16)
	for i, v := range arr {
		assert.Equal(t, uint8(resumeTestKey[i]), v)
	}
}

// TestReconstituteAttachPropagate_TextEnvelope verifies the current recovery
// format: the key rides in metadata as a text envelope shared with ulDataTx.
func TestReconstituteAttachPropagate_TextEnvelope(t *testing.T) {
	env := newDetachTestEnv(t, &Config{MessageEncoding: EncodingJSON}, nil)

	encKey, err := env.server.cipher.EncryptString(resumeTestKey)
	require.NoError(t, err)

	msg, err := env.server.reconstitueAttachPropagateMessage(
		map[string]interface{}{"command": mioty.CmdAttachPropagate},
		map[string]interface{}{"encryptedKey": encKey},
	)
	require.NoError(t, err)
	assertRestoredKey(t, msg)
}

// TestReconstituteAttachPropagate_LegacyBinaryEnvelope verifies a record
// written before the format unification (base64 of the binary envelope) still
// resumes.
func TestReconstituteAttachPropagate_LegacyBinaryEnvelope(t *testing.T) {
	env := newDetachTestEnv(t, &Config{MessageEncoding: EncodingJSON}, nil)

	envelope, err := env.server.cipher.Encrypt(resumeTestKey)
	require.NoError(t, err)

	msg, err := env.server.reconstitueAttachPropagateMessage(
		map[string]interface{}{"command": mioty.CmdAttachPropagate},
		map[string]interface{}{"encryptedKey": base64.StdEncoding.EncodeToString(envelope)},
	)
	require.NoError(t, err)
	assertRestoredKey(t, msg)
}

// TestReconstituteAttachPropagate_PreSanitizationRecord verifies a record
// persisted before key sanitization - no envelope in metadata, the key still
// in the message - passes through unchanged instead of rejecting the resume.
func TestReconstituteAttachPropagate_PreSanitizationRecord(t *testing.T) {
	env := newDetachTestEnv(t, &Config{MessageEncoding: EncodingJSON}, nil)

	keyArray := make([]interface{}, 16)
	for i := range resumeTestKey {
		keyArray[i] = uint8(resumeTestKey[i])
	}
	msg, err := env.server.reconstitueAttachPropagateMessage(
		map[string]interface{}{
			"command":         mioty.CmdAttachPropagate,
			wireFieldNwkSnKey: keyArray,
		},
		map[string]interface{}{"epEui": "70B3D59CD0000001"},
	)
	require.NoError(t, err)
	assertRestoredKey(t, msg)
}

// TestReconstituteAttachPropagate_MissingKeyEverywhere verifies a record with
// neither an envelope nor an in-message key is reported unrecoverable.
func TestReconstituteAttachPropagate_MissingKeyEverywhere(t *testing.T) {
	env := newDetachTestEnv(t, &Config{MessageEncoding: EncodingJSON}, nil)

	_, err := env.server.reconstitueAttachPropagateMessage(
		map[string]interface{}{"command": mioty.CmdAttachPropagate},
		map[string]interface{}{"epEui": "70B3D59CD0000001"},
	)
	require.Error(t, err)
}

// TestReconstituteULDataTx_PreSanitizationRecord verifies a ulDataTx record
// persisted before key sanitization passes through with its in-message key.
func TestReconstituteULDataTx_PreSanitizationRecord(t *testing.T) {
	env := newDetachTestEnv(t, &Config{MessageEncoding: EncodingJSON}, nil)

	keyArray := make([]interface{}, 16)
	for i := range resumeTestKey {
		keyArray[i] = uint8(resumeTestKey[i])
	}
	msg, err := env.server.reconstitueULDataTxMessage(
		map[string]interface{}{
			"command":         mioty.CmdULDataTransmit,
			wireFieldNwkSnKey: keyArray,
			"opId":            int64(-5),
		},
		map[string]interface{}{"epEui": "70B3D59CD0000001"},
		&PendingOperation{OperationID: -5, OperationType: mioty.CmdULDataTransmit},
	)
	require.NoError(t, err)
	assertRestoredKey(t, msg)
}

// TestReconstituteResumeOperations_DropsUnrecoverable verifies one
// unrecoverable pending operation degrades alone: it is dropped from the
// resume set with a durable system event while the remaining operations
// survive - the connect is never rejected for it.
func TestReconstituteResumeOperations_DropsUnrecoverable(t *testing.T) {
	env := newDetachTestEnv(t, &Config{MessageEncoding: EncodingJSON}, nil)

	goodKey, err := env.server.cipher.EncryptString(resumeTestKey)
	require.NoError(t, err)

	ops := []*PendingOperation{
		{
			OperationID:   -1,
			OperationType: mioty.CmdAttachPropagate,
			Message:       map[string]interface{}{"command": mioty.CmdAttachPropagate},
			Metadata:      map[string]interface{}{"encryptedKey": goodKey},
		},
		{
			OperationID:   -2,
			OperationType: mioty.CmdAttachPropagate,
			Message:       map[string]interface{}{"command": mioty.CmdAttachPropagate},
			// A key that is neither a valid envelope nor decodable legacy
			// material: the operation cannot be rebuilt.
			Metadata: map[string]interface{}{"encryptedKey": "not-an-envelope"},
		},
	}

	kept := env.server.reconstituteResumeOperations(testutil.TestContext(), env.session, ops)

	require.Len(t, kept, 1, "the recoverable operation must survive")
	assert.Equal(t, int64(-1), kept[0].OperationID)
	assertRestoredKey(t, kept[0].Message)

	env.events.mu.Lock()
	events := append([]*models.SystemEvent(nil), env.events.created...)
	env.events.mu.Unlock()
	var recorded bool
	for _, ev := range events {
		if ev.EventType == models.EventTypeSessionPendingOpDropped {
			recorded = true
		}
	}
	assert.True(t, recorded, "the dropped operation must be durably recorded")
}
