package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const certificateCategoryStationEUI = "70b3d59cd00181a1"

// seedCategorizedEvent stores an event of the given type and category and
// returns its id.
func (h *migrationHarness) seedCategorizedEvent(tenantID int64, eventType, category string) string {
	h.t.Helper()
	var id string
	require.NoError(h.t, h.db.QueryRow(`INSERT INTO system_events (tenant_id, event_type, event_category, severity,
		source_type, source_name, title, description, data, status, occurred_at, recorded_at)
		VALUES ($1, $2, $3, 'info', 'service_center', 'certificates', 'event', 'event', '{}'::jsonb, 'new', NOW(), NOW())
		RETURNING id::text`, tenantID, eventType, category).Scan(&id))
	return id
}

func (h *migrationHarness) eventCategory(id string) string {
	h.t.Helper()
	var category string
	require.NoError(h.t, h.db.QueryRow(`SELECT event_category FROM system_events WHERE id = $1::uuid`, id).Scan(&category))
	return category
}

// TestMigration181FilesStationCertificatesUnderBaseStation: the station
// certificate events stored under audit move to basestation, the server
// certificate events stay under audit, the down migration files every
// station certificate event under audit again, and a second up moves them
// back.
func TestMigration181FilesStationCertificatesUnderBaseStation(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(180)
	tenantID, _ := h.seedTenantAndBaseStation(certificateCategoryStationEUI)

	stored := h.seedCategorizedEvent(tenantID, models.EventTypeCertificateGenerated, models.EventCategoryAudit)
	current := h.seedCategorizedEvent(tenantID, models.EventTypeCertificateGenerated, models.EventCategoryBaseStation)
	server := h.seedCategorizedEvent(tenantID, models.EventTypeCertificateServerGenerated, models.EventCategoryAudit)
	renewed := h.seedCategorizedEvent(tenantID, models.EventTypeCertificateServerRenewed, models.EventCategoryAudit)

	h.migrateTo(181)
	assert.Equal(t, models.EventCategoryBaseStation, h.eventCategory(stored), "a stored station certificate event moves to basestation")
	assert.Equal(t, models.EventCategoryBaseStation, h.eventCategory(current))
	assert.Equal(t, models.EventCategoryAudit, h.eventCategory(server), "server certificate events stay under audit")
	assert.Equal(t, models.EventCategoryAudit, h.eventCategory(renewed), "server certificate events stay under audit")
	validateCertificateGeneratedBaseStationCategory(t, h.db)

	h.migrateTo(180)
	assert.Equal(t, models.EventCategoryAudit, h.eventCategory(stored), "the previous release reads station certificates under audit")
	assert.Equal(t, models.EventCategoryAudit, h.eventCategory(current), "the previous release reads station certificates under audit")
	assert.Equal(t, models.EventCategoryAudit, h.eventCategory(server))

	h.migrateTo(181)
	assert.Equal(t, models.EventCategoryBaseStation, h.eventCategory(stored), "a second up moves the event again")
	assert.Equal(t, models.EventCategoryBaseStation, h.eventCategory(current), "a second up moves the event again")
	assert.Equal(t, models.EventCategoryAudit, h.eventCategory(renewed))
}
