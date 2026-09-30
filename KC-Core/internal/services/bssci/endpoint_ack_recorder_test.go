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
	// ackRecorderMessageID is the stored uplink that carries the acknowledgement.
	ackRecorderMessageID = "7c4f9a52-3d2e-4b8a-9f61-0d5c2a1b8e40"
)

var (
	errAckStoreDown     = errors.New("downlink queue down")
	errAckEventsDown    = errors.New("event store down")
	ackRecorderDownlink = &storage.DownlinkMessage{QueID: ackRecorderQueueID, EPEUI: mioty.FormatEUI64(ackRecorderEndpointEUI), TenantID: "3"}
	ackRecorderChannels = []models.DeliveryChannel{models.DeliveryChannelMQTTDownlinkAck}
)

// recordingAckStore records every acknowledgement and answers with one outcome.
type recordingAckStore struct {
	acks   []models.EndpointAckRequest
	marked bool
	err    error
}

func (s *recordingAckStore) MarkEndpointAcknowledged(_ context.Context, ack models.EndpointAckRequest) (*storage.DownlinkMessage, bool, error) {
	s.acks = append(s.acks, ack)
	if !s.marked {
		return nil, false, s.err
	}
	return ackRecorderDownlink, true, s.err
}

// recordedAckEvent is one acknowledgement recorded in the owner tenant's events.
type recordedAckEvent struct {
	downlink  *storage.DownlinkMessage
	packetCnt uint32
}

// recordingAckEvents records the acknowledgement events.
type recordingAckEvents struct {
	mu       sync.Mutex
	recorded []recordedAckEvent
	err      error
}

func (e *recordingAckEvents) RecordDownlinkAcknowledged(_ context.Context, downlink *storage.DownlinkMessage, packetCnt uint32) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.recorded = append(e.recorded, recordedAckEvent{downlink, packetCnt})
	return e.err
}

func newAckRecorder(t *testing.T, store EndpointAckStore, events DownlinkAckEvents) *EndpointAckRecorder {
	t.Helper()
	recorder, err := NewEndpointAckRecorder(store, events, ackRecorderChannels, logger.NewNop())
	require.NoError(t, err)
	return recorder
}

// TestRecordEndpointAck_AcknowledgesThePreviousWindow pins BSSCI §3.10.1:
// dlAck on the uplink with packet counter N acknowledges the downlink the
// owner tenant transmitted in the window of N - 1; the mark queues its
// publication for the uplink that carried it, and the owner's events record
// it once.
func TestRecordEndpointAck_AcknowledgesThePreviousWindow(t *testing.T) {
	store := &recordingAckStore{marked: true}
	events := &recordingAckEvents{}

	require.NoError(t, newAckRecorder(t, store, events).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, 42, ackRecorderMessageID))

	assert.Equal(t, []models.EndpointAckRequest{{TenantID: 3, EpEUI: ackRecorderEndpointEUI, WindowPacketCnt: 41,
		MessageID: ackRecorderMessageID, Channels: ackRecorderChannels}}, store.acks)
	assert.Equal(t, []recordedAckEvent{{downlink: ackRecorderDownlink, packetCnt: 41}}, events.recorded)
}

// TestRecordEndpointAck_RecordsNothingTheStoreDidNotMark: the queue's mark is
// the source of truth, so the first uplink after an over-the-air attach, a
// dlAck no transmission matches and a repeated or replayed dlAck record no
// event, and a failed mark is reported.
func TestRecordEndpointAck_RecordsNothingTheStoreDidNotMark(t *testing.T) {
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
			events := &recordingAckEvents{}

			err := newAckRecorder(t, tc.store, events).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, tc.packetCnt, ackRecorderMessageID)

			if tc.store.err != nil {
				require.ErrorIs(t, err, errMarkEndpointAck)
				require.ErrorIs(t, err, errAckStoreDown)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, tc.store.acks, tc.wantAcks)
			assert.Empty(t, events.recorded)
		})
	}
}

// TestRecordEndpointAck_AnEventFailureKeepsTheAcknowledgement: the mark and
// its queued publication commit before the event is recorded, so a failed
// event neither fails the uplink nor withdraws the acknowledgement.
func TestRecordEndpointAck_AnEventFailureKeepsTheAcknowledgement(t *testing.T) {
	store := &recordingAckStore{marked: true}
	events := &recordingAckEvents{err: errAckEventsDown}

	require.NoError(t, newAckRecorder(t, store, events).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, 7, ackRecorderMessageID))

	assert.Len(t, store.acks, 1)
	assert.Len(t, events.recorded, 1)
}

