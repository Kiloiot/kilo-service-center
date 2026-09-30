package bssciservices

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	ackRecorderEndpointEUI uint64 = 0x70B3D59CD0000341
	ackRecorderQueueID     int64  = 840001
)

var errAckStoreDown = errors.New("downlink queue down")

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

func (s *recordingAckStore) MarkEndpointAcknowledged(_ context.Context, tenantID int64, epEUI uint64, window int64) (int64, bool, error) {
	s.acks = append(s.acks, endpointAck{tenantID: tenantID, epEUI: epEUI, window: window})
	if !s.marked {
		return 0, false, s.err
	}
	return ackRecorderQueueID, true, s.err
}

// acknowledgedDownlink is one acknowledgement event the recorder filed.
type acknowledgedDownlink struct {
	tenantID  int64
	epEUI     uint64
	queueID   int64
	packetCnt uint32
}

// recordingAckEvents records the acknowledgement events filed.
type recordingAckEvents struct {
	events []acknowledgedDownlink
}

func (e *recordingAckEvents) RecordDownlinkAcknowledged(_ context.Context, tenantID int64, epEUI uint64, queueID int64, packetCnt uint32) error {
	e.events = append(e.events, acknowledgedDownlink{tenantID, epEUI, queueID, packetCnt})
	return nil
}

func newAckRecorder(t *testing.T, store EndpointAckStore, events DownlinkAckEvents) *EndpointAckRecorder {
	t.Helper()
	recorder, err := NewEndpointAckRecorder(store, events, logger.NewNop())
	require.NoError(t, err)
	return recorder
}

// TestRecordEndpointAck_AcknowledgesThePreviousWindow pins BSSCI §3.10.1:
// dlAck on the uplink with packet counter N acknowledges the downlink the
// owner tenant transmitted in the window of N - 1, and the owner sees an event.
func TestRecordEndpointAck_AcknowledgesThePreviousWindow(t *testing.T) {
	store := &recordingAckStore{marked: true}
	events := &recordingAckEvents{}

	require.NoError(t, newAckRecorder(t, store, events).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, 42))

	assert.Equal(t, []endpointAck{{tenantID: 3, epEUI: ackRecorderEndpointEUI, window: 41}}, store.acks)
	assert.Equal(t, []acknowledgedDownlink{{tenantID: 3, epEUI: ackRecorderEndpointEUI, queueID: ackRecorderQueueID, packetCnt: 41}}, events.events)
}

// TestRecordEndpointAck_CounterZeroHasNoPreviousWindow: the first uplink
// after an over-the-air attach cannot acknowledge an earlier window.
func TestRecordEndpointAck_CounterZeroHasNoPreviousWindow(t *testing.T) {
	store := &recordingAckStore{}
	events := &recordingAckEvents{}

	require.NoError(t, newAckRecorder(t, store, events).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, 0))

	assert.Empty(t, store.acks)
	assert.Empty(t, events.events)
}

func TestRecordEndpointAck_WithoutAMatchingDownlinkRecordsNoEvent(t *testing.T) {
	store := &recordingAckStore{marked: false}
	events := &recordingAckEvents{}

	require.NoError(t, newAckRecorder(t, store, events).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, 7))

	assert.Empty(t, events.events, "a repeated or unmatched dlAck is not an acknowledgement")
}

func TestRecordEndpointAck_WrapsAStoreFailure(t *testing.T) {
	store := &recordingAckStore{err: errAckStoreDown}

	err := newAckRecorder(t, store, &recordingAckEvents{}).RecordEndpointAck(testutil.TestContext(), 3, ackRecorderEndpointEUI, 7)

	require.ErrorIs(t, err, errMarkEndpointAck)
	require.ErrorIs(t, err, errAckStoreDown)
}

func TestNewEndpointAckRecorder_RejectsMissingCollaborators(t *testing.T) {
	_, err := NewEndpointAckRecorder(nil, &recordingAckEvents{}, logger.NewNop())
	require.ErrorIs(t, err, ErrNilEndpointAckStore)
	_, err = NewEndpointAckRecorder(&recordingAckStore{}, nil, logger.NewNop())
	require.ErrorIs(t, err, ErrNilEndpointAckEvents)
	_, err = NewEndpointAckRecorder(&recordingAckStore{}, &recordingAckEvents{}, nil)
	require.ErrorIs(t, err, ErrNilEndpointAckLogger)
}
