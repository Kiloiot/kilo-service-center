package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	alertsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/alerts"
	analyticsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/analytics"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/certificates"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	scacimonitoring "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci_monitoring"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/statistics"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// These tests cover the public RPCs no in-tree client calls. They are part of
// the published CoreService contract, so every one is exercised for tenant
// propagation, cross-tenant denial, request validation and response mapping.

const (
	retainedOwnerTenant   int64  = 41
	retainedForeignTenant int64  = 42
	retainedBsEUI         uint64 = 0x0011223344556677
	retainedBsEUIHex             = "0011223344556677"
	retainedBsID          int64  = 7
	retainedSessionID            = "9"
	retainedMessageID            = "msg-9"
	retainedMessageOpID          = int64(9007199254740993)
	retainedOpID          int64  = -13
	retainedCertType             = "ca"
	retainedIntervalSecs  int64  = 3600
	retainedBadStatus            = "closed"
)

// retainedFakes are tenant-scoped doubles: they answer only for the owner
// tenant and record every tenant they were asked about.
type retainedFakes struct {
	seenTenants       []int64
	connected         bool
	statErr           error
	lastSessionFilter grpcservices.ScaciSessionFilter
	lastAlertFilters  *grpcservices.AlertFilters
	alertListErr      error
}

func (f *retainedFakes) owns(tenantID int64) bool {
	f.seenTenants = append(f.seenTenants, tenantID)
	return tenantID == retainedOwnerTenant
}

// grpcservices.ScaciMonitoringService

func (f *retainedFakes) ListSessions(_ context.Context, tenantID int64, filter grpcservices.ScaciSessionFilter, _, _ int) ([]*grpcservices.ScaciSession, int64, error) {
	f.lastSessionFilter = filter
	if filter.Status != nil && *filter.Status == retainedBadStatus {
		return nil, 0, scacimonitoring.ErrInvalidSessionStatus
	}
	if !f.owns(tenantID) {
		return nil, 0, nil
	}
	return []*grpcservices.ScaciSession{{ID: retainedSessionID, AcEUI: retainedBsEUIHex, Status: "active", ConnectedAt: time.Unix(0, 0)}}, 1, nil
}

func (f *retainedFakes) GetSession(_ context.Context, tenantID int64, sessionID string) (*grpcservices.ScaciSession, error) {
	if !f.owns(tenantID) || sessionID != retainedSessionID {
		return nil, scacimonitoring.ErrSessionNotFound
	}
	return &grpcservices.ScaciSession{ID: sessionID, AcEUI: retainedBsEUIHex, Status: "active", ConnectedAt: time.Unix(0, 0)}, nil
}

func (f *retainedFakes) GetStatistics(_ context.Context, tenantID int64, window grpcservices.ScaciWindow) (*grpcservices.ScaciStatistics, error) {
	if window.From != nil && window.To != nil && window.From.After(*window.To) {
		return nil, scacimonitoring.ErrInvalidTimeRange
	}
	if !f.owns(tenantID) {
		return &grpcservices.ScaciStatistics{}, nil
	}
	return &grpcservices.ScaciStatistics{TotalSessions: 3, ActiveSessions: 1, TotalOperations: 10, SuccessfulOperations: 9, FailedOperations: 1, SuccessRate: 0.9}, nil
}

func (f *retainedFakes) ListErrors(_ context.Context, tenantID int64, _ grpcservices.ScaciWindow, _, _ int) ([]*grpcservices.ScaciError, int64, error) {
	if !f.owns(tenantID) {
		return nil, 0, nil
	}
	return []*grpcservices.ScaciError{{ID: "e1", ErrorCode: "1", ErrorMessage: "m", OccurredAt: time.Unix(0, 0)}}, 1, nil
}

func (f *retainedFakes) ListQueues(_ context.Context, tenantID int64, _ uuid.UUID, _ *[8]byte, _, _ int) ([]*grpcservices.ScaciQueue, int64, error) {
	if !f.owns(tenantID) {
		return nil, 0, nil
	}
	return []*grpcservices.ScaciQueue{{ID: "q1", EpEUI: retainedBsEUIHex, Status: "pending", QueuedAt: time.Unix(0, 0)}}, 1, nil
}

func (f *retainedFakes) GetStatus(_ context.Context, tenantID int64, _ uuid.UUID, _ grpcservices.ScaciWindow) (*grpcservices.ScaciStatus, error) {
	f.owns(tenantID)
	return &grpcservices.ScaciStatus{ServiceOnline: true, ActiveSessions: 1, ProtocolVersion: "1.0.0"}, nil
}

