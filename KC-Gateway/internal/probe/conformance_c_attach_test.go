//go:build integration

package probe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	firstAttachCnt  uint32 = 1
	stationShAddr          = 0x1234
	counterBefore   uint32 = 100
	counterRestart  uint32 = 1
	detachPacketCnt uint32 = 5
)

// otaAttach runs a spec-conformant over-the-air attach through the station
// and returns the attRsp; the SIM-BS completes it with attCmp.
func otaAttach(t *testing.T, bs *simBS, ep testEndpoint, attachCnt uint32, nonce []byte) wireFrame {
	t.Helper()
	rsp := bs.request(t, attMsg(t, ep, attachCnt, nonce))
	requireRsp(t, rsp, cmdAtt)
	return rsp
}

// C1 BSSCI §3.6.1 l.364-385: att mandatory fields are checked; a missing nonce
// is a protocol error.
func TestConformanceC1_AttachMandatoryFields(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	msg := attMsg(t, ep, firstAttachCnt, randomBytes(t, nonceLen))
	delete(msg, keyNonce)
	requireError(t, bs.request(t, msg), posixEPROTO)
}

// C2 RADIO §3.7.1.3 Fig. 3-15: an attach signed over the 16-byte IV with a
// 4-byte attach counter is authentic and is accepted.
func TestConformanceC2_AttachSignature16ByteIV(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	otaAttach(t, bs, ep, firstAttachCnt, randomBytes(t, nonceLen))
}

// C4 RADIO §3.6.5.3 l.766-771: a stale attach counter is refused (the first
// attach must succeed, so this row also needs C2).
func TestConformanceC4_StaleAttachCounter(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	otaAttach(t, bs, ep, firstAttachCnt, randomBytes(t, nonceLen))
	requireProtocolError(t, bs.request(t, attMsg(t, ep, firstAttachCnt, randomBytes(t, nonceLen))))
}

// C5 BSSCI §3.6.2 l.393, RADIO §3.7.1.3 Fig. 3-16: attRsp.nwkSnKey is the
// Numeric[16] session key AES-128-ECB(pre-shared key, EUI | nonce | sign).
func TestConformanceC5_AttachSessionKey(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	nonce := randomBytes(t, nonceLen)
	msg := attMsg(t, ep, firstAttachCnt, nonce)
	rsp := bs.request(t, msg)
	requireRsp(t, rsp, cmdAtt)
	key, ok := numericArray(rsp.fields[keyNwkSnKey])
	require.True(t, ok, "attRsp.nwkSnKey is not Numeric[16]: %T", rsp.fields[keyNwkSnKey])
	sign, _ := numericArray(msg[keySign])
	require.Equal(t, specSessionKey(t, ep.eui, nonce, sign, ep.key), key)
}

// C6 BSSCI §3.6.1 l.381, §3.6.2 l.394: when the station assigned the short
// address, attRsp carries none.
func TestConformanceC6_AttachStationAssignedShAddr(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	msg := attMsg(t, ep, firstAttachCnt, randomBytes(t, nonceLen))
	msg[keyShAddr] = stationShAddr
	rsp := bs.request(t, msg)
	requireRsp(t, rsp, cmdAtt)
	require.False(t, rsp.has(keyShAddr), "attRsp must omit shAddr the station assigned: %v", rsp.fields)
}

// C8 RADIO §3.7.1.2 l.1064-1066: when two stations receive the same attach the
// SC completes it through exactly one of them.
func TestConformanceC8_ConcurrentAttachOneWinner(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bsA := connectStation(t, stationA)
	bsB := connectStation(t, stationB)
	msg := attMsg(t, ep, firstAttachCnt, randomBytes(t, nonceLen))
	fromA, fromB := bsA.mark(), bsB.mark()
	opA := bsA.send(t, copyMsg(msg))
	opB := bsB.send(t, copyMsg(msg))
	answered := func(op int64) func(wireFrame) bool {
		return func(f wireFrame) bool { return f.opID == op && (f.command == cmdAttRsp || f.command == cmdError) }
	}
	rspA, okA := bsA.awaitFrom(fromA, frameWait, answered(opA))
	rspB, okB := bsB.awaitFrom(fromB, frameWait, answered(opB))
	accepted := 0
	for _, got := range []struct {
		f  wireFrame
		ok bool
	}{{rspA, okA}, {rspB, okB}} {
		if got.ok && got.f.command == cmdAttRsp {
			accepted++
		}
	}
	require.Equal(t, 1, accepted, "attRsp count; A=%v B=%v", rspA.fields, rspB.fields)
}

