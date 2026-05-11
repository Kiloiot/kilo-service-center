package scaci

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// DLDataResult Validator Tests (SCACI §3.12.1)
// ============================================================================

func TestValidateDLDataResult(t *testing.T) {
	// Helper to create a valid "sent" message
	validSentMsg := func() *DLDataResult {
		bsEui := uint64(0x0102030405060708)
		txTime := int64(1234567890)
		packetCnt := uint32(42)
		return &DLDataResult{
			BaseMessage: BaseMessage{OpId: -1},
			EpEui:       0x0102030405060708,
			QueID:       123,
			Result:      ResultSent,
			BsEui:       &bsEui,
			TxTime:      &txTime,
			PacketCnt:   &packetCnt,
		}
	}

	// Helper: sent with nil bsEui
	sentMsgNoBsEui := func() *DLDataResult {
		msg := validSentMsg()
		msg.BsEui = nil
		return msg
	}

	// Helper: sent with zero bsEui
	sentMsgZeroBsEui := func() *DLDataResult {
		msg := validSentMsg()
		zero := uint64(0)
		msg.BsEui = &zero
		return msg
	}

	// Helper: sent with nil txTime
	sentMsgNoTxTime := func() *DLDataResult {
		msg := validSentMsg()
		msg.TxTime = nil
		return msg
	}

	// Helper: sent with zero txTime (IS valid per spec)
	sentMsgZeroTxTime := func() *DLDataResult {
		msg := validSentMsg()
		zero := int64(0)
		msg.TxTime = &zero
		return msg
	}

	// Helper: sent with nil packetCnt
	sentMsgNoPktCnt := func() *DLDataResult {
		msg := validSentMsg()
		msg.PacketCnt = nil
		return msg
	}

	// Helper: sent with zero packetCnt (IS valid per spec)
	sentMsgZeroPktCnt := func() *DLDataResult {
		msg := validSentMsg()
		zero := uint32(0)
		msg.PacketCnt = &zero
		return msg
	}

	tests := []struct {
		name        string
		msg         *DLDataResult
		wantErr     string // empty = no error
		specSection string
	}{
		// Mandatory fields
		{
			name:        "missing_epEui",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: -1}, QueID: 1, Result: ResultSent},
			wantErr:     errEpEuiZero,
			specSection: "§3.12.1",
		},
		{
			name:        "missing_queId",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: -1}, EpEui: 123, Result: ResultSent},
			wantErr:     errQueIDZero, // Reuse existing token per §3.12.1
			specSection: "§3.12.1",
		},
		{
			name:        "missing_result",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: -1}, EpEui: 123, QueID: 1},
			wantErr:     errDLDataResultMissingResult,
			specSection: "§3.12.1",
		},
		{
			name:        "invalid_result_enum",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: -1}, EpEui: 123, QueID: 1, Result: "unknown"},
			wantErr:     errDLDataResultInvalidResultEnum,
			specSection: "§3.12.1",
		},

		// OpId direction (CRITICAL per §3.2)
		{
			name:        "opId_zero_rejected",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: 0}, EpEui: 123, QueID: 1, Result: ResultExpired},
			wantErr:     errDLDataResultOpIDNotNegative,
			specSection: "§3.12.1",
		},
		{
			name:        "opId_positive_rejected",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: 42}, EpEui: 123, QueID: 1, Result: ResultExpired},
			wantErr:     errDLDataResultOpIDNotNegative,
			specSection: "§3.12.1",
		},
		{
			name:        "opId_negative_accepted",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: -5}, EpEui: 123, QueID: 1, Result: ResultExpired},
			wantErr:     "",
			specSection: "§3.12.1",
		},

		// Conditional: result == ResultSent (§3.12.1 requires presence, not specific values except bsEui)
		{
			name:        "sent_missing_bsEui",
			msg:         sentMsgNoBsEui(),
			wantErr:     errDLDataResultSentMissingBsEui,
			specSection: "§3.12.1",
		},
		{
			name:        "sent_invalid_bsEui_zero",
			msg:         sentMsgZeroBsEui(),
			wantErr:     errDLDataResultSentInvalidBsEui,
			specSection: "§3.12.1",
		},
		{
			name:        "sent_missing_txTime",
			msg:         sentMsgNoTxTime(),
			wantErr:     errDLDataResultSentMissingTxTime,
			specSection: "§3.12.1",
		},
		{
			name:        "sent_zero_txTime_valid",
			msg:         sentMsgZeroTxTime(),
			wantErr:     "", // zero txTime IS valid per spec
			specSection: "§3.12.1",
		},
		{
			name:        "sent_missing_packetCnt",
			msg:         sentMsgNoPktCnt(),
			wantErr:     errDLDataResultSentMissingPktCnt,
			specSection: "§3.12.1",
		},
		{
			name:        "sent_zero_packetCnt_valid",
			msg:         sentMsgZeroPktCnt(),
			wantErr:     "", // zero packetCnt IS valid per spec
			specSection: "§3.12.1",
		},
		{
			name:        "sent_valid",
			msg:         validSentMsg(),
			wantErr:     "",
			specSection: "§3.12.1",
		},

		// Non-sent results (optional fields not required)
		{
			name:        "expired_valid",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: -1}, EpEui: 123, QueID: 1, Result: ResultExpired},
			wantErr:     "",
			specSection: "§3.12.1",
		},
		{
			name:        "invalid_valid",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: -1}, EpEui: 123, QueID: 1, Result: ResultInvalid},
			wantErr:     "",
			specSection: "§3.12.1",
		},
		// "revoked" is NOT a valid wire enum per SCACI §3.12.1 - it is internal DB/BSSCI status only
		{
			name:        "revoked_rejected",
			msg:         &DLDataResult{BaseMessage: BaseMessage{OpId: -1}, EpEui: 123, QueID: 1, Result: ResultRevoked},
			wantErr:     errDLDataResultInvalidResultEnum,
			specSection: "§3.12.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ValidateDLDataResult(tt.msg)
			if tt.wantErr == "" {
				assert.Empty(t, gotErr, "Expected no error for %s (spec: %s)", tt.name, tt.specSection)
			} else {
				assert.Equal(t, tt.wantErr, gotErr, "Expected error %s for %s (spec: %s)", tt.wantErr, tt.name, tt.specSection)
			}
		})
	}
}