// grpcservices.AnalyticsService

func (f *retainedFakes) GetOverview(_ context.Context, tenantID int64, _, _ *time.Time) (*grpcservices.AnalyticsOverview, error) {
	f.owns(tenantID)
	return &grpcservices.AnalyticsOverview{}, nil
}

// Figures the analytics and alert fakes report for the owner tenant.
const (
	retainedCriticalAlerts     = 1
	retainedErrorAlerts        = 3
	retainedFirstDayMessages   = 4
	retainedFirstDayEndpoints  = 2
	retainedSecondDayMessages  = 6
	retainedSecondDayEndpoints = 3
	retainedStationMessages    = 5
)

func (f *retainedFakes) GetActivity(_ context.Context, tenantID int64, _, _ *time.Time, granularity string) (*grpcservices.ActivityAnalytics, error) {
	if granularity != "" && granularity != analyticsservice.GranularityDay {
		return nil, analyticsservice.ErrUnsupportedGranularity
	}
	if !f.owns(tenantID) {
		return &grpcservices.ActivityAnalytics{}, nil
	}
	return &grpcservices.ActivityAnalytics{
		StartTime:          time.Unix(0, 0),
		EndTime:            time.Unix(2*86400, 0),
		TotalMessages:      10,
		UniqueEndpoints:    3,
		UniqueBaseStations: 2,
		Slots: []grpcservices.ActivitySlot{
			{Slot: time.Unix(0, 0), MessageCount: retainedFirstDayMessages, EndpointCount: retainedFirstDayEndpoints},
			{Slot: time.Unix(86400, 0), MessageCount: retainedSecondDayMessages, EndpointCount: retainedSecondDayEndpoints},
		},
	}, nil
}

func (f *retainedFakes) GetSignalQuality(_ context.Context, tenantID int64, _, _ *time.Time) (*grpcservices.SignalQualityAnalytics, error) {
	if !f.owns(tenantID) {
		return &grpcservices.SignalQualityAnalytics{}, nil
	}
	return &grpcservices.SignalQualityAnalytics{
		AverageRSSI: -80, AverageSNR: 9, MedianRSSI: -82, MedianSNR: 8,
		RSSIRange: [2]float64{-100, -60}, SNRRange: [2]float64{2, 15},
		ByBaseStation: []grpcservices.BaseStationSignalQuality{{EUI: retainedBsEUIHex, AverageRSSI: -81, AverageSNR: 7, MessageCount: retainedStationMessages}},
	}, nil
}

// grpcservices.AlertService

func (f *retainedFakes) List(_ context.Context, tenantID int64, filters *grpcservices.AlertFilters, _, _ int) ([]*grpcservices.Alert, int64, error) {
	f.lastAlertFilters = filters
	if f.alertListErr != nil {
		return nil, 0, f.alertListErr
	}
	if !f.owns(tenantID) {
		return nil, 0, nil
	}
	return []*grpcservices.Alert{{ID: "a1", TenantID: tenantID, Severity: "warning", Timestamp: time.Unix(0, 0), Status: "new"}}, 1, nil
}

func (f *retainedFakes) GetSummary(_ context.Context, tenantID int64) (*grpcservices.AlertSummary, error) {
	if !f.owns(tenantID) {
		return &grpcservices.AlertSummary{}, nil
	}
	return &grpcservices.AlertSummary{Critical: retainedCriticalAlerts, Error: retainedErrorAlerts}, nil
}

// grpcservices.CertificateService

func (f *retainedFakes) GenerateCertificate(context.Context, *grpcservices.CertificateRequest) (*grpcservices.CertificateResponse, error) {
	return nil, nil
}

func (f *retainedFakes) DownloadCertificateByID(context.Context, int64, string, string) ([]byte, string, error) {
	return nil, "", nil
}

func (f *retainedFakes) GetStoredCertificate(_ context.Context, tenantID int64, _ []byte, certType string) ([]byte, string, error) {
	if !f.owns(tenantID) {
		return nil, "", certificates.ErrBaseStationNotFound
	}
	return []byte("PEM"), certType + ".pem", nil
}
func (f *retainedFakes) GenerateServerCertificates(context.Context) error { return nil }
func (f *retainedFakes) RenewServerCertificates(context.Context) error    { return nil }
func (f *retainedFakes) GetServerCertificateStatus(context.Context) (*grpcservices.CertificateStatus, error) {
	return nil, nil
}

// retainedStatisticsSvc implements grpcservices.StatisticsService.
type retainedStatisticsSvc struct{ f *retainedFakes }

