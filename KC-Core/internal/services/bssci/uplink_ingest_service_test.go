package bssciservices

import (
	"context"
	"crypto/x509"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUplinkDedupWindow     = 5 * time.Minute
	testUplinkReceptionWindow = 500 * time.Millisecond

	uplinkIngestTestTenantID       = int64(42)
	uplinkIngestTestEpEUI          = uint64(0x12345678)
	uplinkIngestTestBsEUI          = uint64(0xABCDEF1234567890)
	uplinkIngestTestSyntheticEUI   = uint64(0xFEED000000000001)
	uplinkIngestTestDuplicateCount = 1
	uplinkIngestTestOpID           = int64(4242)
	uplinkIngestTestOwnerTenantID  = int64(77)
)

var errUplinkIngestTestStore = errors.New("store unavailable")

var testUplinkWindows = UplinkWindows{Duplicate: testUplinkDedupWindow, Reception: testUplinkReceptionWindow}

// fakeUplinkStore records every persist request and answers with a scripted
// outcome, so the tests observe exactly what the service hands the classifier.
type fakeUplinkStore struct {
	mu       sync.Mutex
	requests []models.UplinkPersistRequest
	outcome  models.UplinkPersistOutcome
	err      error
}

func (s *fakeUplinkStore) Persist(_ context.Context, req models.UplinkPersistRequest) (models.UplinkPersistOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	if s.err != nil {
		return models.UplinkPersistOutcome{}, s.err
	}
	out := s.outcome
	if out.MessageID == "" {
		out.MessageID = req.Message.ID
	}
	return out, nil
}

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
func (r *uplinkIngestEndpointRepo) EndpointRegistrationUpdate(context.Context, int64, int64, models.EndpointRegistrationParams) error {
	return nil
}

func (r *uplinkIngestEndpointRepo) EndpointAttachmentStateUpdate(context.Context, int64, int64, models.EndpointAttachmentStateParams) error {
	return nil
}

func (r *uplinkIngestEndpointRepo) EndpointAttachSessionUpdate(context.Context, int64, int64, models.EndpointAttachSessionParams) error {
	return nil
}

func (r *uplinkIngestEndpointRepo) EndpointDetachStateUpdate(context.Context, int64, int64, models.EndpointDetachStateParams) error {
	return nil
}

func (r *uplinkIngestEndpointRepo) UpdateLastSeen(context.Context, int64, models.EUI, uint32) error {
	return nil
}

func (r *uplinkIngestEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, models.RadioMetricsUpdate) error {
	return nil
}

func (r *uplinkIngestEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil
}

func (r *uplinkIngestEndpointRepo) DeleteByTenant(context.Context, int64, []byte) (int64, error) {
	return 0, nil
}

func (r *uplinkIngestEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (r *uplinkIngestEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error { return nil }

type recordedEndpointAck struct {
	ownerTenantID int64
	epEUI         uint64
	packetCnt     uint32
}

// recordingDownlinkAcks records every endpoint acknowledgement handed on.
type recordingDownlinkAcks struct {
	mu   sync.Mutex
	acks []recordedEndpointAck
}

func (r *recordingDownlinkAcks) RecordEndpointAck(_ context.Context, ownerTenantID int64, epEUI uint64, packetCnt uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.acks = append(r.acks, recordedEndpointAck{ownerTenantID: ownerTenantID, epEUI: epEUI, packetCnt: packetCnt})
	return nil
}

type ingestFixture struct {
	store     *fakeUplinkStore
	endpoints *uplinkIngestEndpointRepo
	acks      *recordingDownlinkAcks
	svc       *UplinkIngestServiceImpl
}

func newIngestFixture(t *testing.T, orgMapping map[int64]uuid.UUID, channels []models.DeliveryChannel, syntheticBsEUI uint64) *ingestFixture {
	t.Helper()
	endpointRepo := &uplinkIngestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			uplinkIngestTestEpEUI: {ID: 1, TenantID: uplinkIngestTestTenantID, OwnerTenantID: uplinkIngestTestTenantID},
		},
	}
	store := &fakeUplinkStore{}
	acks := &recordingDownlinkAcks{}
	endpointOwners, err := NewEndpointOwnerResolver(endpointRepo)
	require.NoError(t, err)
	svc, err := NewUplinkIngestService(
		store,
		testUplinkWindows,
		channels,
		nil, // no DL RX metrics in these scenarios
		&fakeOrgResolver{defaultOrgByTenant: orgMapping},
		nil, // roamingSvc nil takes the direct endpoint lookup path
		endpointRepo,
		endpointOwners,
		nil, // blueprintResolver
		nil, // blueprintDecoder
		acks,
		logger.NewNop(),
		uplinkIngestTestTenantID,
		syntheticBsEUI,
	)
	require.NoError(t, err)
	return &ingestFixture{store: store, endpoints: endpointRepo, acks: acks, svc: svc}
}

func allChannels() []models.DeliveryChannel {
	return []models.DeliveryChannel{models.DeliveryChannelSCACI, models.DeliveryChannelMQTT}
}

