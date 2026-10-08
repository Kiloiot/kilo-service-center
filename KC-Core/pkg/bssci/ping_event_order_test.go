package bssci

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// gatedConn holds its first write until released, as a slow peer does.
type gatedConn struct {
	*bsscitest.TestConn
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func newGatedConn(conn *bsscitest.TestConn) *gatedConn {
	return &gatedConn{TestConn: conn, entered: make(chan struct{}), release: make(chan struct{})}
}

func (g *gatedConn) Write(b []byte) (int, error) {
	g.once.Do(func() {
		close(g.entered)
		<-g.release
	})
	return g.TestConn.Write(b)
}

// orderedStationEvents is the event store behind the production recorder chain.
type orderedStationEvents struct {
	mu       sync.Mutex
	events   []*models.SystemEvent
	answered chan struct{}
}

func (o *orderedStationEvents) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
	if event.EventType == models.EventTypeBaseStationPingAnswered {
		close(o.answered)
	}
	return nil
}

func (o *orderedStationEvents) occurredAt(t *testing.T, eventType string) models.SystemEvent {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, event := range o.events {
		if event.EventType == eventType {
			return *event
		}
	}
	require.FailNow(t, "event not recorded", eventType)
	return models.SystemEvent{}
}

type pingStationDirectory struct{}

func (pingStationDirectory) GetBaseStation(context.Context, [8]byte) (*basestation.BaseStation, error) {
	return &basestation.BaseStation{Name: "tims base"}, nil
}

// The station's answer can be recorded before the ping that caused it; the
// ping still occurred first, when its frame was written.
func TestPing_SentOccursBeforeAnsweredRecordedFirst(t *testing.T) {
	store := &orderedStationEvents{answered: make(chan struct{})}
	server, session, conn := newPingServer(t, &stationEventLog{})
	server.stationEvents = basestation.NewNamingEventRecorder(pingStationDirectory{},
		basestation.NewPersistentEventRecorder(logger.NewNop(), store, "1"), logger.NewNop())
	gated := newGatedConn(conn)
	session.Conn = gated

	pinged := make(chan error, 1)
	go func() {
		_, err := server.InitiatePing(testutil.TestContext(), uint64(TestBsEui01), pingTestTenant)
		pinged <- err
	}()
	<-gated.entered

	answeredDone := make(chan error, 1)
	go func() {
		answeredDone <- server.handlePingResponse(session, &Message{Command: mioty.CmdPingResponse, OpId: pingTestRspOpID}, map[string]interface{}{})
	}()
	<-store.answered
	close(gated.release)
	require.NoError(t, <-pinged)
	require.NoError(t, <-answeredDone)

	sent := store.occurredAt(t, models.EventTypeBaseStationPingSent)
	answered := store.occurredAt(t, models.EventTypeBaseStationPingAnswered)
	assert.True(t, sent.CreatedAt.Before(answered.CreatedAt),
		"ping sent at %s must precede its answer at %s", sent.CreatedAt.Format(time.RFC3339Nano), answered.CreatedAt.Format(time.RFC3339Nano))
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// A station's answers occurred when their frames arrived, the server's clock
// reading on receipt.
func TestStationAnswers_OccurWhenTheFrameArrives(t *testing.T) {
	arrived := time.Date(2026, time.September, 29, 12, 40, 35, 321186000, time.UTC)
	events := &stationEventLog{}
	server, session, _ := newPingServer(t, events)
	server.clock = fixedClock{now: arrived}

	require.NoError(t, server.handlePingResponse(session, &Message{Command: mioty.CmdPingResponse, OpId: pingTestRspOpID}, map[string]interface{}{}))
	opID, err := server.SendStatusRequest(session)
	require.NoError(t, err)
	answerStatus(t, server, session, opID)

	require.Len(t, events.events, 2)
	for _, event := range events.events {
		assert.Equal(t, arrived, event.occurredAt, event.eventType)
	}
}
