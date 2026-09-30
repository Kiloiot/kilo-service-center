package mioty

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

// MarshalJSON implements custom JSON marshaling to numeric array
func (n Numeric4) MarshalJSON() ([]byte, error) {
	// Serialize as [1,2,3,4], not base64
	return json.Marshal([4]uint8(n))
}

// UnmarshalJSON implements custom JSON unmarshaling with length validation
func (n *Numeric4) UnmarshalJSON(data []byte) error {
	// First unmarshal to a slice to check length
	var slice []uint8
	if err := json.Unmarshal(data, &slice); err != nil {
		return fmt.Errorf("%s: %w", errWrapNonceSignMustBe4ElementNumericArray, err)
	}
	if len(slice) != 4 {
		return fmt.Errorf(errFmtNonceSignMustBe4ElementNumericArray, len(slice))
	}
	// Copy to fixed array
	copy(n[:], slice)
	return nil
}

// EncodeMsgpack implements vmihailenco/msgpack/v5 custom encoding
func (n Numeric4) EncodeMsgpack(enc *msgpack.Encoder) error {
	// Encode as 4-element array
	if err := enc.EncodeArrayLen(numeric4Len); err != nil {
		return err
	}
	for _, v := range n {
		if err := enc.EncodeUint8(v); err != nil {
			return err
		}
	}
	return nil
}

// DecodeMsgpack implements vmihailenco/msgpack/v5 custom decoding
func (n *Numeric4) DecodeMsgpack(dec *msgpack.Decoder) error {
	arrLen, err := dec.DecodeArrayLen()
	if err != nil {
		return err
	}
	if arrLen != numeric4Len {
		return fmt.Errorf(errFmtNonceSignMustBe4ElementsGot, arrLen)
	}

	for i := 0; i < numeric4Len; i++ {
		v, err := dec.DecodeUint8()
		if err != nil {
			return err
		}
		n[i] = v
	}
	return nil
}

// encodeNumericBytes writes bytes in the Numeric[n] shape: an array of byte
// values, never a MessagePack binary.
func encodeNumericBytes(enc *msgpack.Encoder, data []byte) error {
	if err := enc.EncodeArrayLen(len(data)); err != nil {
		return err
	}
	for _, b := range data {
		if err := enc.EncodeUint8(b); err != nil {
			return err
		}
	}
	return nil
}

// decodeNumericBytes reads a Numeric[n] byte field in the specification's
// array shape and also accepts the binary form of clients that encode bytes
// natively; a nil value reads as nil.
func decodeNumericBytes(dec *msgpack.Decoder) ([]byte, error) {
	code, err := dec.PeekCode()
	if err != nil {
		return nil, err
	}
	if msgpcode.IsBin(code) || msgpcode.IsString(code) {
		return dec.DecodeBytes()
	}
	length, err := dec.DecodeArrayLen()
	if err != nil || length < 0 {
		return nil, err
	}
	values := make([]byte, length)
	for i := range values {
		value, err := dec.DecodeInterfaceLoose()
		if err != nil {
			return nil, err
		}
		if values[i], err = rangedValue[byte](value); err != nil {
			return nil, err
		}
	}
	return values, nil
}

// unmarshalNumericBytes reads a JSON Numeric[n] byte field: an array of byte
// values, or the base64 string Go encodes byte slices as; null reads as nil.
func unmarshalNumericBytes(data []byte) ([]byte, error) {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	switch v := value.(type) {
	case nil:
		return nil, nil
	case string:
		return base64.StdEncoding.DecodeString(v)
	case []interface{}:
		values := make([]byte, len(v))
		for i, element := range v {
			var err error
			if values[i], err = rangedValue[byte](element); err != nil {
				return nil, err
			}
		}
		return values, nil
	default:
		return nil, fmt.Errorf(errFmtNumericType, value)
	}
}

// copyFixedNumeric copies a fixed Numeric array into dst, refusing another length.
func copyFixedNumeric(values, dst []byte) error {
	if len(values) != len(dst) {
		return fmt.Errorf(errFmtNumericLength, ErrNumericLength, len(values), len(dst))
	}
	copy(dst, values)
	return nil
}

// decodeFixedNumeric reads a Numeric[len(dst)] byte field into dst.
func decodeFixedNumeric(dec *msgpack.Decoder, dst []byte) error {
	values, err := decodeNumericBytes(dec)
	if err != nil {
		return err
	}
	return copyFixedNumeric(values, dst)
}

