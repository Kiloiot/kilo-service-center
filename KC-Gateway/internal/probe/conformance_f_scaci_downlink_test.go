//go:build integration

package probe

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const revokeCounter uint32 = 42

// awaitStationQueue returns the station's dlDataQue whose single user data
// entry is payload, so a test can map an AC downlink to its BSSCI queId.
func (bs *simBS) awaitStationQueue(t *testing.T, from int, ep testEndpoint, payload []byte) wireFrame {
	t.Helper()
	return bs.awaitCommand(t, from, cmdDLDataQue, func(f wireFrame) bool {
		if !forEndpoint(ep.eui)(f) {
			return false
		}
		entries, _ := f.fields[keyUserData].([]interface{})
		for _, entry := range entries {
			if b, ok := anyBytes(entry); ok && bytes.Equal(b, payload) {
				return true
			}
		}
		return false
	})
}

// F1 SCACI §2.4, §3.10.1 l.505-512: userData is mandatory; a dlDataQue without
// it is a protocol error.
func TestConformanceF1_DLDataQueMandatoryUserData(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	msg := acDLDataQue(ep, acQueID(), []byte{0x01})
	delete(msg, keyUserData)
	requireProtocolError(t, ac.request(t, msg))
}

// F4 RADIO §3.6.6.3 l.924: a downlink payload entry is at most 200 bytes.
func TestConformanceF4_DLPayloadLimit(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	requireProtocolError(t, ac.request(t, acDLDataQue(ep, acQueID(), make([]byte, oversizedRadioLen))))
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), make([]byte, maxRadioPayload))), cmdDLDataQue)
}

// F5 SCACI §3.10.2-3: dlDataQue is answered with dlDataQueRsp and the AC's
// dlDataQueCmp completes it without an error.
func TestConformanceF5_DLDataQueHandshake(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	from := ac.mark()
	rsp := ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x01}))
	requireRsp(t, rsp, cmdDLDataQue)
	ac.assertQuiet(t, from, cmdError, nil)
}

// F6 SCACI §3.10: a persisted downlink is accepted exactly once: a station lost
// during the immediate dispatch does not turn into an error the AC would
// retry, and the downlink is delivered once later.
func TestConformanceF6_DispatchFailureAcceptedOnce(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	// The station vanishes between the paired dlRxStatQry and the dlDataQue
	// frame, so the SC's immediate dispatch fails after persisting the row.
	bs.onRequest(cmdDLRxStatQry, replyRule{action: replyDrop, match: forEndpoint(ep.eui), once: true})
	msg := acDLDataQue(ep, acQueID(), []byte{0x06})
	msg[keyDlRxStatQry] = true
	requireRsp(t, ac.request(t, msg), cmdDLDataQue)

	fresh := connectStation(t, stationA)
	from := fresh.mark()
	requireRsp(t, fresh.request(t, ulDataOpen(ep, uplinkCounter, []byte{0x01})), cmdULData)
	fresh.awaitStationQueue(t, from, ep, []byte{0x06})
	time.Sleep(quietWindow)
	require.Len(t, fresh.framesFrom(from, forCommand(cmdDLDataQue, ep.eui)), 1, "the downlink must be sent exactly once")
}

// F7 and I4 SCACI §3.10 l.499-500: an empty userData queues a pure
// acknowledgement, whether sent as [] or as one empty entry [[]].
func TestConformanceF7I4_EmptyUserDataFromAC(t *testing.T) {
	for name, userData := range map[string][]interface{}{
		"empty array":     {},
		"one empty entry": {[]interface{}{}},
	} {
		t.Run(name, func(t *testing.T) {
			ep := newEndpoint(t, tenantPrimary, preAttached)
			ac := newAC(t)
			bs := connectServing(t, stationA, ep)
			from := bs.mark()
			msg := acDLDataQue(ep, acQueID(), nil)
			msg[keyUserData] = userData
			requireRsp(t, ac.request(t, msg), cmdDLDataQue)
			que := bs.awaitCommand(t, from, cmdDLDataQue, forEndpoint(ep.eui))
			require.Equal(t, [][]byte{{}}, userDataEntries(t, que))
		})
	}
}

