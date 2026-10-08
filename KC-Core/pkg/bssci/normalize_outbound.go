package bssci

import (

	// Shared MIOTY helpers (FormatEUI64, EPStatus)

	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// NormalizeSubpackets converts raw subpackets map to typed Subpackets struct.
// Reusable helper for attach, detach, UL data handlers, and federation frame parsing (BSSCI §5.6.1, §5.7.1, §5.10.1).
// Returns (subpackets, nil) on success or (nil, error) if no valid arrays found.
func NormalizeSubpackets(raw map[string]interface{}) (*mioty.Subpackets, error) {
	sp := &mioty.Subpackets{}

	// Extract SNR array (signal to noise ratio in dB)
	if snrVals, ok := raw["snr"].([]interface{}); ok {
		if extracted, valid := extractFloatSlice(snrVals); valid {
			sp.SNR = extracted
		}
	}

	// Extract RSSI array (signal strength in dBm)
	if rssiVals, ok := raw["rssi"].([]interface{}); ok {
		if extracted, valid := extractFloatSlice(rssiVals); valid {
			sp.RSSI = extracted
		}
	}

	// Extract frequency array (Hz)
	if freqVals, ok := raw["frequency"].([]interface{}); ok {
		if extracted, valid := extractHertzSlice(freqVals); valid {
			sp.Frequency = extracted
		}
	}

	// Extract phase array (degrees ±180)
	if phaseVals, ok := raw["phase"].([]interface{}); ok {
		if extracted, valid := extractFloatSlice(phaseVals); valid {
			sp.Phase = extracted
		}
	}

	// Validate at least one measurement array is present
	if len(sp.SNR)+len(sp.RSSI)+len(sp.Frequency)+len(sp.Phase) == 0 {
		return nil, errNoValidSubpacketArraysFound
	}

	return sp, nil
}

// parseSubpackets reads a present subpackets object (BSSCI §3.10.1): snr, rssi
// and frequency are Numeric[m] of one length, phase is optional of the same length.
func parseSubpackets(raw interface{}) (*mioty.Subpackets, bool) {
	fields, ok := raw.(map[string]interface{})
	if !ok {
		return nil, false
	}
	sp, err := NormalizeSubpackets(fields)
	if err != nil {
		return nil, false
	}
	m := len(sp.SNR)
	phaseValid := fields[wireFieldPhase] == nil || len(sp.Phase) == m
	return sp, m > 0 && len(sp.RSSI) == m && len(sp.Frequency) == m && phaseValid
}

// normalizeAttPrpMessage reconstructs an attPrp message with correct BSSCI types
func normalizeAttPrpMessage(msg map[string]interface{}) map[string]interface{} {
	// Start with original map to preserve unknown/optional fields
	normalized := make(map[string]interface{})
	for k, v := range msg {
		normalized[k] = v
	}

	// Command stays as string
	if cmd, ok := msg["command"].(string); ok {
		normalized["command"] = cmd
	}

	// opId must be int64 per BSSCI 3.2 (supports full 64-bit range)
	if opId, ok := msg["opId"].(float64); ok {
		normalized["opId"] = int64(opId)
	}

	// epEui must be uint64 per BSSCI 3.8.1
	if epEui, ok := msg["epEui"].(float64); ok {
		normalized["epEui"] = uint64(epEui)
	}

	// Helper to coerce boolean fields (handles bool, float64 0/1)
	getBool := func(key string) bool {
		if val, ok := msg[key].(bool); ok {
			return val
		}
		// Fallback for older data that might store booleans as numbers
		if val, ok := msg[key].(float64); ok {
			return val != 0
		}
		return false
	}

	// Boolean fields with coercion support
	normalized["bidi"] = getBool("bidi")
	normalized["dualChan"] = getBool("dualChan")
	normalized["repetition"] = getBool("repetition")
	normalized["wideCarrOff"] = getBool("wideCarrOff")
	normalized["longBlkDist"] = getBool("longBlkDist")

	// nwkSnKey must be Numeric[16] array - ALWAYS 16 elements
	normalizedKey := make([]interface{}, dbconfig.SessionKeySize)
	if keyArray, ok := msg["nwkSnKey"].([]interface{}); ok {
		for i := 0; i < dbconfig.SessionKeySize; i++ {
			if i < len(keyArray) {
				if val, ok := keyArray[i].(float64); ok {
					normalizedKey[i] = uint8(val)
				} else {
					normalizedKey[i] = uint8(0)
				}
			} else {
				normalizedKey[i] = uint8(0) // Pad with zeros
			}
		}
	} else {
		// Fill with zeros if missing
		for i := 0; i < dbconfig.SessionKeySize; i++ {
			normalizedKey[i] = uint8(0)
		}
	}
	normalized["nwkSnKey"] = normalizedKey

	// shAddr must be uint16 per BSSCI spec
	if shAddr, ok := msg["shAddr"].(float64); ok {
		normalized["shAddr"] = uint16(shAddr)
	}

	// lastPacketCnt must be uint32
	if packetCnt, ok := msg["lastPacketCnt"].(float64); ok {
		normalized["lastPacketCnt"] = uint32(packetCnt)
	}

	return normalized
}

// normalizeDetPrpMessage reconstructs a detPrp message with correct types
func normalizeDetPrpMessage(msg map[string]interface{}) map[string]interface{} {
	// Start with original to preserve optional fields
	normalized := make(map[string]interface{})
	for k, v := range msg {
		normalized[k] = v
	}

	// Fix required numeric types
	if opId, ok := msg["opId"].(float64); ok {
		normalized["opId"] = int64(opId)
	}
	if epEui, ok := msg["epEui"].(float64); ok {
		normalized["epEui"] = uint64(epEui)
	}

	return normalized
}

// normalizeDlDataRevMessage fixes numeric types for dlDataRev messages
func normalizeDlDataRevMessage(msg map[string]interface{}) map[string]interface{} {
	// Start with all existing fields to preserve any future optional data
	normalized := make(map[string]interface{})
	for k, v := range msg {
		normalized[k] = v
	}

	// Ensure command is present
	normalized["command"] = mioty.CmdDLDataRevoke

	// Coerce numeric types to ensure proper MessagePack encoding
	if opId, ok := msg["opId"].(float64); ok {
		normalized["opId"] = int64(opId)
	}

	if epEui, ok := msg["epEui"].(float64); ok {
		normalized["epEui"] = uint64(epEui)
	}

	// queId should be uint64 per canonical MIOTY type
	if queId, ok := msg["queId"].(float64); ok {
		normalized["queId"] = uint64(queId)
	} else if queId, ok := msg["queId"].(int64); ok {
		// Handle int64 input and convert to uint64
		queIdSafe, errToken := safeUint64(queId)
		if errToken != "" {
			// NOTE: Normalization function has no logger access, overflow handled with 0 default
			// Downstream protocol validation will catch invalid queId=0
			queIdSafe = 0
		}
		normalized["queId"] = queIdSafe
	}

	return normalized
}
