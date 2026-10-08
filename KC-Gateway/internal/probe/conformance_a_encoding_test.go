//go:build integration

package probe

import (
	"strconv"
	"testing"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/stretchr/testify/require"
)

const (
	keyFutureField = "futureField"
	prioFractional = 1.5
	prioHigh       = 0.9
)

// A1 BSSCI §2.5 l.186-192: across a full session the SC adds no field beyond
// the spec table of any command it sends to the station.
func TestConformanceA1_NoExtraFieldsInSCFrames(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	bs.awaitCommand(t, 0, cmdAttPrp, forEndpoint(ep.eui))
	bs.settle(cmdAttPrp)

	full := dlRequest([]byte{0x01})
	full.Format, full.Priority, full.DlRxStatQry = formatCustom, prioFractional, true
	full.ResponseExp, full.ResponsePrio, full.DlWindReq, full.ExpOnly = true, true, true, true
	queID := queueDownlink(t, ep, full)
	queueDownlink(t, ep, &pb.SendDownlinkRequest{CntDepend: true, PacketCnt: []int64{int64(uplinkCounter)}, Payloads: [][]byte{{0x02}}})
	bs.dlDataQueFor(t, 0, ep, queID)
	bs.settle(cmdDLDataQueCmp)
	_, err := coreClient(t).RevokeDownlink(apiCtx(t, ep.tenant), &pb.RevokeDownlinkRequest{EpEui: euiHex(ep.eui), QueueId: strconv.FormatInt(queID, 10)})
	require.NoError(t, err, "RevokeDownlink")
	_, err = coreClient(t).QueryDLRXStatus(apiCtx(t, ep.tenant), &pb.QueryDLRXStatusRequest{EpEui: euiHex(ep.eui)})
	require.NoError(t, err, "QueryDLRXStatus")
	_, err = coreClient(t).SendULTransmit(apiCtx(t, ep.tenant), &pb.SendULTransmitRequest{
		EpEui: euiHex(ep.eui), UserData: []byte{0x03}, PacketCnt: uplinkCounter, NwkSnKey: ep.key, ShAddr: uint32(ep.shAddr), Format: formatCustom,
	})
	require.NoError(t, err, "SendULTransmit")
	_, err = coreClient(t).InitiatePing(apiCtx(t, ep.tenant), &pb.InitiatePingRequest{BsEuiHex: euiHex(stationA.eui)})
	require.NoError(t, err, "InitiatePing")
	bs.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x04}))
	bs.request(t, dlRxStatMsg(ep, uplinkCounter))
	bs.request(t, attMsg(t, ep, firstAttachCnt, randomBytes(t, nonceLen)))
	bs.request(t, detMsg(ep, uplinkCounter+1, randomBytes(t, nonceLen)))
	requireRsp(t, ac.request(t, map[string]interface{}{keyCommand: cmdDereg, keyEpEui: ep.eui}), cmdDereg)
	bs.awaitCommand(t, 0, cmdDetPrp, forEndpoint(ep.eui))
	bs.awaitCommand(t, 0, cmdStatus, nil)
	bs.settle(cmdStatusCmp)

	for _, f := range bs.framesFrom(0, func(wireFrame) bool { return true }) {
		allowed, known := bssciSCFields[f.command]
		require.True(t, known, "the SC sent %q, which the SC never sends to a station: %v", f.command, f.fields)
		require.Subset(t, allowed, keysOf(f.fields), "%s carries a field beyond BSSCI §3: %v", f.command, f.fields)
	}
}

// A2 BSSCI §2.4 l.178-184: an unknown field is ignored; a missing mandatory
// field is a protocol error.
func TestConformanceA2_MessageInterpretation(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	extra := ulDataMsg(ep, 1, []byte{0x01})
	extra[keyFutureField] = 1
	requireRsp(t, bs.request(t, extra), cmdULData)
	missing := ulDataMsg(ep, 2, []byte{0x01})
	delete(missing, keyRssi)
	requireError(t, bs.request(t, missing), posixEPROTO)
}

