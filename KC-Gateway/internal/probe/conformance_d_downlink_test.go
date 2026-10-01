//go:build integration

package probe

import (
	"database/sql"
	"math"
	"strconv"
	"testing"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	_ "github.com/lib/pq" // PostgreSQL driver for the D28 lifetime setup
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	formatCustom             = 0xC1 // APP §2.1: custom range 0xC0-0xFF
	uplinkCounter     uint32 = 7
	dlRxSnrReport            = 4.5
	dlRxRssiReport           = -97.0
	sqlExpireDownlink        = `UPDATE downlink_queue SET earliest_at = NOW() - INTERVAL '2 hours', latest_at = NOW() - INTERVAL '1 hour' WHERE que_id = $1`
	queuedStatus             = "queued"
	revokedResult            = "revoked"
)

// dlDataQueFor waits for the station to receive the dlDataQue of the queue.
func (bs *simBS) dlDataQueFor(t *testing.T, from int, ep testEndpoint, queID int64) wireFrame {
	t.Helper()
	return bs.awaitCommand(t, from, cmdDLDataQue, forQueue(ep.eui, queID))
}

// userDataEntries decodes dlDataQue.userData (Numeric[m][n]).
func userDataEntries(t *testing.T, f wireFrame) [][]byte {
	t.Helper()
	outer, ok := f.fields[keyUserData].([]interface{})
	require.True(t, ok, "dlDataQue.userData is not an array: %v", f.fields[keyUserData])
	entries := make([][]byte, 0, len(outer))
	for _, entry := range outer {
		b, isBytes := anyBytes(entry)
		require.True(t, isBytes, "dlDataQue.userData entry %v", entry)
		entries = append(entries, b)
	}
	return entries
}

// D1 and D5 BSSCI §3.12.1 l.614-633: dlDataQue carries epEui, queId, cntDepend
// and userData; packetCnt only with cntDepend, and one userData entry without.
func TestConformanceD1D5_DLDataQueFields(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	queID := queueDownlink(t, ep, dlRequest([]byte{0x01}))
	que := bs.dlDataQueFor(t, from, ep, queID)
	for _, key := range []string{keyEpEui, keyQueID, keyCntDepend, keyUserData} {
		require.True(t, que.has(key), "dlDataQue lacks mandatory %s: %v", key, que.fields)
	}
	require.Subset(t, []string{keyCommand, keyOpID, keyEpEui, keyQueID, keyCntDepend, keyUserData, keyPrio}, keysOf(que.fields),
		"dlDataQue of a plain downlink carries extra keys")
	require.Equal(t, [][]byte{{0x01}}, userDataEntries(t, que))

	from = bs.mark()
	cntQueID := queueDownlink(t, ep, &pb.SendDownlinkRequest{
		CntDepend: true, PacketCnt: []int64{int64(uplinkCounter), int64(uplinkCounter + 1)}, Payloads: [][]byte{{0x01}, {0x02}},
	})
	cnt := bs.dlDataQueFor(t, from, ep, cntQueID)
	counters, ok := cnt.fields[keyPacketCnt].([]interface{})
	require.True(t, ok, "counter-dependent dlDataQue lacks packetCnt[]: %v", cnt.fields)
	require.Len(t, counters, len(userDataEntries(t, cnt)))
}

// D2 BSSCI §3.12.1 l.620: queId is 64-bit, so an AC may choose one above 2^63.
func TestConformanceD2_QueueID64Bit(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	requireRsp(t, ac.request(t, acDLDataQue(ep, uint64AboveInt64, []byte{0x01})), cmdDLDataQue)
}

// D3 BSSCI §3.12.1 l.620, SCACI §3.10.1 l.509: an AC's queId never collides
// with another tenant's downlink, and the SC's BSSCI queIds stay unique.
func TestConformanceD3_QueueIDAcrossTenants(t *testing.T) {
	foreign := newEndpoint(t, tenantSecondary)
	foreignQueID := queueDownlink(t, foreign, dlRequest([]byte{0x04}))
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	requireRsp(t, ac.request(t, acDLDataQue(ep, uint64(foreignQueID), []byte{0x01})), cmdDLDataQue) //nolint:gosec // positive queue id
	que := bs.awaitCommand(t, from, cmdDLDataQue, forEndpoint(ep.eui))
	bssciQueID, _ := que.int(keyQueID)
	require.NotEqual(t, foreignQueID, bssciQueID, "BSSCI queId reused the other tenant's")
}

