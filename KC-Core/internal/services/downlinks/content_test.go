package downlinks

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	contentFormat   = uint32(7)
	contentCounter  = int64(12)
	contentPriority = float32(2.0)
)

func TestContentCheck_RefusesEachSCACIRule(t *testing.T) {
	payload := []byte{0x01}
	cases := map[string]struct {
		content Content
		want    error
	}{
		"counter-dependent without payload":    {Content{CntDepend: true}, ErrPayloadRequired},
		"counters and payloads unpaired":       {Content{CntDepend: true, Payloads: [][]byte{payload}, PacketCnt: []int64{1, 2}}, ErrPacketCountersUnpaired},
		"counter-independent with two entries": {Content{Payloads: [][]byte{payload, payload}}, ErrTooManyPayloads},
		"payload over the radio limit":         {Content{Payloads: [][]byte{make([]byte, mioty.MaxDLUserDataBytes+1)}}, ErrPayloadTooLarge},
		"priority not a number":                {Content{Priority: float32(math.NaN())}, ErrPriorityInvalid},
		"priority infinite":                    {Content{Priority: float32(math.Inf(1))}, ErrPriorityInvalid},
		"format over 8 bits":                   {Content{Format: math.MaxUint8 + 1}, ErrFormatInvalid},
		"negative counter":                     {Content{CntDepend: true, Payloads: [][]byte{payload}, PacketCnt: []int64{-1}}, ErrPacketCounterInvalid},
		"counter over 32 bits":                 {Content{CntDepend: true, Payloads: [][]byte{payload}, PacketCnt: []int64{math.MaxUint32 + 1}}, ErrPacketCounterInvalid},
	}
	for name, tc := range cases {
		_, err := tc.content.check()
		assert.ErrorIs(t, err, tc.want, name)
	}
}

func TestContentCheck_AcceptsTheWireBounds(t *testing.T) {
	for name, content := range map[string]Content{
		"pure acknowledgement":   {},
		"priority above one":     {Payloads: [][]byte{{0x01}}, Priority: contentPriority},
		"largest format":         {Format: math.MaxUint8},
		"largest counter":        {CntDepend: true, Payloads: [][]byte{{0x01}}, PacketCnt: []int64{math.MaxUint32}},
		"payload at radio limit": {Payloads: [][]byte{make([]byte, mioty.MaxDLUserDataBytes)}},
	} {
		_, err := content.check()
		assert.NoError(t, err, name)
	}
}

func TestContentCheck_NarrowsToTheWireWidths(t *testing.T) {
	content := Content{
		Payloads: [][]byte{{0xAA}}, Priority: contentPriority, CntDepend: true, PacketCnt: []int64{contentCounter},
		Format: contentFormat, ResponseExp: true, DlRxStatQry: true,
	}
	checked, err := content.check()
	require.NoError(t, err)

	req := checked.queueRequest(0x0102030405060708)
	assert.Equal(t, uint64(0x0102030405060708), req.EpEui)
	assert.Equal(t, []uint32{uint32(contentCounter)}, req.PacketCnt)
	assert.Equal(t, uint8(contentFormat), *req.Format)
	assert.Equal(t, contentPriority, *req.Prio)
	assert.True(t, *req.ResponseExp)
	assert.True(t, *req.DlRxStatQry)

	patch := checked.patch()
	assert.Equal(t, uint8(contentFormat), patch.Format)
	assert.Equal(t, []int64{contentCounter}, patch.PacketCnt)
	assert.Equal(t, content.Payloads, patch.Payloads)
}