func (s retainedStatisticsSvc) GetStatistics(_ context.Context, tenantID int64, _, _ *time.Time, _ string) (*grpcservices.StatisticsResult, error) {
	if s.f.statErr != nil {
		return nil, s.f.statErr
	}
	if !s.f.owns(tenantID) {
		return &grpcservices.StatisticsResult{}, nil
	}
	return &grpcservices.StatisticsResult{
		TotalMessages: 5, TotalEndpoints: 2, TotalBaseStations: 1,
		MessageCounts: []grpcservices.TimeSeriesPoint{{Timestamp: time.Unix(0, 0), Value: 5}},
	}, nil
}

// retainedMessageListing implements grpcservices.MessageListingService with tenant scoping.
type retainedMessageListing struct{ f *retainedFakes }

func (m retainedMessageListing) GetMessage(_ context.Context, tenantID int64, id string) (*mioty.ULDataMessage, error) {
	if !m.f.owns(tenantID) || id != retainedMessageID {
		return nil, storage.ErrNotFound
	}
	return &mioty.ULDataMessage{EpEui: retainedBsEUI, PacketCnt: 3, OpId: retainedMessageOpID}, nil
}

func (m retainedMessageListing) ListMessages(_ context.Context, tenantID int64, _ *grpcservices.MessageFilters, _, _ int) ([]*mioty.ULDataMessage, int64, error) {
	if !m.f.owns(tenantID) {
		return nil, 0, nil
	}
	return []*mioty.ULDataMessage{{EpEui: retainedBsEUI, PacketCnt: 3, OpId: retainedMessageOpID}}, 1, nil
}

func (m retainedMessageListing) StreamMessages(_ context.Context, tenantID int64, _ *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	if !m.f.owns(tenantID) {
		return nil, storage.ErrNotFound
	}
	ch := make(chan *mioty.ULDataMessage, 1)
	ch <- &mioty.ULDataMessage{EpEui: retainedBsEUI, PacketCnt: 3, OpId: retainedMessageOpID}
	close(ch)
	return ch, nil
}

func (m retainedMessageListing) ListBaseStationMessages(context.Context, int64, []byte, *grpcservices.MessageFilters, int, int) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}

func (m retainedMessageListing) GetBaseStationMessage(context.Context, int64, []byte, string) (*mioty.ULDataMessage, error) {
	return nil, nil
}

func (m retainedMessageListing) GetBaseStationMessageStats(context.Context, int64, []byte, *time.Time, *time.Time) (*mioty.BaseStationMessageStats, error) {
	return nil, nil
}

func (m retainedMessageListing) SearchBaseStationMessages(context.Context, int64, []byte, string, *grpcservices.MessageFilters, int, int) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}

func (m retainedMessageListing) ExportBaseStationMessages(context.Context, int64, []byte, *grpcservices.MessageFilters, string) ([]byte, error) {
	return nil, nil
}

func (m retainedMessageListing) StreamBaseStationMessages(context.Context, int64, []byte, *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	return nil, nil
}

// retainedBasestationSvc is a tenant-scoped grpcservices.BaseStationService.
type retainedBasestationSvc struct {
	fakeBasestationSvc
	f *retainedFakes
}

func (b *retainedBasestationSvc) GetByEUI(_ context.Context, _ []byte, tenantID int64) (*models.BaseStation, error) {
	if !b.f.owns(tenantID) {
		return nil, storage.ErrNotFound
	}
	return &models.BaseStation{ID: retainedBsID, TenantID: tenantID, Name: "bs"}, nil
}

// retainedSessionDir reports the base station as connected when configured.
type retainedSessionDir struct {
	fakeSessionDirectory
	f *retainedFakes
}

func (d *retainedSessionDir) GetSessionByEUI(_ uint64) interface{} {
	if d.f.connected {
		return struct{}{}
	}
	return nil
}

type retainedStatusReq struct{}

func (retainedStatusReq) SendStatusRequest(_ interface{}) (int64, error) { return retainedOpID, nil }

type retainedPingCmd struct{ f *retainedFakes }

func (p retainedPingCmd) InitiatePing(_ context.Context, _ uint64, tenantID int64) (int64, error) {
	p.f.owns(tenantID)
	return retainedOpID, nil
}

// retainedStatsStore scopes GetBaseStationMessageStats to the owner tenant.
type retainedStatsStore struct {
	stubMessageStore
	f *retainedFakes
}

func (s *retainedStatsStore) GetBaseStationMessageStats(_ context.Context, tenantID int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	if !s.f.owns(tenantID) {
		return nil, storage.ErrNotFound
	}
	return &mioty.BaseStationMessageStats{TotalMessages: 12}, nil
}