// D6 BSSCI §3.12 l.610-611: an empty userData queues a pure acknowledgement,
// sent as userData [[]].
func TestConformanceD6_AckOnlyDownlink(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	// The ingress side of an ack-only downlink is I4 and F7; this row uses the
	// one ingress form the SC accepts today so it measures the BSSCI frame.
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{})), cmdDLDataQue)
	que := bs.awaitCommand(t, from, cmdDLDataQue, forEndpoint(ep.eui))
	require.Equal(t, [][]byte{{}}, userDataEntries(t, que))
}

// D7 BSSCI §3.12.1 l.624 and H8 APP §2.1: format is passed through as an
// 8-bit value.
func TestConformanceD7H8_DLFormat(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	req := dlRequest([]byte{0x01})
	req.Format = formatCustom
	que := bs.dlDataQueFor(t, from, ep, queueDownlink(t, ep, req))
	format, ok := que.int(keyFormat)
	require.True(t, ok, "dlDataQue lacks format: %v", que.fields)
	require.EqualValues(t, formatCustom, format)
}

// D8 BSSCI §3.12.1 l.625: prio is a finite single-precision priority; NaN is an
// invalid value and is refused at the API.
func TestConformanceD8_PriorityNaN(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	req := dlRequest([]byte{0x01})
	req.EpEui = euiHex(ep.eui)
	req.Priority = float32(math.NaN())
	_, err := coreClient(t).SendDownlink(apiCtx(t, ep.tenant), req)
	require.Equal(t, codes.InvalidArgument, status.Code(err), "SendDownlink with prio NaN: %v", err)
}

// D9 BSSCI §3.12.1 l.626-633 and H7 RADIO §3.6.6.1: responseExp, responsePrio,
// dlWindReq and expOnly reach the station.
func TestConformanceD9H7_DLFlagsPassThrough(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	req := dlRequest([]byte{0x01})
	req.ResponseExp, req.ResponsePrio, req.DlWindReq, req.ExpOnly = true, true, true, true
	que := bs.dlDataQueFor(t, from, ep, queueDownlink(t, ep, req))
	for _, key := range []string{keyResponseExp, keyResponsePrio, keyDlWindReq, keyExpOnly} {
		v, present := que.bool(key)
		require.True(t, present && v, "dlDataQue.%s must be true: %v", key, que.fields)
	}
}

// D10 and F8 SCACI §3.10.1 l.519, BSSCI §3.16: an AC's dlRxStatQry becomes a
// BSSCI dlRxStatQry operation ahead of the dlDataQue, never a dlDataQue field.
func TestConformanceD10F8_DLRxStatQryPairing(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	msg := acDLDataQue(ep, acQueID(), []byte{0x01})
	msg[keyDlRxStatQry] = true
	requireRsp(t, ac.request(t, msg), cmdDLDataQue)
	qry := bs.awaitCommand(t, from, cmdDLRxStatQry, forEndpoint(ep.eui))
	que := bs.awaitCommand(t, from, cmdDLDataQue, forEndpoint(ep.eui))
	require.False(t, que.has(keyDlRxStatQry), "dlDataQue carries dlRxStatQry: %v", que.fields)
	require.Greater(t, qry.opID, que.opID, "dlRxStatQry must precede dlDataQue (SC opIds decrement)")
}

// D11 BSSCI §3.2 l.231-235: SC operations use negative, strictly decrementing
// opIds.
func TestConformanceD11_SCOperationIDs(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	first := bs.dlDataQueFor(t, from, ep, queueDownlink(t, ep, dlRequest([]byte{0x01})))
	second := bs.dlDataQueFor(t, from, ep, queueDownlink(t, ep, dlRequest([]byte{0x02})))
	require.Negative(t, first.opID)
	require.Less(t, second.opID, first.opID)
}

