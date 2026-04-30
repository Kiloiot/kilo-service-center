package scaci

import (
	"encoding/json"
	"testing"

	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// TestEPStatusStructure verifies EPStatus struct conforms to SCACI §3.13.1
func TestEPStatusStructure(t *testing.T) {
	t.Run("RequiredFieldsOnly", func(t *testing.T) {
		status := EPStatus{
			BaseMessage: BaseMessage{
				Command: "epStat",
				OpId:    -1,
			},
			EpEui:    0x0102030405060708,
			EpStatus: EPStatusAttached,
		}

		// Verify JSON encoding
		jsonData, err := json.Marshal(status)
		require.NoError(t, err, "JSON encoding should succeed")

		var decoded EPStatus
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err, "JSON decoding should succeed")
		assert.Equal(t, status.EpEui, decoded.EpEui)
		assert.Equal(t, status.EpStatus, decoded.EpStatus)
		assert.Nil(t, decoded.AttachCnt, "Optional field should be nil")
		assert.Nil(t, decoded.Nonce, "Optional field should be nil")
		assert.Nil(t, decoded.Sign, "Optional field should be nil")

		// Verify MessagePack encoding
		msgpackData, err := msgpack.Marshal(status)
		require.NoError(t, err, "MessagePack encoding should succeed")

		var decodedMP EPStatus
		err = msgpack.Unmarshal(msgpackData, &decodedMP)
		require.NoError(t, err, "MessagePack decoding should succeed")
		assert.Equal(t, status.EpEui, decodedMP.EpEui)
		assert.Equal(t, status.EpStatus, decodedMP.EpStatus)
	})

	t.Run("OTAAttachWithNumeric4Fields", func(t *testing.T) {
		attachCnt := uint32(42)
		nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
		sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
		snr := 15.5
		rssi := -85.0

		status := EPStatus{
			BaseMessage: BaseMessage{
				Command: "epStat",
				OpId:    -1,
			},
			EpEui:     0x0102030405060708,
			EpStatus:  EPStatusAttached,
			AttachCnt: &attachCnt,
			Nonce:     &nonce,
			Sign:      &sign,
			Snr:       &snr,
			Rssi:      &rssi,
		}

		// Verify JSON encoding
		jsonData, err := json.Marshal(status)
		require.NoError(t, err, "JSON encoding should succeed")

		var decoded EPStatus
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err, "JSON decoding should succeed")
		assert.Equal(t, status.EpEui, decoded.EpEui)
		assert.Equal(t, status.EpStatus, decoded.EpStatus)
		require.NotNil(t, decoded.AttachCnt, "AttachCnt should be present")
		assert.Equal(t, uint32(42), *decoded.AttachCnt)
		require.NotNil(t, decoded.Nonce, "Nonce should be present")
		assert.Equal(t, nonce, *decoded.Nonce, "Nonce should roundtrip as array")
		require.NotNil(t, decoded.Sign, "Sign should be present")
		assert.Equal(t, sign, *decoded.Sign, "Sign should roundtrip as array")
		require.NotNil(t, decoded.Snr)
		assert.Equal(t, 15.5, *decoded.Snr)
		require.NotNil(t, decoded.Rssi)
		assert.Equal(t, -85.0, *decoded.Rssi)

		// Verify MessagePack encoding
		msgpackData, err := msgpack.Marshal(status)
		require.NoError(t, err, "MessagePack encoding should succeed")

		var decodedMP EPStatus
		err = msgpack.Unmarshal(msgpackData, &decodedMP)
		require.NoError(t, err, "MessagePack decoding should succeed")
		assert.Equal(t, status.EpEui, decodedMP.EpEui)
		assert.Equal(t, status.EpStatus, decodedMP.EpStatus)
		require.NotNil(t, decodedMP.Nonce, "Nonce should be present after MessagePack roundtrip")
		assert.Equal(t, nonce, *decodedMP.Nonce, "Nonce should roundtrip correctly via MessagePack")
		require.NotNil(t, decodedMP.Sign, "Sign should be present after MessagePack roundtrip")
		assert.Equal(t, sign, *decodedMP.Sign, "Sign should roundtrip correctly via MessagePack")
	})

	t.Run("DetachedStatus", func(t *testing.T) {
		sign := mioty.Numeric4{0x11, 0x22, 0x33, 0x44}
		snr := 12.3
		rssi := -90.0

		status := EPStatus{
			BaseMessage: BaseMessage{
				Command: "epStat",
				OpId:    -2,
			},
			EpEui:    0x0102030405060708,
			EpStatus: EPStatusDetached,
			Sign:     &sign,
			Snr:      &snr,
			Rssi:     &rssi,
		}

		// Verify JSON encoding
		jsonData, err := json.Marshal(status)
		require.NoError(t, err, "JSON encoding should succeed")

		var decoded EPStatus
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err, "JSON decoding should succeed")
		assert.Equal(t, EPStatusDetached, decoded.EpStatus)
		assert.Nil(t, decoded.AttachCnt, "AttachCnt should not be present for detach")
		assert.Nil(t, decoded.Nonce, "Nonce should not be present for detach")
		require.NotNil(t, decoded.Sign, "Sign should be present for detach")
		assert.Equal(t, sign, *decoded.Sign)

		// Verify MessagePack encoding
		msgpackData, err := msgpack.Marshal(status)
		require.NoError(t, err, "MessagePack encoding should succeed")

		var decodedMP EPStatus
		err = msgpack.Unmarshal(msgpackData, &decodedMP)
		require.NoError(t, err, "MessagePack decoding should succeed")
		assert.Equal(t, EPStatusDetached, decodedMP.EpStatus)
		require.NotNil(t, decodedMP.Sign, "Sign should roundtrip for detach via MessagePack")
		assert.Equal(t, sign, *decodedMP.Sign)
	})

	t.Run("WithOptionalEqSnrAndSubpackets", func(t *testing.T) {
		eqSnr := 14.2
		subpackets := mioty.Subpackets{
			SNR:       []float64{12.5, 13.0, 11.8},
			RSSI:      []float64{-88.0, -87.5, -89.0},
			Frequency: []int64{868100000, 868200000, 868300000},
		}

		status := EPStatus{
			BaseMessage: BaseMessage{
				Command: "epStat",
				OpId:    -3,
			},
			EpEui:      0x0102030405060708,
			EpStatus:   EPStatusAttached,
			EqSnr:      &eqSnr,
			Subpackets: &subpackets,
		}

		// Verify JSON roundtrip
		jsonData, err := json.Marshal(status)
		require.NoError(t, err)

		var decoded EPStatus
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		require.NotNil(t, decoded.EqSnr, "EqSnr should be present")
		assert.Equal(t, 14.2, *decoded.EqSnr)
		require.NotNil(t, decoded.Subpackets, "Subpackets should be present")
		assert.Equal(t, 3, len(decoded.Subpackets.SNR), "Should have 3 subpacket SNR values")
		assert.Equal(t, 12.5, decoded.Subpackets.SNR[0])

		// Verify MessagePack roundtrip
		msgpackData, err := msgpack.Marshal(status)
		require.NoError(t, err)

		var decodedMP EPStatus
		err = msgpack.Unmarshal(msgpackData, &decodedMP)
		require.NoError(t, err)
		require.NotNil(t, decodedMP.EqSnr)
		assert.Equal(t, 14.2, *decodedMP.EqSnr)
		require.NotNil(t, decodedMP.Subpackets)
		assert.Equal(t, 3, len(decodedMP.Subpackets.SNR))
		assert.Equal(t, 12.5, decodedMP.Subpackets.SNR[0])
	})
}

