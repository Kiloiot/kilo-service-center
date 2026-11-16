package bssci

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	tenantScopeForeignStation = "attach-foreign-station"
	tenantScopeUnknownEUI     = uint64(0x70B3D5677011FFFF)
)

// tenantScopeStations are two connected stations of the endpoint's tenant
// and one of another tenant.
type tenantScopeStations struct {
	f       *attachFixture
	own     []*bsscitest.TestConn
	foreign *bsscitest.TestConn
}

func newTenantScopeStations(t *testing.T) *tenantScopeStations {
	t.Helper()
	f := newOwnAttachFixture(t)
	f.server.RegisterSession(f.session)
	_, second := f.newSecondStation()
	_, foreign := f.newStation(tenantScopeForeignStation, TestBsEui03, attachRoamingOwnerTenant)
	return &tenantScopeStations{f: f, own: []*bsscitest.TestConn{f.conn, second}, foreign: foreign}
}

func (s *tenantScopeStations) attachPropagate(tenantID int64) []error {
	return s.f.server.SendAttachPropagateToAll(tenantID, TestEpEui01, testPresharedKey(), 0x1505, false, 0, false, 0, false, false)
}

// BSSCI §5.8, multi-tenant isolation: an endpoint's network session key is
// sent only to the stations of the endpoint's tenant.
func TestAttachPropagateReachesOnlyTheEndpointTenantsStations(t *testing.T) {
	s := newTenantScopeStations(t)

	require.Empty(t, s.attachPropagate(attachResponseTenant))

	for _, conn := range s.own {
		assert.True(t, conn.SeenCommand(mioty.CmdAttachPropagate), "a station of the endpoint's tenant receives attPrp")
	}
	assert.False(t, s.foreign.SeenCommand(mioty.CmdAttachPropagate), "another tenant's station receives no attPrp")
}

// The filter follows the endpoint's tenant, not the server's default tenant.
func TestAttachPropagateOfAnotherTenantsEndpointReachesOnlyThatTenant(t *testing.T) {
	s := newTenantScopeStations(t)

	require.Empty(t, s.attachPropagate(attachRoamingOwnerTenant))

	assert.True(t, s.foreign.SeenCommand(mioty.CmdAttachPropagate))
	for _, conn := range s.own {
		assert.False(t, conn.SeenCommand(mioty.CmdAttachPropagate), "the server tenant's station receives no attPrp")
	}
}

// BSSCI §5.9: detPrp goes only to the stations of the endpoint's tenant.
func TestDetachPropagateReachesOnlyTheEndpointTenantsStations(t *testing.T) {
	s := newTenantScopeStations(t)

	require.Empty(t, s.f.server.SendDetachPropagateToAll(attachResponseTenant, TestEpEui01))

	for _, conn := range s.own {
		assert.True(t, conn.SeenCommand(mioty.CmdDetachPropagate), "a station of the endpoint's tenant receives detPrp")
	}
	assert.False(t, s.foreign.SeenCommand(mioty.CmdDetachPropagate), "another tenant's station receives no detPrp")
}

// A fan-out that does not know the endpoint's tenant sends to no station and
// records no event that would name the endpoint.
func TestPropagateWithoutTheEndpointTenantReachesNoStation(t *testing.T) {
	for name, propagate := range map[string]func(*tenantScopeStations) []error{
		"attPrp": func(s *tenantScopeStations) []error { return s.attachPropagate(0) },
		"detPrp": func(s *tenantScopeStations) []error { return s.f.server.SendDetachPropagateToAll(0, TestEpEui01) },
	} {
		t.Run(name, func(t *testing.T) {
			s := newTenantScopeStations(t)

			errs := propagate(s)

			require.Len(t, errs, 1)
			assert.ErrorIs(t, errs[0], ErrEndpointTenantUnknown)
			for _, conn := range append(s.own, s.foreign) {
				assert.Zero(t, conn.MessageCount(), "no station is sent anything")
			}
			assert.Empty(t, s.f.events.created, "no event names the endpoint")
		})
	}
}

// The operation events of a propagation name only the stations it reaches.
func TestConnectedSessionEUIsAreTheTenantsStations(t *testing.T) {
	f := newOwnAttachFixture(t)
	f.session.Connected = time.Now()
	f.server.RegisterSession(f.session)
	foreign, _ := f.newStation(tenantScopeForeignStation, TestBsEui03, attachRoamingOwnerTenant)
	foreign.Connected = time.Now()

	assert.Equal(t, []string{mioty.FormatEUI64(TestBsEui01)}, f.server.GetConnectedSessionEUIs(attachResponseTenant))
	assert.Equal(t, []string{mioty.FormatEUI64(TestBsEui03)}, f.server.GetConnectedSessionEUIs(attachRoamingOwnerTenant))
	assert.Empty(t, f.server.GetConnectedSessionEUIs(0))
}

func TestEndpointOwnerTenantNamesTheOwner(t *testing.T) {
	f := newOwnAttachFixture(t)

	owner, err := f.server.EndpointOwnerTenant(TestEpEui01)
	require.NoError(t, err)
	assert.Equal(t, attachResponseTenant, owner)

	_, err = f.server.EndpointOwnerTenant(tenantScopeUnknownEUI)
	assert.Error(t, err, "an endpoint no tenant owns has no owner tenant")
}

// Radio spec §3.7.1: an over-the-air detach is propagated only to the other
// stations of the endpoint's tenant.
func TestOverTheAirDetachIsNotPropagatedToAnotherTenantsStation(t *testing.T) {
	const opID = int64(809)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, buildTestEndpoint(TestEpEui01, 1))
	env.session.HandshakeComplete = true
	env.server.RegisterSession(env.session)
	own := env.connectStation("detach-own-station", TestBsEui02, true)
	foreign := &bsscitest.TestConn{Encoding: EncodingJSON}
	env.server.RegisterSession(&Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "detach-foreign-station",
			BaseStationEUI:    TestBsEui03,
			ResolvedTenantID:  attachRoamingOwnerTenant,
			DbSessionID:       3,
			Encoding:          EncodingJSON,
			SessionUUID:       uuidBytes(),
			HandshakeComplete: true,
		},
		Conn: foreign,
	})

	payload := buildDetachPayload(TestEpEui01)
	require.NoError(t, env.server.handleDetach(env.session, &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}, payload))
	require.NoError(t, env.server.handleDetachComplete(env.session, &Message{Command: mioty.CmdDetachComplete, OpId: opID}, map[string]interface{}{}))
	env.server.wg.Wait()

	assert.True(t, own.SeenCommand(mioty.CmdDetachPropagate), "the tenant's other station is told to drop the endpoint")
	assert.False(t, foreign.SeenCommand(mioty.CmdDetachPropagate), "another tenant's station is told nothing")
}