// F9 SCACI §3.11.1 l.543-546: dlDataRev names scheduled data by packetCnt and
// revokes every entry scheduled for that counter, and nothing else.
func TestConformanceF9_RevokeByCounter(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	payloads := [][]byte{{0x91}, {0x92}, {0x93}}
	for _, p := range payloads[:2] {
		msg := acDLDataQue(ep, acQueID(), p)
		msg[keyCntDepend] = true
		msg[keyPacketCnt] = []interface{}{revokeCounter}
		requireRsp(t, ac.request(t, msg), cmdDLDataQue)
	}
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), payloads[2])), cmdDLDataQue)
	bssciQueIDs := make([]int64, len(payloads))
	for i, p := range payloads {
		q, _ := bs.awaitStationQueue(t, from, ep, p).int(keyQueID)
		bssciQueIDs[i] = q
	}
	bs.settle(cmdDLDataQueCmp)

	from = bs.mark()
	requireRsp(t, ac.request(t, map[string]interface{}{keyCommand: cmdDLDataRev, keyEpEui: ep.eui, keyPacketCnt: revokeCounter}), cmdDLDataRev)
	time.Sleep(quietWindow)
	revoked := map[int64]bool{}
	for _, f := range bs.framesFrom(from, forCommand(cmdDLDataRev, ep.eui)) {
		q, _ := f.int(keyQueID)
		revoked[q] = true
	}
	require.True(t, revoked[bssciQueIDs[0]] && revoked[bssciQueIDs[1]], "both entries for counter %d must be revoked: %v", revokeCounter, revoked)
	require.False(t, revoked[bssciQueIDs[2]], "the counter-independent entry must stay scheduled")
}

// F10 SCACI §3.11.2-3: dlDataRev is answered with dlDataRevRsp and the AC's
// dlDataRevCmp completes it without an error.
func TestConformanceF10_RevokeHandshake(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x10})), cmdDLDataQue)
	bs.awaitStationQueue(t, from, ep, []byte{0x10})
	bs.settle(cmdDLDataQueCmp)
	fromAC := ac.mark()
	requireRsp(t, ac.request(t, map[string]interface{}{keyCommand: cmdDLDataRev, keyEpEui: ep.eui, keyPacketCnt: uplinkCounter}), cmdDLDataRev)
	ac.assertQuiet(t, fromAC, cmdError, nil)
}

// F11 and F13 SCACI §3.12.1 l.572-577, §3.12.2-3: dlDataRes carries bsEui,
// txTime and packetCnt only for sent; the AC's txDataResRsp is completed with
// txDataResCmp.
func TestConformanceF11F13_DLDataResult(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x11})), cmdDLDataQue)
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x12})), cmdDLDataQue)
	sentQ, _ := bs.awaitStationQueue(t, from, ep, []byte{0x11}).int(keyQueID)
	expiredQ, _ := bs.awaitStationQueue(t, from, ep, []byte{0x12}).int(keyQueID)

	fromAC := ac.mark()
	requireRsp(t, bs.request(t, dlDataResSent(ep.eui, sentQ, uplinkCounter)), cmdDLDataRes)
	sent := ac.awaitCommand(t, fromAC, cmdDLDataRes, func(f wireFrame) bool { return forEndpoint(ep.eui)(f) && f.fields[keyResult] == resultSent })
	bsEui, _ := sent.uint(keyBsEui)
	cnt, _ := sent.int(keyPacketCnt)
	require.Equal(t, stationA.eui, bsEui, "sent result bsEui: %v", sent.fields)
	require.True(t, sent.has(keyTxTime), "sent result lacks txTime: %v", sent.fields)
	require.EqualValues(t, uplinkCounter, cnt)
	ac.awaitCommand(t, fromAC, cmdTxDataResCmp, func(f wireFrame) bool { return f.opID == sent.opID })

	requireRsp(t, bs.request(t, dlDataResMsg(ep.eui, expiredQ, resultExpired)), cmdDLDataRes)
	expired := ac.awaitCommand(t, fromAC, cmdDLDataRes, func(f wireFrame) bool { return forEndpoint(ep.eui)(f) && f.fields[keyResult] == resultExpired })
	for _, key := range []string{keyBsEui, keyTxTime, keyPacketCnt} {
		require.False(t, expired.has(key), "expired result carries %s: %v", key, expired.fields)
	}
}

