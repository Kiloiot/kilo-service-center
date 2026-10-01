package adapters

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	messagesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	exportTestTenant    = int64(1)
	exportTestEpEUI     = uint64(0x70B3D56770111505)
	exportTestBsEUI     = uint64(0x70B3D59CD00009E6)
	exportTestOpID      = int64(9007199254740993) // 2^53 + 1: lost as a JSON number
	exportTestRxTime    = int64(1790600960702975291)
	exportTestEqSNR     = 11.5
	exportTestProfile   = "eu1"
	exportTestMode      = "ulp"
	exportTestPacketCnt = uint32(72)
)

// exportStore returns one fully populated uplink.
type exportStore struct {
	messageListingStore
}

func (exportStore) ListULDataMessages(context.Context, mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	eqSnr, profile, mode, duplicate := exportTestEqSNR, exportTestProfile, exportTestMode, true
	return []*mioty.ULDataMessage{{
		ID: "7", OpId: exportTestOpID, EpEui: exportTestEpEUI, BsEui: exportTestBsEUI, RxTime: exportTestRxTime,
		PacketCnt: exportTestPacketCnt, SNR: 9.25, RSSI: -101.5, EqSnr: &eqSnr, Profile: &profile, Mode: &mode,
		UserData: []byte{0x00, 0x2A}, DlOpen: true, DlAck: true, Duplicate: &duplicate,
		Subpackets: &mioty.Subpackets{SNR: []float64{9}, RSSI: []float64{-101}, Frequency: []int64{868180000}},
	}}, 1, nil
}

func exportOf(t *testing.T, format string) []byte {
	t.Helper()
	out, err := NewMessageListingStoreAdapter(exportStore{}).Export(testutil.TestContext(), exportTestTenant, nil, nil, format)
	require.NoError(t, err)
	return out
}

// 64-bit identifiers and the nanosecond reception time survive JSON parsing
// as strings; EUIs and payloads read as the CSV and the web interface show them.
func TestExport_JSONCarriesExactIdentifiers(t *testing.T) {
	var rows []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(exportOf(t, messagesservice.ExportFormatJSON)))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&rows))
	require.Len(t, rows, 1)
	row := rows[0]

	assert.Equal(t, mioty.FormatEUI64(exportTestEpEUI), row["epEui"])
	assert.Equal(t, mioty.FormatEUI64(exportTestBsEUI), row["bsEui"])
	assert.Equal(t, "9007199254740993", row["opId"])
	assert.Equal(t, time.Unix(0, exportTestRxTime).UTC().Format(time.RFC3339Nano), row["rxTime"])
	assert.Equal(t, "002a", row["userData"])
	for _, key := range []string{"eqSnr", "profile", "mode", "dlOpen", "dlAck", "duplicate", "subpackets"} {
		assert.Contains(t, row, key)
	}
}

// Both formats carry the same fields under the same names.
func TestExport_CSVAndJSONShareTheFields(t *testing.T) {
	records, err := csv.NewReader(bytes.NewReader(exportOf(t, messagesservice.ExportFormatCSV))).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(exportOf(t, messagesservice.ExportFormatJSON), &rows))
	keys := make([]string, 0, len(rows[0]))
	for key := range rows[0] {
		keys = append(keys, key)
	}
	assert.ElementsMatch(t, keys, records[0])

	byHeader := make(map[string]string, len(records[0]))
	for i, header := range records[0] {
		byHeader[header] = records[1][i]
	}
	assert.Equal(t, mioty.FormatEUI64(exportTestEpEUI), byHeader["epEui"])
	assert.Equal(t, "11.50", byHeader["eqSnr"])
	assert.Equal(t, "true", byHeader["dlAck"])
}
