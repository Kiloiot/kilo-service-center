package scacimonitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	testTenant     int64 = 7
	testSession    int64 = 42
	testSCEui            = uint64(0x70b3d59cd0000001)
	errBoom              = "boom"
	sessionsCount        = 3
	sessionOpCount       = 11
	failedCount          = 4
	missedPings          = 2
	reconnects           = 5
	pingRTT              = 250 * time.Millisecond
	logLevelError        = "ERROR"
)

var (
	now          = time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	serviceStart = now.Add(-time.Hour)
	errRepo      = errors.New(errBoom)
	testOrg      = uuid.MustParse("6aa6b3db-ceaa-4a71-8ece-59cc2263f019")
)

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

var _ clock.Clock = fixedClock{}

type fakeSessions struct {
	lastFilter *models.SCACISessionFilter
	sessions   []*models.SCACISession
	err        error
}

func (f *fakeSessions) GetSessionByID(_ context.Context, _ int64, id int64) (*models.SCACISession, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, s := range f.sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, storage.ErrNotFound
}

func (f *fakeSessions) GetSessionStatistics(_ context.Context, tenantID int64) (*models.SCACISessionStatistics, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &models.SCACISessionStatistics{TenantID: tenantID, TotalSessions: sessionsCount, ActiveSessions: 1}, nil
}

func (f *fakeSessions) ListSessions(_ context.Context, filter *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	f.lastFilter = filter
	if f.err != nil {
		return nil, 0, f.err
	}
	if filter.ConnectedFrom != nil {
		return nil, reconnects, nil
	}
	return f.sessions, int64(len(f.sessions)), nil
}

type fakeOperations struct {
	lastFrom, lastTo time.Time
	lastStale        time.Duration
	groups           []*models.SCACIOperationErrorGroup
	latest           *models.SCACIOperation
	err              error
}

func (f *fakeOperations) GetTenantOperationSummaryBetween(_ context.Context, tenantID int64, from, to time.Time) (*models.SCACIOperationSummary, error) {
	f.lastFrom, f.lastTo = from, to
	if f.err != nil {
		return nil, f.err
	}
	return &models.SCACIOperationSummary{TenantID: tenantID, TotalOperations: 10, StateCounts: map[string]int64{
		string(models.OperationStateCompleted):             6,
		string(models.OperationStateCompletedWithWarnings): 2,
		string(models.OperationStateFailed):                2,
	}}, nil
}

func (f *fakeOperations) CountOperationsBySession(_ context.Context, _ int64, ids []int64) (map[int64]int64, error) {
	if f.err != nil {
		return nil, f.err
	}
	counts := map[int64]int64{}
	for _, id := range ids {
		counts[id] = sessionOpCount
	}
	return counts, nil
}

func (f *fakeOperations) ListFailedOperationGroups(_ context.Context, _ int64, from, to time.Time, _, _ int) ([]*models.SCACIOperationErrorGroup, int64, error) {
	f.lastFrom, f.lastTo = from, to
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.groups, int64(len(f.groups)), nil
}

func (f *fakeOperations) GetPingSummary(_ context.Context, _ int64, _ string, from, to time.Time, stale time.Duration) (*models.SCACIPingSummary, error) {
	f.lastFrom, f.lastTo, f.lastStale = from, to, stale
	if f.err != nil {
		return nil, f.err
	}
	at := now.Add(-time.Minute)
	rtt := pingRTT
	return &models.SCACIPingSummary{LastPingAt: &at, LastPingRTT: &rtt, MissedPings: missedPings}, nil
}

func (f *fakeOperations) GetLatestOperation(_ context.Context, _ int64, _ string) (*models.SCACIOperation, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.latest, nil
}

type fakeQueue struct {
	entries []*storage.DownlinkMessage
	err     error
	filters []storage.DownlinkQueueFilter
}