// ============================================================================
// EPStatus Validator Tests (SCACI §§3.2, 3.13.1)
// ============================================================================

func TestValidateEPStatus(t *testing.T) {
	// Per SCACI §3.2: SC-originated messages require negative opId
	// Per SCACI §3.13.1: OTA field requirements:
	// - attached: attachCnt, nonce, sign required
	// - detached: sign required

	tests := []struct {
		name        string
		msg         *EPStatus
		opId        int64
		wantErr     string
		specSection string
	}{
		// opId validation (§3.2)
		{
			name: "opid_positive_rejected",
			msg: func() *EPStatus {
				attachCnt := uint32(42)
				nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
				sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
				return &EPStatus{
					EpEui:     0x0102030405060708,
					EpStatus:  EPStatusAttached,
					AttachCnt: &attachCnt,
					Nonce:     &nonce,
					Sign:      &sign,
				}
			}(),
			opId:        1, // positive = invalid for SC-originated
			wantErr:     errOpIDSignMismatch,
			specSection: "§3.2",
		},
		{
			name: "opid_zero_rejected",
			msg: func() *EPStatus {
				attachCnt := uint32(42)
				nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
				sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
				return &EPStatus{
					EpEui:     0x0102030405060708,
					EpStatus:  EPStatusAttached,
					AttachCnt: &attachCnt,
					Nonce:     &nonce,
					Sign:      &sign,
				}
			}(),
			opId:        0, // zero = invalid for SC-originated
			wantErr:     errOpIDSignMismatch,
			specSection: "§3.2",
		},

		// Mandatory fields (§3.13.1)
		{
			name:        "missing_epEui",
			msg:         &EPStatus{EpStatus: EPStatusAttached},
			opId:        -1,
			wantErr:     errEpEuiZero,
			specSection: "§3.13.1",
		},
		{
			name:        "missing_epStatus",
			msg:         &EPStatus{EpEui: 123},
			opId:        -1,
			wantErr:     errEPStatusMissingStatus,
			specSection: "§3.13.1",
		},
		{
			name:        "invalid_status_enum",
			msg:         &EPStatus{EpEui: 123, EpStatus: "unknown"},
			opId:        -1,
			wantErr:     errEPStatusInvalidStatusEnum,
			specSection: "§3.13.1",
		},

		// OTA field requirements for attached (§3.13.1)
		{
			name: "attached_missing_attachCnt",
			msg: func() *EPStatus {
				nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
				sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
				return &EPStatus{
					EpEui:    0x0102030405060708,
					EpStatus: EPStatusAttached,
					Nonce:    &nonce,
					Sign:     &sign,
				}
			}(),
			opId:        -1,
			wantErr:     errEPStatusMissingAttachCnt,
			specSection: "§3.13.1",
		},
		{
			name: "attached_missing_nonce",
			msg: func() *EPStatus {
				attachCnt := uint32(42)
				sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
				return &EPStatus{
					EpEui:     0x0102030405060708,
					EpStatus:  EPStatusAttached,
					AttachCnt: &attachCnt,
					Sign:      &sign,
				}
			}(),
			opId:        -1,
			wantErr:     errEPStatusMissingNonce,
			specSection: "§3.13.1",
		},
		{
			name: "attached_missing_sign",
			msg: func() *EPStatus {
				attachCnt := uint32(42)
				nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
				return &EPStatus{
					EpEui:     0x0102030405060708,
					EpStatus:  EPStatusAttached,
					AttachCnt: &attachCnt,
					Nonce:     &nonce,
				}
			}(),
			opId:        -1,
			wantErr:     errEPStatusMissingSign,
			specSection: "§3.13.1",
		},

		// OTA field requirements for detached (§3.13.1)
		{
			name: "detached_missing_sign",
			msg: &EPStatus{
				EpEui:    0x0102030405060708,
				EpStatus: EPStatusDetached,
			},
			opId:        -1,
			wantErr:     errEPStatusMissingSign,
			specSection: "§3.13.1",
		},

		// Valid cases - attached with all OTA fields
		{
			name: "attached_valid_with_ota_fields",
			msg: func() *EPStatus {
				attachCnt := uint32(42)
				nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
				sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
				return &EPStatus{
					EpEui:     0x0102030405060708,
					EpStatus:  EPStatusAttached,
					AttachCnt: &attachCnt,
					Nonce:     &nonce,
					Sign:      &sign,
				}
			}(),
			opId:        -1,
			wantErr:     "",
			specSection: "§3.13.1",
		},

		// Valid cases - detached with sign
		{
			name: "detached_valid_with_sign",
			msg: func() *EPStatus {
				sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
				return &EPStatus{
					EpEui:    0x0102030405060708,
					EpStatus: EPStatusDetached,
					Sign:     &sign,
				}
			}(),
			opId:        -1,
			wantErr:     "",
			specSection: "§3.13.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ValidateEPStatus(tt.msg, tt.opId)
			if tt.wantErr == "" {
				assert.Empty(t, gotErr, "Expected no error for %s (spec: %s)", tt.name, tt.specSection)
			} else {
				assert.Equal(t, tt.wantErr, gotErr, "Expected error %s for %s (spec: %s)", tt.wantErr, tt.name, tt.specSection)
			}
		})
	}
}