func buildUplinkPayload() *bssci.UplinkPayload {
	return &bssci.UplinkPayload{
		OpID:      uplinkIngestTestOpID,
		EpEUI:     uplinkIngestTestEpEUI,
		BsEUI:     uplinkIngestTestBsEUI,
		PacketCnt: 42,
		UserData:  []byte{0x01, 0x02, 0x03},
		SNR:       12.3,
		RSSI:      -85.5,
		RxTime:    time.Now().UnixNano(),
	}
}

func ingestBSSCI(t *testing.T, f *ingestFixture, payload *bssci.UplinkPayload) (*bssci.IngestResult, error) {
	t.Helper()
	return f.svc.Ingest(testutil.TestContextWithTenant(uplinkIngestTestTenantID), payload,
		bssci.UplinkIngestOptions{Source: bssci.UplinkSourceBSSCI})
}

func TestUplinkIngest_FirstReceptionQueuesEveryChannel(t *testing.T) {
	t.Parallel()
	ownerOrg := uuid.New()
	f := newIngestFixture(t, map[int64]uuid.UUID{uplinkIngestTestTenantID: ownerOrg}, allChannels(), 0)
	payload := buildUplinkPayload()

	res, err := ingestBSSCI(t, f, payload)
	require.NoError(t, err)

	require.Len(t, f.store.requests, 1)
	req := f.store.requests[0]
	assert.Equal(t, testUplinkDedupWindow, req.Window)
	assert.Equal(t, testUplinkReceptionWindow, req.ReceptionWindow, "the delivery waits for the other stations' receptions")
	assert.Equal(t, allChannels(), req.Channels)
	msg := req.Message
	require.NotNil(t, msg.OrgUUID)
	assert.Equal(t, ownerOrg.String(), *msg.OrgUUID)
	assert.Equal(t, uplinkIngestTestTenantID, msg.TenantID)
	assert.Equal(t, payload.EpEUI, msg.EpEui)
	assert.Equal(t, payload.BsEUI, msg.BsEui)
	assert.Equal(t, payload.PacketCnt, msg.PacketCnt)
	assert.Equal(t, payload.UserData, msg.UserData)
	require.Len(t, msg.BaseStations, 1)
	assert.Equal(t, payload.BsEUI, msg.BaseStations[0].BsEui)
	assert.Equal(t, payload.RSSI, msg.BaseStations[0].Rssi)
	assert.Equal(t, payload.SNR, msg.BaseStations[0].Snr)
	assert.NotEmpty(t, msg.ID)

	assert.False(t, res.IsDuplicate)
	assert.Equal(t, msg.ID, res.MessageID)
	assert.Equal(t, uplinkIngestTestTenantID, res.OwnerTenantID)
	assert.Equal(t, ownerOrg, res.OwnerOrgUUID)
}

func TestUplinkIngest_OrgUnresolvedDropsMQTTChannel(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)

	res, err := ingestBSSCI(t, f, buildUplinkPayload())
	require.NoError(t, err)

	require.Len(t, f.store.requests, 1)
	assert.Equal(t, []models.DeliveryChannel{models.DeliveryChannelSCACI}, f.store.requests[0].Channels,
		"MQTT topics are organization-scoped, so an unresolved org must not queue MQTT")
	assert.Nil(t, f.store.requests[0].Message.OrgUUID)
	assert.Equal(t, uuid.Nil, res.OwnerOrgUUID)
}

// TestUplinkIngest_DuplicateReportsTheTelegramsMessage: a merged reception
// creates no message of its own and names the telegram's first message, whose
// downlink window it shares.
func TestUplinkIngest_DuplicateReportsTheTelegramsMessage(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)
	first := uuid.New().String()
	f.store.outcome = models.UplinkPersistOutcome{
		Classification: models.UplinkDuplicate,
		MessageID:      first,
		DuplicateCount: uplinkIngestTestDuplicateCount,
	}

	res, err := ingestBSSCI(t, f, buildUplinkPayload())
	require.NoError(t, err)

	assert.True(t, res.IsDuplicate)
	assert.Equal(t, first, res.MessageID)
	assert.NotEqual(t, f.store.requests[0].Message.ID, res.MessageID, "the merged reception's own id was never stored")
	assert.Equal(t, uplinkIngestTestTenantID, res.OwnerTenantID)
}

func TestUplinkIngest_ClassifierRefusalsMapToCatalogErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		cause error
		token string
		posix int
	}{
		{"collision", storage.ErrPacketCounterCollision, bssci.ErrTokenPacketCounterCollision, bssci.POSIX_EEXIST},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)
			f.store.err = tc.cause

			res, err := ingestBSSCI(t, f, buildUplinkPayload())

			require.Error(t, err)
			assert.Nil(t, res)
			var catalogErr *bssci.CatalogError
			require.ErrorAs(t, err, &catalogErr, "the base station is answered from the catalog")
			assert.Equal(t, tc.token, catalogErr.Token)
			assert.Equal(t, tc.posix, catalogErr.Posix)
			assert.ErrorIs(t, err, tc.cause, "the storage cause stays matchable")
		})
	}
}

