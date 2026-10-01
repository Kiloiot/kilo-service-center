package postgres

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// eventDeviceEUIKeys are the detail keys the device scopes match
// (appendOneDeviceScope) by containment, so they are stored in one form.
var eventDeviceEUIKeys = []string{models.EventDetailKeyBsEui, models.EventDetailKeyEpEui}

// canonicalEventEUI renders an EUI given as hex (any case, dashes or colons)
// in the stored form, mioty.FormatEUI64; false when it is not an EUI.
func canonicalEventEUI(eui string) (string, bool) {
	value, err := validation.ParseEUI(eui)
	if err != nil {
		return "", false
	}
	return mioty.FormatEUI64(value), true
}

// canonicalEventDetails stores the device EUIs an event's details name, given
// as hex strings or as numbers, in the form the device scopes match. Details
// that are not an object, and values that are not EUIs, are kept as written.
func canonicalEventDetails(details []byte) []byte {
	decoder := json.NewDecoder(bytes.NewReader(details))
	decoder.UseNumber()
	var fields map[string]interface{}
	if err := decoder.Decode(&fields); err != nil || fields == nil {
		return details
	}
	changed := false
	for _, key := range eventDeviceEUIKeys {
		canonical, ok := canonicalDetailEUI(fields[key])
		if ok && canonical != fields[key] {
			fields[key] = canonical
			changed = true
		}
	}
	if !changed {
		return details
	}
	rewritten, err := json.Marshal(fields)
	if err != nil {
		return details
	}
	return rewritten
}

func canonicalDetailEUI(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return canonicalEventEUI(v)
	case json.Number:
		numeric, err := strconv.ParseUint(v.String(), 10, 64)
		if err != nil {
			return "", false
		}
		return mioty.FormatEUI64(numeric), true
	default:
		return "", false
	}
}
