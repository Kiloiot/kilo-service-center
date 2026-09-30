package bssci

import (
	"encoding/base64"
	"fmt"
	// Import neutral scheduler contracts
)

// buildDLDataQueUserData keeps userData Numeric[m][n]: with cntDepend false the outer array
// holds exactly one (possibly zero-length) entry, or the base station rejects the frame with code=22.
func buildDLDataQueUserData(payloads [][]byte, cntDepend bool) []interface{} {
	entries := payloads
	if !cntDepend {
		entries = [][]byte{nil}
		if len(payloads) > 0 {
			entries = payloads[:1]
		}
	}

	userData := make([]interface{}, len(entries))
	for i, payload := range entries {
		numericPayload := make([]interface{}, len(payload))
		for j, b := range payload {
			numericPayload[j] = uint8(b)
		}
		userData[i] = numericPayload
	}
	return userData
}

// reconstitueDLDataQueMessage reconstitutes a dlDataQue message from sanitized storage
// This is used during session resume to rebuild the message with the full payload structure
func (s *Server) reconstitueDLDataQueMessage(sanitizedMsg map[string]interface{}, metadata map[string]interface{}, pendingOp *PendingOperation) (map[string]interface{}, error) {
	// Start with the sanitized message
	msg := make(map[string]interface{})
	for k, v := range sanitizedMsg {
		msg[k] = v
	}

	// Extract payloads from metadata
	var payloads [][]byte
	if encodedPayloads, ok := metadata["payloads"].([]interface{}); ok {
		payloads = make([][]byte, len(encodedPayloads))
		for i, encoded := range encodedPayloads {
			if encodedStr, ok := encoded.(string); ok {
				decoded, err := base64.StdEncoding.DecodeString(encodedStr)
				if err != nil {
					return nil, fmt.Errorf("%s %d: %w", ResolveErrorMessage(errFailedToDecodePayload), i, err)
				}
				payloads[i] = decoded
			}
		}
	} else if encodedPayloads, ok := metadata["payloads"].([]string); ok {
		// Handle direct string slice
		payloads = make([][]byte, len(encodedPayloads))
		for i, encodedStr := range encodedPayloads {
			decoded, err := base64.StdEncoding.DecodeString(encodedStr)
			if err != nil {
				return nil, fmt.Errorf("%s %d: %w", ResolveErrorMessage(errFailedToDecodePayload), i, err)
			}
			payloads[i] = decoded
		}
	} else {
		// Fallback to single payload from PendingOperation.Data
		if pendingOp != nil && len(pendingOp.Data) > 0 {
			payloads = [][]byte{pendingOp.Data}
		} else {
			payloads = [][]byte{{}} // Empty payload
		}
	}

	// Validate reconstructed payload sizes (MIOTY radio protocol §3.6.6.3)
	if err := validatePayloadSizes(payloads); err != nil {
		return nil, err
	}

	// Extract counter-dependency flag
	cntDepend, _ := metadata["cntDepend"].(bool)

	if !cntDepend && len(payloads) > 1 {
		return nil, fmt.Errorf(errFmtTokenGotValue, ResolveErrorMessage(errNonCounterDependentMultiPayload), len(payloads))
	}
	msg["userData"] = buildDLDataQueUserData(payloads, cntDepend)

	if cntDepend {
		msg["cntDepend"] = true

		// Restore packetCnt array - handle both []int64 and []interface{}
		switch packetCnt := metadata["packetCnt"].(type) {
		case []int64:
			// Native []int64 from session storage
			packetCntArray := make([]interface{}, len(packetCnt))
			for i, cnt := range packetCnt {
				packetCntArray[i] = cnt
			}
			msg["packetCnt"] = packetCntArray
		case []interface{}:
			// JSON-decoded []interface{}
			packetCntArray := make([]interface{}, len(packetCnt))
			for i, cnt := range packetCnt {
				switch v := cnt.(type) {
				case float64:
					packetCntArray[i] = int64(v)
				case int64:
					packetCntArray[i] = v
				case int:
					packetCntArray[i] = int64(v)
				}
			}
			msg["packetCnt"] = packetCntArray
		}
	}

	// Restore optional MIOTY fields (canonical numeric coercion covers
	// float64, native integers, and the strict json.Number resume decode)
	if format, err := coerceInt64(metadata["format"]); err == nil && format > 0 {
		formatU8, errToken := safeUint8(format)
		if errToken != "" {
			return nil, fmt.Errorf(errFmtTokenWithFormat, ResolveErrorMessage(errToken), format)
		}
		msg["format"] = formatU8
	}
	if responseExp, ok := metadata["responseExp"].(bool); ok && responseExp {
		msg["responseExp"] = true
	}
	if responsePrio, ok := metadata["responsePrio"].(bool); ok && responsePrio {
		msg["responsePrio"] = true
	}
	if dlWindReq, ok := metadata["dlWindReq"].(bool); ok && dlWindReq {
		msg["dlWindReq"] = true
	}
	if expOnly, ok := metadata["expOnly"].(bool); ok && expOnly {
		msg["expOnly"] = true
	}

	return msg, nil
}
