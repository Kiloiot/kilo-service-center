//go:build integration

package probe

import (
	"crypto/tls"
	"net"
	"sync/atomic"
	"testing"
	"time"

	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/require"
)

const (
	acEUIBase   uint64 = 0x70B3D5AC00000000
	simVendor          = "KiloCenter conformance"
	simModel           = "simulator"
	statusOK           = "ok"
	simSNR             = 12.5
	simRSSI            = -80.0
	scaciTLSMin        = tls.VersionTLS13
	bssciTLSMin        = tls.VersionTLS12
	// stationSettle gives the SC time to retire a closed station session, so
	// the next test's downlinks are never routed to it.
	stationSettle = 500 * time.Millisecond
	// settleInterval is the silence that ends a burst of SC frames.
	settleInterval = 700 * time.Millisecond
)

// simBS is a simulated base station (SIM-BS); session is the snBsUuid of its
// BSSCI session.
type simBS struct {
	*peer
	session []byte
}

// connectStation opens a fresh BSSCI session for the station (a new
// snBsUuid): TLS with the station's CA-signed certificate, con (opId 0),
// conRsp, conCmp (BSSCI §3.3).
func connectStation(t *testing.T, st station) *simBS {
	t.Helper()
	return openStation(t, st, randomBytes(t, sessionKeyLen), 0)
}

// resumeStation reconnects the station under prev's snBsUuid, so the SC
// resumes prev's session (BSSCI §1), with operation ids continuing from prev's.
func resumeStation(t *testing.T, st station, prev *simBS) *simBS {
	t.Helper()
	return openStation(t, st, prev.session, prev.nextOp.Load())
}

// openStation connects the station under the session uuid, its own
// operation ids continuing after lastOp.
func openStation(t *testing.T, st station, session []byte, lastOp int64) *simBS {
	t.Helper()
	conn := dialTLS(t, envBSSCIAddr, st.cert, bssciTLSMin)
	t.Cleanup(func() { time.Sleep(stationSettle) }) // runs after the peer's close
	bs := &simBS{peer: newPeer(t, st.cert, conn, mioty.MIOTYFrameIdentifier), session: session}
	bs.nextOp.Store(lastOp)
	bs.replyFields[cmdStatus] = func() map[string]interface{} {
		return map[string]interface{}{keyCode: 0, keyMessage: statusOK, keyTime: time.Now().UnixNano(), keyDutyCycle: 0.0}
	}
	bs.start()
	con := map[string]interface{}{
		keyCommand: cmdCon, keyOpID: 0, keyVersion: protocolVersion, keyBsEui: st.eui, keyVendor: simVendor,
		keyModel: simModel, keyName: st.cert, keyBidi: true, keySnBsUUID: numeric(session),
	}
	from := bs.mark()
	require.NoError(t, bs.write(con))
	rsp := bs.awaitCommand(t, from, cmdConRsp, nil)
	require.NoError(t, bs.write(map[string]interface{}{keyCommand: cmdConCmp, keyOpID: rsp.opID}))
	bs.syncActive(t)
	return bs
}

// connectServing connects the station and returns once it serves every
// endpoint: the SC sent the endpoint's attPrp, completed it with attPrpCmp
// and, before reading the station's next frame, recorded the station as the
// endpoint's attachment, which makes it the endpoint's serving station.
func connectServing(t *testing.T, st station, eps ...testEndpoint) *simBS {
	t.Helper()
	bs := connectStation(t, st)
	for _, ep := range eps {
		prp := bs.awaitCommand(t, 0, cmdAttPrp, forEndpoint(ep.eui))
		bs.awaitCommand(t, 0, cmdAttPrpCmp, func(f wireFrame) bool { return f.opID == prp.opID })
	}
	bs.syncActive(t)
	return bs
}

