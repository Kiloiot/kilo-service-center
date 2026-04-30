// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kilocenter/KC-DB/storage/models"
)

// TestBuildBaseStationStatus_NoTelemetry verifies that telemetry fields
// (Code/Message/Uptime/GeoLocation) are nil when LastStatusAt is nil.
// This prevents fabricating a "healthy" status when no real telemetry exists.
func TestBuildBaseStationStatus_NoTelemetry(t *testing.T) {
	// Create base station with no LastStatusAt (no telemetry received)
	bs := &models.BaseStation{
		EUI:           models.EUI{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:          "Test BS",
		StatusCode:    0,                 // Would be healthy if emitted
		LastStatusAt:  nil,               // No real telemetry
		UptimeSeconds: ptr(int64(3600)),  // Would be 1 hour if emitted
		Latitude:      ptr(48.1351),      // Would be set if emitted
		Longitude:     ptr(11.5820),      // Would be set if emitted
		Altitude:      ptr(500.0),        // Would be set if emitted
		Vendor:        ptr("TestVendor"), // Identity - always emitted
		Model:         ptr("TestModel"),  // Identity - always emitted
	}

	result := BuildBaseStationStatus(bs)

	// Verify identity fields ARE emitted
	if result.Eui == nil {
		t.Error("expected Eui to be emitted")
	}
	if result.Vendor == nil || *result.Vendor != "TestVendor" {
		t.Error("expected Vendor to be emitted")
	}
	if result.Model == nil || *result.Model != "TestModel" {
		t.Error("expected Model to be emitted")
	}
	if result.Name == nil || *result.Name != "Test BS" {
		t.Error("expected Name to be emitted")
	}

	// Verify telemetry fields are NOT emitted (nil = unknown)
	if result.Code != nil {
		t.Errorf("expected Code to be nil when no telemetry, got %d", *result.Code)
	}
	if result.Message != nil {
		t.Errorf("expected Message to be nil when no telemetry, got %s", *result.Message)
	}
	if result.Uptime != nil {
		t.Errorf("expected Uptime to be nil when no telemetry, got %d", *result.Uptime)
	}
	if result.GeoLocation != nil {
		t.Errorf("expected GeoLocation to be nil when no telemetry, got %v", *result.GeoLocation)
	}
}

// TestBuildBaseStationStatus_WithTelemetry verifies that telemetry fields
// are populated when LastStatusAt is set (real telemetry exists).
func TestBuildBaseStationStatus_WithTelemetry(t *testing.T) {
	now := time.Now()
	statusMsg := "Base station operational"

	// Create base station with LastStatusAt set (real telemetry received)
	bs := &models.BaseStation{
		EUI:           models.EUI{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:          "Test BS",
		StatusCode:    0, // Healthy
		StatusMessage: &statusMsg,
		LastStatusAt:  &now,             // Real telemetry exists
		UptimeSeconds: ptr(int64(3600)), // 1 hour
		Latitude:      ptr(48.1351),
		Longitude:     ptr(11.5820),
		Altitude:      ptr(500.0),
		Vendor:        ptr("TestVendor"),
		Model:         ptr("TestModel"),
	}

	result := BuildBaseStationStatus(bs)

	// Verify identity fields are emitted
	if result.Eui == nil {
		t.Error("expected Eui to be emitted")
	}

	// Verify telemetry fields ARE emitted when LastStatusAt is set
	if result.Code == nil {
		t.Error("expected Code to be emitted with telemetry")
	} else if *result.Code != 0 {
		t.Errorf("expected Code=0, got %d", *result.Code)
	}

	if result.Message == nil {
		t.Error("expected Message to be emitted with telemetry")
	} else if *result.Message != statusMsg {
		t.Errorf("expected Message=%q, got %q", statusMsg, *result.Message)
	}

	if result.Uptime == nil {
		t.Error("expected Uptime to be emitted with telemetry")
	} else if *result.Uptime != 3600 {
		t.Errorf("expected Uptime=3600, got %d", *result.Uptime)
	}

	if result.GeoLocation == nil {
		t.Error("expected GeoLocation to be emitted with telemetry")
	} else {
		if (*result.GeoLocation)[0] != 48.1351 {
			t.Errorf("expected Latitude=48.1351, got %f", (*result.GeoLocation)[0])
		}
		if (*result.GeoLocation)[1] != 11.5820 {
			t.Errorf("expected Longitude=11.5820, got %f", (*result.GeoLocation)[1])
		}
		if (*result.GeoLocation)[2] != 500.0 {
			t.Errorf("expected Altitude=500.0, got %f", (*result.GeoLocation)[2])
		}
	}
}

// TestBuildBaseStationStatus_AlwaysEmitsIdentity verifies that identity fields
// (EUI, Bidi, Vendor, Model, Name) are always emitted regardless of telemetry state.
func TestBuildBaseStationStatus_AlwaysEmitsIdentity(t *testing.T) {
	bidi := true
	vendor := "Fraunhofer"
	model := "AVA-BS"

	testCases := []struct {
		name         string
		lastStatusAt *time.Time
	}{
		{"with telemetry", ptr(time.Now())},
		{"without telemetry", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := &models.BaseStation{
				EUI:          models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
				Name:         "Production BS 01",
				Bidi:         &bidi,
				Vendor:       &vendor,
				Model:        &model,
				LastStatusAt: tc.lastStatusAt,
			}

			result := BuildBaseStationStatus(bs)

			// Identity fields should always be present
			if result.Eui == nil {
				t.Error("Eui should always be emitted")
			}
			if result.Bidi == nil || *result.Bidi != true {
				t.Error("Bidi should be emitted when set")
			}
			if result.Vendor == nil || *result.Vendor != "Fraunhofer" {
				t.Error("Vendor should be emitted when set")
			}
			if result.Model == nil || *result.Model != "AVA-BS" {
				t.Error("Model should be emitted when set")
			}
			if result.Name == nil || *result.Name != "Production BS 01" {
				t.Error("Name should be emitted when set")
			}
		})
	}
}

// TestBuildBaseStationStatus_LoadMetricsNotGated verifies that UlLoad/DlLoad
// are emitted regardless of LastStatusAt (they're not gated to telemetry).
func TestBuildBaseStationStatus_LoadMetricsNotGated(t *testing.T) {
	ulLoad := float32(0.45)
	dlLoad := float32(0.23)

	// No telemetry but load metrics should still be emitted
	bs := &models.BaseStation{
		EUI:          models.EUI{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:         "Load Test BS",
		LastStatusAt: nil, // No telemetry
		ULLoad:       &ulLoad,
		DLLoad:       &dlLoad,
	}

	result := BuildBaseStationStatus(bs)

	// Load metrics should be emitted regardless of telemetry
	if result.UlLoad == nil {
		t.Error("expected UlLoad to be emitted")
	} else if *result.UlLoad != float64(ulLoad) {
		t.Errorf("expected UlLoad=%f, got %f", float64(ulLoad), *result.UlLoad)
	}

	if result.DlLoad == nil {
		t.Error("expected DlLoad to be emitted")
	} else if *result.DlLoad != float64(dlLoad) {
		t.Errorf("expected DlLoad=%f, got %f", float64(dlLoad), *result.DlLoad)
	}

	// But telemetry fields should be nil
	if result.Code != nil {
		t.Error("Code should be nil when no telemetry")
	}
}

// TestBuildBaseStationStatus_InfoNotGated verifies that Info/BSConfig
// is emitted regardless of LastStatusAt.
func TestBuildBaseStationStatus_InfoNotGated(t *testing.T) {
	configJSON := `{"firmware": "v2.1.0", "region": "EU868"}`

	bs := &models.BaseStation{
		EUI:          models.EUI{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Name:         "Config Test BS",
		LastStatusAt: nil, // No telemetry
		BSConfig: models.NullJSON{
			Valid: true,
			Data:  json.RawMessage(configJSON),
		},
	}

	result := BuildBaseStationStatus(bs)

	// Info should be emitted regardless of telemetry
	if result.Info == nil {
		t.Error("expected Info to be emitted")
	}

	// Verify JSON structure
	infoMap, ok := result.Info.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Info to be map, got %T", result.Info)
	}
	if infoMap["firmware"] != "v2.1.0" {
		t.Errorf("expected firmware=v2.1.0, got %v", infoMap["firmware"])
	}
}

// TestBuildBaseStationStatus_PartialGeoLocation verifies that GeoLocation
// requires all three coordinates to be set.
func TestBuildBaseStationStatus_PartialGeoLocation(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name      string
		lat       *float64
		lon       *float64
		alt       *float64
		expectNil bool
	}{
		{"all coordinates", ptr(48.1351), ptr(11.5820), ptr(500.0), false},
		{"missing altitude", ptr(48.1351), ptr(11.5820), nil, true},
		{"missing longitude", ptr(48.1351), nil, ptr(500.0), true},
		{"missing latitude", nil, ptr(11.5820), ptr(500.0), true},
		{"all missing", nil, nil, nil, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := &models.BaseStation{
				EUI:          models.EUI{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				LastStatusAt: &now,
				Latitude:     tc.lat,
				Longitude:    tc.lon,
				Altitude:     tc.alt,
			}

			result := BuildBaseStationStatus(bs)

			if tc.expectNil && result.GeoLocation != nil {
				t.Error("expected GeoLocation to be nil with partial coordinates")
			}
			if !tc.expectNil && result.GeoLocation == nil {
				t.Error("expected GeoLocation to be set with all coordinates")
			}
		})
	}
}

// TestBuildBaseStationStatus_UptimeFallback verifies uptime field fallback logic.
func TestBuildBaseStationStatus_UptimeFallback(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name           string
		uptimeSeconds  *int64
		uptimeLegacy   int64
		expectedUptime int64
	}{
		{"prefer uptime_seconds", ptr(int64(7200)), 3600, 7200},
		{"fallback to uptime", nil, 3600, 3600},
		{"zero legacy ignored", nil, 0, 0}, // No uptime emitted
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := &models.BaseStation{
				EUI:           models.EUI{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				LastStatusAt:  &now,
				UptimeSeconds: tc.uptimeSeconds,
				Uptime:        tc.uptimeLegacy,
			}

			result := BuildBaseStationStatus(bs)

			if tc.expectedUptime == 0 && tc.uptimeSeconds == nil {
				if result.Uptime != nil {
					t.Errorf("expected nil Uptime for zero legacy, got %d", *result.Uptime)
				}
			} else if result.Uptime == nil {
				t.Error("expected Uptime to be set")
			} else if *result.Uptime != tc.expectedUptime {
				t.Errorf("expected Uptime=%d, got %d", tc.expectedUptime, *result.Uptime)
			}
		})
	}
}

// ptr is a helper to create pointers to values
func ptr[T any](v T) *T {
	return &v
}