func newRetainedService(t *testing.T, f *retainedFakes) *CoreService {
	t.Helper()
	svc, err := NewCoreService(CoreServiceDeps{
		Log:          logger.NewNop(),
		Audit:        &captureAuditRecorder{},
		Endpoints:    EndpointHandlerDeps{Endpoints: &fakeEndpointSvc{}, Attachment: &mockEndpointAttachmentSvc{}, Clock: clock.SystemClock{}, KeyReveals: &captureAuditRecorder{}},
		BaseStations: BaseStationHandlerDeps{BaseStations: &retainedBasestationSvc{f: f}, Stats: &retainedStatsStore{f: f}, StatusReq: retainedStatusReq{}, Ping: retainedPingCmd{f: f}, Sessions: &retainedSessionDir{f: f}},
		Downlinks:    downlinkHandlerDeps(t, downlinkFakes{endpoints: &fakeEndpointSvc{}, messages: &fakeMessageSvc{}}),
		ULTransmit:   ULTransmitHandlerDeps{Sessions: &retainedSessionDir{f: f}, Transmitter: &fakeULTransmit{}, BaseStations: &retainedBasestationSvc{f: f}},
		DLRX:         DLRXHandlerDeps{Queries: &stubDLRXStorage{}, Statuses: &fakeMessageSvc{}, Stations: &fakeMessageSvc{}, Commander: &fakeDownlinkCmd{}, Sessions: &retainedSessionDir{f: f}},
		System:       SystemHandlerDeps{StartedAt: testServiceStart, SCEui: retainedBsEUI, SCVendor: "test", SCModel: "test", SCName: "test", SCSwVersion: "1.0.0"},
	})
	require.NoError(t, err)
	svc.scaciMonitorSvc = f
	svc.analyticsSvc = f
	svc.alertSvc = f
	svc.msgListingSvc = retainedMessageListing{f: f}
	svc.statisticsSvc = retainedStatisticsSvc{f: f}
	svc.certSvc = f
	svc.availabilityReader = stubAvailabilityReader{}
	svc.messageBucketReader = stubMessageBucketReader{}
	return svc
}

// ownerCtx is an administrator of the owner tenant, as the authorization interceptor admits one.
func ownerCtx() context.Context {
	return authz.WithRoles(testutil.TestContextWithTenantAndOrg(retainedOwnerTenant, retainedOwnerOrg), authz.AllRoles)
}
func foreignCtx() context.Context {
	return testutil.TestContextWithTenantAndOrg(retainedForeignTenant, retainedForeignOrg)
}

var (
	retainedOwnerOrg   = uuid.MustParse("6aa6b3db-ceaa-4a71-8ece-59cc2263f019")
	retainedForeignOrg = uuid.MustParse("fe7fe002-6880-4ea6-84ed-a69911dbdf8c")
)

func assertCode(t *testing.T, err error, token string) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, grpcerrors.GetGRPCCode(token), status.Code(err))
}

func assertSawOnly(t *testing.T, f *retainedFakes, tenantID int64) {
	t.Helper()
	require.NotEmpty(t, f.seenTenants, "the service must be consulted with the caller's tenant")
	for _, seen := range f.seenTenants {
		assert.Equal(t, tenantID, seen, "every lookup must carry the caller's tenant, never another")
	}
}

func metricsWindow() (*timestamppb.Timestamp, *timestamppb.Timestamp) {
	start, end := metricsTestWindow()
	return timestamppb.New(time.Unix(start, 0)), timestamppb.New(time.Unix(end, 0))
}

// --- SCACI observability ---------------------------------------------------

