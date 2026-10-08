package logger

import (
	"strconv"
	"unicode/utf8"
)

// UntrustedValue renders a value a remote peer chose for a log field: cut to
// at most limit bytes on a UTF-8 boundary and quoted, so it can neither flood
// the log nor forge a line with control characters.
func UntrustedValue(value string, limit int) string {
	if len(value) > limit {
		value = withoutPartialTrailingRune(value[:limit])
	}
	return strconv.Quote(value)
}

// withoutPartialTrailingRune drops only a rune the cut split; other invalid
// bytes stay and are quoted as escapes.
func withoutPartialTrailingRune(value string) string {
	for i := len(value) - 1; i >= 0 && i > len(value)-utf8.UTFMax; i-- {
		if utf8.RuneStart(value[i]) {
			if !utf8.FullRuneInString(value[i:]) {
				return value[:i]
			}
			return value
		}
	}
	return value
}