// ============================================================================
// ConnectResponse Validator Tests (SCACI §3.3.2)
// ============================================================================

func TestValidateConnectResponse(t *testing.T) {
	validVersion := ProtocolVersionString
	validScEui := uint64(0x0102030405060708)
	validSnScUUID := UUID16{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00}

	tests := []struct {
		name        string
		msg         *ConnectResponse
		wantErr     string
		specSection string
	}{
		{
			name:        "nil_message",
			msg:         nil,
			wantErr:     errInvalidMessageFormat,
			specSection: "§3.3.2",
		},
		// SCACI §3.3.2: version is OPTIONAL - nil/empty version is valid
		// (SendConnectResponse applies default before wire emission)
		{
			name: "nil_version_valid",
			msg: &ConnectResponse{
				BaseMessage: BaseMessage{Command: CmdConnectResponse, OpId: 0},
				ScEui:       validScEui,
				SnScUUID:    validSnScUUID,
			},
			wantErr:     "", // CHANGED: version is optional per §3.3.2
			specSection: "§3.3.2",
		},
		{
			name: "empty_version_valid",
			msg: func() *ConnectResponse {
				empty := ""
				return &ConnectResponse{
					BaseMessage: BaseMessage{Command: CmdConnectResponse, OpId: 0},
					Version:     &empty,
					ScEui:       validScEui,
					SnScUUID:    validSnScUUID,
				}
			}(),
			wantErr:     "", // CHANGED: version is optional per §3.3.2
			specSection: "§3.3.2",
		},
		{
			name: "zero_scEui",
			msg: &ConnectResponse{
				BaseMessage: BaseMessage{Command: CmdConnectResponse, OpId: 0},
				Version:     &validVersion,
				ScEui:       0,
				SnScUUID:    validSnScUUID,
			},
			wantErr:     errConnectResponseMissingScEui,
			specSection: "§3.3.2",
		},
		{
			name: "zero_snScUUID",
			msg: &ConnectResponse{
				BaseMessage: BaseMessage{Command: CmdConnectResponse, OpId: 0},
				Version:     &validVersion,
				ScEui:       validScEui,
				SnScUUID:    UUID16{}, // All zeros
			},
			wantErr:     errConnectResponseMissingSnScUUID,
			specSection: "§3.3.2",
		},
		{
			name: "valid_message_with_version",
			msg: &ConnectResponse{
				BaseMessage: BaseMessage{Command: CmdConnectResponse, OpId: 0},
				Version:     &validVersion,
				ScEui:       validScEui,
				SnScUUID:    validSnScUUID,
			},
			wantErr:     "",
			specSection: "§3.3.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ValidateConnectResponse(tt.msg)
			if tt.wantErr == "" {
				assert.Empty(t, gotErr, "Expected no error for %s (spec: %s)", tt.name, tt.specSection)
			} else {
				assert.Equal(t, tt.wantErr, gotErr, "Expected error %s for %s (spec: %s)", tt.wantErr, tt.name, tt.specSection)
			}
		})
	}
}

