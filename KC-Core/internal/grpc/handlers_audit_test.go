package grpc

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	auditTestTenant int64 = 1
	auditTestActor        = "0b0d7c1e-2f5c-4c3a-9c2e-1f8a6f0a1d2e"
	auditTestEpEUI        = "70B3D59CD0000002"
)

var errAuditStore = errors.New("audit store down")

type captureAuditRecorder struct {
	events []audit.Event
}

func (c *captureAuditRecorder) Record(_ context.Context, ev audit.Event) {
	c.events = append(c.events, ev)
}

func (c *captureAuditRecorder) RecordRequired(_ context.Context, ev audit.Event) error {
	c.events = append(c.events, ev)
	return nil
}

// captureAuditEmitter captures the events the production recorder writes.
type captureAuditEmitter struct {
	events []audit.Event
}

func (c *captureAuditEmitter) EmitAudit(_ context.Context, ev audit.Event) error {
	c.events = append(c.events, ev)
	return nil
}

type discardDrops struct{}

func (discardDrops) Inc(string) {}

// productionRecorder records through audit.Recorder into a capture, so the
// acting user is stamped exactly as in production.
func productionRecorder(t *testing.T) (*audit.Recorder, *captureAuditEmitter) {
	t.Helper()
	capture := &captureAuditEmitter{}
	recorder, err := audit.NewRecorder(capture, logger.NewNop(), discardDrops{})
	require.NoError(t, err)
	return recorder, capture
}

func actorCtx() context.Context {
	return pkgcontext.WithUserID(testutil.TestContextWithTenantAndOrg(auditTestTenant, auditTestOrg), auditTestActor)
}

var auditTestOrg = uuid.MustParse("6aa6b3db-ceaa-4a71-8ece-59cc2263f019")

func TestNewCoreService_RequiresAnAuditRecorder(t *testing.T) {
	_, err := NewCoreService(CoreServiceDeps{
		Log:       logger.NewNop(),
		Endpoints: EndpointHandlerDeps{Endpoints: &fakeEndpointSvc{}, Attachment: &mockEndpointAttachmentSvc{}, Clock: clock.SystemClock{}, KeyReveals: &captureAuditRecorder{}},
	})
	require.ErrorIs(t, err, audit.ErrNilRecorder, "a core service that could drop every audit event unreported is not built")
}

func TestSendDownlink_RecordsQueuedAudit(t *testing.T) {
	scaciQueuer := &fakeSCACIQueuer{}
	recorder, emitter := productionRecorder(t)
	svc, err := NewCoreService(CoreServiceDeps{
		Log:          logger.NewNop(),
		Audit:        recorder,
		Endpoints:    EndpointHandlerDeps{Endpoints: &fakeEndpointSvc{}, Attachment: &mockEndpointAttachmentSvc{}, Clock: clock.SystemClock{}, KeyReveals: &captureAuditRecorder{}},
		BaseStations: BaseStationHandlerDeps{BaseStations: &fakeBasestationSvc{}, Stats: &fakeLegacyStorage{}, StatusReq: &fakeStatusReq{}, Ping: &fakePingCmd{}, Sessions: &fakeSessionDirectory{}},
		Downlinks:    downlinkHandlerDeps(t, downlinkFakes{endpoints: &fakeEndpointSvc{}, messages: &fakeMessageSvc{}, queuer: scaciQueuer, audit: recorder}),
		ULTransmit:   ULTransmitHandlerDeps{Sessions: &fakeSessionDirectory{}, Transmitter: &fakeULTransmit{}, BaseStations: &fakeBasestationSvc{}},
		DLRX:         DLRXHandlerDeps{Queries: &fakeLegacyStorage{}, Statuses: &fakeMessageSvc{}, Stations: &fakeMessageSvc{}, Commander: &fakeDownlinkCmd{}, Sessions: &fakeSessionDirectory{}},
		System:       SystemHandlerDeps{StartedAt: testServiceStart, SCEui: 1, SCVendor: "Test", SCModel: "TestCenter", SCName: "test-instance", SCSwVersion: "test-1.0"},
	})
	require.NoError(t, err)

	resp, err := svc.SendDownlink(actorCtx(), &pb.SendDownlinkRequest{EpEui: auditTestEpEUI, Payloads: [][]byte{{0x01}}})
	require.NoError(t, err)
	require.Len(t, emitter.events, 1)
	got := emitter.events[0]
	assert.Equal(t, models.EventTypeDownlinkQueued, got.EventType)
	assert.Equal(t, auditTestTenant, got.TenantID)
	assert.Equal(t, auditTestActor, got.UserID)
	assert.Empty(t, got.SourceName, "the operator's action is not on the endpoint's timeline, which records the station accepting the downlink")
	assert.Empty(t, got.SourceType, "the audit record is the actor's, filed under the API source")
	assert.Equal(t, auditTestEpEUI, got.Details["epEui"])
	assert.Equal(t, testServiceCenterQueueID, got.Details["queId"])
	assert.Equal(t, "0102030405060708", got.Details["bsEui"])
	assert.Equal(t, strconv.FormatUint(testServiceCenterQueueID, 10), resp.Id)
}