// TestNumeric4InEPStatus specifically tests Numeric4 serialization within EPStatus
func TestNumeric4InEPStatus(t *testing.T) {
	nonce := mioty.Numeric4{0xDE, 0xAD, 0xBE, 0xEF}
	sign := mioty.Numeric4{0xCA, 0xFE, 0xBA, 0xBE}

	status := EPStatus{
		BaseMessage: BaseMessage{
			Command: "epStat",
			OpId:    -100,
		},
		EpEui:    0xFFEEDDCCBBAA9988,
		EpStatus: EPStatusAttached,
		Nonce:    &nonce,
		Sign:     &sign,
	}

	t.Run("JSONArrayFormat", func(t *testing.T) {
		jsonData, err := json.Marshal(status)
		require.NoError(t, err)

		// Verify nonce and sign are encoded as arrays [1,2,3,4], not base64
		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, `"nonce":[222,173,190,239]`, "Nonce should be JSON array")
		assert.Contains(t, jsonStr, `"sign":[202,254,186,190]`, "Sign should be JSON array")
		assert.NotContains(t, jsonStr, "base64", "Should not use base64 encoding")

		// Verify roundtrip
		var decoded EPStatus
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err)
		require.NotNil(t, decoded.Nonce)
		assert.Equal(t, nonce, *decoded.Nonce)
		require.NotNil(t, decoded.Sign)
		assert.Equal(t, sign, *decoded.Sign)
	})

	t.Run("MessagePackArrayFormat", func(t *testing.T) {
		msgpackData, err := msgpack.Marshal(status)
		require.NoError(t, err)

		// Verify roundtrip
		var decoded EPStatus
		err = msgpack.Unmarshal(msgpackData, &decoded)
		require.NoError(t, err)
		require.NotNil(t, decoded.Nonce)
		assert.Equal(t, nonce, *decoded.Nonce, "Nonce should roundtrip via MessagePack")
		require.NotNil(t, decoded.Sign)
		assert.Equal(t, sign, *decoded.Sign, "Sign should roundtrip via MessagePack")
	})
}
