package postgres

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	alertTenant        int64 = 710
	alertForeignTenant int64 = 711
)

type alertFixture struct {
	t     *testing.T
	db    *sqlx.DB
	store *SystemEventStore
	ctx   context.Context
}

func newAlertFixture(t *testing.T) *alertFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSystemEventsTestDB(t)
	createSystemEventsTestTenant(t, db, alertTenant, "Alert Tenant")
	createSystemEventsTestTenant(t, db, alertForeignTenant, "Alert Foreign Tenant")
	ctx, cancel := context.WithTimeout(testutil.TestContext(), testContextTimeout)
	t.Cleanup(cancel)
	return &alertFixture{t: t, db: db, store: NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get()), ctx: ctx}
}

// alert stores a warning through the production insert and moves it to status.
func (f *alertFixture) alert(tenantID int64, title, status string) {
	f.t.Helper()
	event := &models.SystemEvent{
		TenantID:   strconv.FormatInt(tenantID, 10),
		EventType:  models.EventTypeConnectionError,
		Category:   models.EventCategoryBaseStation,
		Severity:   models.EventSeverityWarning,
		SourceType: models.SourceTypeBaseStation,
		SourceName: title,
		Title:      title,
	}
	require.NoError(f.t, f.store.CreateEvent(f.ctx, event))
	if status == models.EventStatusNew {
		return
	}
	_, err := f.db.Exec(`UPDATE system_events SET status = $1 WHERE id = $2::uuid`, status, event.ID)
	require.NoError(f.t, err)
}

func (f *alertFixture) titles(filter models.AlertFilter) []string {
	f.t.Helper()
	events, err := f.store.GetActiveAlerts(f.ctx, filter)
	require.NoError(f.t, err)
	count, err := f.store.CountActiveAlerts(f.ctx, filter)
	require.NoError(f.t, err)
	titles := make([]string, len(events))
	for i, e := range events {
		titles[i] = e.Title
	}
	assert.Equal(f.t, int64(len(events)), count, "the count pages the same alerts")
	return titles
}

func TestCreateEvent_StoresNewWhenNoStatusIsGiven(t *testing.T) {
	f := newAlertFixture(t)
	f.alert(alertTenant, "fresh", models.EventStatusNew)

	var status string
	require.NoError(t, f.db.Get(&status, `SELECT status FROM system_events WHERE title = 'fresh'`))

	assert.Equal(t, models.EventStatusNew, status)
}

func TestGetActiveAlerts_HonoursTheStatusFilter(t *testing.T) {
	f := newAlertFixture(t)
	f.alert(alertTenant, "fresh", models.EventStatusNew)
	f.alert(alertTenant, "seen", models.EventStatusAcknowledged)
	f.alert(alertTenant, "closed", models.EventStatusResolved)
	f.alert(alertForeignTenant, "foreign", models.EventStatusNew)
	tenant := strconv.FormatInt(alertTenant, 10)

	assert.ElementsMatch(t, []string{"fresh", "seen"}, f.titles(models.AlertFilter{TenantID: tenant}))
	assert.Equal(t, []string{"fresh"}, f.titles(models.AlertFilter{TenantID: tenant, Statuses: []string{models.EventStatusNew}}))
	assert.Equal(t, []string{"seen"}, f.titles(models.AlertFilter{TenantID: tenant, Statuses: []string{models.EventStatusAcknowledged}}))
	assert.Equal(t, []string{"closed"}, f.titles(models.AlertFilter{TenantID: tenant, Statuses: []string{models.EventStatusResolved}}))
}

func TestGetActiveAlerts_ForeignTenantSeesNoneOfTheStatuses(t *testing.T) {
	f := newAlertFixture(t)
	f.alert(alertTenant, "closed", models.EventStatusResolved)

	assert.Empty(t, f.titles(models.AlertFilter{
		TenantID: strconv.FormatInt(alertForeignTenant, 10),
		Statuses: []string{models.EventStatusResolved},
		Since:    ptrTime(time.Now().Add(-time.Hour)),
	}))
}

const (
	alertPageSize   = 1
	alertPageOffset = 1
)