// F12 SCACI §3.12.1 l.573: dlDataRes names the downlink by the queId the AC
// assigned, so an AC can always tell its own result from any other.
func TestConformanceF12_ResultCorrelatesToACQueueID(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	apiQueID := queueDownlink(t, ep, dlRequest([]byte{0x21}))
	// The AC picks the queId the SC gave the API downlink.
	requireRsp(t, ac.request(t, acDLDataQue(ep, uint64(apiQueID), []byte{0x22})), cmdDLDataQue) //nolint:gosec // positive queue id
	apiBSSCI, _ := bs.awaitStationQueue(t, from, ep, []byte{0x21}).int(keyQueID)
	acBSSCI, _ := bs.awaitStationQueue(t, from, ep, []byte{0x22}).int(keyQueID)
	fromAC := ac.mark()
	requireRsp(t, bs.request(t, dlDataResMsg(ep.eui, apiBSSCI, resultExpired)), cmdDLDataRes)
	requireRsp(t, bs.request(t, dlDataResSent(ep.eui, acBSSCI, uplinkCounter)), cmdDLDataRes)
	time.Sleep(quietWindow)
	own := ac.framesFrom(fromAC, func(f wireFrame) bool {
		q, _ := f.int(keyQueID)
		return f.command == cmdDLDataRes && forEndpoint(ep.eui)(f) && q == apiQueID
	})
	require.Len(t, own, 1, "results the AC receives under its queId")
	require.Equal(t, resultSent, own[0].fields[keyResult], "the result under the AC's queId is not the AC's downlink")
}

// F14 SCACI §1 l.118-123, §3.12 l.562-564: a result produced while the AC
// was offline reaches it when it resumes the session, and a new session
// starts from discarded state.
func TestConformanceF14_ResultWhileACOffline(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x14})), cmdDLDataQue)
	q, _ := bs.awaitStationQueue(t, from, ep, []byte{0x14}).int(keyQueID)
	ac.close()
	requireRsp(t, bs.request(t, dlDataResSent(ep.eui, q, uplinkCounter)), cmdDLDataRes)

	back := connectAC(t, ac.eui, ac)
	res := back.awaitCommand(t, 0, cmdDLDataRes, forEndpoint(ep.eui))
	require.Less(t, res.opID, int64(0), "the result is a service center operation")
}

// F14b SCACI §1 l.121-123: a new session discards the previous session's
// state, so a result produced while the AC was offline is not sent to it.
func TestConformanceF14b_NewSessionGetsNoOfflineResult(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x14})), cmdDLDataQue)
	q, _ := bs.awaitStationQueue(t, from, ep, []byte{0x14}).int(keyQueID)
	ac.close()
	requireRsp(t, bs.request(t, dlDataResSent(ep.eui, q, uplinkCounter)), cmdDLDataRes)

	fresh := connectAC(t, ac.eui, nil)
	fresh.assertQuiet(t, 0, cmdDLDataRes, forEndpoint(ep.eui))
}

// F15 SCACI §3.9 l.458-491: an AC's ulDataTx reaches a station as a BSSCI
// ulDataTx with the same fields.
func TestConformanceF15_ULDataTransmitFromAC(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	from := bs.mark()
	payload := []byte{0x15, 0x16}
	requireRsp(t, ac.request(t, map[string]interface{}{
		keyCommand: cmdULDataTx, keyEpEui: ep.eui, keyNwkSnKey: acBytes(ep.key), keyShAddr: ep.shAddr,
		keyPacketCnt: uplinkCounter, keyUserData: acBytes(payload),
	}), cmdULDataTx)
	tx := bs.awaitCommand(t, from, cmdULDataTx, forEndpoint(ep.eui))
	key, _ := anyBytes(tx.fields[keyNwkSnKey])
	data, _ := anyBytes(tx.fields[keyUserData])
	shAddr, _ := tx.int(keyShAddr)
	cnt, _ := tx.int(keyPacketCnt)
	require.Equal(t, ep.key, key)
	require.Equal(t, payload, data)
	require.EqualValues(t, ep.shAddr, shAddr)
	require.EqualValues(t, uplinkCounter, cnt)
}