// ============================================================================
// StatusResponse Validator Tests (SCACI §3.5.2)
// ============================================================================

func TestValidateStatusResponse(t *testing.T) {
	tests := []struct {
		name        string
		msg         *StatusResponse
		wantErr     string
		specSection string
	}{
		{
			name:        "nil_message",
			msg:         nil,
			wantErr:     errInvalidMessageFormat,
			specSection: "§3.5.2",
		},
		{
			name: "missing_message",
			msg: &StatusResponse{
				BaseMessage: BaseMessage{Command: CmdStatusResponse, OpId: 1},
				Time:        1234567890,
			},
			wantErr:     errStatusResponseMissingMessage,
			specSection: "§3.5.2",
		},
		{
			name: "empty_message",
			msg: &StatusResponse{
				BaseMessage: BaseMessage{Command: CmdStatusResponse, OpId: 1},
				Message:     "",
				Time:        1234567890,
			},
			wantErr:     errStatusResponseMissingMessage,
			specSection: "§3.5.2",
		},
		{
			name: "missing_time",
			msg: &StatusResponse{
				BaseMessage: BaseMessage{Command: CmdStatusResponse, OpId: 1},
				Message:     "Service Center operational",
			},
			wantErr:     errStatusResponseMissingTime,
			specSection: "§3.5.2",
		},
		{
			name: "valid_message",
			msg: &StatusResponse{
				BaseMessage: BaseMessage{Command: CmdStatusResponse, OpId: 1},
				Message:     "Service Center operational",
				Time:        1234567890,
			},
			wantErr:     "",
			specSection: "§3.5.2",
		},
		{
			name: "valid_with_optional_fields",
			msg: func() *StatusResponse {
				uptime := int64(3600)
				return &StatusResponse{
					BaseMessage:  BaseMessage{Command: CmdStatusResponse, OpId: 1},
					Message:      "Service Center operational",
					Time:         1234567890,
					Uptime:       &uptime,
					BaseStations: []BaseStationStatus{},
				}
			}(),
			wantErr:     "",
			specSection: "§3.5.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ValidateStatusResponse(tt.msg)
			if tt.wantErr == "" {
				assert.Empty(t, gotErr, "Expected no error for %s (spec: %s)", tt.name, tt.specSection)
			} else {
				assert.Equal(t, tt.wantErr, gotErr, "Expected error %s for %s (spec: %s)", tt.wantErr, tt.name, tt.specSection)
			}
		})
	}
}

// ============================================================================
// ULData Validator Tests (SCACI §3.8.1)
// ============================================================================