func TestUplinkIngest_PersistFailureIsWrapped(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)
	f.store.err = errUplinkIngestTestStore

	res, err := ingestBSSCI(t, f, buildUplinkPayload())

	require.Error(t, err)
	assert.Nil(t, res)
	assert.ErrorIs(t, err, errPersistUplinkFailed)
	assert.ErrorIs(t, err, errUplinkIngestTestStore)
	var catalogErr *bssci.CatalogError
	assert.NotErrorAs(t, err, &catalogErr)
}

func TestUplinkIngest_OwnerIsTheEndpointOwner(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)
	f.endpoints.endpoints[uplinkIngestTestEpEUI].OwnerTenantID = uplinkIngestTestOwnerTenantID

	res, err := ingestBSSCI(t, f, buildUplinkPayload())
	require.NoError(t, err)

	require.Len(t, f.store.requests, 1)
	assert.Equal(t, uplinkIngestTestOwnerTenantID, f.store.requests[0].Message.TenantID,
		"without roaming the owner comes from the same column the roaming resolver reads")
	assert.Equal(t, uplinkIngestTestOwnerTenantID, res.OwnerTenantID)
}

func TestUplinkIngest_CarriesTheFrameOpID(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)

	_, err := ingestBSSCI(t, f, buildUplinkPayload())
	require.NoError(t, err)

	require.Len(t, f.store.requests, 1)
	assert.Equal(t, uplinkIngestTestOpID, f.store.requests[0].Message.OpId, "the stored message keeps the base station's ulData opId")
}

func TestUplinkIngest_UnknownEndpointNeverReachesStore(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)
	payload := buildUplinkPayload()
	payload.EpEUI = uplinkIngestTestEpEUI + 1

	res, err := ingestBSSCI(t, f, payload)

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Empty(t, f.store.requests)
}

func TestUplinkIngest_FederationUsesSyntheticBaseStation(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), uplinkIngestTestSyntheticEUI)
	payload := buildUplinkPayload()

	_, err := f.svc.Ingest(testutil.TestContextWithTenant(uplinkIngestTestTenantID), payload,
		bssci.UplinkIngestOptions{Source: bssci.UplinkSourceFederation})
	require.NoError(t, err)

	require.Len(t, f.store.requests, 1)
	msg := f.store.requests[0].Message
	assert.Equal(t, uplinkIngestTestSyntheticEUI, msg.BsEui)
	require.Len(t, msg.BaseStations, 1)
	assert.Equal(t, uplinkIngestTestSyntheticEUI, msg.BaseStations[0].BsEui,
		"tenant-visible records must never reference the relaying CE base station")
}

// TestUplinkIngest_RecordsTheEndpointAckOfEveryPath pins BSSCI §3.10.1: an
// uplink with dlAck acknowledges its owner's downlink of the previous window
// whichever path delivered it, a federation relay included.
func TestUplinkIngest_RecordsTheEndpointAckOfEveryPath(t *testing.T) {
	t.Parallel()
	for _, source := range []bssci.UplinkSource{bssci.UplinkSourceBSSCI, bssci.UplinkSourceFederation} {
		f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), uplinkIngestTestSyntheticEUI)
		payload := buildUplinkPayload()
		payload.DlAck = true

		_, err := f.svc.Ingest(testutil.TestContextWithTenant(uplinkIngestTestTenantID), payload,
			bssci.UplinkIngestOptions{Source: source})
		require.NoError(t, err)

		assert.Equal(t, []recordedEndpointAck{{ownerTenantID: uplinkIngestTestTenantID, epEUI: payload.EpEUI, packetCnt: payload.PacketCnt}},
			f.acks.acks, "an ingest with source %d records the endpoint acknowledgement", source)
	}
}

func TestUplinkIngest_WithoutDlAckRecordsNoAcknowledgement(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)

	_, err := ingestBSSCI(t, f, buildUplinkPayload())
	require.NoError(t, err)

	assert.Empty(t, f.acks.acks)
}

// TestUplinkIngest_AFailedPersistRecordsNoAcknowledgement: an uplink the
// store refused was never received, so it acknowledges nothing.
func TestUplinkIngest_AFailedPersistRecordsNoAcknowledgement(t *testing.T) {
	t.Parallel()
	f := newIngestFixture(t, map[int64]uuid.UUID{}, allChannels(), 0)
	f.store.err = errUplinkIngestTestStore
	payload := buildUplinkPayload()
	payload.DlAck = true

	_, err := ingestBSSCI(t, f, payload)
	require.Error(t, err)

	assert.Empty(t, f.acks.acks)
}

func TestNewUplinkIngestService_RequiresTheAckRecorder(t *testing.T) {
	endpoints := &uplinkIngestEndpointRepo{}
	endpointOwners, err := NewEndpointOwnerResolver(endpoints)
	require.NoError(t, err)
	svc, err := NewUplinkIngestService(&fakeUplinkStore{}, testUplinkWindows, allChannels(), nil, nil, nil,
		endpoints, endpointOwners, nil, nil, nil, logger.NewNop(), uplinkIngestTestTenantID, 0)
	require.ErrorIs(t, err, ErrNilDownlinkAckRecorder)
	assert.Nil(t, svc)
}
