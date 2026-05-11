// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"encoding/json"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// BuildBaseStationStatus transforms a models.BaseStation into scaci.BaseStationStatus per SCACI §3.5.2
//
// This helper maps canonical MIOTY base station fields to the SCACI protocol structure,
// reusing existing data without fabrication. Designed for reuse across SCACI handlers
// and gRPC endpoints.
//
// Telemetry Gating (§3.5.2 optional fields):
// Fields Code/Message/Uptime/GeoLocation are only emitted when real telemetry exists
// (bs.LastStatusAt != nil), indicating the base station has reported actual status.
// When LastStatusAt is nil, these fields remain nil to signal "status unknown" rather
// than fabricating a healthy status with Code=0.
//
// Field Mapping (SCACI §3.5.2):
//   - Eui: Base station EUI64 from database (always emitted)
//   - Bidi: Hardware bidirectional capability flag (always emitted if set)
//   - Code/Message: POSIX status code and message (gated to LastStatusAt)
//   - GeoLocation: [Lat, Lon, Alt] array (gated to LastStatusAt AND coordinates present)
//   - Uptime: Base station uptime in seconds (gated to LastStatusAt)
//   - Vendor/Model/Name: Hardware identification fields (always emitted if set)
//   - Info: Raw configuration JSON from bs_config (always emitted if valid)
//   - UlLoad/DlLoad: Channel utilization (always emitted if set)
//
// Parameters:
//   - bs: Base station record from database (must not be nil)
//
// Returns:
//   - BaseStationStatus: SCACI protocol message ready for StatusResponse.BaseStations array
func BuildBaseStationStatus(bs *models.BaseStation) BaseStationStatus {
	status := BaseStationStatus{}

	// ========================================================================
	// Identity Fields (always emitted regardless of telemetry)
	// ========================================================================

	// EUI64 (mandatory for identification)
	eui := bs.EUI.ToUint64()
	status.Eui = &eui

	// Bidirectional capability
	if bs.Bidi != nil {
		status.Bidi = bs.Bidi
	}

	// Hardware identification fields
	if bs.Vendor != nil {
		status.Vendor = bs.Vendor
	}
	if bs.Model != nil {
		status.Model = bs.Model
	}
	if bs.Name != "" {
		status.Name = &bs.Name
	}

	// ========================================================================
	// Telemetry Fields (gated to LastStatusAt per §3.5.2 optional semantics)
	// Only emit when real status telemetry exists; nil signals "unknown"
	// ========================================================================

	if bs.LastStatusAt != nil {
		// Status code and message from last status report
		// Only emit when real telemetry exists; Code=0 means "explicitly healthy"
		status.Code = &bs.StatusCode
		if bs.StatusMessage != nil && *bs.StatusMessage != "" {
			status.Message = bs.StatusMessage
		}

		// Uptime in seconds (prefer uptime_seconds, fallback to uptime)
		if bs.UptimeSeconds != nil {
			status.Uptime = bs.UptimeSeconds
		} else if bs.Uptime != 0 {
			status.Uptime = &bs.Uptime
		}

		// Geographic location (only when all three coordinates are present)
		if bs.Latitude != nil && bs.Longitude != nil && bs.Altitude != nil {
			location := [3]float64{*bs.Latitude, *bs.Longitude, *bs.Altitude}
			status.GeoLocation = &location
		}
	}
	// If LastStatusAt == nil, Code/Message/Uptime/GeoLocation remain nil (unknown)

	// ========================================================================
	// Configuration and Load Metrics (not gated - always emitted if present)
	// ========================================================================

	// Configuration JSON (reuse existing JSON from database)
	if bs.BSConfig.Valid && len(bs.BSConfig.Data) > 0 {
		// Validate JSON structure before including
		var testJSON interface{}
		if err := json.Unmarshal(bs.BSConfig.Data, &testJSON); err == nil {
			status.Info = testJSON
		}
		// Invalid JSON silently omitted (Info remains nil)
	}

	// Channel load metrics (cast float32 → float64 for protocol compliance)
	if bs.ULLoad != nil {
		ulLoad := float64(*bs.ULLoad)
		status.UlLoad = &ulLoad
	}
	if bs.DLLoad != nil {
		dlLoad := float64(*bs.DLLoad)
		status.DlLoad = &dlLoad
	}

	return status
}