func TestValidateULData(t *testing.T) {
	validBaseStations := []BaseStationReception{
		{BsEui: 0x0102030405060708, RxTime: 1234567890},
	}

	tests := []struct {
		name        string
		msg         *ULData
		wantErr     string
		specSection string
	}{
		{
			name:        "nil_message",
			msg:         nil,
			wantErr:     errInvalidMessageFormat,
			specSection: "§3.8.1",
		},
		{
			name: "missing_epEui",
			msg: &ULData{
				BaseMessage:  BaseMessage{Command: CmdULData, OpId: -1},
				BaseStations: validBaseStations,
				UserData:     []byte{0x01, 0x02},
			},
			wantErr:     errEpEuiZero,
			specSection: "§3.8.1",
		},
		{
			name: "missing_baseStations",
			msg: &ULData{
				BaseMessage: BaseMessage{Command: CmdULData, OpId: -1},
				EpEui:       0x0102030405060708,
				UserData:    []byte{0x01, 0x02},
			},
			wantErr:     errULDataMissingBaseStations,
			specSection: "§3.8.1",
		},
		{
			name: "empty_baseStations",
			msg: &ULData{
				BaseMessage:  BaseMessage{Command: CmdULData, OpId: -1},
				EpEui:        0x0102030405060708,
				BaseStations: []BaseStationReception{},
				UserData:     []byte{0x01, 0x02},
			},
			wantErr:     errULDataMissingBaseStations,
			specSection: "§3.8.1",
		},
		{
			name: "missing_userData",
			msg: &ULData{
				BaseMessage:  BaseMessage{Command: CmdULData, OpId: -1},
				EpEui:        0x0102030405060708,
				BaseStations: validBaseStations,
			},
			wantErr:     errUserDataEmpty,
			specSection: "§3.8.1",
		},
		{
			name: "valid_message",
			msg: &ULData{
				BaseMessage:  BaseMessage{Command: CmdULData, OpId: -1},
				EpEui:        0x0102030405060708,
				BaseStations: validBaseStations,
				UserData:     []byte{0x01, 0x02, 0x03},
				PacketCnt:    42,
			},
			wantErr:     "",
			specSection: "§3.8.1",
		},
		{
			name: "valid_message_empty_userData_slice",
			msg: &ULData{
				BaseMessage:  BaseMessage{Command: CmdULData, OpId: -1},
				EpEui:        0x0102030405060708,
				BaseStations: validBaseStations,
				UserData:     []byte{}, // Empty slice is valid (non-nil)
				PacketCnt:    0,
			},
			wantErr:     "",
			specSection: "§3.8.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ValidateULData(tt.msg)
			if tt.wantErr == "" {
				assert.Empty(t, gotErr, "Expected no error for %s (spec: %s)", tt.name, tt.specSection)
			} else {
				assert.Equal(t, tt.wantErr, gotErr, "Expected error %s for %s (spec: %s)", tt.wantErr, tt.name, tt.specSection)
			}
		})
	}
}

// ============================================================================
// ULDataTransmit Validator Tests (SCACI §3.9.1)
// ============================================================================

func TestValidateULDataTransmit(t *testing.T) {
	// Helper to create a valid ULDataTransmit message
	validULDataTx := func() *ULDataTransmit {
		format := uint8(0)
		return &ULDataTransmit{
			EpEui:     0x0102030405060708,
			ShAddr:    uint16(0x1234), // ShAddr is uint16 per mioty.ULDataTransmit
			PacketCnt: 42,
			NwkSnKey:  [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			UserData:  []byte{0xDE, 0xAD, 0xBE, 0xEF},
			Format:    &format,
		}
	}

	tests := []struct {
		name        string
		msg         *ULDataTransmit
		wantErr     string
		specSection string
	}{
		// Nil message
		{
			name:        "nil_message",
			msg:         nil,
			wantErr:     errInvalidMessageFormat,
			specSection: "§3.9.1",
		},
		// Mandatory field: epEui
		{
			name: "missing_epEui_zero",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				m.EpEui = 0
				return m
			}(),
			wantErr:     errEpEuiZero,
			specSection: "§3.9.1",
		},
		// Mandatory field: userData
		{
			name: "missing_userData_nil",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				m.UserData = nil
				return m
			}(),
			wantErr:     errUserDataEmpty,
			specSection: "§3.9.1",
		},
		// Empty userData slice IS valid (non-nil)
		{
			name: "userData_empty_slice_valid",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				m.UserData = []byte{}
				return m
			}(),
			wantErr:     "",
			specSection: "§3.9.1",
		},
		// Zero shAddr IS valid (value type, spec mandates presence not value)
		{
			name: "shAddr_zero_valid",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				m.ShAddr = 0
				return m
			}(),
			wantErr:     "",
			specSection: "§3.9.1",
		},
		// Zero packetCnt IS valid (value type, spec mandates presence not value)
		{
			name: "packetCnt_zero_valid",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				m.PacketCnt = 0
				return m
			}(),
			wantErr:     "",
			specSection: "§3.9.1",
		},
		// All-zero nwkSnKey IS valid (type system enforces 16 bytes)
		{
			name: "nwkSnKey_zero_valid",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				m.NwkSnKey = [16]byte{}
				return m
			}(),
			wantErr:     "",
			specSection: "§3.9.1",
		},
		// Optional format nil IS valid (handler defaults to 0 per §2.4)
		{
			name: "format_nil_valid",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				m.Format = nil
				return m
			}(),
			wantErr:     "",
			specSection: "§3.9.1",
		},
		// Valid complete message
		{
			name:        "valid_message",
			msg:         validULDataTx(),
			wantErr:     "",
			specSection: "§3.9.1",
		},
		// Valid message with optional fields
		{
			name: "valid_with_optional_bsEui",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				bsEui := uint64(0xAABBCCDDEEFF0011)
				m.BsEui = &bsEui
				return m
			}(),
			wantErr:     "",
			specSection: "§3.9.1",
		},
		{
			name: "valid_with_optional_profile",
			msg: func() *ULDataTransmit {
				m := validULDataTx()
				profile := "standard" // Profile is *string per mioty.ULDataTransmit
				m.Profile = &profile
				return m
			}(),
			wantErr:     "",
			specSection: "§3.9.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ValidateULDataTransmit(tt.msg)
			if tt.wantErr == "" {
				assert.Empty(t, gotErr, "Expected no error for %s (spec: %s)", tt.name, tt.specSection)
			} else {
				assert.Equal(t, tt.wantErr, gotErr, "Expected error %s for %s (spec: %s)", tt.wantErr, tt.name, tt.specSection)
			}
		})
	}
}

