package bssci

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	stationlessDefaultTenant = int64(1)
	stationlessOwnerTenant   = int64(4)
	stationlessEndpointEUI   = uint64(0x70B3D56770111505)
)

// ownersByEUI owns each listed endpoint for its tenant.
type ownersByEUI map[uint64]int64

func (o ownersByEUI) ResolveOwner(_ context.Context, eui models.EUI) (EndpointOwner, error) {
	tenantID, ok := o[eui.ToUint64()]
	if !ok {
		return EndpointOwner{}, storage.ErrNotFound
	}
	return EndpointOwner{TenantID: tenantID, Endpoint: &models.EndPoint{TenantID: tenantID, EUI: eui}}, nil
}

// stationlessServer is a server with no base station connected.
func stationlessServer(owners EndpointOwnerResolver) (*Server, *recordingEventStore) {
	events := &recordingEventStore{}
	return &Server{
		clock:           clock.SystemClock{},
		logger:          logger.NewNop(),
		eventStore:      events,
		sessions:        newSessionRegistry(),
		defaultTenantID: stationlessDefaultTenant,
		endpointOwners:  owners,
		orgResolver:     unresolvedOrganizations{},
	}, events
}

var propagationsToAllStations = map[string]func(*Server) []error{
	"detPrp": func(s *Server) []error { return s.SendDetachPropagateToAll(stationlessEndpointEUI) },
	"attPrp": func(s *Server) []error {
		return s.SendAttachPropagateToAll(stationlessEndpointEUI, make([]byte, 16), 1, true, 0, false, 0, false, false)
	},
}

// BSSCI §5.8-§5.9, multi-tenant roaming: a propagation no base station can
// receive fails, and its failure event, which names the endpoint, is
// recorded for the tenant that owns the endpoint, never for the server's
// default tenant.
func TestPropagationWithoutStationsIsRecordedForTheEndpointOwner(t *testing.T) {
	for name, propagate := range propagationsToAllStations {
		t.Run(name, func(t *testing.T) {
			server, events := stationlessServer(ownersByEUI{stationlessEndpointEUI: stationlessOwnerTenant})

			require.Len(t, propagate(server), 1, "no base station received the propagation")

			require.Len(t, events.created, 1)
			assert.Equal(t, strconv.FormatInt(stationlessOwnerTenant, 10), events.created[0].TenantID)
		})
	}
}

// An endpoint no tenant owns has no one to tell: the propagation still fails,
// and no tenant is shown its EUI.
func TestPropagationWithoutStationsOfAnUnownedEndpointRecordsNoEvent(t *testing.T) {
	for name, propagate := range propagationsToAllStations {
		t.Run(name, func(t *testing.T) {
			server, events := stationlessServer(ownersByEUI{})

			require.Len(t, propagate(server), 1, "no base station received the propagation")

			assert.Empty(t, events.created)
		})
	}
}
