package bssci

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// statusLocationUpdates are the base station columns a statusRsp geoLocation
// writes: a fix sets the GPS position; a report without a fix marks a station
// that has no location yet as GPS without coordinates, so the UI can say it
// reports no fix, and never replaces a manual location or an earlier fix.
func statusLocationUpdates(report geoReport, fix geoFix, station *models.BaseStation, now time.Time) map[string]interface{} {
	switch {
	case report == geoFixed:
		return map[string]interface{}{
			locationColumnLatitude:  fix.latitude,
			locationColumnLongitude: fix.longitude,
			locationColumnAltitude:  fix.altitude,
			locationColumnSource:    models.LocationSourceGPS,
			locationColumnUpdatedAt: now,
		}
	case report == geoNoFix && station.LocationSource == nil:
		return map[string]interface{}{
			locationColumnSource:    models.LocationSourceGPS,
			locationColumnUpdatedAt: now,
		}
	default:
		return nil
	}
}
