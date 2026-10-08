package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// TestAssignAttachFields_LastAttachedStation: a NULL or malformed
// last_attached_bs_eui leaves the station unknown, while a stored zero EUI is
// still a station.
func TestAssignAttachFields_LastAttachedStation(t *testing.T) {
	zero := uint64(0)
	station := uint64(0x70B3D59CD00009E6)
	tests := []struct {
		name   string
		stored []byte
		want   *uint64
	}{
		{name: "NULL", stored: nil, want: nil},
		{name: "malformed", stored: []byte{0x70, 0xB3, 0xD5}, want: nil},
		{name: "zero EUI", stored: make([]byte, 8), want: &zero},
		{name: "station", stored: []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6}, want: &station},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ep models.EndPoint
			assignAttachFields(&ep, attachNullables{LastAttachedBsEui: tt.stored})
			assert.Equal(t, tt.want, ep.LastAttachedBsEui)
		})
	}
}
