// Package downlinks queues, edits and revokes an organization's downlinks
// and audits each request.
package downlinks

import (
	"math"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Content is the SCACI §3.10.1 content of a downlink as a client sends it,
// before it is checked against the wire widths.
type Content struct {
	Payloads     [][]byte
	Priority     float32
	CntDepend    bool
	PacketCnt    []int64
	Format       uint32
	ResponseExp  bool
	ResponsePrio bool
	DlWindReq    bool
	ExpOnly      bool
	DlRxStatQry  bool
}

// checkedContent is content that passed check, narrowed to its wire widths.
type checkedContent struct {
	Content
	format    uint8
	packetCnt []uint32
}

// check validates the content against SCACI §3.10.1: a counter-dependent
// downlink carries one payload per packet counter, any other at most one
// (none for a pure acknowledgement); every payload fits the radio limit; the
// priority is a finite single-precision number; the format fits 8 bits and
// each packet counter 32.
func (c Content) check() (checkedContent, error) {
	if err := c.checkPayloads(); err != nil {
		return checkedContent{}, err
	}
	if !mioty.ValidPriority(c.Priority) {
		return checkedContent{}, ErrPriorityInvalid
	}
	if c.Format > math.MaxUint8 {
		return checkedContent{}, ErrFormatInvalid
	}
	checked := checkedContent{Content: c, format: uint8(c.Format)}
	if len(c.PacketCnt) == 0 {
		return checked, nil
	}
	checked.packetCnt = make([]uint32, len(c.PacketCnt))
	for i, cnt := range c.PacketCnt {
		if cnt < 0 || cnt > math.MaxUint32 {
			return checkedContent{}, ErrPacketCounterInvalid
		}
		checked.packetCnt[i] = uint32(cnt)
	}
	return checked, nil
}

// checkPayloads checks how many payloads the downlink carries and their size.
func (c Content) checkPayloads() error {
	switch {
	case c.CntDepend && len(c.Payloads) == 0:
		return ErrPayloadRequired
	case c.CntDepend && len(c.PacketCnt) != len(c.Payloads):
		return ErrPacketCountersUnpaired
	case !c.CntDepend && len(c.Payloads) > 1:
		return ErrTooManyPayloads
	}
	for _, payload := range c.Payloads {
		if len(payload) > mioty.MaxDLUserDataBytes {
			return ErrPayloadTooLarge
		}
	}
	return nil
}

// patch is the content as the pending downlink it replaces.
func (c checkedContent) patch() storage.DownlinkPatch {
	return storage.DownlinkPatch{
		Payloads:     c.Payloads,
		Priority:     c.Priority,
		CntDepend:    c.CntDepend,
		PacketCnt:    c.PacketCnt,
		Format:       c.format,
		ResponseExp:  c.ResponseExp,
		ResponsePrio: c.ResponsePrio,
		DlWindReq:    c.DlWindReq,
		ExpOnly:      c.ExpOnly,
		DlRxStatQry:  c.DlRxStatQry,
	}
}

// queueRequest is the content as the dlDataQue the SCACI handler core
// queues; the service center assigns the queue id.
func (c checkedContent) queueRequest(epEUI uint64) *mioty.DLDataQueue {
	return &mioty.DLDataQueue{
		EpEui:        epEUI,
		UserData:     c.Payloads,
		Prio:         &c.Priority,
		CntDepend:    c.CntDepend,
		PacketCnt:    c.packetCnt,
		Format:       &c.format,
		ResponseExp:  &c.ResponseExp,
		ResponsePrio: &c.ResponsePrio,
		DlWindReq:    &c.DlWindReq,
		ExpOnly:      &c.ExpOnly,
		DlRxStatQry:  &c.DlRxStatQry,
	}
}
