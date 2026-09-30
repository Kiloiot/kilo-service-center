package mioty

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

const checkedTestEpEui = uint64(0x70B3D56770111505)

// A required field is present only when the message carries a value: a
// missing key and a null both leave it absent, in MessagePack and in JSON.
func TestRequiredRecordsPresence(t *testing.T) {
	type wire struct {
		EpEui Required[RangedUint[uint64]] `json:"epEui" msgpack:"epEui"`
	}
	for name, fields := range map[string]map[string]interface{}{
		"missing": {},
		"null":    {"epEui": nil},
	} {
		encoded, err := msgpack.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		var fromMsgpack wire
		if err := msgpack.Unmarshal(encoded, &fromMsgpack); err != nil || fromMsgpack.EpEui.Present() {
			t.Errorf("msgpack %s: present = %v, err = %v; want absent", name, fromMsgpack.EpEui.Present(), err)
		}
		text, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		var fromJSON wire
		if err := json.Unmarshal(text, &fromJSON); err != nil || fromJSON.EpEui.Present() {
			t.Errorf("JSON %s: present = %v, err = %v; want absent", name, fromJSON.EpEui.Present(), err)
		}
	}

	encoded, err := msgpack.Marshal(map[string]interface{}{"epEui": checkedTestEpEui})
	if err != nil {
		t.Fatal(err)
	}
	var carried wire
	if err := msgpack.Unmarshal(encoded, &carried); err != nil || !carried.EpEui.Present() || carried.EpEui.Value.Value != checkedTestEpEui {
		t.Fatalf("carried epEui = %#x present %v, err = %v", carried.EpEui.Value.Value, carried.EpEui.Present(), err)
	}
}

// A checked message is refused when its table's mandatory row is missing and
// keeps the range check of the field it carries.
func TestDecodeCheckedRefusesMissingAndOutOfRangeFields(t *testing.T) {
	missing, err := msgpack.Marshal(map[string]interface{}{"queId": 1, "cntDepend": false, "userData": [][]int{{1}}})
	if err != nil {
		t.Fatal(err)
	}
	var queue DLDataQueue
	if err := msgpack.Unmarshal(missing, &queue); !errors.Is(err, ErrMissingMandatoryField) {
		t.Errorf("dlDataQue without epEui: err = %v, want ErrMissingMandatoryField", err)
	}

	outOfRange, err := msgpack.Marshal(map[string]interface{}{"epEui": -1, "queId": 1, "cntDepend": false, "userData": [][]int{{1}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := msgpack.Unmarshal(outOfRange, &queue); !errors.Is(err, ErrNumericOutOfRange) {
		t.Errorf("dlDataQue with a negative epEui: err = %v, want ErrNumericOutOfRange", err)
	}
}
