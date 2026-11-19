package scaciservices

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	detachOwnerTenant   = int64(3)
	detachForeignTenant = int64(4)
	detachEndpointEUI   = uint64(0x70B3D56770111505)
)

// tenantScopedEndpointRepo finds an endpoint only for the tenant that owns it.
type tenantScopedEndpointRepo struct {
	*mockEndpointRepo
	owner int64
}

func (r *tenantScopedEndpointRepo) GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error) {
	if tenantID != r.owner {
		return nil, storage.ErrNotFound
	}
	return r.mockEndpointRepo.GetByEUI(ctx, tenantID, eui)
}

// recordingDetachPropagator records the endpoints it sent detPrp for.
type recordingDetachPropagator struct {
	sent []uint64
}

func (p *recordingDetachPropagator) SendDetachPropagateToAll(epEUI uint64) []error {
	p.sent = append(p.sent, epEUI)
	return nil
}

func detachPropagationService(t *testing.T) (*recordingDetachPropagator, func(tenantID int64) []error) {
	t.Helper()
	repo := &tenantScopedEndpointRepo{mockEndpointRepo: newMockEndpointRepo(), owner: detachOwnerTenant}
	owned := &models.EndPoint{ID: 1, TenantID: detachOwnerTenant}
	binary.BigEndian.PutUint64(owned.EUI[:], detachEndpointEUI)
	repo.endpoints[detachEndpointEUI] = owned
	decider, err := bssciservices.NewEndpointAttachmentDecider(repo, &announcedDecisions{})
	require.NoError(t, err)
	stations := &recordingDetachPropagator{}
	svc, err := NewEndpointService(repo, stations, decider, logger.NewNop())
	require.NoError(t, err)
	return stations, func(tenantID int64) []error {
		return svc.PropagateDetachToAll(testutil.TestContext(), tenantID, detachEndpointEUI)
	}
}

// SCACI §3.7.3: a completed deregistration sends detPrp for an endpoint of
// the tenant that deregistered it; an EUI that tenant does not own is sent to
// no base station.
func TestPropagateDetachToAllSendsOnlyTheTenantsOwnEndpoint(t *testing.T) {
	stations, propagate := detachPropagationService(t)

	errs := propagate(detachForeignTenant)

	require.Len(t, errs, 1, "another tenant's endpoint is refused")
	assert.ErrorIs(t, errs[0], storage.ErrNotFound)
	assert.Empty(t, stations.sent, "no base station is sent detPrp for another tenant's endpoint")
}

func TestPropagateDetachToAllSendsTheOwnersEndpoint(t *testing.T) {
	stations, propagate := detachPropagationService(t)

	assert.Empty(t, propagate(detachOwnerTenant))
	assert.Equal(t, []uint64{detachEndpointEUI}, stations.sent)
}