// unmarshalFixedNumeric reads a JSON Numeric[len(dst)] byte field into dst;
// null, like a MessagePack nil, leaves dst untouched.
func unmarshalFixedNumeric(data []byte, dst []byte) error {
	values, err := unmarshalNumericBytes(data)
	if err != nil || values == nil {
		return err
	}
	return copyFixedNumeric(values, dst)
}

// EncodeMsgpack writes the key as Numeric[16].
func (k NetworkKey) EncodeMsgpack(enc *msgpack.Encoder) error {
	return encodeNumericBytes(enc, k[:])
}

// DecodeMsgpack reads the key as Numeric[16] or as a 16-byte binary.
func (k *NetworkKey) DecodeMsgpack(dec *msgpack.Decoder) error {
	return decodeFixedNumeric(dec, k[:])
}

// UnmarshalJSON reads the key as Numeric[16] or as a base64 string.
func (k *NetworkKey) UnmarshalJSON(data []byte) error {
	return unmarshalFixedNumeric(data, k[:])
}

// EncodeMsgpack writes the user data as Numeric[n].
func (d UplinkUserData) EncodeMsgpack(enc *msgpack.Encoder) error {
	return encodeNumericBytes(enc, d)
}

// DecodeMsgpack reads the user data as Numeric[n] or as a binary.
func (d *UplinkUserData) DecodeMsgpack(dec *msgpack.Decoder) error {
	values, err := decodeNumericBytes(dec)
	*d = values
	return err
}

// MarshalJSON writes the user data as an array of byte values, not base64.
func (d UplinkUserData) MarshalJSON() ([]byte, error) {
	values := make([]uint16, len(d))
	for i, b := range d {
		values[i] = uint16(b)
	}
	return json.Marshal(values)
}

// UnmarshalJSON reads the user data as Numeric[n] or as a base64 string.
func (d *UplinkUserData) UnmarshalJSON(data []byte) error {
	values, err := unmarshalNumericBytes(data)
	*d = values
	return err
}

// EncodeMsgpack writes every entry as Numeric[n].
func (d DownlinkUserData) EncodeMsgpack(enc *msgpack.Encoder) error {
	if err := enc.EncodeArrayLen(len(d)); err != nil {
		return err
	}
	for _, entry := range d {
		if err := encodeNumericBytes(enc, entry); err != nil {
			return err
		}
	}
	return nil
}

// DecodeMsgpack reads each entry in the specification's numeric-array shape
// and also accepts binary entries from clients that encode bytes natively.
func (d *DownlinkUserData) DecodeMsgpack(dec *msgpack.Decoder) error {
	count, err := dec.DecodeArrayLen()
	if err != nil {
		return err
	}
	if count < 0 {
		*d = nil
		return nil
	}
	entries := make(DownlinkUserData, count)
	for i := range entries {
		entry, err := decodeNumericBytes(dec)
		if err != nil {
			return fmt.Errorf(errFmtUserDataEntry, i, err)
		}
		entries[i] = entry
	}
	*d = entries
	return nil
}

// UnmarshalJSON reads each entry as Numeric[n] or as a base64 string.
func (d *DownlinkUserData) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw == nil {
		*d = nil
		return nil
	}
	entries := make(DownlinkUserData, len(raw))
	for i, element := range raw {
		entry, err := unmarshalNumericBytes(element)
		if err != nil {
			return fmt.Errorf(errFmtUserDataEntry, i, err)
		}
		entries[i] = entry
	}
	*d = entries
	return nil
}

// MarshalJSON marshals the SessionUUID as an array of integers [1,2,3,...] instead of base64
// BSSCI §4-4.5 requires UUIDs to be marshaled as arrays of integers, not base64 strings
func (u SessionUUID) MarshalJSON() ([]byte, error) {
	arr := make([]int, 16)
	for i, b := range u {
		arr[i] = int(b)
	}
	return json.Marshal(arr)
}

// EncodeMsgpack marshals the UUID as an array of integers for MessagePack
// Uses value receiver to work with non-addressable values (struct fields passed by value)
func (u SessionUUID) EncodeMsgpack(enc *msgpack.Encoder) error {
	arr := make([]int, 16)
	for i, b := range u {
		arr[i] = int(b)
	}
	return enc.Encode(arr)
}

// DecodeMsgpack reads the UUID as Numeric[16] or as a 16-byte binary.
func (u *SessionUUID) DecodeMsgpack(dec *msgpack.Decoder) error {
	return decodeFixedNumeric(dec, u[:])
}

// UnmarshalJSON reads the UUID as Numeric[16] or as a base64 string.
func (u *SessionUUID) UnmarshalJSON(data []byte) error {
	return unmarshalFixedNumeric(data, u[:])
}
