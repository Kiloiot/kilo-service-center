package scaciservices

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
)

// SCACI §3.6.1: a reg of an End Point the service center does not know
// creates it, classed by the registration's bidi flag (radio protocol §3.1).
func TestEndpointService_RegisterCreatesAnUnknownEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	tenantID, _ := seedTenantWithOrganization(t, db, "reg-unknown-endpoint")
	endpoints := postgres.NewRepositories(db).Endpoints
	svc, _ := newTestEndpointService(t, endpoints)

	register := func(eui uint64, bidi bool) {
		t.Helper()
		req := &scaci.Register{EpEui: eui, Bidi: bidi, NwkKey: mioty.NetworkKey{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}
		require.Empty(t, svc.Register(ctx, req, tenantID), "reg of an End Point stores it")

		euiBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(euiBytes, eui)
		stored, err := endpoints.GetByEUI(ctx, tenantID, euiBytes)
		require.NoError(t, err)
		assert.Equal(t, mioty.EndpointClass(bidi), stored.EPClass)
		assert.Equal(t, bidi, stored.Bidi)
	}

	register(0x70B3D56770111505, true)
	register(0x70B3D56770111506, false)
	register(0x70B3D56770111505, false)
}

// SCACI §3.6.1: a pre-attachment attaches a registered endpoint once; the
// attach reports the change only to the call that made it.
func TestEndpointService_AttachReportsTheChangeOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	tenantID, _ := seedTenantWithOrganization(t, db, "pre-attach-endpoint")
	endpoints := postgres.NewRepositories(db).Endpoints
	svc, notices := newTestEndpointService(t, endpoints)
	const eui = uint64(0x70B3D56770111507)
	require.Empty(t, svc.Register(ctx, &scaci.Register{EpEui: eui, PreAttach: true, NwkKey: mioty.NetworkKey{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}, tenantID))
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, eui)
	registered, err := endpoints.GetByEUI(ctx, tenantID, euiBytes)
	require.NoError(t, err)
	require.Equal(t, endpoint.EndpointStatusDetached, registered.EpStatus, "a registration alone leaves the endpoint detached")

	require.Empty(t, svc.Attach(ctx, registered))
	require.Empty(t, svc.Attach(ctx, registered))
	assert.Equal(t, []string{endpoint.EndpointStatusAttached}, notices.statuses, "the owner is told of the attachment once")

	stored, err := endpoints.GetByEUI(ctx, tenantID, euiBytes)
	require.NoError(t, err)
	assert.Equal(t, endpoint.EndpointStatusAttached, stored.EpStatus)
}

// SCACI §3.7: a deregistration detaches an attached endpoint once; it reports
// the change only to the call that made it.
func TestEndpointService_DeregisterReportsTheDetachmentOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	tenantID, _ := seedTenantWithOrganization(t, db, "deregister-endpoint")
	endpoints := postgres.NewRepositories(db).Endpoints
	svc, notices := newTestEndpointService(t, endpoints)
	const eui = uint64(0x70B3D56770111508)
	require.Empty(t, svc.Register(ctx, &scaci.Register{EpEui: eui, NwkKey: mioty.NetworkKey{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}, tenantID))
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, eui)
	registered, err := endpoints.GetByEUI(ctx, tenantID, euiBytes)
	require.NoError(t, err)
	require.Empty(t, svc.Attach(ctx, registered))

	require.Empty(t, svc.Deregister(ctx, eui, tenantID))
	require.Empty(t, svc.Deregister(ctx, eui, tenantID))
	assert.Equal(t, []string{endpoint.EndpointStatusAttached, endpoint.EndpointStatusDetached}, notices.statuses,
		"the owner is told of the detachment once")
	stored, err := endpoints.GetByEUI(ctx, tenantID, euiBytes)
	require.NoError(t, err)
	assert.Equal(t, endpoint.EndpointStatusDetached, stored.EpStatus)
}