func TestRetained_ListScaciSessions(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.ListScaciSessions(ownerCtx(), &pb.ListScaciSessionsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Sessions, 1)
	assert.Equal(t, retainedSessionID, resp.Sessions[0].Id)
	assert.Equal(t, int32(1), resp.TotalCount)

	foreign, err := svc.ListScaciSessions(foreignCtx(), &pb.ListScaciSessionsRequest{})
	require.NoError(t, err)
	assert.Empty(t, foreign.Sessions, "a foreign tenant sees no sessions of another tenant")
	assert.Contains(t, f.seenTenants, retainedForeignTenant)

	_, err = svc.ListScaciSessions(ownerCtx(), &pb.ListScaciSessionsRequest{PageToken: "not-a-token"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidPageToken)

	_, err = svc.ListScaciSessions(testutil.TestContext(), &pb.ListScaciSessionsRequest{})
	require.Error(t, err, "missing tenant context must fail closed")

	nonResumable := false
	_, err = svc.ListScaciSessions(ownerCtx(), &pb.ListScaciSessionsRequest{Status: "resumed", CanResumeFilter: &nonResumable})
	require.NoError(t, err)
	require.NotNil(t, f.lastSessionFilter.Status)
	assert.Equal(t, "resumed", *f.lastSessionFilter.Status)
	require.NotNil(t, f.lastSessionFilter.CanResume, "explicit filter false must reach the service")
	assert.False(t, *f.lastSessionFilter.CanResume)

	_, err = svc.ListScaciSessions(ownerCtx(), &pb.ListScaciSessionsRequest{CanResume: true}) //nolint:staticcheck // legacy field set deliberately
	require.NoError(t, err)
	require.NotNil(t, f.lastSessionFilter.CanResume, "legacy true still filters to resumable sessions")
	assert.True(t, *f.lastSessionFilter.CanResume)

	resumable := true
	_, err = svc.ListScaciSessions(ownerCtx(), &pb.ListScaciSessionsRequest{CanResume: false, CanResumeFilter: &resumable}) //nolint:staticcheck // legacy field set deliberately
	require.NoError(t, err)
	require.NotNil(t, f.lastSessionFilter.CanResume, "the explicit filter wins over the legacy field")
	assert.True(t, *f.lastSessionFilter.CanResume)

	_, err = svc.ListScaciSessions(ownerCtx(), &pb.ListScaciSessionsRequest{})
	require.NoError(t, err)
	assert.Nil(t, f.lastSessionFilter.CanResume, "absent filter means no resumability filter")

	_, err = svc.ListScaciSessions(ownerCtx(), &pb.ListScaciSessionsRequest{Status: retainedBadStatus})
	assertCode(t, err, grpcerrors.ErrTokenScaciInvalidSessionStatus)
}

func TestRetained_GetScaciStatisticsRejectsInvertedRange(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	later := timestamppb.New(time.Unix(200, 0))
	earlier := timestamppb.New(time.Unix(100, 0))
	_, err := svc.GetScaciStatistics(ownerCtx(), &pb.GetScaciStatisticsRequest{StartTime: later, EndTime: earlier})
	assertCode(t, err, grpcerrors.ErrTokenInvalidTimeRange)
}

func TestRetained_ListScaciQueuesRejectsMalformedEndpoint(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	_, err := svc.ListScaciQueues(ownerCtx(), &pb.ListScaciQueuesRequest{EpEui: "not-hex"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidEndpointEUIFormat)
}

func TestRetained_GetScaciSession(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.GetScaciSession(ownerCtx(), &pb.GetScaciSessionRequest{Id: retainedSessionID})
	require.NoError(t, err)
	assert.Equal(t, retainedSessionID, resp.Session.Id)

	_, err = svc.GetScaciSession(foreignCtx(), &pb.GetScaciSessionRequest{Id: retainedSessionID})
	assertCode(t, err, grpcerrors.ErrTokenScaciSessionNotFound)

	_, err = svc.GetScaciSession(ownerCtx(), &pb.GetScaciSessionRequest{})
	assertCode(t, err, grpcerrors.ErrTokenIDRequired)
}

func TestRetained_GetScaciStatistics(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.GetScaciStatistics(ownerCtx(), &pb.GetScaciStatisticsRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), resp.Statistics.TotalSessions)
	assert.Equal(t, int64(9), resp.Statistics.SuccessfulOperations)
	assert.InDelta(t, 0.9, resp.Statistics.SuccessRate, 0.0001)
	assertSawOnly(t, f, retainedOwnerTenant)
}

func TestRetained_ListScaciErrorsAndQueues(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	errs, err := svc.ListScaciErrors(ownerCtx(), &pb.ListScaciErrorsRequest{})
	require.NoError(t, err)
	require.Len(t, errs.Errors, 1)
	assert.Equal(t, "e1", errs.Errors[0].Id)

	queues, err := svc.ListScaciQueues(ownerCtx(), &pb.ListScaciQueuesRequest{})
	require.NoError(t, err)
	require.Len(t, queues.QueueEntries, 1)
	assert.Equal(t, "q1", queues.QueueEntries[0].Id)

	foreignErrs, err := svc.ListScaciErrors(foreignCtx(), &pb.ListScaciErrorsRequest{})
	require.NoError(t, err)
	assert.Empty(t, foreignErrs.Errors)
	foreignQueues, err := svc.ListScaciQueues(foreignCtx(), &pb.ListScaciQueuesRequest{})
	require.NoError(t, err)
	assert.Empty(t, foreignQueues.QueueEntries)
}

func TestRetained_GetScaciStatus(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.GetScaciStatus(ownerCtx(), &pb.GetScaciStatusRequest{})
	require.NoError(t, err)
	assert.True(t, resp.Status.ServiceOnline)
	assert.Equal(t, "1.0.0", resp.Status.ProtocolVersion)
	assertSawOnly(t, f, retainedOwnerTenant)
}

func TestRetained_ScaciHandlersRequireService(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	svc.scaciMonitorSvc = nil

	_, err := svc.GetScaciStatus(ownerCtx(), &pb.GetScaciStatusRequest{})
	assertCode(t, err, grpcerrors.ErrTokenServiceNotConfigured)
}

// --- analytics, statistics, alerts ------------------------------------------

func TestRetained_GetActivityAnalytics(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.GetActivityAnalytics(ownerCtx(), &pb.GetActivityAnalyticsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Activity.TimeSlots, 2)
	assert.Equal(t, int64(10), resp.Activity.TotalMessages)
	assert.Equal(t, int64(3), resp.Activity.UniqueEndpoints)
	assert.Equal(t, int64(2), resp.Activity.UniqueBaseStations)
	assert.Equal(t, time.Unix(0, 0).UTC(), resp.Activity.TimeSlots[0].Slot.AsTime())
	assert.Equal(t, int64(retainedSecondDayMessages), resp.Activity.TimeSlots[1].MessageCount)
	assert.Equal(t, int64(retainedSecondDayEndpoints), resp.Activity.TimeSlots[1].EndpointCount)
	assert.Equal(t, time.Unix(2*86400, 0).UTC(), resp.Activity.EndTime.AsTime())

	foreign, err := svc.GetActivityAnalytics(foreignCtx(), &pb.GetActivityAnalyticsRequest{})
	require.NoError(t, err)
	assert.Empty(t, foreign.Activity.TimeSlots)
	assert.Zero(t, foreign.Activity.TotalMessages)
}

func TestRetained_GetActivityAnalytics_RejectsUnsupportedGranularity(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})

	_, err := svc.GetActivityAnalytics(ownerCtx(), &pb.GetActivityAnalyticsRequest{Granularity: "week"})
	assertCode(t, err, grpcerrors.ErrTokenUnsupportedGranularity)
}

