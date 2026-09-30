package mioty

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
)

func TestFormatEUI64_CasePolicy(t *testing.T) {
	const eui = uint64(0xCAFEcafe0000BEEF)
	if got := FormatEUI64(eui); got != "CAFECAFE0000BEEF" {
		t.Errorf("FormatEUI64 = %q, want uppercase", got)
	}
	if got := FormatEUI64Lower(eui); got != "cafecafe0000beef" {
		t.Errorf("FormatEUI64Lower = %q, want lowercase", got)
	}
	if got := FormatEUI64(1); got != "0000000000000001" {
		t.Errorf("FormatEUI64 must zero-pad to 16 digits, got %q", got)
	}
	bytes := []byte{0xCA, 0xFE, 0xCA, 0xFE, 0x00, 0x00, 0xBE, 0xEF}
	if got := FormatEUIBytes(bytes); got != "CAFECAFE0000BEEF" {
		t.Errorf("FormatEUIBytes = %q, want uppercase", got)
	}
}

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

func TestOptionalEUI64FromBytes(t *testing.T) {
	zero := uint64(0)
	station := uint64(0x70B3D59CD00009E6)
	tests := []struct {
		name  string
		bytes []byte
		want  *uint64
	}{
		{name: "null column", bytes: nil, want: nil},
		{name: "short", bytes: []byte{0x70, 0xB3}, want: nil},
		{name: "long", bytes: make([]byte, 9), want: nil},
		{name: "zero EUI", bytes: make([]byte, 8), want: &zero},
		{name: "EUI", bytes: []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6}, want: &station},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OptionalEUI64FromBytes(tt.bytes)
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Errorf("OptionalEUI64FromBytes(%x) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}
