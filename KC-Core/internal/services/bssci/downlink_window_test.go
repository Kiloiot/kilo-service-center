package bssciservices

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// dispatchTestMessageID names the telegram whose window the dispatcher tests fill.
const dispatchTestMessageID = "5f7c0c8e-2b1f-4f5a-9f60-6f0c1d2e3a4b"

// windowClaims is the in-memory claim table of telegram downlink windows.
type windowClaims struct {
	mu       sync.Mutex
	claimed  map[string]bool
	releases []string
}

func (w *windowClaims) ClaimDownlinkWindow(_ context.Context, _ int64, messageID string) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.claimed == nil {
		w.claimed = map[string]bool{}
	}
	if w.claimed[messageID] {
		return false, nil
	}
	w.claimed[messageID] = true
	return true, nil
}

func (w *windowClaims) ReleaseDownlinkWindow(_ context.Context, _ int64, messageID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.releases = append(w.releases, messageID)
	delete(w.claimed, messageID)
	return nil
}

func windowDispatcher(t *testing.T, reserver DownlinkReserver, queue DownlinkConfirmer, windows *windowClaims, send SendDLQueueFunc) bssci.DownlinkDispatcher {
	t.Helper()
	dispatcher, err := NewDownlinkDispatcher(&mockLoggerForDispatch{}, reserver, queue, windows, send, testutil.NewFakeClock(dispatchTestNow))
	require.NoError(t, err)
	return dispatcher
}

func stationSession(bsEUI uint64, bidirectional bool) *bssci.Session {
	session := testDispatchSession()
	session.BaseStationEUI = bsEUI
	session.Bidirectional = bidirectional
	return session
}

// TestDispatchIfAvailable_FillsTheWindowOnceThroughTheFirstBidirectionalStation
// pins radio §3.6.1 for a telegram heard by several stations: a receive-only
// first reception leaves the window to the next bidirectional one, which
// dispatches, and a third reception finds the window taken.
func TestDispatchIfAvailable_FillsTheWindowOnceThroughTheFirstBidirectionalStation(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{QueID: 12400, Payload: []byte{0x01}})
	windows := &windowClaims{}
	dispatcher := windowDispatcher(t, reserver, dlRepo, windows, sendFn.Send)
	ctx := testutil.TestContext()

	receiveOnly, err := dispatcher.DispatchIfAvailable(ctx, 42, stationSession(0x70B3D59CD0000011, false), 0xAABBCCDDEEFF0011, dispatchTestMessageID, true)
	require.NoError(t, err)
	first, err := dispatcher.DispatchIfAvailable(ctx, 42, stationSession(0x70B3D59CD0000012, true), 0xAABBCCDDEEFF0011, dispatchTestMessageID, true)
	require.NoError(t, err)
	second, err := dispatcher.DispatchIfAvailable(ctx, 42, stationSession(0x70B3D59CD0000013, true), 0xAABBCCDDEEFF0011, dispatchTestMessageID, true)
	require.NoError(t, err)

	assert.False(t, receiveOnly)
	assert.True(t, first)
	assert.False(t, second)
	assert.Equal(t, 1, sendFn.calls, "one dlDataQue per window")
	assert.Equal(t, 1, reserver.reserveNextCalls, "a claimed window reserves nothing more")
}

// TestDispatchIfAvailable_UnusedWindowIsGivenBack: a reception that finds
// nothing to send, or cannot send it, leaves the window to the next one.
func TestDispatchIfAvailable_UnusedWindowIsGivenBack(t *testing.T) {
	cases := map[string]struct {
		row     *storage.DownlinkMessage
		sendErr error
	}{
		"nothing pending":  {},
		"definite failure": {row: &storage.DownlinkMessage{QueID: 12401, Payload: []byte{0x01}}, sendErr: errTestSendFailed},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dlRepo, reserver, sendFn := newDispatchFixture(tc.row)
			sendFn.err = tc.sendErr
			windows := &windowClaims{}
			dispatcher := windowDispatcher(t, reserver, dlRepo, windows, sendFn.Send)

			dispatched, _ := dispatcher.DispatchIfAvailable(testutil.TestContext(), 42, stationSession(0x70B3D59CD0000012, true), 0xAABBCCDDEEFF0011, dispatchTestMessageID, true)

			assert.False(t, dispatched)
			assert.Equal(t, []string{dispatchTestMessageID}, windows.releases)
		})
	}
}

// TestDispatchIfAvailable_AmbiguousSendKeepsTheWindow: a frame that may be on
// the wire keeps its window, or a second downlink could be queued into it.
func TestDispatchIfAvailable_AmbiguousSendKeepsTheWindow(t *testing.T) {
	dlRepo, reserver, sendFn := newDispatchFixture(&storage.DownlinkMessage{QueID: 12402, Payload: []byte{0x01}})
	sendFn.err = errTestWritePayloadAmbiguous
	windows := &windowClaims{}
	dispatcher := windowDispatcher(t, reserver, dlRepo, windows, sendFn.Send)

	_, err := dispatcher.DispatchIfAvailable(testutil.TestContext(), 42, stationSession(0x70B3D59CD0000012, true), 0xAABBCCDDEEFF0011, dispatchTestMessageID, true)

	require.ErrorIs(t, err, bssci.ErrAmbiguousWrite)
	assert.Empty(t, windows.releases)
}
