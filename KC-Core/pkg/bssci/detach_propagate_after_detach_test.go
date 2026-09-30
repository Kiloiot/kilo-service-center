package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// connectStation registers another base station session of the endpoint's tenant.
func (env *detachTestEnv) connectStation(id string, bsEUI uint64, handshakeComplete bool) *bsscitest.TestConn {
	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	env.server.RegisterSession(&Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                id,
			BaseStationEUI:    bsEUI,
			ResolvedTenantID:  env.session.ResolvedTenantID,
			DbSessionID:       int64(len(id)),
			Encoding:          EncodingJSON,
			SessionUUID:       uuidBytes(),
			HandshakeComplete: handshakeComplete,
		},
		Conn: conn,
	})
	return conn
}

// Radio spec §3.7.1: an endpoint that detached over the air at one base
// station is dropped by the others through detPrp; the reporting station,
// which already dropped it, and a station still in its handshake get none.
func TestOverTheAirDetachIsPropagatedToTheOtherStations(t *testing.T) {
	const opID = int64(808)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, buildTestEndpoint(TestEpEui01, 1))
	env.session.HandshakeComplete = true
	env.server.RegisterSession(env.session)
	other := env.connectStation("detach-other-station", TestBsEui02, true)
	connecting := env.connectStation("detach-connecting-station", TestBsEui03, false)

	payload := buildDetachPayload(TestEpEui01)
	require.NoError(t, env.server.handleDetach(env.session, &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}, payload))
	require.NoError(t, env.server.handleDetachComplete(env.session, &Message{Command: mioty.CmdDetachComplete, OpId: opID}, map[string]interface{}{}))
	env.server.wg.Wait()

	detPrp := other.LastMessage(mioty.CmdDetachPropagate)
	require.NotNil(t, detPrp, "the other station is told to drop the endpoint")
	assert.InDelta(t, float64(TestEpEui01), detPrp["epEui"], 0)
	assert.False(t, env.conn.SeenCommand(mioty.CmdDetachPropagate), "the reporting station already dropped it")
	assert.False(t, connecting.SeenCommand(mioty.CmdDetachPropagate), "a station in its handshake gets no operation")
}
