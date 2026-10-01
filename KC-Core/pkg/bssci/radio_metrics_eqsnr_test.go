package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// BSSCI §3.6.1: eqSnr is optional; an att without it records no eqSnr rather
// than the snr.
func TestAttachWithoutEqSnrRecordsNoEqSnr(t *testing.T) {
	fixture := newProfileTestFixture(t, "")

	fixture.sendAttach(nil)

	update := fixture.repo.GetLastRadioUpdate()
	require.NotNil(t, update, "the attach records its radio metrics")
	assert.Nil(t, update.EqSNR, "an absent eqSnr is not recorded as the snr")
}

// BSSCI §3.7.1: eqSnr is optional; a det without it records no eqSnr rather
// than the snr, and a det with it records the reported value.
func TestDetachRecordsOnlyAReportedEqSnr(t *testing.T) {
	reported := 7.5
	for name, eqSnr := range map[string]*float64{"absent": nil, "reported": &reported} {
		t.Run(name, func(t *testing.T) {
			const opID = int64(515)
			env := newDetachTestEnv(t, &Config{
				DetachSignatureValidationEnabled: detachSigValidationOff,
				MessageEncoding:                  EncodingJSON,
			}, buildTestEndpoint(TestEpEui01, 1))
			payload := buildDetachPayload(TestEpEui01)
			if eqSnr != nil {
				payload[wireFieldEqSnr] = *eqSnr
			}

			require.NoError(t, env.server.handleDetach(env.session, &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}, payload))
			require.NoError(t, env.server.handleDetachComplete(env.session, &Message{Command: mioty.CmdDetachComplete, OpId: opID}, map[string]interface{}{}))

			require.Len(t, env.repo.radioMetricsUpdates, 1, "the detach records its radio metrics")
			assert.Equal(t, eqSnr, env.repo.radioMetricsUpdates[0].EqSNR, "only a reported eqSnr is recorded, never the snr")
		})
	}
}
