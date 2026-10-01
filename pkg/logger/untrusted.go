package logger

import (
	"strconv"
	"strings"
)

// UntrustedValue renders a value a remote peer chose for a log field: cut to
// at most limit bytes on a UTF-8 boundary and quoted, so it can neither flood
// the log nor forge a line with control characters.
func UntrustedValue(value string, limit int) string {
	if len(value) > limit {
		value = strings.ToValidUTF8(value[:limit], "")
	}
	return strconv.Quote(value)
}
