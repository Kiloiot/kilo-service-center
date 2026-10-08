package mioty

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

const rangedTestEUI = uint64(0x70B3D56770111505)

// A 64-bit field keeps every EUI exactly, in MessagePack and in JSON, where a
// float64 would round it, and refuses a number outside 0 to 2^64-1.
func TestRangedUint64KeepsEUIsAndRefusesOtherNumbers(t *testing.T) {
	encoded, err := msgpack.Marshal(rangedTestEUI)
	if err != nil {
		t.Fatal(err)
	}
	var fromMsgpack RangedUint[uint64]
	if err := msgpack.Unmarshal(encoded, &fromMsgpack); err != nil || fromMsgpack.Value != rangedTestEUI {
		t.Fatalf("msgpack decode = %#x, %v; want %#x", fromMsgpack.Value, err, rangedTestEUI)
	}
	var fromJSON RangedUint[uint64]
	if err := json.Unmarshal([]byte("8121069193317651717"), &fromJSON); err != nil || fromJSON.Value != rangedTestEUI {
		t.Fatalf("JSON decode = %#x, %v; want %#x", fromJSON.Value, err, rangedTestEUI)
	}
	var maxJSON RangedUint[uint64]
	if err := json.Unmarshal([]byte("18446744073709551615"), &maxJSON); err != nil || maxJSON.Value != math.MaxUint64 {
		t.Fatalf("JSON decode of 2^64-1 = %d, %v", maxJSON.Value, err)
	}

	for name, value := range map[string]interface{}{"negative": -1, "fraction": 1.5, "2^64 as float": math.Pow(2, 64)} {
		encoded, err := msgpack.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var decoded RangedUint[uint64]
		if err := msgpack.Unmarshal(encoded, &decoded); !errors.Is(err, ErrNumericOutOfRange) {
			t.Errorf("msgpack %s: err = %v, want ErrNumericOutOfRange", name, err)
		}
	}
	for _, text := range []string{"-1", "18446744073709551616", "1.5"} {
		var decoded RangedUint[uint64]
		if err := json.Unmarshal([]byte(text), &decoded); !errors.Is(err, ErrNumericOutOfRange) {
			t.Errorf("JSON %s: err = %v, want ErrNumericOutOfRange", text, err)
		}
	}
}
