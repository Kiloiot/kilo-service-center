package scaci

import (
	"bytes"
	"encoding/json"
	"math"

	"github.com/vmihailenco/msgpack/v5"
)

// normalizeInt64 converts a decoded frame integer to int64: MessagePack picks
// the smallest integer width that holds the value, a JSON frame carries a
// json.Number (see decodePayload).
//
// Parameters:
//   - v: Value from the payload decode (interface{})
//
// Returns:
//   - int64: Normalized 64-bit signed integer
//   - bool: True if conversion succeeded
func normalizeInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case json.Number:
		n, err := val.Int64()
		return n, err == nil
	case int64:
		return val, true
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int16:
		return int64(val), true
	case int8:
		return int64(val), true
	case uint64:
		if val <= math.MaxInt64 {
			return int64(val), true
		}
	case uint32:
		return int64(val), true
	case uint16:
		return int64(val), true
	case uint8:
		return int64(val), true
	}
	return 0, false
}

// decodePayload decodes a frame payload in the codec it was framed in: SCACI
// frames carry a JSON or a MessagePack object (§1, §3), and a MessagePack map
// never starts with '{' or JSON whitespace. JSON numbers stay exact as
// json.Number, so a 64-bit opId survives the envelope decode.
func decodePayload(payload []byte, v interface{}) error {
	trimmed := bytes.TrimLeft(payload, jsonWhitespace)
	if len(trimmed) == 0 || trimmed[0] != jsonObjectStart {
		return msgpack.Unmarshal(payload, v)
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if decoder.More() {
		return errTrailingJSONData
	}
	return nil
}
