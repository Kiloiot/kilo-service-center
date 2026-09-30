package bssciservices

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	decisionTestTenant   = int64(78)
	decisionTestEUI      = "70B3D56770111505"
	decisionTestEndpoint = int64(5)
)

// decisionEndpoints holds one endpoint whose status the attach and detach
// decisions move, as the endpoint repository does.
type decisionEndpoints struct {
	mu       sync.Mutex
	endpoint models.EndPoint
	created  []string
}

func (d *decisionEndpoints) GetByEUI(_ context.Context, tenantID int64, _ []byte) (*models.EndPoint, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	clone := d.endpoint
	clone.TenantID = tenantID
	return &clone, nil
}

func (d *decisionEndpoints) EndpointDetachStateUpdate(context.Context, int64, int64, models.EndpointDetachStateParams) error {
	return nil
}

func (d *decisionEndpoints) TransitionEndpointStatus(_ context.Context, _ int64, _ int64, status string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	changed := d.endpoint.EpStatus != status
	d.endpoint.EpStatus = status
	return changed, nil
}

func (d *decisionEndpoints) RestateEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error) {
	return d.TransitionEndpointStatus(ctx, tenantID, endpointID, status)
}

func (d *decisionEndpoints) CreateWithStatus(_ context.Context, ep *models.EndPoint, status string) (*models.EndPoint, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.created = append(d.created, status)
	stored := *ep
	stored.ID, stored.EpStatus = decisionTestEndpoint, status
	return &stored, nil
}

func (d *decisionEndpoints) status() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.endpoint.EpStatus
}

func decisionKey() []byte {
	return []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
}

func newDecisionFixture(t *testing.T, epStatus string) (*attachmentFixture, *decisionEndpoints) {
	t.Helper()
	endpoints := &decisionEndpoints{endpoint: models.EndPoint{
		ID:       decisionTestEndpoint,
		EUI:      models.EUIFromString(decisionTestEUI),
		EpStatus: epStatus,
		NwkSnKey: decisionKey(),
	}}
	return newAttachmentFixture(t, endpoints, endpoints, idleStations{}, fixedSessionKeys{}, discardedEvents{}), endpoints
}

// An attach through the service center attaches the endpoint when it is
// made, with no base station connected, so a station that connects later is
// sent it; the owner is told once (SCACI §3.13).
func TestAttachThroughTheServiceCenterAttachesTheEndpointWhenMade(t *testing.T) {
	f, endpoints := newDecisionFixture(t, endpoint.EndpointStatusDetached)

	for range 2 {
		_, err := f.svc.AttachEndPoint(testutil.TestContext(), decisionTestEUI, decisionTestTenant)
		require.NoError(t, err)
	}
	f.wait(t)

	assert.Equal(t, pkgbssci.EndpointStatusAttached, endpoints.status())
	notices := f.notices.told()
	require.Len(t, notices, 1, "one attachment is announced once")
	assert.Equal(t, decisionTestTenant, notices[0].TenantID, "to the endpoint owner")
	assert.Equal(t, pkgbssci.EndpointStatusAttached, notices[0].Status.EpStatus)
	assert.Nil(t, notices[0].HeardBy, "an attachment decided in the service center was heard by no base station")
}

// A detach through the service center detaches the endpoint when it is made,
// with no base station connected, so no reconnecting station is sent it
// again; the owner is told once (SCACI §3.13).
func TestDetachThroughTheServiceCenterDetachesTheEndpointWhenMade(t *testing.T) {
	f, endpoints := newDecisionFixture(t, pkgbssci.EndpointStatusAttached)

	for range 2 {
		_, err := f.svc.DetachEndPoint(testutil.TestContext(), decisionTestEUI, decisionTestTenant)
		require.NoError(t, err)
	}
	f.wait(t)

	assert.Equal(t, endpoint.EndpointStatusDetached, endpoints.status())
	assert.Equal(t, []string{endpoint.EndpointStatusDetached}, f.notices.statuses(), "one detachment is announced once")
}

// A pre-attached endpoint is stored attached, then its owner is told once.
func TestCreateAttachedStoresTheEndpointAttachedAndAnnouncesIt(t *testing.T) {
	f, endpoints := newDecisionFixture(t, endpoint.EndpointStatusDetached)

	created, err := f.svc.CreateAttached(testutil.TestContext(), &models.EndPoint{
		EUI: models.EUIFromString(decisionTestEUI), TenantID: decisionTestTenant, NwkSnKey: decisionKey(),
	})
	require.NoError(t, err)
	f.wait(t)

	assert.Equal(t, []string{pkgbssci.EndpointStatusAttached}, endpoints.created, "stored with its attachment")
	assert.Equal(t, pkgbssci.EndpointStatusAttached, created.EpStatus)
	assert.Equal(t, []string{pkgbssci.EndpointStatusAttached}, f.notices.statuses())
}

// An endpoint whose key cannot be sent in an attPrp is refused before it is
// stored, so nothing is left to retry against.
func TestCreateAttachedRefusesAnEndpointThatCannotBeAttached(t *testing.T) {
	f, endpoints := newDecisionFixture(t, endpoint.EndpointStatusDetached)

	_, err := f.svc.CreateAttached(testutil.TestContext(), &models.EndPoint{
		EUI: models.EUIFromString(decisionTestEUI), TenantID: decisionTestTenant, NwkSnKey: make([]byte, endpointKeyLength),
	})

	require.ErrorIs(t, err, grpcservices.ErrEndpointNotAttachable)
	assert.Empty(t, endpoints.created, "nothing is stored")
	assert.Empty(t, f.notices.told(), "nothing is announced")
}

func TestNewEndpointAttachmentServiceRefusesMissingCollaborators(t *testing.T) {
	f, endpoints := newDecisionFixture(t, endpoint.EndpointStatusDetached)
	svc := f.svc
	cases := map[string]error{
		"endpoints": func() error {
			_, err := NewEndpointAttachmentService(nil, endpoints, svc.decider, svc.notifier, svc.propagation)
			return err
		}(),
		"creator": func() error {
			_, err := NewEndpointAttachmentService(endpoints, nil, svc.decider, svc.notifier, svc.propagation)
			return err
		}(),
		"decider": func() error {
			_, err := NewEndpointAttachmentService(endpoints, endpoints, nil, svc.notifier, svc.propagation)
			return err
		}(),
		"notifier": func() error {
			_, err := NewEndpointAttachmentService(endpoints, endpoints, svc.decider, nil, svc.propagation)
			return err
		}(),
		"propagation": func() error {
			_, err := NewEndpointAttachmentService(endpoints, endpoints, svc.decider, svc.notifier, nil)
			return err
		}(),
	}
	want := map[string]error{
		"endpoints": errNilAttachmentEndpoints, "creator": errNilAttachedEndpointCreator, "decider": errNilAttachmentDecider,
		"notifier": errNilEndpointStatusNotifier, "propagation": errNilAttachmentPropagation,
	}
	for name, err := range cases {
		assert.ErrorIs(t, err, want[name], name)
	}
}
