package scaciservices

import (
	"testing"

	"github.com/kilocenter/KC-Core/pkg/scaci"
)

func TestValidateConnectFields_ValidFreshSession(t *testing.T) {
	validator := NewSessionValidator()

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0x123456789ABCDEF0,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
		SnAcOpId: nil, // Fresh session - no resume fields
		SnScOpId: nil,
	}

	errToken := validator.ValidateConnectFields(req, 0)
	if errToken != "" {
		t.Errorf("expected valid fresh session, got error: %s", errToken)
	}
}

func TestValidateConnectFields_ValidResumeSession(t *testing.T) {
	validator := NewSessionValidator()

	acOpId := int64(100)
	scOpId := int64(200)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16{
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
			0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00,
		},
		SnAcOpId: &acOpId, // Resume session - both fields present
		SnScOpId: &scOpId,
	}

	errToken := validator.ValidateConnectFields(req, 0)
	if errToken != "" {
		t.Errorf("expected valid resume session, got error: %s", errToken)
	}
}

func TestValidateConnectFields_MissingVersion(t *testing.T) {
	validator := NewSessionValidator()

	req := &scaci.Connect{
		Version: "", // Empty version
		AcEui:   0x123456789ABCDEF0,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	errToken := validator.ValidateConnectFields(req, 0)
	if errToken != "scaci.error.missing_version" {
		t.Errorf("expected missing_version error, got: %s", errToken)
	}
}

func TestValidateConnectFields_ZeroAcEui(t *testing.T) {
	validator := NewSessionValidator()

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0, // Zero AcEui
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	errToken := validator.ValidateConnectFields(req, 0)
	if errToken != "scaci.error.missing_ac_eui" {
		t.Errorf("expected missing_ac_eui error, got: %s", errToken)
	}
}

func TestValidateConnectFields_ZeroSnAcUUID(t *testing.T) {
	validator := NewSessionValidator()

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0x123456789ABCDEF0,
		SnAcUUID: scaci.UUID16{ // All zeros
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		},
	}

	errToken := validator.ValidateConnectFields(req, 0)
	if errToken != "scaci.error.sn_ac_uuid_zero" {
		t.Errorf("expected sn_ac_uuid_zero error, got: %s", errToken)
	}
}

func TestValidateConnectFields_UnpairedResumeFields_AcOpIdOnly(t *testing.T) {
	validator := NewSessionValidator()

	acOpId := int64(300)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0x123456789ABCDEF0,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
		SnAcOpId: &acOpId, // SnAcOpId present
		SnScOpId: nil,     // SnScOpId missing - INVALID
	}

	errToken := validator.ValidateConnectFields(req, 0)
	if errToken != "scaci.error.sn_sc_op_id_required" {
		t.Errorf("expected sn_sc_op_id_required error, got: %s", errToken)
	}
}

func TestValidateConnectFields_UnpairedResumeFields_ScOpIdOnly(t *testing.T) {
	validator := NewSessionValidator()

	scOpId := int64(400)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0x123456789ABCDEF0,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
		SnAcOpId: nil,     // SnAcOpId missing - INVALID
		SnScOpId: &scOpId, // SnScOpId present
	}

	errToken := validator.ValidateConnectFields(req, 0)
	if errToken != "scaci.error.sn_ac_op_id_required" {
		t.Errorf("expected sn_ac_op_id_required error, got: %s", errToken)
	}
}

func TestIsUUIDZero(t *testing.T) {
	// Test zero UUID
	zeroUUID := scaci.UUID16{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	if !isUUIDZero(zeroUUID) {
		t.Error("expected zero UUID to be detected as zero")
	}

	// Test non-zero UUID
	nonZeroUUID := scaci.UUID16{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // Last byte non-zero
	}
	if isUUIDZero(nonZeroUUID) {
		t.Error("expected non-zero UUID to be detected as non-zero")
	}
}
