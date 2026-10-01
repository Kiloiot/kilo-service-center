package mioty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

// rangedValue converts a decoded number to T, refusing one outside T's range.
func rangedValue[T uint8 | uint16 | uint32 | uint64](value interface{}) (T, error) {
	number, err := unsignedValue(value, uint64(^T(0)))
	return T(number), err
}

// unsignedValue checks a decoded number against [0, limit]; an integral float
// counts as the integer it holds.
func unsignedValue(value interface{}, limit uint64) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		if v <= limit {
			return v, nil
		}
	case int64:
		if v >= 0 && uint64(v) <= limit {
			return uint64(v), nil
		}
	case float64:
		// float64(limit)+1 is exact below 64 bits and rounds to 2^64 at 64 bits.
		if v >= 0 && v < float64(limit)+1 && v == math.Trunc(v) {
			return uint64(v), nil
		}
	case json.Number:
		if n, err := strconv.ParseUint(v.String(), 10, 64); err == nil {
			return unsignedValue(n, limit)
		}
		f, err := v.Float64()
		if err != nil {
			return 0, fmt.Errorf(errFmtNumericType, value)
		}
		return unsignedValue(f, limit)
	default:
		return 0, fmt.Errorf(errFmtNumericType, value)
	}
	return 0, fmt.Errorf(errFmtNumericValue, ErrNumericOutOfRange, value)
}

// RangedUint is an unsigned Numeric field of T's width. It decodes from any
// numeric wire width and refuses a value outside T instead of truncating it
// (SCACI §2.4).
type RangedUint[T uint8 | uint16 | uint32 | uint64] struct{ Value T }

// DecodeMsgpack reads the number and checks it against T's range.
func (r *RangedUint[T]) DecodeMsgpack(dec *msgpack.Decoder) error {
	value, err := dec.DecodeInterfaceLoose()
	if err != nil {
		return err
	}
	r.Value, err = rangedValue[T](value)
	return err
}

// UnmarshalJSON reads the number exactly and checks it against T's range;
// null, like a MessagePack nil, leaves the zero value.
func (r *RangedUint[T]) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil || value == nil {
		return err
	}
	var err error
	r.Value, err = rangedValue[T](value)
	return err
}

// Pointer returns the value of a present optional field, nil for an absent one.
func (r *RangedUint[T]) Pointer() *T {
	if r == nil {
		return nil
	}
	return &r.Value
}

// ValidPriority reports a downlink priority the specifications admit: a
// finite single-precision number (SCACI §3.10.1, BSSCI §3.12.1).
func ValidPriority(prio float32) bool {
	number := float64(prio)
	return !math.IsNaN(number) && !math.IsInf(number, 0)
}

// priority is the dlDataQue prio, decoded from any numeric wire width.
type priority struct{ value float32 }

// DecodeMsgpack reads the number as a finite single-precision value.
func (p *priority) DecodeMsgpack(dec *msgpack.Decoder) error {
	value, err := dec.DecodeInterfaceLoose()
	if err != nil {
		return err
	}
	prio, err := priorityValue(value)
	p.value = prio
	return err
}

// UnmarshalJSON reads the number as a finite single-precision value.
func (p *priority) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	prio, err := priorityValue(value)
	p.value = prio
	return err
}

func (p *priority) pointer() *float32 {
	if p == nil {
		return nil
	}
	return &p.value
}

func priorityValue(value interface{}) (float32, error) {
	var number float64
	switch v := value.(type) {
	case float64:
		number = v
	case int64:
		number = float64(v)
	case uint64:
		number = float64(v)
	default:
		return 0, fmt.Errorf(errFmtNumericType, value)
	}
	prio := float32(number)
	if !ValidPriority(prio) {
		return 0, fmt.Errorf(errFmtNumericValue, ErrNumericOutOfRange, value)
	}
	return prio, nil
}

// jsonNull is the JSON literal of an absent value.
var jsonNull = []byte("null")

// Required is a mandatory message field: its decoders record whether the
// decoded message carried it, a null counting as absent (SCACI §2.4).
type Required[T any] struct {
	Value   T
	present bool
}

// Present reports whether the decoded message carried the field.
func (r *Required[T]) Present() bool {
	return r.present
}

// DecodeMsgpack reads the field value; a nil leaves the field absent.
func (r *Required[T]) DecodeMsgpack(dec *msgpack.Decoder) error {
	code, err := dec.PeekCode()
	if err != nil {
		return err
	}
	if code == msgpcode.Nil {
		return dec.DecodeNil()
	}
	r.present = true
	return dec.Decode(&r.Value)
}

// UnmarshalJSON reads the field value; null leaves the field absent.
func (r *Required[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, jsonNull) {
		return nil
	}
	r.present = true
	return json.Unmarshal(data, &r.Value)
}

// Presence reports whether a decoded message carried a field.
type Presence interface {
	Present() bool
}

// CheckedWire is the decode form of a message M: it names the fields the
// message's specification table makes mandatory and gives the checked values
// M's Go types.
type CheckedWire[M any] interface {
	MandatoryFields() map[string]Presence
	Message() (M, error)
}

// DecodeChecked decodes a MessagePack message once through W, its checked
// wire form, so no mandatory field is missing and no value is truncated into
// the message's Go types.
func DecodeChecked[W any, P interface {
	*W
	CheckedWire[M]
}, M any](dec *msgpack.Decoder, dst *M) error {
	var wire W
	if err := dec.Decode(&wire); err != nil {
		return err
	}
	return adoptChecked[W, P](&wire, dst)
}

// UnmarshalChecked is DecodeChecked for a JSON message.
func UnmarshalChecked[W any, P interface {
	*W
	CheckedWire[M]
}, M any](data []byte, dst *M) error {
	var wire W
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	return adoptChecked[W, P](&wire, dst)
}

// adoptChecked refuses a wire form that lacks a mandatory field, naming the
// first one in wire-name order, and otherwise stores its message in dst.
func adoptChecked[W any, P interface {
	*W
	CheckedWire[M]
}, M any](wire *W, dst *M) error {
	checked := P(wire)
	mandatory := checked.MandatoryFields()
	for _, name := range slices.Sorted(maps.Keys(mandatory)) {
		if !mandatory[name].Present() {
			return fmt.Errorf(errFmtMissingField, ErrMissingMandatoryField, name)
		}
	}
	message, err := checked.Message()
	if err != nil {
		return err
	}
	*dst = message
	return nil
}
