package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const detailKeysStationEUI = "70b3d59cd00179a1"

// TestMigration179RenamesSnakeCaseEventDetailKeys: the snake_case detail keys
// of stored events become the camelCase keys the writers use, a row that
// already holds the new key keeps its value, the down migration restores what
// the previous release reads (certificate and SCACI events keep their keys),
// and a second up renames again.
func TestMigration179RenamesSnakeCaseEventDetailKeys(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(178)
	tenantID, _ := h.seedTenantAndBaseStation(detailKeysStationEUI)

	online := h.seedEventDetails(tenantID, models.EventTypeBaseStationOnline,
		`{"basestation_name":"tims base","bsEui":"70B3D59CD00179A1","connection_type":"bssci","is_online":true,"session_id":"s-1"}`)
	propagate := h.seedEventDetails(tenantID, "attach_propagate_initiated",
		`{"operation_id":"attach-7","operation_type":"attach","endpoint_id":7,"target_bs":"none","target_bs_list":[],"target_bs_count":0}`)
	both := h.seedEventDetails(tenantID, "dl_data_revoked", `{"basestation_name":"old","baseStationName":"kept","queId":3}`)
	certificate := h.seedEventDetails(tenantID, models.EventTypeCertificateGenerated, `{"baseStationName":"tims base","validityDays":365}`)
	scaciError := h.seedEventDetails(tenantID, "scaciErr", `{"sessionID":12,"count":2}`)

	camel := map[string]map[string]interface{}{
		online: {"baseStationName": "tims base", "bsEui": "70B3D59CD00179A1", "connectionType": "bssci", "isOnline": true, "sessionID": "s-1"},
		propagate: {"operationId": "attach-7", "operationType": "attach", "endpointId": float64(7), "targetBs": "none",
			"targetBsList": []interface{}{}, "targetBsCount": float64(0)},
		both:        {"baseStationName": "kept", "queId": float64(3)},
		certificate: {"baseStationName": "tims base", "validityDays": float64(365)},
		scaciError:  {"sessionID": float64(12), "count": float64(2)},
	}
	h.migrateTo(179)
	for id, details := range camel {
		assert.Equal(t, details, h.eventDetails(id), "event %s after the rename", id)
	}
	validateEventDetailKeysCamelCase(t, h.db)

	h.migrateTo(178)
	assert.Equal(t, map[string]interface{}{"basestation_name": "tims base", "bsEui": "70B3D59CD00179A1",
		"connection_type": "bssci", "is_online": true, "session_id": "s-1"}, h.eventDetails(online))
	assert.Equal(t, map[string]interface{}{"operation_id": "attach-7", "operation_type": "attach", "endpoint_id": float64(7),
		"target_bs": "none", "target_bs_list": []interface{}{}, "target_bs_count": float64(0)}, h.eventDetails(propagate))
	assert.Equal(t, camel[certificate], h.eventDetails(certificate), "the certificate event always named its station baseStationName")
	assert.Equal(t, camel[scaciError], h.eventDetails(scaciError), "SCACI events read sessionID")

	h.migrateTo(179)
	for id, details := range camel {
		assert.Equal(t, details, h.eventDetails(id), "a second up renames event %s again", id)
	}
}