// syncActive returns once the SC has processed everything the peer sent: the
// SC handles one connection's frames in order, so the answer to a ping means
// the preceding conCmp activated the session (BSSCI §3.4, SCACI §3.4).
func (p *peer) syncActive(t *testing.T) {
	t.Helper()
	requireRsp(t, p.request(t, map[string]interface{}{keyCommand: cmdPing}), cmdPing)
}

// dropAbortively ends the TCP connection with a reset, the way a base station
// that loses power or network does.
func dropAbortively(conn net.Conn) {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		if tcp, isTCP := tlsConn.NetConn().(*net.TCPConn); isTCP {
			_ = tcp.SetLinger(0)
		}
	}
	_ = conn.Close()
}

// simAC is a simulated application center (SIM-AC).
type simAC struct {
	*peer
	eui     uint64
	session []byte
}

// acBytes encodes a byte field of an AC message for rows that test something
// other than byte encoding. The SC decodes these fields only from msgpack bin
// (A4 tests the spec's Numeric arrays and fails while that holds), so the
// remaining rows send bin and measure their own obligation instead.
func acBytes(b []byte) interface{} { return b }

type acOption func(msg map[string]interface{})

// connectAC opens a SCACI session: con (opId 0), conRsp, conCmp (SCACI §3.3).
// With prev set it resumes prev's session: same snAcUuid, snScOpId naming the
// last SC operation prev saw, and AC opIds continuing from prev's.
func connectAC(t *testing.T, eui uint64, prev *simAC, opts ...acOption) *simAC {
	t.Helper()
	conn := dialTLS(t, envSCACIAddr, acCertName, scaciTLSMin)
	ac := &simAC{peer: newPeer(t, acCertName, conn, mioty.SCACIFrameIdentifier), eui: eui, session: randomBytes(t, sessionKeyLen)}
	ac.responseNames[cmdDLDataRes] = cmdTxDataResRsp
	ac.start()
	con := map[string]interface{}{
		keyCommand: cmdCon, keyOpID: 0, keyVersion: protocolVersion, keyAcEui: eui,
		keyVendor: simVendor, keyModel: simModel,
	}
	if prev != nil {
		ac.session = prev.session
		ac.nextOp.Store(prev.nextOp.Load())
		var lastSC int64
		for _, f := range prev.framesFrom(0, func(f wireFrame) bool { return f.opID < 0 }) {
			lastSC = min(lastSC, f.opID)
		}
		// snAcOpId 0 asks the SC to confirm no particular AC operation.
		con[keySnAcOpID] = 0
		con[keySnScOpID] = lastSC
	}
	con[keySnAcUUID] = acBytes(ac.session)
	for _, opt := range opts {
		opt(con)
	}
	from := ac.mark()
	require.NoError(t, ac.write(con))
	rsp, ok := ac.awaitFrom(from, frameWait, func(f wireFrame) bool { return f.command == cmdConRsp || f.command == cmdError })
	require.True(t, ok, "no answer to SCACI con")
	require.Equal(t, cmdConRsp, rsp.command, "SCACI con refused: %v", rsp.fields)
	require.NoError(t, ac.write(map[string]interface{}{keyCommand: cmdConCmp, keyOpID: rsp.opID}))
	ac.syncActive(t)
	return ac
}

var acSeq atomic.Uint64

// newAC connects an application center with its own EUI and session. The SC
// keeps one session per AC EUI, so every simulated AC gets a fresh EUI.
func newAC(t *testing.T) *simAC {
	t.Helper()
	return connectAC(t, acEUIBase|endpointRun|acSeq.Add(1), nil)
}

// Message builders. Every builder emits the spec's mandatory fields in the
// spec's shapes: EUIs as numbers, byte fields as Numeric arrays.

func ulDataMsg(ep testEndpoint, cnt uint32, data []byte) map[string]interface{} {
	return map[string]interface{}{
		keyCommand: cmdULData, keyEpEui: ep.eui, keyRxTime: time.Now().UnixNano(), keyPacketCnt: cnt,
		keySnr: simSNR, keyRssi: simRSSI, keyUserData: numeric(data),
		keyDlOpen: false, keyResponseExp: false, keyDlAck: false,
	}
}

