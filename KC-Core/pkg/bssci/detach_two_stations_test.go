package bssci

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// commandCount counts the frames of one command a station received.
func commandCount(conn *bsscitest.TestConn, command string) int {
	var count int
	for i := range conn.MessageCount() {
		if conn.GetMessage(i)["command"] == command {
			count++
		}
	}
	return count
}

// Radio spec §3.7.1, SCACI §3.13: a det heard by two base stations detaches
// the endpoint once, so only the completion that detached it announces the
// detachment and propagates it to the other stations.
func TestSameDetachFromTwoStationsIsAnnouncedOnce(t *testing.T) {
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, buildTestEndpoint(TestEpEui01, 1))
	org := uuid.New()
	env.server.orgResolver = &fakeOrgResolver{tenantToOrg: map[int64]uuid.UUID{1: org}, orgToTenant: map[uuid.UUID]int64{org: 1}}
	broadcaster := NewMockSCACIEPStatusBroadcaster()
	env.server.SetSCACIEPStatusBroadcaster(broadcaster)
	env.session.HandshakeComplete = true
	env.server.RegisterSession(env.session)
	env.connectStation("detach-second-station", TestBsEui02, true)
	unaware := env.connectStation("detach-unaware-station", TestBsEui03, true)

	for opID, session := range map[int64]*Session{901: env.session, 902: env.server.GetSession("detach-second-station")} {
		payload := buildDetachPayload(TestEpEui01)
		require.NoError(t, env.server.handleDetach(session, &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}, payload))
		require.NoError(t, env.server.handleDetachComplete(session, &Message{Command: mioty.CmdDetachComplete, OpId: opID}, map[string]interface{}{}))
	}
	env.server.wg.Wait()

	assert.Equal(t, 1, broadcaster.CallCount(), "one epStat for one detachment")
	assert.Equal(t, 1, commandCount(unaware, mioty.CmdDetachPropagate), "the station that did not hear the det is told once")
}