func TestRetained_GetSignalQualityAnalytics(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.GetSignalQualityAnalytics(ownerCtx(), &pb.GetSignalQualityAnalyticsRequest{})
	require.NoError(t, err)
	assert.InDelta(t, -80, resp.SignalQuality.Overall.AvgRssi, 0.0001)
	assert.InDelta(t, -100, resp.SignalQuality.Overall.MinRssi, 0.0001)
	assert.InDelta(t, -60, resp.SignalQuality.Overall.MaxRssi, 0.0001)
	assert.InDelta(t, -82, resp.SignalQuality.Overall.MedianRssi, 0.0001)
	assert.InDelta(t, 8, resp.SignalQuality.Overall.MedianSnr, 0.0001)
	require.Len(t, resp.SignalQuality.ByBaseStation, 1)
	assert.Equal(t, retainedBsEUIHex, resp.SignalQuality.ByBaseStation[0].Eui)
	assert.Equal(t, int64(retainedStationMessages), resp.SignalQuality.ByBaseStation[0].MessageCount)
	assertSawOnly(t, f, retainedOwnerTenant)

	foreign, err := svc.GetSignalQualityAnalytics(foreignCtx(), &pb.GetSignalQualityAnalyticsRequest{})
	require.NoError(t, err)
	assert.Empty(t, foreign.SignalQuality.ByBaseStation)
}

func TestRetained_GetStatistics(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.GetStatistics(ownerCtx(), &pb.GetStatisticsRequest{Granularity: "hour"})
	require.NoError(t, err)
	assert.Equal(t, int64(5), resp.TotalMessages)
	require.Len(t, resp.MessageCounts, 1)
	assert.Equal(t, int64(5), resp.MessageCounts[0].Value)

	f.statErr = statistics.ErrUnsupportedGranularity
	_, err = svc.GetStatistics(ownerCtx(), &pb.GetStatisticsRequest{Granularity: "eon"})
	assertCode(t, err, grpcerrors.ErrTokenUnsupportedGranularity)

	f.statErr = errors.New("db down")
	_, err = svc.GetStatistics(ownerCtx(), &pb.GetStatisticsRequest{})
	assertCode(t, err, grpcerrors.ErrTokenMessageStatsFailed)
}