// ulDataOpen is an uplink that opens a downlink window.
func ulDataOpen(ep testEndpoint, cnt uint32, data []byte) map[string]interface{} {
	msg := ulDataMsg(ep, cnt, data)
	msg[keyDlOpen] = true
	return msg
}

func attMsg(t *testing.T, ep testEndpoint, attachCnt uint32, nonce []byte) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		keyCommand: cmdAtt, keyEpEui: ep.eui, keyRxTime: time.Now().UnixNano(), keyAttachCnt: attachCnt,
		keySnr: simSNR, keyRssi: simRSSI, keyNonce: numeric(nonce),
		keySign:     numeric(specAttachSignature(t, ep.eui, attachCnt, ep.key)),
		keyDualChan: false, keyRepetition: false, keyWideCarrOff: false, keyLongBlkDist: false,
	}
}

func detMsg(ep testEndpoint, cnt uint32, sign []byte) map[string]interface{} {
	return map[string]interface{}{
		keyCommand: cmdDet, keyEpEui: ep.eui, keyRxTime: time.Now().UnixNano(), keyPacketCnt: cnt,
		keySnr: simSNR, keyRssi: simRSSI, keySign: numeric(sign),
	}
}

func dlDataResMsg(eui uint64, queID int64, result string) map[string]interface{} {
	return map[string]interface{}{keyCommand: cmdDLDataRes, keyEpEui: eui, keyQueID: queID, keyResult: result}
}

func dlDataResSent(eui uint64, queID int64, cnt uint32) map[string]interface{} {
	msg := dlDataResMsg(eui, queID, resultSent)
	msg[keyTxTime] = time.Now().UnixNano()
	msg[keyPacketCnt] = cnt
	return msg
}

// Frame predicates.

func forEndpoint(eui uint64) func(wireFrame) bool {
	return func(f wireFrame) bool {
		v, ok := f.uint(keyEpEui)
		return ok && v == eui
	}
}

func forQueue(eui uint64, queID int64) func(wireFrame) bool {
	return func(f wireFrame) bool {
		v, ok := f.int(keyQueID)
		return forEndpoint(eui)(f) && ok && v == queID
	}
}

// settle waits until the SC has sent no frame of the command for
// settleInterval, so a burst such as the attPrp reconciliation after connect
// is over before a test marks its starting point.
func (p *peer) settle(command string) {
	deadline := time.Now().Add(frameWait)
	for time.Now().Before(deadline) {
		if _, got := p.awaitFrom(p.mark(), settleInterval, func(f wireFrame) bool { return f.command == command }); !got {
			return
		}
	}
}

// assertQuiet observes the quiet window and fails if the SC sent a frame of
// the command that matches.
func (p *peer) assertQuiet(t *testing.T, from int, command string, match func(wireFrame) bool) {
	t.Helper()
	f, got := p.awaitFrom(from, quietWindow, func(f wireFrame) bool {
		return f.command == command && (match == nil || match(f))
	})
	require.False(t, got, "%s: unexpected %s: %v", p.name, command, f.fields)
}

// requireRsp asserts that the SC answered an operation with its response.
func requireRsp(t *testing.T, got wireFrame, command string) {
	t.Helper()
	require.Equal(t, command+suffixRsp, got.command, "answer to %s: %v", command, got.fields)
}

// requireError asserts that the SC answered an operation with an error frame
// carrying the POSIX code.
func requireError(t *testing.T, got wireFrame, code int) {
	t.Helper()
	require.Equal(t, cmdError, got.command, "answer: %v", got.fields)
	gotCode, _ := got.int(keyCode)
	require.EqualValues(t, code, gotCode, "error code of %v", got.fields)
}