func TestGetActiveAlerts_FiltersSeverityAndCategoryAndPages(t *testing.T) {
	f := newAlertFixture(t)
	tenant := strconv.FormatInt(alertTenant, 10)
	f.alert(alertTenant, "older warning", models.EventStatusNew)
	f.alert(alertTenant, "newer warning", models.EventStatusNew)
	require.NoError(t, f.store.CreateEvent(f.ctx, &models.SystemEvent{
		TenantID: tenant, EventType: models.EventTypeSCACISessionOpened, Category: models.EventCategorySCACI,
		Severity: models.EventSeverityInfo, SourceType: models.SourceTypeApplicationCenter, Title: "info",
	}))

	assert.ElementsMatch(t, []string{"older warning", "newer warning"}, f.titles(models.AlertFilter{TenantID: tenant}),
		"informational events are not alerts unless asked for")
	assert.Equal(t, []string{"info"}, f.titles(models.AlertFilter{TenantID: tenant, Severities: []string{models.EventSeverityInfo}}))
	assert.Equal(t, []string{"info"}, f.titles(models.AlertFilter{TenantID: tenant, Categories: []string{models.EventCategorySCACI},
		Severities: []string{models.EventSeverityInfo, models.EventSeverityWarning}}))

	page, err := f.store.GetActiveAlerts(f.ctx, models.AlertFilter{TenantID: tenant, Limit: alertPageSize, Offset: alertPageOffset})
	require.NoError(t, err)
	require.Len(t, page, alertPageSize)
	assert.Equal(t, "older warning", page[0].Title, "newest first, the offset skips the newer one")
}

// summaryEvent stores an event of severity through the production insert,
// occurred age ago, and moves it to status.
func (f *alertFixture) summaryEvent(severity, status string, age time.Duration) {
	f.t.Helper()
	event := &models.SystemEvent{
		TenantID:   strconv.FormatInt(alertTenant, 10),
		EventType:  models.EventTypeConnectionError,
		Category:   models.EventCategoryBaseStation,
		Severity:   severity,
		SourceType: models.SourceTypeBaseStation,
		Title:      severity + " " + status,
		CreatedAt:  time.Now().Add(-age),
	}
	require.NoError(f.t, f.store.CreateEvent(f.ctx, event))
	_, err := f.db.Exec(`UPDATE system_events SET status = $1 WHERE id = $2::uuid`, status, event.ID)
	require.NoError(f.t, err)
}

const alertSummaryOldAge = 48 * time.Hour

var alertSummarySeverities = []string{models.EventSeverityWarning, models.EventSeverityError, models.EventSeverityCritical}

func TestAlertSummary_CountsEachSeverityAsItsListFilterDoes(t *testing.T) {
	f := newAlertFixture(t)
	f.summaryEvent(models.EventSeverityError, models.EventStatusNew, alertSummaryOldAge)
	f.summaryEvent(models.EventSeverityWarning, models.EventStatusResolved, time.Minute)
	f.summaryEvent(models.EventSeverityWarning, models.EventStatusAcknowledged, time.Minute)
	f.summaryEvent(models.EventSeverityInfo, models.EventStatusNew, time.Minute)
	f.summaryEvent(models.EventSeverityCritical, models.EventStatusNew, time.Minute)
	f.alert(alertForeignTenant, "foreign", models.EventStatusNew)
	tenant := strconv.FormatInt(alertTenant, 10)

	counts, err := f.store.CountAlertsBySeverity(f.ctx, models.AlertFilter{TenantID: tenant, Severities: alertSummarySeverities})
	require.NoError(t, err)

	for _, severity := range alertSummarySeverities {
		listed := f.titles(models.AlertFilter{TenantID: tenant, Severities: []string{severity}})
		assert.Equal(t, int64(len(listed)), counts[severity], "the %s count equals its filter's rows", severity)
	}
	assert.Equal(t, map[string]int64{
		models.EventSeverityWarning:  1,
		models.EventSeverityError:    1,
		models.EventSeverityCritical: 1,
	}, counts, "info and resolved events are no alerts; an old open error still is")
}