func (f *fakeQueue) ListTenantQueue(_ context.Context, _ int64, filter storage.DownlinkQueueFilter, _, _ int) ([]*storage.DownlinkMessage, error) {
	f.filters = append(f.filters, filter)
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

func (f *fakeQueue) CountTenantQueue(_ context.Context, _ int64, filter storage.DownlinkQueueFilter) (int64, error) {
	f.filters = append(f.filters, filter)
	if f.err != nil {
		return 0, f.err
	}
	return int64(len(f.entries)), nil
}

type fakeListener struct{ listening bool }

func (f fakeListener) Listening() bool { return f.listening }

func newService(sessions *fakeSessions, ops *fakeOperations, queue *fakeQueue, listener ListenerState) *Service {
	return New(Deps{
		Sessions:     sessions,
		Operations:   ops,
		Queue:        queue,
		Listener:     listener,
		ServiceStart: serviceStart,
		SCEui:        testSCEui,
		Clock:        fixedClock{at: now},
		Log:          logger.NewNop(),
	})
}

func sampleSession() *models.SCACISession {
	hb := now.Add(-time.Minute)
	return &models.SCACISession{
		ID:                testSession,
		TenantID:          testTenant,
		AcEUI:             [8]byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x02},
		SnAcUUID:          [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SnScUUID:          [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		LastOpIDAc:        17,
		LastOpIDSc:        -9,
		Status:            models.SCACISessionStatusResumed,
		NegotiatedVersion: scaci.ProtocolVersionString,
		ConnectedAt:       now.Add(-2 * time.Hour),
		LastHeartbeat:     &hb,
		CanResume:         true,
	}
}

func TestListSessions_MapsSessionAndHonoursFilter(t *testing.T) {
	sessions := &fakeSessions{sessions: []*models.SCACISession{sampleSession()}}
	svc := newService(sessions, &fakeOperations{}, &fakeQueue{}, nil)
	status := models.SCACISessionStatusResumed
	canResume := true

	got, total, err := svc.ListSessions(testutil.TestContext(), testTenant, grpcservices.ScaciSessionFilter{Status: &status, CanResume: &canResume}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, got, 1)
	assert.Equal(t, "42", got[0].ID)
	assert.Equal(t, "70B3D59CD0000002", got[0].AcEUI)
	assert.Equal(t, "01020304-0506-0708-090a-0b0c0d0e0f10", got[0].SnAcUUID)
	assert.Equal(t, "100f0e0d-0c0b-0a09-0807-060504030201", got[0].SnScUUID)
	assert.Equal(t, int64(17), got[0].LastOpIDAc)
	assert.Equal(t, int64(-9), got[0].LastOpIDSc)
	assert.Equal(t, int64(sessionOpCount), got[0].OperationsCount)
	require.NotNil(t, sessions.lastFilter)
	assert.Equal(t, &status, sessions.lastFilter.Status)
	assert.Equal(t, &canResume, sessions.lastFilter.CanResume)
	assert.Equal(t, testTenant, *sessions.lastFilter.TenantID)
}

func TestListSessions_RejectsUnknownStatus(t *testing.T) {
	svc := newService(&fakeSessions{}, &fakeOperations{}, &fakeQueue{}, nil)
	status := "closed"
	_, _, err := svc.ListSessions(testutil.TestContext(), testTenant, grpcservices.ScaciSessionFilter{Status: &status}, 10, 0)
	assert.ErrorIs(t, err, ErrInvalidSessionStatus)
}

func TestGetSession_ParsesDecimalIDAndCountsOperations(t *testing.T) {
	svc := newService(&fakeSessions{sessions: []*models.SCACISession{sampleSession()}}, &fakeOperations{}, &fakeQueue{}, nil)

	got, err := svc.GetSession(testutil.TestContext(), testTenant, "42")
	require.NoError(t, err)
	assert.Equal(t, int64(sessionOpCount), got.OperationsCount)

	_, err = svc.GetSession(testutil.TestContext(), testTenant, "not-a-number")
	assert.ErrorIs(t, err, ErrInvalidSessionID)

	_, err = svc.GetSession(testutil.TestContext(), testTenant, "43")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestGetSession_UnknownSessionIsNotAnErrorCondition(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	svc := New(Deps{
		Sessions:   &fakeSessions{},
		Operations: &fakeOperations{},
		Queue:      &fakeQueue{},
		Clock:      fixedClock{at: now},
		Log:        log,
	})

	_, err := svc.GetSession(testutil.TestContext(), testTenant, "43")

	require.ErrorIs(t, err, ErrSessionNotFound)
	assert.Empty(t, log.AllAtLeast(logLevelError), "a missing session is a client lookup miss, not a service failure")
}

func TestGetSession_RepositoryFailureStaysAnError(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	svc := New(Deps{
		Sessions:   &fakeSessions{err: errRepo},
		Operations: &fakeOperations{},
		Queue:      &fakeQueue{},
		Clock:      fixedClock{at: now},
		Log:        log,
	})

	_, err := svc.GetSession(testutil.TestContext(), testTenant, "43")

	require.ErrorIs(t, err, errRepo)
	assert.NotErrorIs(t, err, ErrSessionNotFound)
	assert.Len(t, log.FilterMessage(LogGetSessionFailed), 1)
}

func TestGetStatistics_UsesRequestedWindowAndDefault(t *testing.T) {
	ops := &fakeOperations{}
	svc := newService(&fakeSessions{}, ops, &fakeQueue{}, nil)

	got, err := svc.GetStatistics(testutil.TestContext(), testTenant, grpcservices.ScaciWindow{})
	require.NoError(t, err)
	assert.Equal(t, now.Add(-config.SCACIDashboardDefaultWindow), ops.lastFrom)
	assert.Equal(t, now, ops.lastTo)
	assert.Equal(t, int64(8), got.SuccessfulOperations)
	assert.Equal(t, int64(2), got.FailedOperations)
	assert.InDelta(t, 80.0, got.SuccessRate, 0.001)
	require.NotNil(t, got.UptimeSince)
	assert.Equal(t, serviceStart, *got.UptimeSince)

	from := now.Add(-3 * time.Hour)
	to := now.Add(-time.Hour)
	_, err = svc.GetStatistics(testutil.TestContext(), testTenant, grpcservices.ScaciWindow{From: &from, To: &to})
	require.NoError(t, err)
	assert.Equal(t, from, ops.lastFrom)
	assert.Equal(t, to, ops.lastTo)

	_, err = svc.GetStatistics(testutil.TestContext(), testTenant, grpcservices.ScaciWindow{From: &to, To: &from})
	assert.ErrorIs(t, err, ErrInvalidTimeRange)
}

func TestResolveWindow_StartOnlyIsBoundedByNow(t *testing.T) {
	svc := newService(&fakeSessions{}, &fakeOperations{}, &fakeQueue{}, nil)
	from := now.Add(-30 * time.Minute)
	gotFrom, gotTo, err := svc.resolveWindow(grpcservices.ScaciWindow{From: &from})
	require.NoError(t, err)
	assert.Equal(t, from, gotFrom)
	assert.Equal(t, now, gotTo)
}

func TestListErrors_MapsGroups(t *testing.T) {
	code := 22
	token := "KC-SCACI-ERR-001"
	message := "invalid argument"
	ops := &fakeOperations{groups: []*models.SCACIOperationErrorGroup{{
		Command: scaci.CmdDLDataQueue, ErrorCode: &code, ErrorToken: &token, ErrorMessage: &message,
		SessionID: testSession, FirstSeen: now.Add(-time.Hour), LastSeen: now, Count: failedCount,
	}}}
	svc := newService(&fakeSessions{}, ops, &fakeQueue{}, nil)

	got, total, err := svc.ListErrors(testutil.TestContext(), testTenant, grpcservices.ScaciWindow{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, got, 1)
	assert.Equal(t, "dlDataQue:22:KC-SCACI-ERR-001", got[0].ID)
	assert.Equal(t, "22", got[0].ErrorCode)
	assert.Equal(t, token, got[0].ErrorToken)
	assert.Equal(t, message, got[0].ErrorMessage)
	assert.Equal(t, "42", got[0].SessionID)
	assert.Equal(t, int64(failedCount), got[0].Count)
	assert.Equal(t, now, got[0].OccurredAt)
}

func TestListQueues_MapsQueueRowsAndPropagatesErrors(t *testing.T) {
	sent := now.Add(-time.Minute)
	queue := &fakeQueue{entries: []*storage.DownlinkMessage{{
		ID: 5, EPEUI: "70B3D59CD0000003", Status: "queued", Payload: []byte{1, 2}, CreatedAt: now.Add(-time.Hour),
		SentAt: &sent, QueID: 99, Priority: 0.5,
	}}}
	svc := newService(&fakeSessions{}, &fakeOperations{}, queue, nil)

	got, total, err := svc.ListQueues(testutil.TestContext(), testTenant, testOrg, nil, 10, 0)
	require.NoError(t, err)
	for _, filter := range queue.filters {
		require.NotNil(t, filter.OrganizationID, "the listing and its count are scoped to the organization")
		assert.Equal(t, testOrg, *filter.OrganizationID)
	}
	assert.Equal(t, int64(1), total)
	require.Len(t, got, 1)
	assert.Equal(t, "5", got[0].ID)
	assert.Equal(t, scaci.CmdDLDataQueue, got[0].OperationType)
	assert.Equal(t, int64(99), got[0].QueID)
	assert.Equal(t, float32(0.5), got[0].Priority)
	assert.Equal(t, &sent, got[0].ProcessedAt)

	queue.err = errRepo
	_, _, err = svc.ListQueues(testutil.TestContext(), testTenant, testOrg, nil, 10, 0)
	assert.ErrorIs(t, err, errRepo)
}

func TestGetStatus_BindsListenerPingAndReconnects(t *testing.T) {
	ops := &fakeOperations{latest: &models.SCACIOperation{Command: scaci.CmdConnect, State: string(models.OperationStateFailed)}}
	queue := &fakeQueue{}
	svc := newService(&fakeSessions{}, ops, queue, fakeListener{listening: true})

	got, err := svc.GetStatus(testutil.TestContext(), testTenant, testOrg, grpcservices.ScaciWindow{})
	require.NoError(t, err)
	require.Len(t, queue.filters, 1)
	require.NotNil(t, queue.filters[0].OrganizationID, "the pending operations are counted for the organization")
	assert.Equal(t, testOrg, *queue.filters[0].OrganizationID)
	assert.True(t, got.ServiceOnline)
	assert.Equal(t, "70B3D59CD0000001", got.SCEui)
	assert.Equal(t, int32(1), got.ActiveSessions)
	require.NotNil(t, got.LastPingRTT)
	assert.Equal(t, pingRTT, *got.LastPingRTT)
	assert.Equal(t, int64(missedPings), got.MissedPings)
	assert.Equal(t, int64(reconnects), got.ReconnectAttempts)
	assert.Equal(t, string(models.OperationStateFailed), got.LastConnectResult)
	assert.Equal(t, serviceStart, *got.UptimeSince)
	assert.Equal(t, now.Add(-config.SCACIDashboardDefaultWindow), ops.lastFrom)

	offline := newService(&fakeSessions{}, &fakeOperations{}, &fakeQueue{}, nil)
	status, err := offline.GetStatus(testutil.TestContext(), testTenant, testOrg, grpcservices.ScaciWindow{})
	require.NoError(t, err)
	assert.False(t, status.ServiceOnline)
	assert.Empty(t, status.LastConnectResult)
}

func TestGetStatus_QueueFailureIsAnError(t *testing.T) {
	svc := newService(&fakeSessions{}, &fakeOperations{}, &fakeQueue{err: errRepo}, fakeListener{listening: true})
	_, err := svc.GetStatus(testutil.TestContext(), testTenant, testOrg, grpcservices.ScaciWindow{})
	assert.ErrorIs(t, err, errRepo)
}
