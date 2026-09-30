package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const subpacketsTestDetachOpID = int64(616)

// malformedSubpackets are subpackets objects BSSCI §3.10.1 does not define:
// arrays of different lengths, or no object at all.
func malformedSubpackets() map[string]interface{} {
	return map[string]interface{}{
		"arrays of different lengths": map[string]interface{}{
			"snr": []interface{}{1.5, 2.5}, "rssi": []interface{}{-80.0}, "frequency": []interface{}{868100000.0, 868300000.0},
		},
		"not an object": []interface{}{1.5, 2.5},
	}
}

// BSSCI §2.4, §3.6.1: an att whose subpackets object is not the one §3.10.1
// defines is a protocol error, as it is for ulData.
func TestAttachRefusesMalformedSubpackets(t *testing.T) {
	for name, subpackets := range malformedSubpackets() {
		t.Run(name, func(t *testing.T) {
			fixture := newProfileTestFixture(t, "")

			fixture.sendAttachWith(func(data map[string]interface{}) { data[wireFieldSubpackets] = subpackets })

			code, message := fixture.testConn.LastError()
			assert.Equal(t, POSIX_EPROTO, code, "a malformed subpackets object is refused")
			assert.Equal(t, ResolveErrorMessage(errInvalidSubpackets), message)
			assert.False(t, fixture.testConn.SeenCommand(mioty.CmdAttachResponse), "no attRsp answers it")
		})
	}
}

// BSSCI §2.4, §3.7.1: a det whose subpackets object is not the one §3.10.1
// defines is a protocol error.
func TestDetachRefusesMalformedSubpackets(t *testing.T) {
	for name, subpackets := range malformedSubpackets() {
		t.Run(name, func(t *testing.T) {
			env := newDetachTestEnv(t, &Config{
				DetachSignatureValidationEnabled: detachSigValidationOff,
				MessageEncoding:                  EncodingJSON,
			}, buildTestEndpoint(TestEpEui01, 1))
			payload := buildDetachPayload(TestEpEui01)
			payload[wireFieldSubpackets] = subpackets

			require.NoError(t, env.server.handleDetach(env.session, &Message{Command: mioty.CmdDetach, OpId: subpacketsTestDetachOpID, Data: payload}, payload))

			code, message := env.conn.LastError()
			assert.Equal(t, POSIX_EPROTO, code, "a malformed subpackets object is refused")
			assert.Equal(t, ResolveErrorMessage(errInvalidSubpackets), message)
			assert.False(t, env.conn.SeenCommand(mioty.CmdDetachResponse), "no detRsp answers it")
		})
	}
}
