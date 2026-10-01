package bssciservices

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	failureEventTestTenant   = int64(77)
	failureEventTestEUI      = "70B3D5677011150A"
	failureEventTestEndpoint = int64(4)
	failureEventWaitBound    = 2 * time.Second
)

var errFailureEventTestPropagate = errors.New("propagate failed")

// failureEventStore reports the context state each failure event was recorded on.
type failureEventStore struct {
	recorded chan error
}

func (s *failureEventStore) CreateEvent(ctx context.Context, event *models.SystemEvent) error {
	if event.EventType == bssci.EventTypeAttachPropagateFailed || event.EventType == bssci.EventTypeDetachPropagateFailed {
		s.recorded <- ctx.Err()
	}
	return ctx.Err()
}

// gatedPropagateSender fails every propagation once the test releases it.
type gatedPropagateSender struct {
	release chan struct{}
}

func (g *gatedPropagateSender) SendAttachPropagateToAll(uint64, []byte, uint16, bool, uint32, bool, uint8, bool, bool) []error {
	<-g.release
	return []error{errFailureEventTestPropagate}
}

func (g *gatedPropagateSender) SendDetachPropagateToAll(uint64) []error {
	<-g.release
	return []error{errFailureEventTestPropagate}
}

func (g *gatedPropagateSender) GetConnectedSessionEUIs() []string { return nil }

type failureEventEndpointLookup struct{}

func (failureEventEndpointLookup) EndpointDetachStateUpdate(context.Context, int64, int64, models.EndpointDetachStateParams) error {
	return nil
}

func (failureEventEndpointLookup) TransitionEndpointStatus(context.Context, int64, int64, string) (bool, error) {
	return true, nil
}

func (f failureEventEndpointLookup) RestateEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error) {
	return f.TransitionEndpointStatus(ctx, tenantID, endpointID, status)
}

func (failureEventEndpointLookup) GetByEUI(_ context.Context, tenantID int64, eui []byte) (*models.EndPoint, error) {
	var modelEUI models.EUI
	copy(modelEUI[:], eui)
	return &models.EndPoint{
		ID:       failureEventTestEndpoint,
		TenantID: tenantID,
		EUI:      modelEUI,
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}, nil
}

// A propagation failure is recorded even though the request that started the
// operation has already returned and cancelled its context.
func TestEndpointAttachmentFailureEventOutlivesRequestContext(t *testing.T) {
	operations := map[string]func(svc *endpointAttachmentService, ctx context.Context) error{
		bssci.OperationTypeAttach: func(svc *endpointAttachmentService, ctx context.Context) error {
			_, err := svc.AttachEndPoint(ctx, failureEventTestEUI, failureEventTestTenant)
			return err
		},
		bssci.OperationTypeDetach: func(svc *endpointAttachmentService, ctx context.Context) error {
			_, err := svc.DetachEndPoint(ctx, failureEventTestEUI, failureEventTestTenant)
			return err
		},
	}
	for name, start := range operations {
		t.Run(name, func(t *testing.T) {
			store := &failureEventStore{recorded: make(chan error, 1)}
			sender := &gatedPropagateSender{release: make(chan struct{})}
			svc := newAttachmentFixture(t, failureEventEndpointLookup{}, noEndpointCreation{}, sender, fixedSessionKeys{}, store).svc

			requestCtx, finishRequest := context.WithCancel(testutil.TestContext())
			require.NoError(t, start(svc, requestCtx))
			finishRequest()
			close(sender.release)

			select {
			case ctxErr := <-store.recorded:
				require.NoError(t, ctxErr, "the failure event must not be recorded on the cancelled request context")
			case <-time.After(failureEventWaitBound):
				t.Fatalf("no %s failure event was recorded", name)
			}
		})
	}
}
