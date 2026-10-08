package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// receptionSubpackets is a BSSCI §3.10.1 subpackets object as a station sends it.
func receptionSubpackets() map[string]interface{} {
	return map[string]interface{}{
		"snr":       []interface{}{10.0, 11.0},
		"rssi":      []interface{}{-70.0, -72.0},
		"frequency": []interface{}{868100000.0, 868300000.0},
	}
}

func assertReceptionSubpackets(t *testing.T, got *mioty.Subpackets) {
	t.Helper()
	require.NotNil(t, got, "the subpackets the station reported are forwarded")
	assert.Equal(t, []float64{10.0, 11.0}, got.SNR)
	assert.Equal(t, []float64{-70.0, -72.0}, got.RSSI)
	assert.Equal(t, []int64{868100000, 868300000}, got.Frequency)
}

// SCACI §3.13.1: epStat attached carries the att's subpackets as received and
// eqSnr only when the station reported one.
func TestAttachEPStatusForwardsTheReceptionFieldsAsReceived(t *testing.T) {
	f := newOwnAttachFixture(t)
	broadcaster := NewSyncMockSCACIEPStatusBroadcaster()
	f.server.SetSCACIEPStatusBroadcaster(broadcaster)

	f.attachWith(t, map[string]interface{}{"subpackets": receptionSubpackets()})
	f.attachResponse(t)
	broadcaster.ExpectCall(1)
	complete := map[string]interface{}{"command": mioty.CmdAttachComplete, "opId": int64(1)}
	require.NoError(t, f.server.CallHandleMessage(f.session, &Message{Command: mioty.CmdAttachComplete, OpId: 1, Data: complete}, complete))
	broadcaster.Wait()

	call := broadcaster.LastCall()
	require.NotNil(t, call)
	assert.Equal(t, pkgmioty.EPStatusAttached, call.Data.EpStatus)
	assertReceptionSubpackets(t, call.Data.Subpackets)
	assert.Nil(t, call.Data.EqSnr, "an att without eqSnr yields no eqSnr")
}

// SCACI §3.13.1: epStat detached carries the det's reception fields as
// received, including a 0 dB snr and an eqSnr equal to snr.
func TestDetachEPStatusForwardsTheReceptionFieldsAsReceived(t *testing.T) {
	const opID = int64(707)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, buildTestEndpoint(TestEpEui01, 1))
	broadcaster := NewSyncMockSCACIEPStatusBroadcaster()
	env.server.SetSCACIEPStatusBroadcaster(broadcaster)

	payload := buildDetachPayload(TestEpEui01)
	payload["snr"] = 0.0
	payload["eqSnr"] = 0.0
	payload["subpackets"] = receptionSubpackets()
	require.NoError(t, env.server.handleDetach(env.session, &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}, payload))
	broadcaster.ExpectCall(1)
	require.NoError(t, env.server.handleDetachComplete(env.session, &Message{Command: mioty.CmdDetachComplete, OpId: opID}, map[string]interface{}{}))
	broadcaster.Wait()

	call := broadcaster.LastCall()
	require.NotNil(t, call)
	assert.Equal(t, pkgmioty.EPStatusDetached, call.Data.EpStatus)
	require.NotNil(t, call.Data.Snr, "a 0 dB snr is a reported value")
	assert.Zero(t, *call.Data.Snr)
	require.NotNil(t, call.Data.EqSnr, "an eqSnr equal to snr is still reported")
	assert.Zero(t, *call.Data.EqSnr)
	assertReceptionSubpackets(t, call.Data.Subpackets)
}
