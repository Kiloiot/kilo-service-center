package mqtt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// commandPayload is the JSON payload for command/down messages; data and
// entries are mutually exclusive.
type commandPayload struct {
	Data         *string        `json:"data"`
	Entries      []commandEntry `json:"entries"`
	Confirmed    bool           `json:"confirmed"`
	Format       *uint8         `json:"format"`
	Prio         *float32       `json:"prio"`
	ResponsePrio *bool          `json:"responsePrio"`
	DlWindReq    *bool          `json:"dlWindReq"`
	ExpOnly      *bool          `json:"expOnly"`
	DlRxStatQry  *bool          `json:"dlRxStatQry"`
	Ref          string         `json:"ref"`
	ExpiresAt    *string        `json:"expiresAt"`
}

// commandEntry is the payload for one endpoint packet counter of a
// counter-dependent downlink (BSSCI §3.12.1).
type commandEntry struct {
	PacketCnt *uint32 `json:"packetCnt"`
	Data      string  `json:"data"`
}

// decodeCommand parses the message; a field of the wrong type, and a ref
// beyond the length the queue stores, still yield the rest of the command so
// its ref can be echoed. refReadable tells whether the ref itself decoded
// within that length, so a repeat is recognized before the body is refused.
func decodeCommand(rawPayload []byte) (cmd commandPayload, refReadable bool, err error) {
	switch {
	case len(rawPayload) == 0:
		return cmd, false, refusalEmptyPayload
	case len(rawPayload) > config.MaxMessageSize:
		return cmd, false, &DownlinkRefusal{Code: RejectCodeMessageTooLarge, Message: fmt.Sprintf(RejectMsgMessageTooLargeFmt, config.MaxMessageSize)}
	}
	err = json.Unmarshal(rawPayload, &cmd)
	refReadable = cmd.Ref != "" && len(cmd.Ref) <= storage.MaxDownlinkRefBytes
	var typeErr *json.UnmarshalTypeError
	switch {
	case err == nil && len(cmd.Ref) > storage.MaxDownlinkRefBytes:
		return cmd, false, refusalRefTooLong
	case err == nil:
		return cmd, refReadable, nil
	case errors.As(err, &typeErr) && typeErr.Field != "":
		invalid := &DownlinkRefusal{Code: RejectCodeInvalidField, Message: fmt.Sprintf(RejectMsgInvalidFieldFmt, typeErr.Field)}
		return cmd, refReadable && typeErr.Field != CommandFieldRef, fmt.Errorf("%w: %w", invalid, err)
	default:
		return commandPayload{}, false, fmt.Errorf("%w: %w", refusalInvalidJSON, err)
	}
}

// downlink maps the command onto the canonical dlDataQue request (BSSCI §3.12.1).
func (c commandPayload) downlink(epEUI uint64) (*mioty.DLDataQueue, error) {
	userData, packetCnts, err := c.payloads()
	if err != nil {
		return nil, err
	}
	responseExp := c.Confirmed
	return &mioty.DLDataQueue{
		EpEui:        epEUI,
		CntDepend:    packetCnts != nil,
		PacketCnt:    packetCnts,
		UserData:     userData,
		Format:       c.Format,
		Prio:         c.Prio,
		ResponseExp:  &responseExp,
		ResponsePrio: c.ResponsePrio,
		DlWindReq:    c.DlWindReq,
		ExpOnly:      c.ExpOnly,
		DlRxStatQry:  c.DlRxStatQry,
	}, nil
}

// command reads what the command adds to its downlink: its ref and its
// optional RFC 3339 deadline.
func (c commandPayload) command() (storage.DownlinkCommand, error) {
	command := storage.DownlinkCommand{Ref: c.Ref}
	if c.ExpiresAt == nil {
		return command, nil
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, *c.ExpiresAt)
	if err != nil {
		return command, fmt.Errorf(errFmtInvalidExpiresAt, refusalInvalidExpiresAt, logger.FieldExpiresAt, logger.UntrustedValue(*c.ExpiresAt, len(time.RFC3339Nano)))
	}
	command.ExpiresAt = &expiresAt
	return command, nil
}

// payloads decodes the single data payload, or one payload per packet counter
// for counter-dependent entries; empty data is a pure acknowledgement.
func (c commandPayload) payloads() (mioty.DownlinkUserData, []uint32, error) {
	switch {
	case c.Data != nil && c.Entries != nil:
		return nil, nil, refusalDataWithEntries
	case c.Data != nil:
		payload, err := decodeDownlinkPayload(*c.Data)
		if err != nil {
			return nil, nil, err
		}
		return mioty.DownlinkUserData{payload}, nil, nil
	case c.Entries == nil:
		return nil, nil, refusalMissingData
	case len(c.Entries) == 0:
		return nil, nil, refusalEmptyEntries
	}
	userData := make(mioty.DownlinkUserData, 0, len(c.Entries))
	packetCnts := make([]uint32, 0, len(c.Entries))
	seen := make(map[uint32]bool, len(c.Entries))
	for _, entry := range c.Entries {
		if entry.PacketCnt == nil {
			return nil, nil, refusalMissingPacketCnt
		}
		if seen[*entry.PacketCnt] {
			return nil, nil, refusalDuplicatePacketCnt
		}
		seen[*entry.PacketCnt] = true
		payload, err := decodeDownlinkPayload(entry.Data)
		if err != nil {
			return nil, nil, err
		}
		userData = append(userData, payload)
		packetCnts = append(packetCnts, *entry.PacketCnt)
	}
	return userData, packetCnts, nil
}

// decodeDownlinkPayload decodes one base64 payload within the radio downlink
// limit (radio protocol §3.6.6.3).
func decodeDownlinkPayload(encoded string) ([]byte, error) {
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", refusalInvalidBase64, err)
	}
	if len(payload) > mioty.MaxDLUserDataBytes {
		return nil, refusalPayloadTooLarge
	}
	return payload, nil
}
