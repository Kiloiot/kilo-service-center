package bssci

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// queryRecorder records the tenant every DL RX status query is correlated under.
type queryRecorder struct {
	DLRXStatusStore
	tenants []int64
}

func (r *queryRecorder) CreateDLRXStatusQuery(_ context.Context, tenantID int64, _ *uuid.UUID, _, _ []byte, _ int64) error {
	r.tenants = append(r.tenants, tenantID)
	return nil
}

func newQueryServer(endpoints ...*models.EndPoint) (*Server, *queryRecorder) {
	server := NewTestServerWithMemoryStatusService(logger.NewNop(), nil, nil, 1)
	server.endpointRepo = newFakeEndpointRepo(endpoints...)
	queries := &queryRecorder{}
	server.dlrxStore = queries
	return server, queries
}

// TestSendDLRXStatusQuery_CorrelatesUnderTheEndpointOwner: a query sent
// through another tenant's station is tracked under the endpoint owner, so
// the station owner never lists the foreign endpoint and the owner matches
// the later dlRxStat (BSSCI §3.15/§3.16).
func TestSendDLRXStatusQuery_CorrelatesUnderTheEndpointOwner(t *testing.T) {
	server, queries := newQueryServer(&models.EndPoint{EUI: models.EUIFromString(revokeEndpointEUI), TenantID: revokeOwnerTenant})
	session, _ := registerRoamingStation(server)

	require.NoError(t, server.SendDLRXStatusQuery(session.ID, queueOwnerEndpoint))

	assert.Equal(t, []int64{revokeOwnerTenant}, queries.tenants)
}

// TestSendDLRXStatusQuery_UnknownEndpointIsNotQueried: with no owner to file
// the correlation under, nothing is tracked and no query reaches the station.
func TestSendDLRXStatusQuery_UnknownEndpointIsNotQueried(t *testing.T) {
	server, queries := newQueryServer()
	session, conn := registerRoamingStation(server)

	require.Error(t, server.SendDLRXStatusQuery(session.ID, queueOwnerEndpoint))

	assert.Empty(t, queries.tenants)
	assert.False(t, conn.errorSent, "no dlRxStatQry without a tracked correlation")
}
