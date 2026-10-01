//go:build integration

package probe

import (
	"testing"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/stretchr/testify/require"
)

const (
	registerShAddr    uint16 = 0x2468
	registerAttachCnt uint32 = 3
	registerPacketCnt uint32 = 1000
)

// regMsg is an AC's register message for the endpoint with every SCACI
// §3.6.1 field set.
func regMsg(ep testEndpoint, preAttach bool) map[string]interface{} {
	return map[string]interface{}{
		keyCommand: cmdReg, keyEpEui: ep.eui, keyBidi: true, keyPreAttach: preAttach, keyNwkKey: acBytes(ep.key),
		keyShAddr: registerShAddr, keyAttachCnt: registerAttachCnt, keyPacketCnt: registerPacketCnt,
		keyDualChan: true, keyRepetition: false, keyWideCarrOff: true, keyLongBlkDist: false,
	}
}

// unregisteredEndpoint is an endpoint the SC has never seen, for reg to
// create.
func unregisteredEndpoint(t *testing.T) testEndpoint {
	t.Helper()
	return testEndpoint{eui: nextEndpointEUI(), key: randomBytes(t, sessionKeyLen), shAddr: registerShAddr, tenant: tenantPrimary}
}

// G1 SCACI §3.6.1 l.339-355: reg registers an endpoint and stores every
// field.
func TestConformanceG1_RegisterFieldsStored(t *testing.T) {
	ac := newAC(t)
	ep := unregisteredEndpoint(t)
	requireRsp(t, ac.request(t, regMsg(ep, false)), cmdReg)
	got := getEndpoint(t, ep)
	require.Equal(t, epClassBidirectional, got.GetEpClass(), "bidi")
	require.Empty(t, got.GetNwkSnKey(), "an ordinary read masks nwkKey")
	require.True(t, got.GetNwkSnKeySet(), "nwkKey stored")
	revealed := revealEndpoint(t, ep, pb.EndpointKey_ENDPOINT_KEY_NWK_SN_KEY)
	require.Equal(t, ep.key, revealed.GetNwkSnKey(), "nwkKey")
	require.EqualValues(t, registerShAddr, got.GetShAddr(), "shAddr")
	require.EqualValues(t, registerAttachCnt, got.GetAttachCnt(), "attachCnt")
	require.EqualValues(t, registerPacketCnt, got.GetLastPacketCnt(), "packetCnt")
	require.True(t, got.GetDualChan(), "dualChan")
	require.False(t, got.GetRepetition(), "repetition")
	require.True(t, got.GetWideCarrOff(), "wideCarrOff")
	require.False(t, got.GetLongBlkDist(), "longBlkDist")
	require.False(t, got.GetPreAttach(), "preAttach")
}

// G2 SCACI §3.6.1 l.347, RADIO §3.7.1.1: preAttach propagates the endpoint to
// the stations with its key, short address and last packet counter.
func TestConformanceG2_PreAttachPropagates(t *testing.T) {
	ac := newAC(t)
	bs := connectStation(t, stationA)
	bs.settle(cmdAttPrp)
	ep := newEndpoint(t, tenantPrimary)
	from := bs.mark()
	requireRsp(t, ac.request(t, regMsg(ep, true)), cmdReg)
	prp := bs.awaitCommand(t, from, cmdAttPrp, forEndpoint(ep.eui))
	key, ok := numericArray(prp.fields[keyNwkSnKey])
	require.True(t, ok, "attPrp.nwkSnKey is not Numeric[16]: %v", prp.fields)
	require.Equal(t, ep.key, key, "a pre-attached endpoint's session key is its network key")
	shAddr, _ := prp.int(keyShAddr)
	last, _ := prp.int(keyLastPacketCnt)
	require.EqualValues(t, registerShAddr, shAddr)
	require.EqualValues(t, registerPacketCnt, last)
}

// G3 SCACI §2.4: an out-of-range reg value (a short address that does not fit
// 16 bits) is a protocol error.
func TestConformanceG3_RegisterOutOfRange(t *testing.T) {
	ac := newAC(t)
	msg := regMsg(newEndpoint(t, tenantPrimary), false)
	msg[keyShAddr] = oversizedShAddr
	requireProtocolError(t, ac.request(t, msg))
}

// G4 SCACI §3.7 l.371-396: dereg detaches the endpoint at the stations and
// clears its downlinks, the pending one and the one a station holds.
func TestConformanceG4_DeregisterClears(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	pendingQueID := queueDownlink(t, ep, dlRequest([]byte{0x41}))
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	heldQueID := queueDownlink(t, ep, dlRequest([]byte{0x42}))
	bs.dlDataQueFor(t, from, ep, heldQueID)
	bs.settle(cmdDLDataQueCmp)
	from = bs.mark()
	requireRsp(t, ac.request(t, map[string]interface{}{keyCommand: cmdDereg, keyEpEui: ep.eui}), cmdDereg)
	bs.awaitCommand(t, from, cmdDetPrp, forEndpoint(ep.eui))
	for _, queID := range []int64{pendingQueID, heldQueID} {
		res := awaitDownlinkResult(t, ep, queID)
		require.True(t, res != nil && res.GetResult() == revokedResult,
			"downlink %d of the deregistered endpoint not revoked: %v", queID, res)
	}
}

// G6 SCACI §3.13 l.594-595, l.605-611: an attachment made through the SC (a
// pre-attach) is reported with epStat attached, without the OTA-only fields.
func TestConformanceG6_EPStatForPreAttach(t *testing.T) {
	ac := newAC(t)
	bs := connectStation(t, stationA)
	bs.settle(cmdAttPrp)
	ep := newEndpoint(t, tenantPrimary)
	from := ac.mark()
	requireRsp(t, ac.request(t, regMsg(ep, true)), cmdReg)
	st := ac.awaitCommand(t, from, cmdEPStat, forEndpoint(ep.eui))
	require.Equal(t, epStatusAttached, st.fields[keyEPStatus])
	for _, key := range []string{keyAttachCnt, keyNonce, keySign, keySnr, keyRssi, keyEqSnr, keySubpackets} {
		require.False(t, st.has(key), "epStat for a pre-attach carries OTA-only %s: %v", key, st.fields)
	}
}

// G7 SCACI §3.13.1 l.606-607: epStat nonce and sign are Numeric[4] arrays (an
// OTA attach is needed, so this row also needs C2).
func TestConformanceG7_EPStatNumericNonceSign(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	from := ac.mark()
	otaAttach(t, bs, ep, firstAttachCnt, randomBytes(t, nonceLen))
	st := ac.awaitCommand(t, from, cmdEPStat, forEndpoint(ep.eui))
	for _, key := range []string{keyNonce, keySign} {
		b, ok := numericArray(st.fields[key])
		require.True(t, ok && len(b) == nonceLen, "epStat.%s is not Numeric[4]: %T %v", key, st.fields[key], st.fields[key])
	}
}
