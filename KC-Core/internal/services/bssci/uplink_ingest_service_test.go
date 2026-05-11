package bssciservices

import (
	"context"
	"crypto/x509"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	uplinkIngestTestTenantID = int64(42)
	uplinkIngestTestEpEUI    = uint64(0x12345678)
	uplinkIngestTestBsEUI    = uint64(0xABCDEF1234567890)
)

// --- MQTT publisher fake with channel sync (uplink publishes are async) ---

type mqttUplinkCall struct {
	OrgUUID   string
	EpEUI     uint64
	BsEUI     uint64
	RSSI      float64
	SNR       float64
	RxTime    int64
	PacketCnt uint32
	UserData  []byte
}

type capturingMQTTPublisher struct {
	uplinkCalls chan mqttUplinkCall
}

func newCapturingMQTTPublisher() *capturingMQTTPublisher {
	return &capturingMQTTPublisher{uplinkCalls: make(chan mqttUplinkCall, 4)}
}

func (p *capturingMQTTPublisher) PublishUplink(_ context.Context, orgUUID string,
	epEUI, bsEUI uint64, rssi, snr float64, rxTime int64,
	packetCnt uint32, userData []byte) error {
	p.uplinkCalls <- mqttUplinkCall{
		OrgUUID:   orgUUID,
		EpEUI:     epEUI,
		BsEUI:     bsEUI,
		RSSI:      rssi,
		SNR:       snr,
		RxTime:    rxTime,
		PacketCnt: packetCnt,
		UserData:  userData,
	}
	return nil
}

func (p *capturingMQTTPublisher) PublishAttach(_ context.Context, _ string, _, _ uint64) error {
	return nil
}
func (p *capturingMQTTPublisher) PublishDetach(_ context.Context, _ string, _, _ uint64) error {
	return nil
}
func (p *capturingMQTTPublisher) PublishDownlinkResult(_ context.Context, _ string, _, _ uint64, _ string) error {
	return nil
}

// --- org.Resolver fake mapping tenantID → org UUID ---

type fakeOrgResolver struct {
	defaultOrgByTenant map[int64]uuid.UUID
}

func (r *fakeOrgResolver) LookupTenant(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *fakeOrgResolver) ResolveCert(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
	return uuid.Nil, 0, nil
}
func (r *fakeOrgResolver) GetDefaultOrgForTenant(_ context.Context, tenantID int64) (uuid.UUID, error) {
	if id, ok := r.defaultOrgByTenant[tenantID]; ok {
		return id, nil
	}
	return uuid.Nil, nil
}

// --- endpoint repository fake returning a seeded endpoint by EUI ---

type uplinkIngestEndpointRepo struct {
	endpoints map[uint64]*models.EndPoint
}

func (r *uplinkIngestEndpointRepo) Get(_ context.Context, eui models.EUI) (*models.EndPoint, error) {
	if ep, ok := r.endpoints[eui.ToUint64()]; ok {
		return ep, nil
	}
	return nil, storage.ErrNotFound
}

