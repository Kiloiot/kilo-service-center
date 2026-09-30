package scaci

import (
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
)

// The operation log is stored as JSONB and read back through a generic JSON
// decode, which turns every number into a float64. EUIs, queue ids and
// nanosecond timestamps exceed the 2^53 a float64 represents exactly, so they
// are recorded as text - EUIs as the hex mioty.FormatEUI64 writes, the others
// as decimal strings - and parsed back without loss.

func operationLogInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}

func operationLogUint64(v uint64) string {
	return strconv.FormatUint(v, 10)
}

// extractEUIFromJSON reads an EUI-64 from operation log data, recorded as the
// hex text mioty.FormatEUI64 writes.
func extractEUIFromJSON(data map[string]interface{}, key string) (uint64, bool) {
	text, ok := data[key].(string)
	if !ok {
		return 0, false
	}
	eui, err := validation.ParseEUI(text)
	return eui, err == nil
}

// extractInt64FromJSON reads an integer from operation log data, recorded
// either as a decimal string or as a JSON number.
func extractInt64FromJSON(data map[string]interface{}, key string) (int64, bool) {
	switch v := data[key].(type) {
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	}
	return 0, false
}

// extractUint64FromJSON reads an unsigned integer from operation log data,
// recorded either as a decimal string or as a non-negative JSON number.
func extractUint64FromJSON(data map[string]interface{}, key string) (uint64, bool) {
	if v, ok := data[key].(string); ok {
		n, err := strconv.ParseUint(v, 10, 64)
		return n, err == nil
	}
	n, ok := extractInt64FromJSON(data, key)
	if !ok || n < 0 {
		return 0, false
	}
	return uint64(n), true
}
