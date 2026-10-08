package postgres

import (
	"context"
	"encoding/json"
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
	errorGroupTenant      = int64(610)
	errorGroupOtherTenant = int64(611)
	errorGroupBSEUI       = "70B3D59CD0000001"
	errorGroupEpEUI       = "70B3D59CD0000002"
	errorGroupOfflineType = "basestation_offline"
	errorGroupExpiredType = "dl_data_expired"
	errorGroupOpID        = "-41"
	errorGroupPOSIXCode   = 22
	errorGroupToken       = "KC-SCACI-ERR-GROUP"
	errorGroupPageLimit   = 10
	errorGroupPageOne     = 1
)

type failureRow struct {
	tenantID   int64
	eventType  string
	category   string
	severity   string
	sourceName string
	message    string
	occurredAt time.Time
	opID       string
}

func insertFailure(t *testing.T, db *sqlx.DB, row failureRow) {
	t.Helper()
	data := map[string]interface{}{}
	if row.opID != "" {
		data["opId"] = row.opID
	}
	dataJSON, err := json.Marshal(data)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO system_events (
			tenant_id, event_type, event_category, severity,
			source_type, source_name, title, description,
			data, status, occurred_at, recorded_at
		) VALUES ($1, $2, $3, $4, 'service_center', $5, $2, $6, $7, 'new', $8, NOW())`,
		row.tenantID, row.eventType, row.category, row.severity, row.sourceName, row.message, dataJSON, row.occurredAt)
	require.NoError(t, err)
}

func TestListErrorGroups_BucketsGroupAndIsolate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSystemEventsTestDB(t)
	defer func() { _ = db.Close() }()
	createSystemEventsTestTenant(t, db, errorGroupTenant, "ErrorGroupTenant")
	createSystemEventsTestTenant(t, db, errorGroupOtherTenant, "ErrorGroupOtherTenant")
	now := time.Now().Truncate(time.Second)

	insertFailure(t, db, failureRow{errorGroupTenant, errorGroupOfflineType, models.EventCategoryBaseStation, models.EventSeverityError, errorGroupBSEUI, "first", now.Add(-2 * time.Hour), ""})
	insertFailure(t, db, failureRow{errorGroupTenant, errorGroupOfflineType, models.EventCategoryBaseStation, models.EventSeverityError, errorGroupBSEUI, "latest", now.Add(-time.Hour), errorGroupOpID})
	insertFailure(t, db, failureRow{errorGroupTenant, errorGroupOfflineType, models.EventCategoryBaseStation, models.EventSeverityInfo, errorGroupBSEUI, "informational", now, ""})
	insertFailure(t, db, failureRow{errorGroupTenant, errorGroupExpiredType, models.EventCategoryMessage, models.EventSeverityWarning, errorGroupEpEUI, "expired", now, ""})
	insertFailure(t, db, failureRow{errorGroupTenant, "uplink_received", models.EventCategoryMessage, models.EventSeverityWarning, errorGroupEpEUI, "noise", now, ""})
	insertFailure(t, db, failureRow{errorGroupOtherTenant, errorGroupOfflineType, models.EventCategoryBaseStation, models.EventSeverityError, errorGroupBSEUI, "foreign", now, ""})

	sessionID := createSCACIOperationTestSession(t, db, errorGroupTenant)
	defer cleanupSCACIOperationTestSession(t, db, sessionID)
	ops := NewSCACIOperationRepository(db, logger.Get().WithField("component", "error_groups_test"), clock.SystemClock{})
	ctx, cancel := context.WithTimeout(testutil.TestContext(), testContextTimeout)
	defer cancel()
	recordMonitoringOp(ctx, t, ops, sessionID, errorGroupTenant, 7, monitoringDLQueue)
	require.NoError(t, ops.UpdateOperationStateWithError(ctx, sessionID, 7, models.OperationStateFailed, errorGroupPOSIXCode, errorGroupToken, "queue refused", nil))

	store := NewSystemEventStore(db.DB, clock.SystemClock{}, logger.Get())
	from := now.Add(-24 * time.Hour)
	to := now.Add(time.Hour)

	baseStation, total, err := store.ListErrorGroups(ctx, models.ErrorGroupFilter{
		TenantID: errorGroupTenant, Categories: []string{models.EventCategoryBaseStation},
		Severities: []string{models.EventSeverityError, models.EventSeverityCritical}, From: from, To: to, Limit: errorGroupPageLimit,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, baseStation, 1)
	assert.Equal(t, errorGroupOfflineType, baseStation[0].EventType)
	assert.Equal(t, int64(2), baseStation[0].Count, "the info row stays out of the failure bucket")
	assert.Equal(t, "latest", baseStation[0].Message, "the bucket carries the latest message")
	assert.Equal(t, errorGroupBSEUI, baseStation[0].SourceName)
	assert.Equal(t, errorGroupOpID, baseStation[0].LastOpID)
	assert.True(t, baseStation[0].FirstSeen.Before(baseStation[0].LastSeen))

	downlink, total, err := store.ListErrorGroups(ctx, models.ErrorGroupFilter{
		TenantID: errorGroupTenant, Categories: []string{models.EventCategoryMessage},
		Severities: []string{models.EventSeverityWarning, models.EventSeverityError}, EventTypePrefixes: []string{"dl_data_"}, From: from, To: to, Limit: errorGroupPageLimit,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, downlink, 1)
	assert.Equal(t, errorGroupExpiredType, downlink[0].EventType, "the prefix filter keeps other message warnings out")

	controlPlane, total, err := store.ListErrorGroups(ctx, models.ErrorGroupFilter{
		TenantID: errorGroupTenant, Categories: []string{models.EventCategorySCACI},
		Severities: []string{models.EventSeverityError}, IncludeSCACIFailures: true, From: from, To: to, Limit: errorGroupPageLimit,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, controlPlane, 1)
	assert.Equal(t, monitoringDLQueue, controlPlane[0].EventType)
	assert.Equal(t, "22", controlPlane[0].Code)
	assert.Equal(t, "queue refused", controlPlane[0].Message)
	assert.Equal(t, "7", controlPlane[0].LastOpID)

	page, pageTotal, err := store.ListErrorGroups(ctx, models.ErrorGroupFilter{
		TenantID: errorGroupTenant, Categories: []string{models.EventCategoryBaseStation, models.EventCategoryMessage},
		Severities: []string{models.EventSeverityWarning, models.EventSeverityError}, From: from, To: to, Limit: errorGroupPageOne, Offset: errorGroupPageOne,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), pageTotal)
	assert.Len(t, page, 1)

	foreign, total, err := store.ListErrorGroups(ctx, models.ErrorGroupFilter{
		TenantID: errorGroupOtherTenant, Categories: []string{models.EventCategoryBaseStation},
		Severities: []string{models.EventSeverityError}, IncludeSCACIFailures: true, From: from, To: to, Limit: errorGroupPageLimit,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, foreign, 1)
	assert.Equal(t, "foreign", foreign[0].Message, "another tenant sees only its own failures")
}