func (r *uplinkIngestEndpointRepo) GetByEUI(context.Context, int64, []byte) (*models.EndPoint, error) {
	return nil, nil
}
func (r *uplinkIngestEndpointRepo) Create(context.Context, *models.EndPoint) error { return nil }
func (r *uplinkIngestEndpointRepo) GetByID(context.Context, int64, int64) (*models.EndPoint, error) {
	return nil, nil
}
func (r *uplinkIngestEndpointRepo) GetByTenant(context.Context, int64) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *uplinkIngestEndpointRepo) CountByTenant(context.Context, int64) (int64, error) {
	return 0, nil
}
func (r *uplinkIngestEndpointRepo) ListByTenantPaginated(context.Context, int64, int, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *uplinkIngestEndpointRepo) Update(context.Context, *models.EndPoint) error { return nil }
func (r *uplinkIngestEndpointRepo) UpdateFields(context.Context, int64, int64, map[string]interface{}) error {
	return nil
}
func (r *uplinkIngestEndpointRepo) UpdateLastSeen(context.Context, int64, models.EUI, uint32) error {
	return nil
}
func (r *uplinkIngestEndpointRepo) UpdateRadioMetrics(context.Context, int64, models.EUI, float64, float64, float64, int64, int64, string) error {
	return nil
}
func (r *uplinkIngestEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, interfaces.RadioMetricsUpdate) error {
	return nil
}
func (r *uplinkIngestEndpointRepo) UpdateDetachMetrics(context.Context, int64, models.EUI, interfaces.DetachMetricsUpdate) error {
	return nil
}
func (r *uplinkIngestEndpointRepo) StreamAllForPropagation(context.Context, int64, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *uplinkIngestEndpointRepo) HasEndpointsSince(context.Context, time.Time) (bool, error) {
	return false, nil
}
func (r *uplinkIngestEndpointRepo) GetEndpointWithKeysForDetachValidation(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, nil
}
func (r *uplinkIngestEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil
}
func (r *uplinkIngestEndpointRepo) DeleteByTenant(context.Context, int64, []byte) error {
	return nil
}
func (r *uplinkIngestEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (r *uplinkIngestEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error { return nil }

// --- minimal MIOTYMessageRepository that accepts CreateULDataMessage ---

type uplinkIngestMessageRepo struct{}

func (r *uplinkIngestMessageRepo) CreateULDataMessage(_ context.Context, _ *mioty.ULDataMessage) error {
	return nil
}
func (r *uplinkIngestMessageRepo) CreateDetachMessage(_ context.Context, _ *mioty.DetachMessage, _ map[string]interface{}) error {
	return nil
}
func (r *uplinkIngestMessageRepo) CreateAttachMessage(_ context.Context, _ *mioty.AttachMessage, _ map[string]interface{}) error {
	return nil
}
func (r *uplinkIngestMessageRepo) CreateAttachPropagateMessage(_ context.Context, _ *mioty.AttachPropagateMessage) error {
	return nil
}
func (r *uplinkIngestMessageRepo) CreateDetachPropagateMessage(_ context.Context, _ *mioty.DetachPropagateMessage) error {
	return nil
}
func (r *uplinkIngestMessageRepo) GetULDataMessage(context.Context, string, int64) (*mioty.ULDataMessage, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetDetachMessage(context.Context, string, int64) (*mioty.DetachMessage, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) ListULDataMessages(context.Context, mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}
func (r *uplinkIngestMessageRepo) UpdateULDataBaseStations(context.Context, int64, uint64, uint32, int64, []byte) error {
	return nil
}
func (r *uplinkIngestMessageRepo) GetMessageStatsByBaseStation(context.Context, uint64, int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetExtendedMessageStatsByBaseStation(context.Context, uint64, int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetMessageStatsByEndpoint(context.Context, uint64, int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetOverallStats(context.Context, int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetAnalyticsOverview(context.Context, int64, time.Time, time.Time) (*mioty.AnalyticsOverviewStats, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetHourlyActivity(context.Context, int64, time.Time, time.Time) ([]mioty.HourlyActivity, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetDailyActivity(context.Context, int64, time.Time, time.Time) ([]mioty.DailyActivity, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetTopEndpointsByActivity(context.Context, int64, time.Time, time.Time, int) ([]mioty.EndpointActivity, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetSignalQualityStats(context.Context, int64, time.Time, time.Time) (*mioty.SignalQualityStats, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetSignalQualityByBaseStation(context.Context, int64, time.Time, time.Time) ([]mioty.BaseStationSignalQuality, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetBaseStationMessageStats(context.Context, int64, []byte, *time.Time, *time.Time) (*mioty.BaseStationMessageStats, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetMessageCountsByEndpoint(context.Context, int64, time.Time, time.Time) (map[string]int64, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetMessageCountsByBaseStation(context.Context, int64, time.Time, time.Time) (map[string]int64, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetWeeklyActivity(context.Context, int64, time.Time, time.Time) ([]mioty.WeeklyActivity, error) {
	return nil, nil
}
func (r *uplinkIngestMessageRepo) GetMonthlyActivity(context.Context, int64, time.Time, time.Time) ([]mioty.MonthlyActivity, error) {
	return nil, nil
}

// --- minimal interfaces.Storage exposing MIOTYMessages only ---

type uplinkIngestStorage struct {
	miotyMessages interfaces.MIOTYMessageRepository
}

func (s *uplinkIngestStorage) MIOTYMessages() interfaces.MIOTYMessageRepository {
	return s.miotyMessages
}
func (s *uplinkIngestStorage) EndPoints() interfaces.EndpointRepository          { return nil }
func (s *uplinkIngestStorage) DownlinkQueue() interfaces.DownlinkQueueRepository { return nil }
func (s *uplinkIngestStorage) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	return nil
}
func (s *uplinkIngestStorage) EndPointSessions() interfaces.EndPointSessionRepository { return nil }
func (s *uplinkIngestStorage) EndPointKeys() interfaces.EndPointKeyRepository         { return nil }
func (s *uplinkIngestStorage) RoamingAgreements() interfaces.RoamingAgreementRepository {
	return nil
}
func (s *uplinkIngestStorage) BaseStations() interfaces.BaseStationRepository { return nil }
func (s *uplinkIngestStorage) BaseStationSessions() interfaces.BaseStationSessionRepository {
	return nil
}
func (s *uplinkIngestStorage) DLRXStatus() interfaces.DLRXStatusRepository { return nil }
func (s *uplinkIngestStorage) PendingOperations() interfaces.PendingOperationRepository {
	return nil
}
func (s *uplinkIngestStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
func (s *uplinkIngestStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (s *uplinkIngestStorage) Users() interfaces.UserRepository                 { return nil }
func (s *uplinkIngestStorage) APIKeys() interfaces.APIKeyRepository             { return nil }
func (s *uplinkIngestStorage) Integrations() interfaces.IntegrationRepository   { return nil }
func (s *uplinkIngestStorage) Manufacturers() interfaces.ManufacturerRepository { return nil }
func (s *uplinkIngestStorage) DeviceModels() interfaces.DeviceModelRepository   { return nil }
func (s *uplinkIngestStorage) Blueprints() interfaces.BlueprintRepository       { return nil }
func (s *uplinkIngestStorage) Organizations() interfaces.OrganizationRepository { return nil }
func (s *uplinkIngestStorage) GetSqlxDB() *sqlx.DB                              { return nil }
func (s *uplinkIngestStorage) SystemEvents() interfaces.SystemEventStore        { return nil }
func (s *uplinkIngestStorage) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (s *uplinkIngestStorage) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}
func (s *uplinkIngestStorage) DownlinkQueueReader() interfaces.DownlinkQueueReader { return nil }
func (s *uplinkIngestStorage) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	return nil, nil
}
func (s *uplinkIngestStorage) Ping(_ context.Context) error { return nil }
func (s *uplinkIngestStorage) Close() error                 { return nil }

// --- test setup ---

func newUplinkIngestForTest(t *testing.T, orgMapping map[int64]uuid.UUID, publisher *capturingMQTTPublisher) *UplinkIngestServiceImpl {
	t.Helper()
	endpointRepo := &uplinkIngestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			uplinkIngestTestEpEUI: {
				ID:       1,
				TenantID: uplinkIngestTestTenantID,
			},
		},
	}
	stor := &uplinkIngestStorage{miotyMessages: &uplinkIngestMessageRepo{}}
	dedup := bssci.NewMessageDeduplicator(5 * time.Minute)
	t.Cleanup(func() { dedup.Stop() })
	return NewUplinkIngestService(
		dedup,
		stor,
		&fakeOrgResolver{defaultOrgByTenant: orgMapping},
		nil, // roamingSvc — nil triggers the direct endpoint-repo lookup path
		endpointRepo,
		nil, // blueprintResolver
		nil, // blueprintDecoder
		nil, // broadcaster
		publisher,
		logger.NewNop(),
		uplinkIngestTestTenantID,
		0,
	)
}

func buildUplinkPayload() *bssci.UplinkPayload {
	return &bssci.UplinkPayload{
		EpEUI:     uplinkIngestTestEpEUI,
		BsEUI:     uplinkIngestTestBsEUI,
		PacketCnt: 42,
		UserData:  []byte{0x01, 0x02, 0x03},
		SNR:       12.3,
		RSSI:      -85.5,
		RxTime:    time.Now().UnixNano(),
	}
}

// TestUplinkIngest_PublishesMQTTUplink verifies the real ingest service triggers
// MQTTEventPublisher.PublishUplink with the owner org UUID and payload fields.
func TestUplinkIngest_PublishesMQTTUplink(t *testing.T) {
	t.Parallel()

	ownerOrg := uuid.New()
	publisher := newCapturingMQTTPublisher()
	svc := newUplinkIngestForTest(t,
		map[int64]uuid.UUID{uplinkIngestTestTenantID: ownerOrg},
		publisher,
	)

	payload := buildUplinkPayload()
	_, err := svc.Ingest(testutil.TestContextWithTenant(uplinkIngestTestTenantID), payload, bssci.UplinkIngestOptions{Source: bssci.UplinkSourceBSSCI})
	require.NoError(t, err)

	// Ingest publishes MQTT inside a goroutine — wait on the channel.
	select {
	case call := <-publisher.uplinkCalls:
		assert.Equal(t, ownerOrg.String(), call.OrgUUID, "publish must carry resolved owner-org UUID")
		assert.Equal(t, payload.EpEUI, call.EpEUI)
		assert.Equal(t, payload.BsEUI, call.BsEUI)
		assert.Equal(t, payload.RSSI, call.RSSI)
		assert.Equal(t, payload.SNR, call.SNR)
		assert.Equal(t, payload.RxTime, call.RxTime)
		assert.Equal(t, payload.PacketCnt, call.PacketCnt)
		assert.Equal(t, payload.UserData, call.UserData)
	case <-time.After(2 * time.Second):
		t.Fatal("expected PublishUplink call within timeout")
	}
}

// TestUplinkIngest_OrgUnresolved_SkipsPublish verifies no MQTT publish happens
// when the org resolver yields uuid.Nil for the owner tenant.
func TestUplinkIngest_OrgUnresolved_SkipsPublish(t *testing.T) {
	t.Parallel()

	publisher := newCapturingMQTTPublisher()
	// Empty org mapping → GetDefaultOrgForTenant returns uuid.Nil.
	svc := newUplinkIngestForTest(t, map[int64]uuid.UUID{}, publisher)

	payload := buildUplinkPayload()
	_, err := svc.Ingest(testutil.TestContextWithTenant(uplinkIngestTestTenantID), payload, bssci.UplinkIngestOptions{Source: bssci.UplinkSourceBSSCI})
	require.NoError(t, err)

	select {
	case call := <-publisher.uplinkCalls:
		t.Fatalf("unexpected PublishUplink call: %+v", call)
	case <-time.After(200 * time.Millisecond):
		// good — no publish
	}
}