func TestNewEndpointAckRecorder_RejectsMissingCollaborators(t *testing.T) {
	_, err := NewEndpointAckRecorder(nil, &recordingAckEvents{}, ackRecorderChannels, logger.NewNop())
	require.ErrorIs(t, err, ErrNilEndpointAckStore)
	_, err = NewEndpointAckRecorder(&recordingAckStore{}, nil, ackRecorderChannels, logger.NewNop())
	require.ErrorIs(t, err, ErrNilEndpointAckEvents)
	_, err = NewEndpointAckRecorder(&recordingAckStore{}, &recordingAckEvents{}, ackRecorderChannels, nil)
	require.ErrorIs(t, err, ErrNilEndpointAckLogger)
}

// tenantAckStore holds one transmitted downlink per tenant and marks it once,
// as the queue's update does, keeping the requests that marked one.
type tenantAckStore struct {
	mu     sync.Mutex
	unread map[int64]*storage.DownlinkMessage
	marks  []models.EndpointAckRequest
}

func (s *tenantAckStore) MarkEndpointAcknowledged(_ context.Context, ack models.EndpointAckRequest) (*storage.DownlinkMessage, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	downlink, ok := s.unread[ack.TenantID]
	delete(s.unread, ack.TenantID)
	if ok {
		s.marks = append(s.marks, ack)
	}
	return downlink, ok, nil
}

// TestEndpointAck_RoamingUplinkAcknowledgesTheOwnersDownlinkOnce: an uplink
// heard by another tenant's base station acknowledges the owner's downlink,
// never the station owner's, and queues its publication once; a repeated
// reception of it marks and records nothing more. The queued row names the
// downlink, whose organization the delivery worker publishes to.
func TestEndpointAck_RoamingUplinkAcknowledgesTheOwnersDownlinkOnce(t *testing.T) {
	ownerOrg, stationOrg := uuid.New(), uuid.New()
	ownersDownlink := &storage.DownlinkMessage{QueID: ackRecorderQueueID, EPEUI: mioty.FormatEUI64(uplinkIngestTestEpEUI),
		TenantID: strconv.FormatInt(uplinkIngestTestOwnerTenantID, 10), OrganizationID: &ownerOrg}
	store := &tenantAckStore{unread: map[int64]*storage.DownlinkMessage{
		uplinkIngestTestOwnerTenantID: ownersDownlink,
		uplinkIngestTestTenantID: {QueID: ackRecorderQueueID + 1, EPEUI: mioty.FormatEUI64(uplinkIngestTestEpEUI),
			TenantID: strconv.FormatInt(uplinkIngestTestTenantID, 10), OrganizationID: &stationOrg},
	}}
	events := &recordingAckEvents{}
	endpoints := &uplinkIngestEndpointRepo{endpoints: map[uint64]*models.EndPoint{
		uplinkIngestTestEpEUI: {ID: 1, TenantID: uplinkIngestTestOwnerTenantID, OwnerTenantID: uplinkIngestTestOwnerTenantID},
	}}
	owners, err := NewEndpointOwnerResolver(endpoints)
	require.NoError(t, err)
	uplinks := &fakeUplinkStore{}
	ingest, err := NewUplinkIngestService(uplinks, testUplinkWindows, allChannels(), nil, nil, nil,
		endpoints, owners, nil, nil, newAckRecorder(t, store, events), logger.NewNop(), uplinkIngestTestTenantID, 0)
	require.NoError(t, err)
	payload := buildUplinkPayload()
	payload.DlAck = true

	for range 2 {
		_, err := ingest.Ingest(testutil.TestContextWithTenant(uplinkIngestTestTenantID), payload,
			bssci.UplinkIngestOptions{Source: bssci.UplinkSourceBSSCI, ServingTenantID: uplinkIngestTestTenantID})
		require.NoError(t, err)
	}

	require.Len(t, uplinks.requests, 2)
	assert.Equal(t, []models.EndpointAckRequest{{TenantID: uplinkIngestTestOwnerTenantID, EpEUI: uplinkIngestTestEpEUI,
		WindowPacketCnt: int64(payload.PacketCnt - 1), MessageID: uplinks.requests[0].Message.ID, Channels: ackRecorderChannels}},
		store.marks)
	assert.Equal(t, []recordedAckEvent{{downlink: ownersDownlink, packetCnt: payload.PacketCnt - 1}}, events.recorded)
	assert.Contains(t, store.unread, uplinkIngestTestTenantID, "the station owner's downlink is never touched")
}
