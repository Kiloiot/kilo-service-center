package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// setStationLocation stores a location on a seeded base station.
func (h *migrationHarness) setStationLocation(id int64, lat, lon, alt float64, source string) {
	h.exec(`UPDATE basestations SET latitude = $2, longitude = $3, altitude = $4, location_source = $5 WHERE id = $1`,
		id, lat, lon, alt, source)
}

// locatedAt reports whether the station still holds coordinates and its location source.
func (h *migrationHarness) stationLocation(id int64) (hasCoordinates bool, source string) {
	h.t.Helper()
	var coordinates int64
	if err := h.db.QueryRow(`SELECT count(*) FROM basestations WHERE id = $1 AND latitude IS NOT NULL AND longitude IS NOT NULL AND altitude IS NOT NULL`, id).Scan(&coordinates); err != nil {
		h.t.Fatal(err)
	}
	if err := h.db.QueryRow(`SELECT COALESCE(location_source, '') FROM basestations WHERE id = $1`, id).Scan(&source); err != nil {
		h.t.Fatal(err)
	}
	return coordinates == 1, source
}

// TestMigration174ClearsGPSReportsWithoutAFix: a station that reported
// geoLocation [0,0,0] (no position fix) before the service center stopped
// storing such reports keeps its GPS source without coordinates; a real
// fix and a manual location at 0°/0° are left alone.
func TestMigration174ClearsGPSReportsWithoutAFix(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(172)
	_, noFix := h.seedTenantAndBaseStation("70b3d59cd00174a1")
	_, fixed := h.seedTenantAndBaseStation("70b3d59cd00174a2")
	_, manual := h.seedTenantAndBaseStation("70b3d59cd00174a3")
	h.setStationLocation(noFix, 0, 0, 0, "gps")
	h.setStationLocation(fixed, 48.1351, 11.582, 520, "gps")
	h.setStationLocation(manual, 0, 0, 0, "manual")

	h.migrateTo(174)

	hasCoordinates, source := h.stationLocation(noFix)
	assert.False(t, hasCoordinates, "a report without a fix is not a position")
	assert.Equal(t, "gps", source, "the station still reports through GPS")
	hasCoordinates, source = h.stationLocation(fixed)
	assert.True(t, hasCoordinates, "a real fix stays")
	assert.Equal(t, "gps", source)
	hasCoordinates, source = h.stationLocation(manual)
	assert.True(t, hasCoordinates, "a manual location is never cleared")
	assert.Equal(t, "manual", source)

	h.migrateTo(172)
	hasCoordinates, _ = h.stationLocation(noFix)
	assert.False(t, hasCoordinates, "the downgrade does not restore an invalid position")
}
