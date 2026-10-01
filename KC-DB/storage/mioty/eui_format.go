package mioty

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
)

// EUIDashSeparator joins EUI64 byte pairs in the dashed display format.
const EUIDashSeparator = "-"

// hexDigitsPerByte is the width of one byte in hex text.
const hexDigitsPerByte = 2

// FormatEUI64 renders an EUI64 as 16 uppercase hex digits, the display form
// for APIs, events and logs.
func FormatEUI64(eui uint64) string {
	return fmt.Sprintf("%016X", eui)
}

// FormatEUI64Lower renders an EUI64 as 16 lowercase hex digits, the form
// MQTT topic segments and payloads carry.
func FormatEUI64Lower(eui uint64) string {
	return fmt.Sprintf("%016x", eui)
}

// FormatEUI64Dashed renders an EUI64 as uppercase byte pairs joined by
// dashes (XX-XX-XX-XX-XX-XX-XX-XX), the certificate common-name form.
func FormatEUI64Dashed(eui uint64) string {
	plain := FormatEUI64(eui)
	pairs := make([]string, 0, len(plain)/hexDigitsPerByte)
	for i := 0; i < len(plain); i += hexDigitsPerByte {
		pairs = append(pairs, plain[i:i+hexDigitsPerByte])
	}
	return strings.Join(pairs, EUIDashSeparator)
}

// FormatEUIBytes renders EUI bytes as uppercase hex, the display form.
func FormatEUIBytes(eui []byte) string {
	return strings.ToUpper(hex.EncodeToString(eui))
}

// EUI64Bytes renders an EUI64 as its big-endian bytes, the form BYTEA
// columns and byte-typed filters carry.
func EUI64Bytes(eui uint64) []byte {
	b := make([]byte, dbconfig.EUISize)
	binary.BigEndian.PutUint64(b, eui)
	return b
}

// EUI64FromBytes reads big-endian EUI bytes; anything but exactly one EUI's
// worth of bytes reads as zero, which no endpoint or base station carries.
func EUI64FromBytes(b []byte) uint64 {
	if len(b) != dbconfig.EUISize {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

// OptionalEUI64FromBytes reads a nullable EUI column: nil unless the bytes
// are exactly one EUI's worth, so a stored zero EUI stays distinguishable.
func OptionalEUI64FromBytes(b []byte) *uint64 {
	if len(b) != dbconfig.EUISize {
		return nil
	}
	eui := EUI64FromBytes(b)
	return &eui
}