// C9 and H9 RADIO §3.7.1.2 l.1049-1051: after an OTA attach, other stations
// receive the session key (attRsp.nwkSnKey) by attPrp, never the pre-shared key.
func TestConformanceC9H9_PropagateSessionKey(t *testing.T) {
	bsA := connectStation(t, stationA)
	bsB := connectStation(t, stationB)
	bsB.settle(cmdAttPrp)
	// Registered after the connect reconciliation, so the only attPrp for it
	// is the one the attach triggers.
	ep := newEndpoint(t, tenantPrimary)
	from := bsB.mark()
	rsp := otaAttach(t, bsA, ep, firstAttachCnt, randomBytes(t, nonceLen))
	prp := bsB.awaitCommand(t, from, cmdAttPrp, forEndpoint(ep.eui))
	sessionKey, _ := numericArray(rsp.fields[keyNwkSnKey])
	propagated, ok := numericArray(prp.fields[keyNwkSnKey])
	require.True(t, ok, "attPrp.nwkSnKey is not Numeric[16]: %v", prp.fields)
	require.NotEqual(t, ep.key, propagated, "attPrp leaked the pre-shared key")
	require.Equal(t, sessionKey, propagated, "attPrp.nwkSnKey must equal the attRsp session key")
}

// C10 RADIO §3.6.2 l.426-430, §3.6.5.3: the packet counter restarts with an OTA
// attach, so a reused counter with new data after the attach is new data.
func TestConformanceC10H4_CounterRestartOnAttach(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	requireRsp(t, bs.request(t, ulDataMsg(ep, counterRestart, []byte{0x01})), cmdULData)
	requireRsp(t, bs.request(t, ulDataMsg(ep, counterBefore, []byte{0x02})), cmdULData)
	otaAttach(t, bs, ep, firstAttachCnt, randomBytes(t, nonceLen))
	requireRsp(t, bs.request(t, ulDataMsg(ep, counterRestart, []byte{0x03})), cmdULData)
	stored := 0
	for _, m := range storedMessages(t, tenantPrimary, ep.eui) {
		if m.GetPacketCounter() == counterRestart {
			stored++
		}
	}
	require.Equal(t, 2, stored, "the post-attach uplink with counter %d must be stored as new", counterRestart)
}

// C11 and G5 SCACI §3.13.1 l.600-611: after an OTA attach the AC gets epStat
// attached with the attach's subpackets, and no eqSnr the station did not send.
func TestConformanceC11G5_EPStatAttachedFields(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	from := ac.mark()
	msg := attMsg(t, ep, firstAttachCnt, randomBytes(t, nonceLen))
	msg[keySubpackets] = map[string]interface{}{
		keySnr: []interface{}{simSNR, simSNR}, keyRssi: []interface{}{simRSSI, simRSSI},
		keyFrequency: []interface{}{868100000, 868300000},
	}
	requireRsp(t, bs.request(t, msg), cmdAtt)
	st := ac.awaitCommand(t, from, cmdEPStat, forEndpoint(ep.eui))
	require.Equal(t, epStatusAttached, st.fields[keyEPStatus])
	for _, key := range []string{keyAttachCnt, keyNonce, keySign, keySnr, keyRssi} {
		require.True(t, st.has(key), "epStat attached lacks %s: %v", key, st.fields)
	}
	require.True(t, st.has(keySubpackets), "epStat lacks the attach's subpackets: %v", st.fields)
	require.False(t, st.has(keyEqSnr), "epStat invents an eqSnr the station did not send: %v", st.fields)
}

