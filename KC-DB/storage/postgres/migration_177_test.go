package postgres

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	backfillStationEUI  = "70b3d59cd00177a1"
	backfillStationText = "70B3D59CD00177A1"
	backfillEndpoint    = "70B3D5677011FF01"
	// backfillStationNumber is backfillStationEUI as the unsigned integer a writer stored.
	backfillStationNumber = "8121069422560507809"
	// backfillHighNumber is an EUI at or above 2^63, 0xF0B3D59CD00177A1.
	backfillHighNumber = "17344441459415283617"
	backfillHighText   = "F0B3D59CD00177A1"
)

// sqlStationTimelineOf selects a tenant's event types; the device scope appends to it.
const sqlStationTimelineOf = `SELECT event_type FROM system_events WHERE tenant_id = $1`

// seedEventDetails stores an event of the station's tenant with the given
// details and returns its id.
func (h *migrationHarness) seedEventDetails(tenantID int64, eventType, details string) string {
	h.t.Helper()
	var id string
	require.NoError(h.t, h.db.QueryRow(`INSERT INTO system_events (tenant_id, event_type, event_category, severity,
		source_type, source_name, title, description, data, status, occurred_at, recorded_at)
		VALUES ($1, $2, 'basestation', 'info', 'service_center', 'certificates', 'event', 'event', $3::jsonb, 'new', NOW(), NOW())
		RETURNING id::text`, tenantID, eventType, details).Scan(&id))
	return id
}

func (h *migrationHarness) eventDetails(id string) map[string]interface{} {
	h.t.Helper()
	var raw []byte
	require.NoError(h.t, h.db.QueryRow(`SELECT data FROM system_events WHERE id = $1::uuid`, id).Scan(&raw))
	var details map[string]interface{}
	require.NoError(h.t, json.Unmarshal(raw, &details))
	return details
}

// stationTimeline lists the event types the station's timeline scope matches,
// through the listing's own device predicate; the listing's projection reads
// columns later migrations add, so it cannot run on this schema.
func (h *migrationHarness) stationTimeline(tenantID int64) []string {
	h.t.Helper()
	query, args, _ := appendDeviceScope(sqlStationTimelineOf, []interface{}{tenantID}, 2,
		models.SystemEventFilter{BaseStationEUI: backfillStationText})
	return h.queryStrings(query, args...)
}

// TestMigration177RewritesEventDeviceEUIsToTheCanonicalForm: dashed,
// lowercase and numeric device EUIs become 16 uppercase hex digits, already
// canonical values and values that are not EUIs stay as written, the down
// migration changes nothing, and a second up changes nothing either. A
// certificate event stored with a dashed EUI then shows on its station's timeline.
func TestMigration177RewritesEventDeviceEUIsToTheCanonicalForm(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(174)
	tenantID, _ := h.seedTenantAndBaseStation(backfillStationEUI)

	dashed := h.seedEventDetails(tenantID, models.EventTypeCertificateGenerated, `{"bsEui":"70-B3-D5-9C-D0-01-77-A1","validityDays":365}`)
	lowercase := h.seedEventDetails(tenantID, models.EventTypeBSRegistered, `{"bsEui":"70b3d59cd00177a1"}`)
	numeric := h.seedEventDetails(tenantID, "detach_propagate_completed",
		`{"bsEui":`+backfillStationNumber+`,"epEui":"70:b3:d5:67:70:11:ff:01","opId":-7}`)
	high := h.seedEventDetails(tenantID, "dl_data_sent", `{"bsEui":`+backfillHighNumber+`}`)
	canonical := h.seedEventDetails(tenantID, "basestation.updated", `{"bsEui":"70B3D59CD00177A1","epEui":"70B3D5677011FF01"}`)
	notEUIs := h.seedEventDetails(tenantID, "session.resume_refused",
		`{"bsEui":"unknown","epEui":true,"reason":"70b3d59cd00177a1"}`)
	negative := h.seedEventDetails(tenantID, "dl_data_expired", `{"bsEui":-1,"epEui":1.5}`)

	assert.NotContains(t, h.stationTimeline(tenantID), models.EventTypeCertificateGenerated,
		"before the backfill the dashed certificate event is off the station's timeline")

	h.migrateTo(177)
	want := map[string]map[string]interface{}{
		dashed:    {"bsEui": backfillStationText, "validityDays": float64(365)},
		lowercase: {"bsEui": backfillStationText},
		numeric:   {"bsEui": backfillStationText, "epEui": backfillEndpoint, "opId": float64(-7)},
		high:      {"bsEui": backfillHighText},
		canonical: {"bsEui": backfillStationText, "epEui": backfillEndpoint},
		notEUIs:   {"bsEui": "unknown", "epEui": true, "reason": backfillStationEUI},
		negative:  {"bsEui": float64(-1), "epEui": 1.5},
	}
	for id, details := range want {
		assert.Equal(t, details, h.eventDetails(id), "event %s after the backfill", id)
	}
	assert.Contains(t, h.stationTimeline(tenantID), models.EventTypeCertificateGenerated,
		"the backfilled certificate event is on the station's timeline")
	validateEventDeviceEUIsCanonical(t, h.db)

	h.migrateTo(174)
	for id, details := range want {
		assert.Equal(t, details, h.eventDetails(id), "the down migration leaves event %s as it is", id)
	}
	h.migrateTo(177)
	for id, details := range want {
		assert.Equal(t, details, h.eventDetails(id), "a second up leaves event %s as it is", id)
	}
}
