package mioty

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
)

func TestFormatEUI64Dashed(t *testing.T) {
	tests := []struct {
		name string
		eui  uint64
		want string
	}{
		{name: "high-bit EUI", eui: 0xCAFECAFECAFECAFE, want: "CA-FE-CA-FE-CA-FE-CA-FE"},
		{name: "zero", eui: 0, want: "00-00-00-00-00-00-00-00"},
		{name: "default service center EUI", eui: 0x4B43000000000001, want: "4B-43-00-00-00-00-00-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatEUI64Dashed(tt.eui); got != tt.want {
				t.Errorf("FormatEUI64Dashed(%#016x) = %q, want %q", tt.eui, got, tt.want)
			}
		})
	}
}

func TestFormatEUI64Dashed_RoundTripWithParseEUI(t *testing.T) {
	const eui = uint64(0xCAFECAFECAFECAFE)
	dashed := FormatEUI64Dashed(eui)
	parsed, err := validation.ParseEUI(dashed)
	if err != nil {
		t.Fatalf("ParseEUI(%q) error: %v", dashed, err)
	}
	if parsed != eui {
		t.Errorf("round-trip = %#016x, want %#016x", parsed, eui)
	}
}
