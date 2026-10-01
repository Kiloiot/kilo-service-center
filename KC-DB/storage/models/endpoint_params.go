package models

import "time"

// OptionalBytes represents a byte-array column update that can be left
// untouched, set to a value, or set to SQL NULL. When Set is false the column
// is not part of the UPDATE; when Set is true a nil or empty Value writes NULL.
type OptionalBytes struct {
	Set   bool
	Value []byte
}

// OptionalNullTime represents a timestamp column update that can be left
// untouched, set to a value, or set to SQL NULL. When Set is false the column
// is not part of the UPDATE; when Set is true a nil Time writes NULL.
type OptionalNullTime struct {
	Set  bool
	Time *time.Time
}

// EndpointRegistrationParams carries the SCACI Register field set (§3.6.1).
// Every column is written on each call; NwkKey is encrypted at rest by the
// repository before the row is written. PacketCnt is written to both packet_cnt
// and last_packet_cnt.
type EndpointRegistrationParams struct {
	NwkKey      []byte
	PreAttach   bool
	Bidi        bool
	ShAddr      uint16
	AttachCnt   uint32
	PacketCnt   uint32
	DualChan    bool
	Repetition  bool
	WideCarrOff bool
	LongBlkDist bool
}

// EndpointAttachmentStateParams carries the attach and attach-propagate
// completion field set (BSSCI §5.6.1 / §5.9). Nil pointer fields and nil byte
// slices leave their column untouched so each call site touches only the subset
// it owns.
type EndpointAttachmentStateParams struct {
	AttachCnt            *int64
	LastAttachRxTime     *int64
	LastAttachRxDuration *int64
	Nonce                []byte
	Sign                 []byte
	DualChan             *bool
	Repetition           *bool
	WideCarrOff          *bool
	LongBlkDist          *bool
	ShAddr               *uint16
	LastAttachSubpackets *string
	LastAttachedBsEui    []byte
	LastPropagateTime    *int64
	PropagateStatus      *string
	Propagated           *bool
	PropagatedAt         OptionalNullTime
}

// EndpointAttachSessionParams carries the attach-propagate radio parameters
// (BSSCI §5.8.1). Every column is written on each call.
type EndpointAttachSessionParams struct {
	PropagatedAt  time.Time
	ShAddr        uint16
	Bidi          bool
	LastPacketCnt uint32
	DualChan      bool
	Repetition    bool
	WideCarrOff   bool
	LongBlkDist   bool
}

// EndpointDetachStateParams carries the detach field set for the detach,
// detach-propagate, and shared detach paths (BSSCI §5.7.1). Nil pointer fields
// and nil byte slices leave their column untouched; LastAttachedBsEui and
// PropagatedAt additionally distinguish "set to NULL" from "leave".
type EndpointDetachStateParams struct {
	LastAttachedBsEui   OptionalBytes
	LastPropagateTime   *int64
	LastDetachTime      *int64
	LastDetachSign      []byte
	LastDetachPacketCnt *uint32
	PropagateStatus     *string
	Propagated          *bool
	PropagatedAt        OptionalNullTime
}
