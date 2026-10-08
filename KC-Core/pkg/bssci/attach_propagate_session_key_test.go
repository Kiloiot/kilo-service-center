package bssci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// attachedSessionKeys answers from what an attach persisted - the endpoint's
// attach material and its active session key - the way the storage-backed
// source does.
type attachedSessionKeys struct {
	endpoint    models.EndPoint
	persistence *recordingAttachPersistence
}

func (k attachedSessionKeys) NetworkSessionKey(context.Context, *models.EndPoint) ([]byte, error) {
	endpoint := k.endpoint
	var activeSessionKey []byte
	if attaches := k.persistence.persisted(); len(attaches) > 0 {
		last := attaches[len(attaches)-1]
		endpoint.Nonce, endpoint.Sign = last.EndpointUpdates.Nonce, last.EndpointUpdates.Sign
		activeSessionKey = last.EncryptedKey
	}
	return CurrentNetworkSessionKey(&endpoint, activeSessionKey), nil
}

// wireKey reads a Numeric[16] key field of a captured frame.
func wireKey(t *testing.T, frame map[string]interface{}) []byte {
	t.Helper()
	values, ok := frame[wireFieldNwkSnKey].([]interface{})
	require.True(t, ok, "nwkSnKey is a numeric array, got %T", frame[wireFieldNwkSnKey])
	key := make([]byte, len(values))
	for i, v := range values {
		key[i] = byte(v.(float64))
	}
	return key
}

// newSecondStation connects another base station of the endpoint owner.
func (f *attachFixture) newSecondStation() (*Session, *bsscitest.TestConn) {
	return f.newStation("attach-second-station", TestBsEui02, attachResponseTenant)
}

// newStation connects a base station of the given tenant.
func (f *attachFixture) newStation(id string, bsEUI uint64, tenantID int64) (*Session, *bsscitest.TestConn) {
	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                id,
			BaseStationEUI:    bsEUI,
			ResolvedTenantID:  tenantID,
			DbSessionID:       2,
			Encoding:          EncodingJSON,
			SessionUUID:       uuidBytes(),
			HandshakeComplete: true,
		},
		Conn:          conn,
		Bidirectional: true,
	}
	f.server.RegisterSession(session)
	return session, conn
}

func (f *attachFixture) propagatedKey(t *testing.T, endpoint *models.EndPoint) []byte {
	t.Helper()
	session, conn := f.newSecondStation()
	require.NoError(t, f.server.SendAttachPropagateToSession(testutil.TestContext(), session, endpoint))
	attPrp := conn.LastMessage(mioty.CmdAttachPropagate)
	require.NotNil(t, attPrp, "the second station receives attPrp")
	return wireKey(t, attPrp)
}

func newSessionKeyFixture(t *testing.T) (*attachFixture, *models.EndPoint) {
	t.Helper()
	f := newOwnAttachFixture(t)
	provisioned := *f.endpoints.endpoints[TestEpEui01]
	endpoint := &provisioned
	endpoint.Bidi = true
	f.server.sessionKeys = attachedSessionKeys{endpoint: *endpoint, persistence: f.persistence}
	return f, endpoint
}

// Radio spec §3.7.1.2-§3.7.1.3: after an over-the-air attach at one station the
// others receive the session key that station handed out, never the
// pre-shared key.
func TestAttachPropagateAfterOverTheAirAttachCarriesTheSessionKey(t *testing.T) {
	f, endpoint := newSessionKeyFixture(t)

	f.attach(t)
	sessionKey := wireKey(t, f.attachResponse(t))

	propagated := f.propagatedKey(t, endpoint)
	assert.Equal(t, sessionKey, propagated, "the second station gets the key the first one handed out")
	assert.NotEqual(t, testPresharedKey(), propagated, "the pre-shared key stays in the service center")
}

// A pre-attached endpoint operates on its pre-shared key, which is what
// propagates.
func TestAttachPropagateOfPreAttachedEndpointCarriesThePresharedKey(t *testing.T) {
	f, endpoint := newSessionKeyFixture(t)

	assert.Equal(t, testPresharedKey(), f.propagatedKey(t, endpoint))
}
