package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// downlinkUserDataDocument is the JSON document of downlink_queue.user_data:
// the entries of a counter-dependent downlink, one per packet counter
// (SCACI §3.10.1). The entries are base64 strings; numeric arrays are read too.
type downlinkUserDataDocument struct {
	UserData mioty.DownlinkUserData `json:"userData"`
}

// encodeDownlinkUserData renders the user_data column; a counter-independent
// downlink keeps its single entry in payload and stores none.
func encodeDownlinkUserData(cntDepend bool, entries [][]byte) ([]byte, error) {
	if !cntDepend || len(entries) == 0 {
		return nil, nil
	}
	document, err := json.Marshal(downlinkUserDataDocument{UserData: entries})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapMarshalUserData, err)
	}
	return document, nil
}

// applyDownlinkUserData reads the user_data column into the downlink; its
// first entry stands in for a payload the row does not carry.
func applyDownlinkUserData(downlink *storage.DownlinkMessage, column []byte) error {
	if len(column) == 0 {
		return nil
	}
	var document downlinkUserDataDocument
	if err := json.Unmarshal(column, &document); err != nil {
		return fmt.Errorf("%s: %w", errWrapDecodeUserData, err)
	}
	downlink.UserData = document.UserData
	if len(downlink.UserData) > 0 && len(downlink.Payload) == 0 {
		downlink.Payload = downlink.UserData[0]
	}
	return nil
}