// TestValidateULDataTransmit_POSIXMapping validates that error tokens map to POSIX_EINVAL
func TestValidateULDataTransmit_POSIXMapping(t *testing.T) {
	// All validation errors for ULDataTransmit should map to POSIX_EINVAL per §2.4
	errorTokens := []string{
		errEpEuiZero,
		errUserDataEmpty,
		errInvalidMessageFormat,
	}

	for _, token := range errorTokens {
		t.Run(token, func(t *testing.T) {
			// Verify error token exists in catalog (ensures it's a known error)
			def := GetErrorDefinition(token)
			assert.NotEqual(t, "unknown", def.SpecSection, "Error token %s should be in error catalog", token)

			// Note: The handler maps these to POSIX_EINVAL when calling sendErrorWithCatalog
			// This test documents the expected mapping behavior
		})
	}
}

// ============================================================================
// Error Validator Tests (SCACI §3.14.1)
// ============================================================================

func TestValidateError(t *testing.T) {
	tests := []struct {
		name        string
		msg         *Error
		wantErr     string
		specSection string
	}{
		{
			name:        "nil_message",
			msg:         nil,
			wantErr:     errInvalidMessageFormat,
			specSection: "§3.14.1",
		},
		{
			name: "missing_message",
			msg: &Error{
				BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
				Code:        POSIX_EINVAL,
			},
			wantErr:     errErrorMissingMessage,
			specSection: "§3.14.1",
		},
		{
			name: "empty_message",
			msg: &Error{
				BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
				Code:        POSIX_EINVAL,
				Message:     "",
			},
			wantErr:     errErrorMissingMessage,
			specSection: "§3.14.1",
		},
		{
			name: "valid_message",
			msg: &Error{
				BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
				Code:        POSIX_EINVAL,
				Message:     "Invalid operation",
			},
			wantErr:     "",
			specSection: "§3.14.1",
		},
		{
			name: "zero_code_rejected",
			msg: &Error{
				BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
				Code:        POSIX_OK, // Zero code is invalid for Error messages
				Message:     "Operation succeeded",
			},
			wantErr:     errErrorMissingCode, // code=0 is semantically wrong for Error
			specSection: "§3.14.1",
		},
		{
			name: "valid_non_zero_code",
			msg: &Error{
				BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
				Code:        POSIX_EINVAL, // Non-zero code is valid
				Message:     "Invalid argument",
			},
			wantErr:     "",
			specSection: "§3.14.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := ValidateError(tt.msg)
			if tt.wantErr == "" {
				assert.Empty(t, gotErr, "Expected no error for %s (spec: %s)", tt.name, tt.specSection)
			} else {
				assert.Equal(t, tt.wantErr, gotErr, "Expected error %s for %s (spec: %s)", tt.wantErr, tt.name, tt.specSection)
			}
		})
	}
}