// C12 roaming (CLAUDE.md), RADIO §3.7.1.2: an OTA attach at a foreign-tenant
// station is served under the owner: the attach is recorded for the owner
// tenant and the owner's AC is told.
func TestConformanceC12_RoamingAttach(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	roamer := connectStation(t, stationRoamer)
	from := ac.mark()
	otaAttach(t, roamer, ep, firstAttachCnt, randomBytes(t, nonceLen))
	require.True(t, eventually(func() bool { return getEndpoint(t, ep).GetAttachCnt() == firstAttachCnt }),
		"attach not recorded under the owner tenant")
	ac.awaitCommand(t, from, cmdEPStat, forEndpoint(ep.eui))
}

// C13 BSSCI §3.7.1 l.410-425: det mandatory fields are checked; a missing sign
// is a protocol error.
func TestConformanceC13_DetachMandatoryFields(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	markAttached(t, ep)
	bs := connectStation(t, stationA)
	msg := detMsg(ep, detachPacketCnt, randomBytes(t, nonceLen))
	delete(msg, keySign)
	requireProtocolError(t, bs.request(t, msg))
}

// C14 BSSCI §3.7.2 l.428-433: det is answered by detRsp carrying a 4-byte
// sign. The spec leaves the construction open; the matrix's documented pass
// condition is an echo of the endpoint's sign.
func TestConformanceC14_DetachResponseSign(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	markAttached(t, ep)
	bs := connectStation(t, stationA)
	sign := randomBytes(t, nonceLen)
	rsp := bs.request(t, detMsg(ep, detachPacketCnt, sign))
	requireRsp(t, rsp, cmdDet)
	got, ok := numericArray(rsp.fields[keySign])
	require.True(t, ok, "detRsp.sign is not Numeric[4]: %v", rsp.fields)
	require.Equal(t, sign, got)
}

// C15 RADIO §3.7.1 l.966-968, SCACI §3.13: after an OTA detach at one station,
// the other stations drop the endpoint (detPrp) and the AC gets epStat detached.
func TestConformanceC15_DetachPropagates(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	markAttached(t, ep)
	ac := newAC(t)
	bsA := connectStation(t, stationA)
	bsB := connectStation(t, stationB)
	// The connect reconciliation propagates the endpoint to both stations.
	bsA.awaitCommand(t, 0, cmdAttPrp, forEndpoint(ep.eui))
	bsB.awaitCommand(t, 0, cmdAttPrp, forEndpoint(ep.eui))
	bsB.settle(cmdAttPrp)
	fromAC, fromB := ac.mark(), bsB.mark()
	requireRsp(t, bsA.request(t, detMsg(ep, detachPacketCnt, randomBytes(t, nonceLen))), cmdDet)
	st := ac.awaitCommand(t, fromAC, cmdEPStat, forEndpoint(ep.eui))
	require.Equal(t, epStatusDetached, st.fields[keyEPStatus])
	bsB.awaitCommand(t, fromB, cmdDetPrp, forEndpoint(ep.eui))
}

// C17 BSSCI §3.9.1 l.491-496: detPrp carries only command, opId and epEui.
func TestConformanceC17_DetachPropagateFields(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	from := bs.mark()
	requireRsp(t, ac.request(t, map[string]interface{}{keyCommand: cmdDereg, keyEpEui: ep.eui}), cmdDereg)
	prp := bs.awaitCommand(t, from, cmdDetPrp, forEndpoint(ep.eui))
	require.ElementsMatch(t, []string{keyCommand, keyOpID, keyEpEui}, keysOf(prp.fields))
}

func copyMsg(msg map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(msg))
	for k, v := range msg {
		out[k] = v
	}
	return out
}

func keysOf(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// C18 BSSCI §3.8, §1 (decision 2026-09-27): a connecting station is sent the
// endpoints the service center holds attached, and nothing for one registered
// without Pre-Attachment.
func TestConformanceC18_ConnectSendsAttachedEndpointsOnly(t *testing.T) {
	attached := newEndpoint(t, tenantPrimary, preAttached)
	detached := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)

	bs.awaitCommand(t, 0, cmdAttPrp, forEndpoint(attached.eui))
	bs.assertQuiet(t, 0, cmdAttPrp, forEndpoint(detached.eui))
}
