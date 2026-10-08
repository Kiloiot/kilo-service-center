package scaci

import (
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/encoding"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// The operation log records of the operations the service center initiates:
// a resume reissues them from these records (replay.go reads them back).

// buildBaseStationsRequestData converts BaseStationReception array to operation log format
// Returns array of maps with hex EUIs and numeric metrics for SCACI operation tracking
func buildBaseStationsRequestData(basestations []mioty.BaseStationReception) []map[string]interface{} {
	result := make([]map[string]interface{}, len(basestations))
	for i, bs := range basestations {
		entry := map[string]interface{}{
			"bsEui":  mioty.FormatEUI64(bs.BsEui),
			"rxTime": operationLogInt64(bs.RxTime),
			"snr":    bs.Snr,
			"rssi":   bs.Rssi,
		}
		if bs.RxDuration != nil {
			entry["rxDuration"] = *bs.RxDuration
		}
		if bs.EqSnr != nil {
			entry["eqSnr"] = *bs.EqSnr
		}
		if bs.DlRxSnr != nil {
			entry["dlRxSnr"] = *bs.DlRxSnr
		}
		if bs.DlRxRssi != nil {
			entry["dlRxRssi"] = *bs.DlRxRssi
		}
		if bs.Profile != nil {
			entry["profile"] = *bs.Profile
		}
		if bs.Mode != nil {
			entry["mode"] = *bs.Mode
		}
		// Include subpackets for evidence fidelity per SCACI §3.8.1
		if bs.Subpackets != nil {
			subpkts := map[string]interface{}{
				"snr":       bs.Subpackets.SNR,
				"rssi":      bs.Subpackets.RSSI,
				"frequency": bs.Subpackets.Frequency,
			}
			if len(bs.Subpackets.Phase) > 0 {
				subpkts["phase"] = bs.Subpackets.Phase
			}
			entry["subpackets"] = subpkts
		}
		result[i] = entry
	}
	return result
}

// dlDataResultRequestData is the operation log record of a dlDataRes, under
// the queue id the Application Center assigned.
func dlDataResultRequestData(result *mioty.DLDataResult, acQueID uint64) map[string]interface{} {
	requestData := map[string]interface{}{
		MetadataKeyEpEui: mioty.FormatEUI64(result.EpEui),
		"queId":          operationLogUint64(acQueID),
		"result":         result.Result,
	}
	if result.TxTime != nil {
		requestData["txTime"] = operationLogInt64(*result.TxTime)
	}
	if result.PacketCnt != nil {
		requestData["packetCnt"] = *result.PacketCnt
	}
	if result.BsEui != nil {
		requestData["bsEui"] = mioty.FormatEUI64(*result.BsEui)
	}
	return requestData
}

// epStatusRequestData is the operation log record of an epStatus; nonce and
// sign are stored base64-encoded like user data.
func epStatusRequestData(data *EPStatusData) map[string]interface{} {
	requestData := map[string]interface{}{
		MetadataKeyEpEui:    mioty.FormatEUI64(data.EpEui),
		metadataKeyEpStatus: data.EpStatus,
	}
	if data.AttachCnt != nil {
		requestData["attachCnt"] = *data.AttachCnt
	}
	if data.Nonce != nil {
		requestData[metadataKeyNonce] = encoding.EncodeUserData(data.Nonce[:])
	}
	if data.Sign != nil {
		requestData[metadataKeySign] = encoding.EncodeUserData(data.Sign[:])
	}
	if data.Snr != nil {
		requestData["snr"] = *data.Snr
	}
	if data.Rssi != nil {
		requestData["rssi"] = *data.Rssi
	}
	if data.EqSnr != nil {
		requestData["eqSnr"] = *data.EqSnr
	}
	if data.Subpackets != nil {
		requestData[metadataKeySubpackets] = data.Subpackets
	}
	return requestData
}