const (
	auditTestEndpointID int64 = 501
)

func TestDeleteEndPoint_AuditCarriesTheDeletedEndpointID(t *testing.T) {
	emitter := &captureAuditRecorder{}
	endpoints := &mockEndpointSvcIsolation{
		deleteFunc: func(context.Context, []byte, int64) (int64, error) { return auditTestEndpointID, nil },
	}
	svc := testCoreService(coreFields{endpointSvc: endpoints, audit: emitter, log: logger.NewNop()})

	_, err := svc.DeleteEndPoint(actorCtx(), &pb.DeleteEndPointRequest{EpEui: auditTestEpEUI})
	require.NoError(t, err)
	require.Len(t, emitter.events, 1)
	require.NotNil(t, emitter.events[0].EndpointID, "the delete event names the endpoint it removed")
	assert.Equal(t, models.EventCategoryEndpoint, emitter.events[0].Category, "endpoint lifecycle is filed under the endpoint category")
	assert.Equal(t, auditTestEndpointID, *emitter.events[0].EndpointID)
}

func TestDeleteBaseStation_AuditCarriesTheDeletedBaseStationID(t *testing.T) {
	emitter := &captureAuditRecorder{}
	svc := testCoreService(coreFields{basestationSvc: &ownedBaseStations{}, audit: emitter, log: logger.NewNop()})

	ctx := pkgcontext.WithUserID(testutil.TestContextWithTenantAndOrg(testOwnerTenant, auditTestOrg), auditTestActor)
	_, err := svc.DeleteBaseStation(ctx, &pb.DeleteBaseStationRequest{BsEui: testOwnedBsEui})
	require.NoError(t, err)
	require.Len(t, emitter.events, 1)
	require.NotNil(t, emitter.events[0].BaseStationID, "the delete event names the base station it removed")
	assert.Equal(t, models.EventCategoryBaseStation, emitter.events[0].Category, "base station lifecycle is filed under the base station category")
	assert.Equal(t, testOwnedBaseStationID, *emitter.events[0].BaseStationID)
}

// A station's certificate is filed under the base station category, which the
// base station managers who generate and read it may read.
func TestGenerateCertificate_AuditIsFiledForBaseStationManagers(t *testing.T) {
	recorder := &captureAuditRecorder{}
	certs := &persistCertSvc{station: &models.BaseStation{ID: testOwnedBaseStationID}}
	svc := testCoreService(coreFields{certSvc: certs, audit: recorder, log: logger.NewNop()})

	_, err := svc.GenerateCertificate(actorCtx(), &pb.GenerateCertificateRequest{BsEui: persistBsEUIDashed, ValidityDays: 365})
	require.NoError(t, err)
	require.Len(t, recorder.events, 1)
	assert.Equal(t, models.EventTypeCertificateGenerated, recorder.events[0].EventType)
	assert.Equal(t, models.EventCategoryBaseStation, recorder.events[0].Category)
}
