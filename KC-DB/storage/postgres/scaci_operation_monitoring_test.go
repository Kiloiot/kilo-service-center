package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	monitoringTenant   = int64(503)
	monitoringOtherTen = int64(504)
	monitoringPing     = "ping"
	monitoringConnect  = "con"
	monitoringDLQueue  = "dlDataQue"
	monitoringEINVAL   = 22
	monitoringToken    = "KC-SCACI-ERR-TEST"
	monitoringStale    = time.Minute
)

func recordMonitoringOp(ctx context.Context, t *testing.T, repo *SCACIOperationRepository, sessionID, tenantID, opID int64, command string) {
	t.Helper()
	_, err := repo.RecordOperation(ctx, &models.SCACIOperationRequest{
		SessionID: sessionID, TenantID: tenantID, OpId: opID, Command: command,
		Direction: string(models.OperationDirectionInbound), RequestData: map[string]interface{}{},
	})
	require.NoError(t, err)
}

func TestSCACIOperationMonitoring_ReadsAreTenantScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSCACIOperationTestDB(t)
	defer func() { _ = db.Close() }()
	createSCACIOperationTestTenant(t, db, monitoringTenant, "MonitoringTenant")
	defer cleanupSCACIOperationTestTenant(t, db, monitoringTenant)
	createSCACIOperationTestTenant(t, db, monitoringOtherTen, "MonitoringOtherTenant")
	defer cleanupSCACIOperationTestTenant(t, db, monitoringOtherTen)
	sessionID := createSCACIOperationTestSession(t, db, monitoringTenant)
	defer cleanupSCACIOperationTestSession(t, db, sessionID)
	otherSession := createSCACIOperationTestSession(t, db, monitoringOtherTen)
	defer cleanupSCACIOperationTestSession(t, db, otherSession)

	repo := NewSCACIOperationRepository(db, logger.Get().WithField("component", "scaci_operation_monitoring_test"), clock.SystemClock{})
	ctx, cancel := context.WithTimeout(testutil.TestContext(), testContextTimeout)
	defer cancel()

	recordMonitoringOp(ctx, t, repo, sessionID, monitoringTenant, 1, monitoringConnect)
	require.NoError(t, repo.UpdateOperationState(ctx, sessionID, 1, models.OperationStateCompleted, nil))
	recordMonitoringOp(ctx, t, repo, sessionID, monitoringTenant, 2, monitoringPing)
	require.NoError(t, repo.UpdateOperationState(ctx, sessionID, 2, models.OperationStateCompleted, nil))
	recordMonitoringOp(ctx, t, repo, sessionID, monitoringTenant, 3, monitoringPing)
	require.NoError(t, repo.UpdateOperationStateWithError(ctx, sessionID, 3, models.OperationStateFailed, monitoringEINVAL, monitoringToken, "ping failed", nil))
	recordMonitoringOp(ctx, t, repo, sessionID, monitoringTenant, 4, monitoringDLQueue)
	require.NoError(t, repo.UpdateOperationStateWithError(ctx, sessionID, 4, models.OperationStateFailed, monitoringEINVAL, monitoringToken, "first", nil))
	recordMonitoringOp(ctx, t, repo, sessionID, monitoringTenant, 5, monitoringDLQueue)
	require.NoError(t, repo.UpdateOperationStateWithError(ctx, sessionID, 5, models.OperationStateFailed, monitoringEINVAL, monitoringToken, "second", nil))
	recordMonitoringOp(ctx, t, repo, otherSession, monitoringOtherTen, 6, monitoringDLQueue)
	require.NoError(t, repo.UpdateOperationStateWithError(ctx, otherSession, 6, models.OperationStateFailed, monitoringEINVAL, monitoringToken, "foreign", nil))

	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)

	counts, err := repo.CountOperationsBySession(ctx, monitoringTenant, []int64{sessionID, otherSession})
	require.NoError(t, err)
	assert.Equal(t, int64(5), counts[sessionID])
	assert.NotContains(t, counts, otherSession, "another tenant's session must not be counted")

	groups, total, err := repo.ListFailedOperationGroups(ctx, monitoringTenant, from, to, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, groups, 2)
	byCommand := map[string]*models.SCACIOperationErrorGroup{}
	for _, g := range groups {
		byCommand[g.Command] = g
	}
	dl := byCommand[monitoringDLQueue]
	require.NotNil(t, dl)
	assert.Equal(t, int64(2), dl.Count)
	require.NotNil(t, dl.ErrorCode)
	assert.Equal(t, monitoringEINVAL, *dl.ErrorCode)
	require.NotNil(t, dl.ErrorMessage)
	assert.Equal(t, "second", *dl.ErrorMessage, "the bucket carries the latest message")
	assert.Equal(t, sessionID, dl.SessionID)
	assert.False(t, dl.LastSeen.Before(dl.FirstSeen))

	page, pageTotal, err := repo.ListFailedOperationGroups(ctx, monitoringTenant, from, to, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pageTotal)
	assert.Len(t, page, 1)

	summary, err := repo.GetTenantOperationSummaryBetween(ctx, monitoringTenant, from, to)
	require.NoError(t, err)
	assert.Equal(t, int64(5), summary.TotalOperations)
	assert.Equal(t, int64(3), summary.StateCounts[string(models.OperationStateFailed)])

	outside, err := repo.GetTenantOperationSummaryBetween(ctx, monitoringTenant, from.Add(-2*time.Hour), from)
	require.NoError(t, err)
	assert.Zero(t, outside.TotalOperations, "the range bounds the summary")

	pings, err := repo.GetPingSummary(ctx, monitoringTenant, monitoringPing, from, to, monitoringStale)
	require.NoError(t, err)
	require.NotNil(t, pings.LastPingAt)
	require.NotNil(t, pings.LastPingRTT)
	assert.GreaterOrEqual(t, *pings.LastPingRTT, time.Duration(0))
	assert.Equal(t, int64(1), pings.MissedPings)

	latest, err := repo.GetLatestOperation(ctx, monitoringTenant, monitoringConnect)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, string(models.OperationStateCompleted), latest.State)

	none, err := repo.GetLatestOperation(ctx, monitoringOtherTen, monitoringConnect)
	require.NoError(t, err)
	assert.Nil(t, none)
}
