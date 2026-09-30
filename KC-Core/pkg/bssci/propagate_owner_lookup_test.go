package bssci

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

var errPropagateOwnerDirectory = errors.New("endpoint directory unavailable")

// newPropagateOwnerLookupEnv serves a roaming endpoint's propagate on a base
// station of another tenant whose endpoint directory cannot be asked.
func newPropagateOwnerLookupEnv(t *testing.T) *detachTestEnv {
	t.Helper()
	const (
		ownerTenant   = int64(100)
		servingTenant = int64(200)
	)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, buildTestEndpoint(TestEpEuiTenant01, ownerTenant))
	env.server.tenantID = servingTenant
	env.session.ResolvedTenantID = servingTenant
	env.session.HandshakeComplete = true
	env.session.LastScOpId = -1
	env.session.Bidirectional = true
	env.server.RegisterSession(env.session)
	env.repo.lookupErr = errPropagateOwnerDirectory
	return env
}

// A failed owner lookup fails the propagate closed: it is never sent or
// recorded for the tenant of the base station that would carry it. Only a
// genuine not-found makes the endpoint unknown.
func TestPropagateFailsClosedWhenOwnerLookupFails(t *testing.T) {
	t.Run(mioty.CmdDetachPropagate, func(t *testing.T) {
		env := newPropagateOwnerLookupEnv(t)

		err := env.server.SendDetachPropagate(env.session.ID, TestEpEuiTenant01)

		require.ErrorIs(t, err, errPropagateOwnerDirectory)
		assert.False(t, env.conn.SeenCommand(mioty.CmdDetachPropagate), "no detPrp is sent for an unresolved owner")
		msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
		msgRepo.mu.Lock()
		defer msgRepo.mu.Unlock()
		assert.Empty(t, msgRepo.detachPropagates, "no detPrp is recorded under the serving tenant")
	})

	t.Run(mioty.CmdAttachPropagate, func(t *testing.T) {
		env := newPropagateOwnerLookupEnv(t)
		nwkSnKey := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

		err := env.server.SendAttachPropagate(env.session.ID, TestEpEuiTenant01, nwkSnKey, 0x1505, true, 0, false, 0, false, false)

		require.ErrorIs(t, err, errPropagateOwnerDirectory)
		assert.False(t, env.conn.SeenCommand(mioty.CmdAttachPropagate), "no attPrp is sent for an unresolved owner")
	})
}