func TestRetained_ListAlerts(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.ListAlerts(ownerCtx(), &pb.ListAlertsRequest{Severity: "warning"})
	require.NoError(t, err)
	require.Len(t, resp.Alerts, 1)
	assert.Equal(t, "a1", resp.Alerts[0].Id)

	foreign, err := svc.ListAlerts(foreignCtx(), &pb.ListAlertsRequest{})
	require.NoError(t, err)
	assert.Empty(t, foreign.Alerts)
}

func TestRetained_ListAlerts_PassesTheStatusFilter(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	_, err := svc.ListAlerts(ownerCtx(), &pb.ListAlertsRequest{Status: models.EventStatusAcknowledged})

	require.NoError(t, err)
	assert.Equal(t, []string{models.EventStatusAcknowledged}, f.lastAlertFilters.Status)
}

func TestRetained_ListAlerts_RejectsAnUnknownStatus(t *testing.T) {
	f := &retainedFakes{alertListErr: fmt.Errorf("%w: running", alertsservice.ErrInvalidAlertStatus)}
	svc := newRetainedService(t, f)

	_, err := svc.ListAlerts(ownerCtx(), &pb.ListAlertsRequest{Status: "running"})

	assertCode(t, err, grpcerrors.ErrTokenInvalidAlertStatus)
}

func TestRetained_ListAlerts_RejectsASeverityOutsideTheAlerts(t *testing.T) {
	f := &retainedFakes{alertListErr: fmt.Errorf("%w: info", alertsservice.ErrInvalidAlertSeverity)}
	svc := newRetainedService(t, f)

	_, err := svc.ListAlerts(ownerCtx(), &pb.ListAlertsRequest{Severity: models.EventSeverityInfo})

	assertCode(t, err, grpcerrors.ErrTokenInvalidAlertSeverity)
}

func TestRetained_GetAlertSummary_ReportsTheErrorCount(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.GetAlertSummary(ownerCtx(), &pb.GetAlertSummaryRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(retainedErrorAlerts), resp.Summary.Error)
	assert.Equal(t, int32(retainedCriticalAlerts), resp.Summary.Critical)

	foreign, err := svc.GetAlertSummary(foreignCtx(), &pb.GetAlertSummaryRequest{})
	require.NoError(t, err)
	assert.Zero(t, foreign.Summary.Error)
}

// --- base-station statistics and metrics -------------------------------------

func TestRetained_GetBaseStationStats(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.GetBaseStationStats(ownerCtx(), &pb.GetBaseStationStatsRequest{BsEui: retainedBsEUIHex})
	require.NoError(t, err)
	assert.Equal(t, int64(12), resp.TotalMessages)

	_, err = svc.GetBaseStationStats(foreignCtx(), &pb.GetBaseStationStatsRequest{BsEui: retainedBsEUIHex})
	assertCode(t, err, grpcerrors.ErrTokenGetBaseStationStatsFailed)

	_, err = svc.GetBaseStationStats(ownerCtx(), &pb.GetBaseStationStatsRequest{BsEui: "zz"})
	assertCode(t, err, grpcerrors.ErrTokenInvalidBasestationEUIFormat)

	_, err = svc.GetBaseStationStats(ownerCtx(), &pb.GetBaseStationStatsRequest{})
	assertCode(t, err, grpcerrors.ErrTokenBasestationEUIRequired)
}

func TestRetained_BaseStationMetrics_OwnerAndForeign(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	start, end := metricsWindow()

	avail, err := svc.GetBaseStationAvailability(ownerCtx(), &pb.GetBaseStationAvailabilityRequest{
		BsEui: retainedBsEUIHex, StartTime: start, EndTime: end, IntervalSeconds: retainedIntervalSecs,
	})
	require.NoError(t, err)
	assert.Equal(t, retainedBsEUIHex, avail.BsEui)
	assert.Equal(t, retainedIntervalSecs, avail.IntervalSeconds)

	recv, err := svc.GetBaseStationMessagesReceived(ownerCtx(), &pb.GetBaseStationMessagesReceivedRequest{
		BsEui: retainedBsEUIHex, StartTime: start, EndTime: end, IntervalSeconds: retainedIntervalSecs,
	})
	require.NoError(t, err)
	assert.Equal(t, retainedBsEUIHex, recv.BsEui)

	_, err = svc.GetBaseStationAvailability(foreignCtx(), &pb.GetBaseStationAvailabilityRequest{
		BsEui: retainedBsEUIHex, StartTime: start, EndTime: end, IntervalSeconds: retainedIntervalSecs,
	})
	require.Error(t, err, "a foreign tenant must not read another tenant's base station metrics")
	_, err = svc.GetBaseStationMessagesReceived(foreignCtx(), &pb.GetBaseStationMessagesReceivedRequest{
		BsEui: retainedBsEUIHex, StartTime: start, EndTime: end, IntervalSeconds: retainedIntervalSecs,
	})
	require.Error(t, err)
}