// A3 SCACI §3.8.1 l.412, §2.5: the SC sends Numeric fields as arrays of
// numbers: ulData.userData, and conRsp.snScUuid (SCACI §3.3.2).
func TestConformanceA3_SCACINumericEncoding(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	t.Run("ulData.userData", func(t *testing.T) {
		bs := connectStation(t, stationA)
		from := ac.mark()
		requireRsp(t, bs.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01, 0xFF})), cmdULData)
		ul := ac.awaitCommand(t, from, cmdULData, forEndpoint(ep.eui))
		data, ok := numericArray(ul.fields[keyUserData])
		require.True(t, ok, "ulData.userData is %T, not a Numeric array", ul.fields[keyUserData])
		require.Equal(t, []byte{0x01, 0xFF}, data)
	})
	t.Run("conRsp.snScUuid", func(t *testing.T) {
		rsp := ac.framesFrom(0, func(f wireFrame) bool { return f.command == cmdConRsp })
		require.Len(t, rsp, 1)
		id, ok := numericArray(rsp[0].fields[keySnScUUID])
		require.True(t, ok && len(id) == sessionKeyLen, "conRsp.snScUuid is %T, not Numeric[16]", rsp[0].fields[keySnScUUID])
	})
}

// A4 and F2 SCACI §2.4, §3.6.1 l.348, §3.9.1 l.476, §3.10.1 l.512: the SC
// accepts the spec's Numeric arrays for byte fields.
func TestConformanceA4F2_SCACINumericArraysAccepted(t *testing.T) {
	t.Run("con.snAcUuid", func(t *testing.T) {
		connectAC(t, acEUIBase|endpointRun|acSeq.Add(1), nil, func(msg map[string]interface{}) {
			id, _ := msg[keySnAcUUID].([]byte)
			msg[keySnAcUUID] = numeric(id)
		})
	})
	ac := newAC(t)
	t.Run("reg.nwkKey", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary)
		msg := regMsg(ep, false)
		msg[keyNwkKey] = numeric(ep.key)
		requireRsp(t, ac.request(t, msg), cmdReg)
	})
	t.Run("ulDataTx", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary)
		connectStation(t, stationA)
		requireRsp(t, ac.request(t, map[string]interface{}{
			keyCommand: cmdULDataTx, keyEpEui: ep.eui, keyNwkSnKey: numeric(ep.key), keyShAddr: ep.shAddr,
			keyPacketCnt: uplinkCounter, keyUserData: numeric([]byte{0x01}),
		}), cmdULDataTx)
	})
	t.Run("dlDataQue.userData", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary)
		msg := acDLDataQue(ep, acQueID(), nil)
		msg[keyUserData] = []interface{}{numeric([]byte{0x01, 0x02})}
		requireRsp(t, ac.request(t, msg), cmdDLDataQue)
	})
}

// A5 SCACI §1 l.125-129, §3: frames may be JSON; a JSON-framed reg and
// dlDataQue are answered like their MessagePack form.
func TestConformanceA5_JSONFrames(t *testing.T) {
	ac := newAC(t)
	ac.encodeJSON.Store(true)
	answer := func(t *testing.T, msg map[string]interface{}) wireFrame {
		t.Helper()
		from := ac.mark()
		op := ac.send(t, msg)
		// An SC that cannot read the opId answers with opId 0.
		f, ok := ac.awaitFrom(from, frameWait, func(f wireFrame) bool {
			return (f.opID == op || f.opID == 0) && (f.command == msg[keyCommand].(string)+suffixRsp || f.command == cmdError)
		})
		require.True(t, ok, "no answer to the JSON %v", msg[keyCommand])
		return f
	}
	t.Run(cmdReg, func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary)
		msg := regMsg(ep, false)
		msg[keyNwkKey] = numeric(ep.key)
		requireRsp(t, answer(t, msg), cmdReg)
	})
	t.Run(cmdDLDataQue, func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary)
		msg := acDLDataQue(ep, acQueID(), nil)
		msg[keyUserData] = []interface{}{numeric([]byte{0x05})}
		requireRsp(t, answer(t, msg), cmdDLDataQue)
	})
}

// A6 and F3 SCACI §2.4, §3.10.1 l.511-514: a present value that does not fit
// its field is a protocol error, and any valid numeric encoding is accepted -
// a float64 prio included.
func TestConformanceA6F3_SCACINumericRanges(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	ac := newAC(t)
	t.Run("format out of range", func(t *testing.T) {
		msg := acDLDataQue(ep, acQueID(), []byte{0x01})
		msg[keyFormat] = oversizedFormat
		requireProtocolError(t, ac.request(t, msg))
	})
	t.Run("float64 prio", func(t *testing.T) {
		bs := connectServing(t, stationA, ep)
		from := bs.mark()
		msg := acDLDataQue(ep, acQueID(), []byte{0x02})
		msg[keyPrio] = prioFractional
		requireRsp(t, ac.request(t, msg), cmdDLDataQue)
		que := bs.awaitStationQueue(t, from, ep, []byte{0x02})
		prio, _ := que.float(keyPrio)
		require.InDelta(t, prioFractional, prio, 1e-6)
	})
}
