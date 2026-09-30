package bssciservices

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	ackRecorderEndpointEUI uint64 = 0x70B3D59CD0000341
	ackRecorderQueueID     int64  = 840001
)

var (
	errAckStoreDown     = errors.New("downlink queue down")
	errAckReporterDown  = errors.New("reporter down")
	ackRecorderDownlink = &storage.DownlinkMessage{QueID: ackRecorderQueueID, EPEUI: mioty.FormatEUI64(ackRecorderEndpointEUI), TenantID: "3"}
)

// endpointAck is one acknowledgement recorded on the queue.
type endpointAck struct {
	tenantID int64
	epEUI    uint64
	window   int64
}

// recordingAckStore records every acknowledgement and answers with one outcome.
type recordingAckStore struct {
	acks   []endpointAck
	marked bool
	err    error
}

func (s *recordingAckStore) MarkEndpointAcknowledged(_ context.Context, tenantID int64, epEUI uint64, window int64) (*storage.DownlinkMessage, bool, error) {
	s.acks = append(s.acks, endpointAck{tenantID: tenantID, epEUI: epEUI, window: window})
	if !s.marked {
		return nil, false, s.err
	}
	return ackRecorderDownlink, true, s.err
}

// reportedAck is one acknowledgement the recorder reported.
type reportedAck struct {
	downlink  *storage.DownlinkMessage
	packetCnt uint32
}

// recordingAckReporter records the acknowledgements reported.
type recordingAckReporter struct {
	reported []reportedAck
	err      error
}

func (r *recordingAckReporter) ReportEndpointAck(_ context.Context, downlink *storage.DownlinkMessage, packetCnt uint32) error {
	r.reported = append(r.reported, reportedAck{downlink, packetCnt})
	return r.err
}

func newAckRecorder(t *testing.T, store EndpointAckStore, reporter DownlinkAckReporter) *EndpointAckRecorder {
	t.Helper()
	recorder, err := NewEndpointAckRecorder(store, reporter, logger.NewNop())
	require.NoError(t, err)
	return recorder
}

// TestRecordEndpointAck_AcknowledgesThePreviousWindow pins BSSCI §3.10.1:
// dlAck on the uplink with packet counter N acknowledges the downlink the
// owner tenant transmitted in the window of N - 1, and its originators learn
// it once.
func TestRecordEndpointAck_AcknowledgesThePreviousWindow(t *testing.T) {
	store := &recordingAckStore{marked: true}
	reporter := &recordingAckReporter{}

	require.NoError(t, newAckRecorder(t, store, reporter).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, 42))

	assert.Equal(t, []endpointAck{{tenantID: 3, epEUI: ackRecorderEndpointEUI, window: 41}}, store.acks)
	assert.Equal(t, []reportedAck{{downlink: ackRecorderDownlink, packetCnt: 41}}, reporter.reported)
}

// TestRecordEndpointAck_ReportsNothingTheStoreDidNotMark: the queue's mark is
// the source of truth, so the first uplink after an over-the-air attach, a
// dlAck no transmission matches and a repeated or replayed dlAck report
// nothing.
func TestRecordEndpointAck_ReportsNothingTheStoreDidNotMark(t *testing.T) {
	for name, tc := range map[string]struct {
		packetCnt uint32
		store     *recordingAckStore
		wantAcks  int
	}{
		"counter zero has no previous window": {packetCnt: 0, store: &recordingAckStore{marked: true}, wantAcks: 0},
		"no unacknowledged transmission":      {packetCnt: 7, store: &recordingAckStore{}, wantAcks: 1},
		"store failure":                       {packetCnt: 7, store: &recordingAckStore{err: errAckStoreDown}, wantAcks: 1},
	} {
		t.Run(name, func(t *testing.T) {
			reporter := &recordingAckReporter{}

			err := newAckRecorder(t, tc.store, reporter).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, tc.packetCnt)

			if tc.store.err != nil {
				require.ErrorIs(t, err, errMarkEndpointAck)
				require.ErrorIs(t, err, errAckStoreDown)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, tc.store.acks, tc.wantAcks)
			assert.Empty(t, reporter.reported)
		})
	}
}

