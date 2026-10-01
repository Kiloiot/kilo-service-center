package scaci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// rangeTestNegative is a negative number in a field that only holds unsigned values.
const rangeTestNegative = -1

// EUIs and queue ids are unsigned 64-bit numbers (SCACI §3.6.1, §3.7.1,
// §3.9.1, §3.10.1, §3.11.1): a negative one is a protocol error, never
// wrapped into the EUI or queue id its bit pattern happens to name (§2.4).
func TestApplicationCenterRequests_RefuseNegativeUnsigned64Fields(t *testing.T) {
	key := numericValues(numericTestKey[:])
	negative := map[string]interface{}{"epEui": rangeTestNegative}
	for name, wire := range map[string]map[string]interface{}{
		"reg epEui":       withFields(regMap(key), negative),
		"dereg epEui":     withFields(deregMap(), negative),
		"ulDataTx epEui":  withFields(ulDataTxMap(key, []int{1}), negative),
		"ulDataTx bsEui":  withFields(ulDataTxMap(key, []int{1}), map[string]interface{}{"bsEui": rangeTestNegative}),
		"dlDataQue epEui": dlDataQueMap(negative),
		"dlDataQue queId": dlDataQueMap(map[string]interface{}{"queId": rangeTestNegative}),
		"dlDataRev epEui": withFields(dlDataRevMap(), negative),
	} {
		t.Run(name, func(t *testing.T) {
			server, downlinks, endpoints, uplinks := requestTestServer()
			conn := &mockConn{}

			handleRequest(t, server, conn, wire)

			assert.Empty(t, endpoints.Calls, "no endpoint is registered or deregistered under a wrapped EUI")
			assert.Empty(t, uplinks.Calls, "no uplink is transmitted for a wrapped EUI")
			downlinks.AssertNotCalled(t, "EnqueueDownlink", mock.Anything, mock.Anything)
			downlinks.AssertNotCalled(t, "GetDownlinksByPacketCnt", mock.Anything, mock.Anything)
			reply := sentError(t, conn)
			assertErrorToken(t, reply, errFieldOutOfRange)
			assert.Equal(t, POSIX_ERANGE, reply.Code)
		})
	}
}

// acEui is an unsigned 64-bit number (SCACI §3.3.1): a con with a negative
// one fails to decode with the out-of-range token handleConnect answers.
func TestConnect_RefusesNegativeAcEui(t *testing.T) {
	payload := mustMsgpack(t, map[string]interface{}{
		"command": CmdConnect, "opId": OpIDConnect, "version": ProtocolVersionString,
		"acEui": rangeTestNegative, "snAcUuid": numericValues(numericTestKey[:]),
	})

	var req Connect
	err := decodePayload(payload, &req)

	require.ErrorIs(t, err, mioty.ErrNumericOutOfRange)
	assert.Equal(t, errFieldOutOfRange, decodeFailureToken(err, errInvalidConnectFormat, errInvalidConnectFormat))
}