// D13 BSSCI §3.17 l.773-781, SCACI §3.12 l.562-564: a dlDataQue the station
// answers with error is discarded; the SC records the failure and reports
// invalid to the originating AC.
func TestConformanceD13_StationRejectsQueue(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectServing(t, stationA, ep)
	bs.onRequest(cmdDLDataQue, replyRule{action: replyError, code: posixEINVAL, match: forEndpoint(ep.eui)})
	from := ac.mark()
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x01})), cmdDLDataQue)
	res := ac.awaitCommand(t, from, cmdDLDataRes, forEndpoint(ep.eui))
	require.Equal(t, resultInvalid, res.fields[keyResult])
	require.False(t, res.has(keyBsEui) || res.has(keyTxTime), "invalid result carries sent-only fields: %v", res.fields)
}

// D14 BSSCI §1 l.131-137: a new session discards the station's state, so a
// downlink it held is queued again or reported failed.
func TestConformanceD14_HeldDownlinkAfterFreshSession(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	queID := queueDownlink(t, ep, dlRequest([]byte{0x01}))
	bs.dlDataQueFor(t, from, ep, queID)
	bs.awaitCommand(t, from, cmdDLDataQueCmp, nil)
	dropAbortively(bs.conn)

	fresh := connectStation(t, stationA)
	from = fresh.mark()
	requireRsp(t, fresh.request(t, ulDataOpen(ep, uplinkCounter, []byte{0x01})), cmdULData)
	_, requeued := fresh.awaitFrom(from, frameWait, func(f wireFrame) bool { return f.command == cmdDLDataQue && forQueue(ep.eui, queID)(f) })
	if requeued {
		return
	}
	res := downlinkResult(t, ep, queID)
	require.True(t, res != nil && res.GetResult() == resultInvalid,
		"downlink held by the discarded session was neither re-queued nor reported invalid: %v", res)
}

// D15 and I10 RADIO §3.6.1, §3.7.1.2: a downlink is queued ahead of time at a
// station that can reach the endpoint - the one that last heard it.
func TestConformanceD15I10_QueueAtServingStation(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bsA := connectStation(t, stationA)
	bsB := connectStation(t, stationB)
	requireRsp(t, bsB.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01})), cmdULData)
	fromA, fromB := bsA.mark(), bsB.mark()
	queID := queueDownlink(t, ep, dlRequest([]byte{0x01}))
	bsB.dlDataQueFor(t, fromB, ep, queID)
	bsA.assertQuiet(t, fromA, cmdDLDataQue, forQueue(ep.eui, queID))
}

// D16 RADIO §3.6.1 l.346-348, BSSCI §3.12 l.603-605: one downlink window gets
// at most one deferred downlink, even when two stations receive the uplink.
func TestConformanceD16_OneDownlinkPerWindow(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	queueDownlink(t, ep, dlRequest([]byte{0x01}))
	queueDownlink(t, ep, dlRequest([]byte{0x02}))
	bsA := connectStation(t, stationA)
	bsB := connectStation(t, stationB)
	fromA, fromB := bsA.mark(), bsB.mark()
	uplink := ulDataOpen(ep, uplinkCounter, []byte{0x01})
	requireRsp(t, bsA.request(t, copyMsg(uplink)), cmdULData)
	requireRsp(t, bsB.request(t, copyMsg(uplink)), cmdULData)
	time.Sleep(quietWindow)
	sent := len(bsA.framesFrom(fromA, forCommand(cmdDLDataQue, ep.eui))) + len(bsB.framesFrom(fromB, forCommand(cmdDLDataQue, ep.eui)))
	require.LessOrEqual(t, sent, 1, "dlDataQue frames for one downlink window")
}

