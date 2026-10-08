package adapters

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"

	messagesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
	miotyformat "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// uplinkExportRow is one uplink of a message export: the SCACI §3.8.1 ulData
// fields of its first reception. 64-bit identifiers and the nanosecond
// reception time are strings, so JSON readers keep every digit; EUIs and the
// payload read as hex, as the web interface shows them.
type uplinkExportRow struct {
	ID          string            `json:"id"`
	OpID        string            `json:"opId"`
	EpEui       string            `json:"epEui"`
	BsEui       string            `json:"bsEui"`
	RxTime      string            `json:"rxTime"`
	RxDuration  *int64            `json:"rxDuration"`
	PacketCnt   uint32            `json:"packetCnt"`
	SNR         float64           `json:"snr"`
	RSSI        float64           `json:"rssi"`
	EqSnr       *float64          `json:"eqSnr"`
	Profile     *string           `json:"profile"`
	Mode        *string           `json:"mode"`
	Format      *uint8            `json:"format"`
	DlOpen      bool              `json:"dlOpen"`
	ResponseExp bool              `json:"responseExp"`
	DlAck       bool              `json:"dlAck"`
	Duplicate   bool              `json:"duplicate"`
	UserData    string            `json:"userData"`
	Subpackets  *mioty.Subpackets `json:"subpackets"`
}

func newUplinkExportRow(m *mioty.ULDataMessage) uplinkExportRow {
	return uplinkExportRow{
		ID:          m.ID,
		OpID:        strconv.FormatInt(m.OpId, 10),
		EpEui:       mioty.FormatEUI64(m.EpEui),
		BsEui:       mioty.FormatEUI64(m.BsEui),
		RxTime:      miotyformat.FormatTimestampRFC3339(m.RxTime),
		RxDuration:  m.RxDuration,
		PacketCnt:   m.PacketCnt,
		SNR:         m.SNR,
		RSSI:        m.RSSI,
		EqSnr:       m.EqSnr,
		Profile:     m.Profile,
		Mode:        m.Mode,
		Format:      m.Format,
		DlOpen:      m.DlOpen,
		ResponseExp: m.ResponseExp,
		DlAck:       m.DlAck,
		Duplicate:   m.Duplicate != nil && *m.Duplicate,
		UserData:    miotyformat.FormatUserDataHex(m.UserData),
		Subpackets:  m.Subpackets,
	}
}

// csvRecord lists the row in messagesservice.ExportCSVHeaders order; an absent
// optional field is empty and the subpackets are their JSON text.
func (r uplinkExportRow) csvRecord() ([]string, error) {
	subpackets := ""
	if r.Subpackets != nil {
		text, err := json.Marshal(r.Subpackets)
		if err != nil {
			return nil, err
		}
		subpackets = string(text)
	}
	return []string{
		r.ID, r.OpID, r.EpEui, r.BsEui, r.RxTime, optionalInt(r.RxDuration),
		strconv.FormatUint(uint64(r.PacketCnt), 10),
		miotyformat.FormatFloat2Dec(r.SNR), miotyformat.FormatFloat2Dec(r.RSSI), optionalFloat(r.EqSnr),
		optionalString(r.Profile), optionalString(r.Mode), optionalFormat(r.Format),
		strconv.FormatBool(r.DlOpen), strconv.FormatBool(r.ResponseExp), strconv.FormatBool(r.DlAck),
		strconv.FormatBool(r.Duplicate), r.UserData, subpackets,
	}, nil
}

func uplinkExportRows(msgs []*mioty.ULDataMessage) []uplinkExportRow {
	rows := make([]uplinkExportRow, len(msgs))
	for i, m := range msgs {
		rows[i] = newUplinkExportRow(m)
	}
	return rows
}

func exportJSON(msgs []*mioty.ULDataMessage) ([]byte, error) {
	return json.Marshal(uplinkExportRows(msgs))
}

func exportCSV(msgs []*mioty.ULDataMessage) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(messagesservice.ExportCSVHeaders); err != nil {
		return nil, err
	}
	for _, row := range uplinkExportRows(msgs) {
		record, err := row.csvRecord()
		if err != nil {
			return nil, err
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func optionalInt(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}

func optionalFloat(v *float64) string {
	if v == nil {
		return ""
	}
	return miotyformat.FormatFloat2Dec(*v)
}

func optionalString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func optionalFormat(v *uint8) string {
	if v == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*v), 10)
}
