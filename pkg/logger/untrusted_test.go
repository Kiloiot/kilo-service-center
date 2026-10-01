package logger

import "testing"

// Limits of the UntrustedValue cases: one no case reaches, one that cuts
// "abcdefgh" to "abc", and one that falls inside the two bytes of "é".
const (
	untrustedRoomyLimit  = 64
	untrustedCutLimit    = 3
	untrustedInRuneLimit = 2
)

func TestUntrustedValue(t *testing.T) {
	tests := map[string]struct {
		value string
		limit int
		want  string
	}{
		"short value is quoted":              {value: "abc", limit: untrustedRoomyLimit, want: `"abc"`},
		"line break cannot forge a line":     {value: "a\nlevel=error msg=forged", limit: untrustedRoomyLimit, want: `"a\nlevel=error msg=forged"`},
		"control characters are escaped":     {value: "a\x1b[31m\r", limit: untrustedRoomyLimit, want: `"a\x1b[31m\r"`},
		"long value is cut to the limit":     {value: "abcdefgh", limit: untrustedCutLimit, want: `"abc"`},
		"cut never splits a UTF-8 rune":      {value: "aé", limit: untrustedInRuneLimit, want: `"a"`},
		"quote in the value stays inside":    {value: `x"y`, limit: untrustedRoomyLimit, want: `"x\"y"`},
		"empty value is an empty quotation":  {value: "", limit: untrustedRoomyLimit, want: `""`},
		"cut keeps invalid bytes as escapes": {value: "a\xffbcdef", limit: untrustedCutLimit, want: `"a\xffb"`},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := UntrustedValue(tt.value, tt.limit); got != tt.want {
				t.Errorf("UntrustedValue(%q, %d) = %s, want %s", tt.value, tt.limit, got, tt.want)
			}
		})
	}
}