// D19 BSSCI §3.13 l.649-675: revoking a downlink the station holds sends
// dlDataRev(epEui, queId), completes it after dlDataRevRsp, and records the
// row as revoked.
func TestConformanceD19_RevokeHeldDownlink(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	queID := queueDownlink(t, ep, dlRequest([]byte{0x01}))
	bs.dlDataQueFor(t, from, ep, queID)
	bs.awaitCommand(t, from, cmdDLDataQueCmp, nil)
	_, err := coreClient(t).RevokeDownlink(apiCtx(t, ep.tenant), &pb.RevokeDownlinkRequest{
		EpEui: euiHex(ep.eui), QueueId: strconv.FormatInt(queID, 10),
	})
	require.NoError(t, err, "RevokeDownlink")
	rev := bs.awaitCommand(t, from, cmdDLDataRev, forQueue(ep.eui, queID))
	require.ElementsMatch(t, []string{keyCommand, keyOpID, keyEpEui, keyQueID}, keysOf(rev.fields))
	bs.awaitCommand(t, from, cmdDLDataRevCmp, func(f wireFrame) bool { return f.opID == rev.opID })
	res := awaitDownlinkResult(t, ep, queID)
	require.NotNil(t, res, "revoked downlink has no terminal row")
	require.Equal(t, revokedResult, res.GetResult())
}

// D20 BSSCI §3.14.1 l.690-696: txTime and packetCnt are required exactly when
// the result is sent.
func TestConformanceD20_ResultConditionalFields(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	sentQueID := queueDownlink(t, ep, dlRequest([]byte{0x01}))
	expiredQueID := queueDownlink(t, ep, dlRequest([]byte{0x02}))
	bs.dlDataQueFor(t, from, ep, expiredQueID)
	requireProtocolError(t, bs.request(t, dlDataResMsg(ep.eui, sentQueID, resultSent)))
	requireRsp(t, bs.request(t, dlDataResMsg(ep.eui, expiredQueID, resultExpired)), cmdDLDataRes)
}

// D23 BSSCI §3.15 l.712-743: dlRxStat is answered and stored.
func TestConformanceD23_DLRxStatStored(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	requireRsp(t, bs.request(t, dlRxStatMsg(ep, uplinkCounter)), cmdDLRxStat)
	var statuses []*pb.DLRXStatus
	require.True(t, eventually(func() bool {
		rsp, err := coreClient(t).GetDLRXStatus(apiCtx(t, ep.tenant), &pb.GetDLRXStatusRequest{EpEui: euiHex(ep.eui)})
		statuses = rsp.GetStatuses()
		return err == nil && len(statuses) > 0
	}), "GetDLRXStatus returned no report")
	require.InDelta(t, dlRxSnrReport, statuses[0].GetDlRxSnr(), 1e-9)
	require.InDelta(t, dlRxRssiReport, statuses[0].GetDlRxRssi(), 1e-9)
}

// D24 BSSCI §3.16 l.745-771: a standalone DL RX status query is a dlRxStatQry
// operation completed with dlRxStatQryCmp.
func TestConformanceD24_DLRxStatQuery(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectStation(t, stationA)
	// The connect reconciliation propagates the endpoint, which makes this
	// station its serving station.
	bs.awaitCommand(t, 0, cmdAttPrp, forEndpoint(ep.eui))
	bs.settle(cmdAttPrp)
	from := bs.mark()
	rsp, err := coreClient(t).QueryDLRXStatus(apiCtx(t, ep.tenant), &pb.QueryDLRXStatusRequest{EpEui: euiHex(ep.eui)})
	require.NoError(t, err, "QueryDLRXStatus")
	require.True(t, rsp.GetQueryInitiated(), "query not initiated: %s", rsp.GetMessage())
	qry := bs.awaitCommand(t, from, cmdDLRxStatQry, forEndpoint(ep.eui))
	require.ElementsMatch(t, []string{keyCommand, keyOpID, keyEpEui}, keysOf(qry.fields))
	bs.awaitCommand(t, from, cmdDLRxStatQryCmp, func(f wireFrame) bool { return f.opID == qry.opID })
}

