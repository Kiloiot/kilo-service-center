package bssci_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/internal/testsupport/bsscitest"
	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// noFixLocations are geoLocation reports of a base station without a
// position fix: (0°, 0°) with any altitude.
var noFixLocations = map[string][]interface{}{
	"all zero":            {float64(0), float64(0), float64(0)},
	"zero with altitude":  {float64(0), float64(0), float64(120)},
	"integer zero triple": {int64(0), int64(0), int64(0)},
}

// coordinateFields are the base station columns a position fix writes.
var coordinateFields = []string{"latitude", "longitude", "altitude"}

// locationFields are the base station columns a position report writes.
var locationFields = append([]string{"location_source", "location_updated_at"}, coordinateFields...)

// statusWithoutFix sends a statusRsp reporting geoLocation to a station the
// repository stores with the given location source.
func statusWithoutFix(t *testing.T, geoLocation []interface{}, stored *string) (*geoLocationTrackingRepo, *mockStatusHistoryRepo) {
	t.Helper()
	log := logger.NewNop()
	trackingRepo := &geoLocationTrackingRepo{locationSource: stored}
	historyRepo := &mockStatusHistoryRepo{}
	storage := &mockStorageWithHistory{historyRepo: historyRepo, baseStationRepo: trackingRepo}
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bsscitest.CreateTestServices(log, nil)
	server := bssci.NewTestServerWithBaseStationRepo(log, storage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{ID: "status-no-fix", BaseStationEUI: bssci.TestBsEui04, Encoding: "msgpack"},
		Conn:                 &statusMockConn{},
	}
	data := map[string]interface{}{
		"code":        0,
		"message":     "ok",
		"time":        int64(1672531200000000000),
		"geoLocation": geoLocation,
	}
	require.NoError(t, server.CallHandleStatusResponse(session, &bssci.Message{Command: mioty.CmdStatusResponse, OpId: -1, Data: data}, data))
	require.True(t, trackingRepo.updateCalled)
	return trackingRepo, historyRepo
}

// A statusRsp without a position fix writes no coordinates and records no
// position in the status history; a station with no location yet is marked
// as reporting GPS without a fix, so the UI says so.
func TestStatusResponseWithoutPositionFixRecordsNoLocation(t *testing.T) {
	for name, geoLocation := range noFixLocations {
		t.Run(name, func(t *testing.T) {
			trackingRepo, historyRepo := statusWithoutFix(t, geoLocation, nil)

			for _, field := range coordinateFields {
				assert.NotContains(t, trackingRepo.updatesMap, field, "a report without a fix must not write %s", field)
			}
			assert.Equal(t, models.LocationSourceGPS, trackingRepo.updatesMap["location_source"])
			assert.Contains(t, trackingRepo.updatesMap, "location_updated_at")
			require.NotNil(t, historyRepo.lastRecord)
			assert.Nil(t, historyRepo.lastRecord.Latitude, "the history records no latitude without a fix")
			assert.Nil(t, historyRepo.lastRecord.Longitude, "the history records no longitude without a fix")
			assert.Nil(t, historyRepo.lastRecord.Altitude, "the history records no altitude without a fix")
		})
	}
}

// A report without a fix never replaces a location the station already has,
// neither a manual one nor an earlier GPS fix.
func TestStatusResponseWithoutPositionFixKeepsAKnownLocation(t *testing.T) {
	for _, stored := range []string{models.LocationSourceManual, models.LocationSourceGPS} {
		t.Run(stored, func(t *testing.T) {
			trackingRepo, _ := statusWithoutFix(t, noFixLocations["all zero"], &stored)

			for _, field := range locationFields {
				assert.NotContains(t, trackingRepo.updatesMap, field, "a known %s location keeps %s", stored, field)
			}
		})
	}
}

// A con without a position fix leaves the stored location alone when the
// connect completes.
func TestConnectWithoutPositionFixRecordsNoLocation(t *testing.T) {
	for name, geoLocation := range noFixLocations {
		t.Run(name, func(t *testing.T) {
			log := &NopLogger{}
			trackingRepo := &geoLocationTrackingRepo{}
			sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssci.CreateTestServices(log, nil)
			server := bssci.NewTestServerWithBaseStationRepo(log, mockStorage, trackingRepo, 1,
				sessionSvc, downlinkSvc, statusSvc, &mockConnectionService{tenantID: 1}, broadcaster, queueSerializer, auditLogger, tenantResolver)
			server.SetConfig(&bssci.Config{ServiceCenterEUI: bssci.TestScEui01, SoftwareVersion: "1.0.0"})

			session := &bssci.Session{
				ProtocolSessionState: bssci.ProtocolSessionState{ID: "connect-no-fix", BaseStationEUI: bssci.TestBsEui01, SessionUUID: make([]byte, 16)},
				Conn:                 &mockConnForConnect{},
			}
			connectData := map[string]interface{}{
				"version":     mioty.MIOTYProtocolVersion,
				"bsEui":       int64(0x0123456789ABCDEF),
				"bidi":        true,
				"geoLocation": geoLocation,
			}
			require.NoError(t, server.CallHandleConnect(session, &bssci.Message{OpId: 0, Command: mioty.CmdConnect, Data: connectData}, connectData))
			assert.Nil(t, session.GeoLocation, "a report without a fix is not a location")

			session.IsResumed = true
			require.NoError(t, server.CallHandleConnectComplete(session, &bssci.Message{OpId: 0, Command: mioty.CmdConnectComplete}, map[string]interface{}{}))

			require.True(t, trackingRepo.updateCalled)
			for _, field := range locationFields {
				assert.NotContains(t, trackingRepo.updatesMap, field, "a report without a fix must not write %s", field)
			}
		})
	}
}
