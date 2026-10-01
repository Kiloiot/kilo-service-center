package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// countingBaseStations owns testOwnedBsEui and counts the lookups the handler
// makes besides the delete itself.
type countingBaseStations struct {
	ownedBaseStations
	lookups int
}

func (c *countingBaseStations) GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	c.lookups++
	return c.ownedBaseStations.GetByEUI(ctx, eui, tenantID)
}

// TestDeleteBaseStation_NamesTheRemovedStationWithoutALookup proves the audit
// record names the station the delete removed, read from the delete itself.
func TestDeleteBaseStation_NamesTheRemovedStationWithoutALookup(t *testing.T) {
	emitter := &captureAuditRecorder{}
	stations := &countingBaseStations{}
	svc := testCoreService(coreFields{basestationSvc: stations, audit: emitter, log: logger.NewNop()})

	ctx := pkgcontext.WithUserID(testutil.TestContextWithTenantAndOrg(testOwnerTenant, auditTestOrg), auditTestActor)
	_, err := svc.DeleteBaseStation(ctx, &pb.DeleteBaseStationRequest{BsEui: testOwnedBsEui})
	require.NoError(t, err)
	assert.Zero(t, stations.lookups, "the delete returns the station it removed; no lookup precedes it")
	require.Len(t, emitter.events, 1)
	require.NotNil(t, emitter.events[0].BaseStationID)
	assert.Equal(t, testOwnedBaseStationID, *emitter.events[0].BaseStationID)
}