// D25 BSSCI §3.11 l.564-599: ulDataTx carries the §3.11.1 fields in their
// spec shapes and is completed with ulDataTxCmp.
func TestConformanceD25_ULDataTransmit(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	from := bs.mark()
	payload := []byte{0x0A, 0x0B}
	_, err := coreClient(t).SendULTransmit(apiCtx(t, ep.tenant), &pb.SendULTransmitRequest{
		EpEui: euiHex(ep.eui), UserData: payload, PacketCnt: uplinkCounter, NwkSnKey: ep.key, ShAddr: uint32(ep.shAddr),
	})
	require.NoError(t, err, "SendULTransmit")
	tx := bs.awaitCommand(t, from, cmdULDataTx, forEndpoint(ep.eui))
	require.Subset(t, bssciSCFields[cmdULDataTx], keysOf(tx.fields))
	key, keyOK := numericArray(tx.fields[keyNwkSnKey])
	data, dataOK := numericArray(tx.fields[keyUserData])
	require.True(t, keyOK && dataOK, "ulDataTx byte fields are not Numeric arrays: %v", tx.fields)
	require.Equal(t, ep.key, key)
	require.Equal(t, payload, data)
	shAddr, _ := tx.int(keyShAddr)
	cnt, _ := tx.int(keyPacketCnt)
	require.EqualValues(t, ep.shAddr, shAddr)
	require.EqualValues(t, uplinkCounter, cnt)
	bs.awaitCommand(t, from, cmdULDataTxCmp, func(f wireFrame) bool { return f.opID == tx.opID })
}

// D27 CLAUDE.md note 9, BSSCI §3.3.1: a downlink needs a bidirectional
// endpoint; one for a unidirectional endpoint is refused.
func TestConformanceD27_DownlinkNeedsBidiEndpoint(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, unidirectional)
	req := dlRequest([]byte{0x01})
	req.EpEui = euiHex(ep.eui)
	_, err := coreClient(t).SendDownlink(apiCtx(t, ep.tenant), req)
	require.Equal(t, codes.FailedPrecondition, status.Code(err), "SendDownlink for a unidirectional endpoint: %v", err)
}

// D28 BSSCI §3.14, SCACI §3.12: a downlink past its lifetime is not sent and
// is reported expired.
func TestConformanceD28_DownlinkLifetime(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	queID := queueDownlink(t, ep, dlRequest([]byte{0x01}))
	db, err := sql.Open("postgres", stackEnv(t, envDBDSN))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	_, err = db.Exec(sqlExpireDownlink, queID)
	require.NoError(t, err, "age the downlink past its lifetime")

	bs := connectStation(t, stationA)
	from := bs.mark()
	requireRsp(t, bs.request(t, ulDataOpen(ep, uplinkCounter, []byte{0x01})), cmdULData)
	bs.assertQuiet(t, from, cmdDLDataQue, forQueue(ep.eui, queID))
	res := awaitDownlinkResult(t, ep, queID)
	require.True(t, res != nil && res.GetResult() == resultExpired, "expired downlink not reported expired: %v", res)
}

func forCommand(command string, eui uint64) func(wireFrame) bool {
	return func(f wireFrame) bool { return f.command == command && forEndpoint(eui)(f) }
}

func dlRxStatMsg(ep testEndpoint, cnt uint32) map[string]interface{} {
	return map[string]interface{}{
		keyCommand: cmdDLRxStat, keyEpEui: ep.eui, keyRxTime: time.Now().UnixNano(), keyPacketCnt: cnt,
		keyDlRxSnr: dlRxSnrReport, keyDlRxRssi: dlRxRssiReport,
	}
}

// acQueID is a queue id an AC chooses, unique within the run.
func acQueID() uint64 { return nextEndpointEUI() }

// acDLDataQue is an AC's counter-independent dlDataQue in the shapes the SC
// decodes today (see acBytes).
func acDLDataQue(ep testEndpoint, queID uint64, payload []byte) map[string]interface{} {
	return map[string]interface{}{
		keyCommand: cmdDLDataQue, keyEpEui: ep.eui, keyQueID: queID, keyCntDepend: false,
		keyUserData: []interface{}{acBytes(payload)},
	}
}

// D18 BSSCI §3.12.1 l.625: a downlink's priority reaches the station as prio,
// so the station can send the higher value first.
func TestConformanceD18_PriorityReachesStation(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	for _, prio := range []float32{0, prioHigh} {
		from := bs.mark()
		req := dlRequest([]byte{0x18})
		req.Priority = prio
		que := bs.dlDataQueFor(t, from, ep, queueDownlink(t, ep, req))
		got, present := que.float(keyPrio)
		if !present {
			got = 0
		}
		require.InDelta(t, float64(prio), got, 1e-6, "dlDataQue prio for a downlink queued with priority %v: %v", prio, que.fields)
	}
}
