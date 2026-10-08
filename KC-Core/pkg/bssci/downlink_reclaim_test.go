package bssci

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// reclaimCall is one ReclaimReservations invocation.
type reclaimCall struct {
	bsEUI    uint64
	reissued []int64
}

// recordingReclaimer records the reservation and queue reclaims the server requests.
type recordingReclaimer struct {
	noopDownlinkReclaimer
	mu              sync.Mutex
	reclaims        []reclaimCall
	discardedQueues []uint64
}

func (d *recordingReclaimer) ReclaimDiscardedQueue(_ context.Context, bsEUI uint64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.discardedQueues = append(d.discardedQueues, bsEUI)
	return 1, nil
}

func (d *recordingReclaimer) discardedQueueCalls() []uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]uint64(nil), d.discardedQueues...)
}

func (d *recordingReclaimer) ReclaimReservations(_ context.Context, bsEUI uint64, reissued []int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reclaims = append(d.reclaims, reclaimCall{bsEUI: bsEUI, reissued: reissued})
	return int64(len(reissued)), nil
}

func (d *recordingReclaimer) calls() []reclaimCall {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]reclaimCall(nil), d.reclaims...)
}

// dlDataQueReissueOp is a persisted dlDataQue operation as resume loads it:
// its queue id comes back as a float64 from the strict decode.
func dlDataQueReissueOp(opID int64, queID int64) *PendingOperation {
	return &PendingOperation{
		OperationID:   opID,
		OperationType: mioty.CmdDLDataQueue,
		Message: map[string]interface{}{
			"command":   mioty.CmdDLDataQueue,
			"opId":      float64(opID),
			"epEui":     float64(TestEpEui01),
			"queId":     float64(queID),
			"userData":  []interface{}{[]interface{}{float64(1)}},
			"prio":      float64(0),
			"cntDepend": false,
		},
		Metadata: map[string]interface{}{
			models.EventDetailKeyQueID: float64(queID),
			"tenantID":                 "1",
		},
	}
}

// TestConnectComplete_FreshSessionReclaimsStationReservations: a session that
// is not resumed discards the previous session's state (BSSCI §1), so every
// downlink the base station held reserved returns to pending.
func TestConnectComplete_FreshSessionReclaimsStationReservations(t *testing.T) {
	server := newResumeReissueServer(t)
	reclaimer := &recordingReclaimer{}
	server.downlinkReclaimer = reclaimer
	session := newActivationSession("fresh-session", &countingConn{})
	t.Cleanup(func() { stopSessionStatus(session) })

	require.NoError(t, server.handleConnectComplete(session, &Message{Command: mioty.CmdConnectComplete}, nil))

	assert.Equal(t, []reclaimCall{{bsEUI: TestBsEui01}}, reclaimer.calls())
	assert.Equal(t, []uint64{TestBsEui01}, reclaimer.discardedQueueCalls(),
		"the downlinks the station held queued are gone with its previous session and return to pending")
}

// TestConnectComplete_ResumeKeepsReissuedReservations: on resume only the
// rows behind the reissued dlDataQue operations stay reserved - a dlDataQueRsp
// can still confirm them - and every other reservation is reclaimed.
func TestConnectComplete_ResumeKeepsReissuedReservations(t *testing.T) {
	server := newResumeReissueServer(t)
	reclaimer := &recordingReclaimer{}
	server.downlinkReclaimer = reclaimer
	session := newResumeSession(&countingConn{}, []*PendingOperation{
		statusReissueOp(-1),
		dlDataQueReissueOp(-2, 777),
	})
	t.Cleanup(func() { stopSessionStatus(session) })

	require.NoError(t, server.handleConnectComplete(session, &Message{Command: mioty.CmdConnectComplete}, nil))

	assert.Equal(t, []reclaimCall{{bsEUI: TestBsEui01, reissued: []int64{777}}}, reclaimer.calls())
	assert.Empty(t, reclaimer.discardedQueueCalls(), "a resumed station still holds its queued downlinks")
}
