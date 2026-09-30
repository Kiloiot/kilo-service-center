package bssci

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// ownerLookup records the revocation the holding station is looked up for.
type ownerLookup struct {
	row  *storage.DownlinkMessage
	seen *storage.DownlinkRevocation
}

func (f *ownerLookup) GetDownlinkByRevocation(_ context.Context, revocation storage.DownlinkRevocation) (*storage.DownlinkMessage, error) {
	f.seen = &revocation
	return f.row, nil
}

// TestRevokeDownlink_NarrowsBothStepsToTheOwner: a reference naming an
// organization and an endpoint narrows the revoke in the queue and the
// lookup of the holding station alike, so another owner's queue id reaches
// neither.
func TestRevokeDownlink_NarrowsBothStepsToTheOwner(t *testing.T) {
	revocations := &pendingRevocations{}
	lookup := &ownerLookup{row: heldDownlink(mioty.DLQueueStatusQueued)}
	server, _ := newRevokeServer(t, queueRowStore{}, revocations)
	server.downlinkQueueStore = lookup
	registerRoamingStation(server)
	orgID, epEUI := uuid.New(), uint64(0x70b3d59cd0000341)

	_, err := server.RevokeDownlink(testutil.TestContext(), scheduler.DownlinkRef{
		TenantID: revokeOwnerTenant, QueID: revokeQueueID, OrganizationID: &orgID, EpEUI: &epEUI,
	})

	require.NoError(t, err)
	want := storage.DownlinkRevocation{QueID: int64(revokeQueueID), TenantID: revokeOwnerTenant, OrganizationID: &orgID, EpEUI: &epEUI}
	assert.Equal(t, []storage.DownlinkRevocation{want}, revocations.calls)
	require.NotNil(t, lookup.seen)
	assert.Equal(t, want, *lookup.seen)
}
