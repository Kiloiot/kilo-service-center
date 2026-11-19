package bssciservices

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// attachmentStore is the endpoint table the attachment service reads and the
// decider records into.
type attachmentStore interface {
	EndpointAttachmentStore
	StatusTransitioner
}

// attachmentFixture is the attachment service over one endpoint table, with
// the notices it announced and the background work it started.
type attachmentFixture struct {
	svc     *endpointAttachmentService
	notices *recordingNotifier
	work    *BackgroundWork
}

func newAttachmentFixture(t *testing.T, endpoints attachmentStore, creator AttachedEndpointCreator,
	stations AttachPropagateSender, sessionKeys bssci.NetworkSessionKeySource, events SystemEventRecorder,
) *attachmentFixture {
	t.Helper()
	notices := &recordingNotifier{}
	work := NewBackgroundWork()
	decider, err := NewEndpointAttachmentDecider(endpoints, notices)
	require.NoError(t, err)
	propagation, err := NewAttachmentPropagation(stations, sessionKeys, events, work, clock.SystemClock{}, logger.NewNop())
	require.NoError(t, err)
	svc, err := NewEndpointAttachmentService(endpoints, creator, decider, notices, propagation)
	require.NoError(t, err)
	return &attachmentFixture{svc: svc.(*endpointAttachmentService), notices: notices, work: work}
}

// wait returns once the background propagation has ended.
func (f *attachmentFixture) wait(t *testing.T) {
	t.Helper()
	require.NoError(t, f.work.Stop(testutil.TestContext()))
}

// recordingNotifier records every notice it is told.
type recordingNotifier struct {
	mu      sync.Mutex
	notices []EndpointStatusNotice
}

func (n *recordingNotifier) NotifyEndpointStatus(_ context.Context, notice EndpointStatusNotice) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.notices = append(n.notices, notice)
}

func (n *recordingNotifier) told() []EndpointStatusNotice {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]EndpointStatusNotice(nil), n.notices...)
}

func (n *recordingNotifier) statuses() []string {
	var statuses []string
	for _, notice := range n.told() {
		statuses = append(statuses, notice.Status.EpStatus)
	}
	return statuses
}

// noEndpointCreation stands in for the creation of attached endpoints in
// tests that attach or detach existing ones.
type noEndpointCreation struct{}

func (noEndpointCreation) CreateWithStatus(_ context.Context, ep *models.EndPoint, _ string) (*models.EndPoint, error) {
	return ep, nil
}

// discardedEvents records no system event.
type discardedEvents struct{}

func (discardedEvents) CreateEvent(context.Context, *models.SystemEvent) error { return nil }

// idleStations connects no base station.
type idleStations struct{}

func (idleStations) SendAttachPropagateToAll(uint64, []byte, uint16, bool, uint32, bool, uint8, bool, bool) []error {
	return nil
}

func (idleStations) SendDetachPropagateToAll(uint64) []error { return nil }

func (idleStations) GetConnectedSessionEUIs() []string { return nil }