// --- messages ----------------------------------------------------------------

func TestRetained_GetMessageAndListMessages(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	msg, err := svc.GetMessage(ownerCtx(), &pb.GetMessageRequest{Id: retainedMessageID})
	require.NoError(t, err)
	assert.Equal(t, uint32(3), msg.PacketCounter)
	assert.Equal(t, retainedMessageOpID, msg.OpId)

	_, err = svc.GetMessage(foreignCtx(), &pb.GetMessageRequest{Id: retainedMessageID})
	assertCode(t, err, grpcerrors.ErrTokenMessageNotFound)

	_, err = svc.GetMessage(ownerCtx(), &pb.GetMessageRequest{})
	assertCode(t, err, grpcerrors.ErrTokenMessageIDRequired)

	list, err := svc.ListMessages(ownerCtx(), &pb.ListMessagesRequest{})
	require.NoError(t, err)
	require.Len(t, list.Messages, 1)
	assert.Equal(t, int32(1), list.TotalCount)
	assert.Equal(t, retainedMessageOpID, list.Messages[0].OpId)

	foreign, err := svc.ListMessages(foreignCtx(), &pb.ListMessagesRequest{})
	require.NoError(t, err)
	assert.Empty(t, foreign.Messages)
}

// --- base-station control actions ---------------------------------------------

func TestRetained_RequestBaseStationStatus(t *testing.T) {
	f := &retainedFakes{connected: true}
	svc := newRetainedService(t, f)

	resp, err := svc.RequestBaseStationStatus(ownerCtx(), &pb.BaseStationStatusRequest{BsEuiHex: retainedBsEUIHex})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, retainedOpID, resp.OpId)

	_, err = svc.RequestBaseStationStatus(foreignCtx(), &pb.BaseStationStatusRequest{BsEuiHex: retainedBsEUIHex})
	assertCode(t, err, grpcerrors.ErrTokenBaseStationNotFound)

	f.connected = false
	offline, err := svc.RequestBaseStationStatus(ownerCtx(), &pb.BaseStationStatusRequest{BsEuiHex: retainedBsEUIHex})
	require.NoError(t, err)
	assert.False(t, offline.Success, "a disconnected base station reports failure, not an error")
	assert.Zero(t, offline.OpId)
}

func TestRetained_InitiatePing(t *testing.T) {
	f := &retainedFakes{connected: true}
	svc := newRetainedService(t, f)

	resp, err := svc.InitiatePing(ownerCtx(), &pb.InitiatePingRequest{BsEuiHex: retainedBsEUIHex})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, retainedOpID, resp.OpId)
	assertSawOnly(t, f, retainedOwnerTenant)

	_, err = svc.InitiatePing(foreignCtx(), &pb.InitiatePingRequest{BsEuiHex: retainedBsEUIHex})
	assertCode(t, err, grpcerrors.ErrTokenBaseStationNotFound)
}

// --- certificates -------------------------------------------------------------

func TestRetained_DownloadBaseStationCertificate(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)

	resp, err := svc.DownloadBaseStationCertificate(ownerCtx(), &pb.DownloadBaseStationCertificateRequest{BsEui: retainedBsEUIHex, CertType: retainedCertType})
	require.NoError(t, err)
	assert.Equal(t, []byte("PEM"), resp.Content)
	assert.Equal(t, retainedCertType+".pem", resp.Filename)

	_, err = svc.DownloadBaseStationCertificate(foreignCtx(), &pb.DownloadBaseStationCertificateRequest{BsEui: retainedBsEUIHex, CertType: retainedCertType})
	assertCode(t, err, grpcerrors.ErrTokenBaseStationNotFound)

	_, err = svc.DownloadBaseStationCertificate(ownerCtx(), &pb.DownloadBaseStationCertificateRequest{BsEui: retainedBsEUIHex})
	assertCode(t, err, grpcerrors.ErrTokenCertTypeRequired)

	_, err = svc.DownloadBaseStationCertificate(ownerCtx(), &pb.DownloadBaseStationCertificateRequest{CertType: retainedCertType})
	assertCode(t, err, grpcerrors.ErrTokenBasestationEUIRequired)
}