func TestRecordEndpointAck_WrapsAReporterFailure(t *testing.T) {
	reporter := &recordingAckReporter{err: errAckReporterDown}

	err := newAckRecorder(t, &recordingAckStore{marked: true}, reporter).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, 7)

	require.ErrorIs(t, err, errReportDownlinkAck)
	require.ErrorIs(t, err, errAckReporterDown)
}

func TestNewEndpointAckRecorder_RejectsMissingCollaborators(t *testing.T) {
	_, err := NewEndpointAckRecorder(nil, &recordingAckReporter{}, logger.NewNop())
	require.ErrorIs(t, err, ErrNilEndpointAckStore)
	_, err = NewEndpointAckRecorder(&recordingAckStore{}, nil, logger.NewNop())
	require.ErrorIs(t, err, ErrNilEndpointAckReporter)
	_, err = NewEndpointAckRecorder(&recordingAckStore{}, &recordingAckReporter{}, nil)
	require.ErrorIs(t, err, ErrNilEndpointAckLogger)
}

// tenantAckStore holds one transmitted downlink per tenant and marks it once,
// as the queue's update does.
type tenantAckStore struct {
	mu     sync.Mutex
	unread map[int64]*storage.DownlinkMessage
}

func (s *tenantAckStore) MarkEndpointAcknowledged(_ context.Context, tenantID int64, _ uint64, _ int64) (*storage.DownlinkMessage, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	downlink, ok := s.unread[tenantID]
	delete(s.unread, tenantID)
	return downlink, ok, nil
}

// TestEndpointAck_RoamingUplinkReachesTheOwnersOrganizationOnce: an uplink
// heard by another tenant's base station acknowledges the owner's downlink,
// on the topic of the organization that queued it and never the station
// owner's, and a repeated reception of it reports nothing more.
func TestEndpointAck_RoamingUplinkReachesTheOwnersOrganizationOnce(t *testing.T) {
	ownerOrg, stationOrg := uuid.New(), uuid.New()
	store := &tenantAckStore{unread: map[int64]*storage.DownlinkMessage{
		uplinkIngestTestOwnerTenantID: {QueID: ackRecorderQueueID, EPEUI: mioty.FormatEUI64(uplinkIngestTestEpEUI),
			TenantID: strconv.FormatInt(uplinkIngestTestOwnerTenantID, 10), OrganizationID: &ownerOrg, Ref: reporterCommandRef},
		uplinkIngestTestTenantID: {QueID: ackRecorderQueueID + 1, EPEUI: mioty.FormatEUI64(uplinkIngestTestEpEUI),
			TenantID: strconv.FormatInt(uplinkIngestTestTenantID, 10), OrganizationID: &stationOrg},
	}}
	results := newReporterFixture(t)
	endpoints := &uplinkIngestEndpointRepo{endpoints: map[uint64]*models.EndPoint{
		uplinkIngestTestEpEUI: {ID: 1, TenantID: uplinkIngestTestOwnerTenantID, OwnerTenantID: uplinkIngestTestOwnerTenantID},
	}}
	owners, err := NewEndpointOwnerResolver(endpoints)
	require.NoError(t, err)
	ingest, err := NewUplinkIngestService(&fakeUplinkStore{}, testUplinkWindows, allChannels(), nil, nil, nil,
		endpoints, owners, nil, nil, newAckRecorder(t, store, results.reporter), logger.NewNop(), uplinkIngestTestTenantID, 0)
	require.NoError(t, err)
	payload := buildUplinkPayload()
	payload.DlAck = true

	for range 2 {
		_, err := ingest.Ingest(testutil.TestContextWithTenant(uplinkIngestTestTenantID), payload,
			bssci.UplinkIngestOptions{Source: bssci.UplinkSourceBSSCI, ServingTenantID: uplinkIngestTestTenantID})
		require.NoError(t, err)
	}
	results.stop(t)

	assert.Equal(t, []publishedAck{{org: ownerOrg.String(), ref: reporterCommandRef, epEUI: uplinkIngestTestEpEUI,
		queID: uint64(ackRecorderQueueID), packetCnt: payload.PacketCnt - 1}}, results.mqtt.acks)
	assert.Contains(t, store.unread, uplinkIngestTestTenantID, "the station owner's downlink is never touched")
}
