package messages

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	testPollInterval  = 10 * time.Millisecond
	testBatchSize     = 10
	testOverlap       = time.Second
	testStreamWindow  = 300 * time.Millisecond
	testCollectWindow = 150 * time.Millisecond
	testTenant        = 1
)

var testStationEUI = []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6}

// uplinkStore lists uplinks newest first and reads them in storage order, as
// the repository does.
type uplinkStore struct {
	MessageStore
	mu       sync.Mutex
	uplinks  []*mioty.ULDataMessage
	readOnce chan struct{}
}

func newUplinkStore(uplinks ...*mioty.ULDataMessage) *uplinkStore {
	s := &uplinkStore{readOnce: make(chan struct{}, 1)}
	for _, uplink := range uplinks {
		s.add(uplink)
	}
	return s
}

// add stores an uplink, stamping it with the time it was stored as the
// database does.
func (s *uplinkStore) add(uplink *mioty.ULDataMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	uplink.StoredAt = time.Now()
	s.uplinks = append(s.uplinks, uplink)
}

// ListStored lists the uplinks stored at or after since, newest stored first.
func (s *uplinkStore) ListStored(_ context.Context, _ int64, _ []byte, _ *grpcservices.MessageFilters, since *time.Time, limit, offset int) ([]*mioty.ULDataMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case s.readOnce <- struct{}{}:
	default:
	}
	var rows []*mioty.ULDataMessage
	for _, uplink := range s.uplinks {
		if since == nil || !uplink.StoredAt.Before(*since) {
			rows = append(rows, uplink)
		}
	}
	slices.SortStableFunc(rows, func(a, b *mioty.ULDataMessage) int { return b.StoredAt.Compare(a.StoredAt) })
	if offset >= len(rows) {
		return nil, nil
	}
	return rows[offset:min(offset+limit, len(rows))], nil
}

func (s *uplinkStore) List(_ context.Context, _ int64, filter *grpcservices.MessageFilters, limit, _ int) ([]*mioty.ULDataMessage, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rows []*mioty.ULDataMessage
	for _, uplink := range s.uplinks {
		if filter == nil || filter.StartTime == nil || uplink.RxTime >= filter.StartTime.UnixNano() {
			rows = append(rows, uplink)
		}
	}
	slices.SortStableFunc(rows, func(a, b *mioty.ULDataMessage) int { return int(b.RxTime - a.RxTime) })
	return rows[:min(limit, len(rows))], int64(len(rows)), nil
}

func uplink(id string, rx time.Time) *mioty.ULDataMessage {
	return &mioty.ULDataMessage{ID: id, RxTime: rx.UnixNano()}
}

func collect(ch <-chan *mioty.ULDataMessage) map[string]int {
	seen := make(map[string]int)
	window := time.After(testCollectWindow)
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return seen
			}
			seen[msg.ID]++
		case <-window:
			return seen
		}
	}
}

type streamOpener func(ctx context.Context, svc *Service) (<-chan *mioty.ULDataMessage, error)

// Every uplink stream delivers the uplinks stored after it opened, once each,
// and neither the history nor a repeat of the newest uplink.
func TestStreams_DeliverEachNewUplinkOnce(t *testing.T) {
	openers := map[string]streamOpener{
		"tenant": func(ctx context.Context, svc *Service) (<-chan *mioty.ULDataMessage, error) {
			return svc.StreamMessages(ctx, testTenant, &grpcservices.MessageFilters{})
		},
		"base station": func(ctx context.Context, svc *Service) (<-chan *mioty.ULDataMessage, error) {
			return svc.StreamBaseStationMessages(ctx, testTenant, testStationEUI, &grpcservices.MessageFilters{})
		},
	}
	for name, open := range openers {
		t.Run(name, func(t *testing.T) {
			now := time.Now()
			store := newUplinkStore(uplink("history", now.Add(-time.Minute)))
			svc := New(store, registrationAt{registered: true}, testPollInterval, testOverlap, streamwake.NewSignal(), testBatchSize, logger.NewNop())
			ctx, cancel := testutil.TestContextWithTimeout(testStreamWindow)
			defer cancel()

			ch, err := open(ctx, svc)
			require.NoError(t, err)
			<-store.readOnce
			store.add(uplink("packet-72", now))
			store.add(uplink("packet-73", now.Add(time.Second)))

			assert.Equal(t, map[string]int{"packet-72": 1, "packet-73": 1}, collect(ch))
		})
	}
}
