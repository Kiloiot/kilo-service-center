//go:build integration

package probe

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	eqSnrFractional     = 7.5
	formatMPFBusy       = 0x80 // APP §2.1: M-Bus
	formatOverflow      = maxFormat + 1
	duplicateReceiveGap = 200 * time.Millisecond
)

// requireProtocolError asserts that the SC refused a message as a protocol
// error (BSSCI/SCACI §2.4). The spec pins no POSIX code, so any code naming a
// protocol or value error counts; an internal failure such as EIO does not.
func requireProtocolError(t *testing.T, got wireFrame) {
	t.Helper()
	require.Equal(t, cmdError, got.command, "answer: %v", got.fields)
	code, _ := got.int(keyCode)
	require.Contains(t, []int64{posixEPROTO, posixEINVAL, posixERANGE}, code,
		"error code %d is not a protocol error: %v", code, got.fields)
}

// anyBytes decodes a byte field in either msgpack encoding; rows that are not
// about encoding (A3 is) accept both.
func anyBytes(v interface{}) ([]byte, bool) {
	if b, ok := v.([]byte); ok {
		return b, true
	}
	return numericArray(v)
}

// B1 BSSCI §3.10.1: every ulData mandatory field is required; a missing one is
// a protocol error and nothing is stored, the complete frame is answered.
func TestConformanceB1_ULDataMandatoryFields(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	mandatory := []string{keyEpEui, keyRxTime, keyPacketCnt, keySnr, keyRssi, keyUserData, keyDlOpen, keyResponseExp, keyDlAck}
	for i, key := range mandatory {
		t.Run(key, func(t *testing.T) {
			msg := ulDataMsg(ep, uint32(i+1), []byte{0x01}) //nolint:gosec // small index
			delete(msg, key)
			requireError(t, bs.request(t, msg), posixEPROTO)
		})
	}
	require.Empty(t, storedMessages(t, tenantPrimary, ep.eui), "a rejected ulData must not be stored")
	requireRsp(t, bs.request(t, ulDataMsg(ep, uint32(len(mandatory)+1), []byte{0x01})), cmdULData)
}

// B2 BSSCI §3.10 l.513, §3.10.1 l.535: empty userData is valid and reaches the
// AC as empty user data.
func TestConformanceB2_EmptyUserData(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	from := ac.mark()
	requireRsp(t, bs.request(t, ulDataMsg(ep, 1, nil)), cmdULData)
	ul := ac.awaitCommand(t, from, cmdULData, forEndpoint(ep.eui))
	data, ok := anyBytes(ul.fields[keyUserData])
	require.True(t, ok, "SCACI ulData.userData %v", ul.fields[keyUserData])
	require.Empty(t, data)
}

// B3 BSSCI §3.10.1 l.538 (and SCACI §3.8.1 l.415, E4): responseExp requires
// dlOpen; the combination is a protocol error and is not stored.
func TestConformanceB3_ResponseExpRequiresDlOpen(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	msg := ulDataMsg(ep, 1, []byte{0x01})
	msg[keyResponseExp] = true
	requireError(t, bs.request(t, msg), posixEPROTO)
	require.Empty(t, storedMessages(t, tenantPrimary, ep.eui))
}

// B4 BSSCI §3.10.1 l.536 and H8 APP §2.1: format is an 8-bit identifier; a
// value that does not fit is a protocol error, a valid one reaches the AC.
func TestConformanceB4H8_ULFormat(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	overflow := ulDataMsg(ep, 1, []byte{0x01})
	overflow[keyFormat] = formatOverflow
	requireProtocolError(t, bs.request(t, overflow))

	from := ac.mark()
	valid := ulDataMsg(ep, 2, []byte{0x01})
	valid[keyFormat] = formatMPFBusy
	requireRsp(t, bs.request(t, valid), cmdULData)
	ul := ac.awaitCommand(t, from, cmdULData, forEndpoint(ep.eui))
	format, ok := ul.int(keyFormat)
	require.True(t, ok, "SCACI ulData carries no format: %v", ul.fields)
	require.EqualValues(t, formatMPFBusy, format)
}

// B5 BSSCI §3.10.1, SCACI §3.8.1 l.428: a present eqSnr is stored and reaches
// baseStations[].eqSnr unchanged, fractional part included.
func TestConformanceB5_EqSnrCarried(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	from := ac.mark()
	msg := ulDataMsg(ep, 1, []byte{0x01})
	msg[keyEqSnr] = eqSnrFractional
	requireRsp(t, bs.request(t, msg), cmdULData)

	stored := messageWithCounter(storedMessages(t, tenantPrimary, ep.eui), 1)
	require.NotNil(t, stored, "uplink not stored")
	require.InDelta(t, eqSnrFractional, stored.GetEqSnr(), 1e-9, "stored eqSnr")

	ul := ac.awaitCommand(t, from, cmdULData, forEndpoint(ep.eui))
	stations, _ := ul.fields[keyBaseStations].([]interface{})
	require.Len(t, stations, 1)
	entry, _ := stations[0].(map[string]interface{})
	eqSnr, ok := toFloat64(entry[keyEqSnr])
	require.True(t, ok, "SCACI baseStations[0] carries no eqSnr: %v", entry)
	require.InDelta(t, eqSnrFractional, eqSnr, 1e-9)
}

// B6 BSSCI §2.4, §3.10.1: a present subpackets object with invalid content
// (arrays of unequal length) is a protocol error.
func TestConformanceB6_InvalidSubpackets(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	msg := ulDataMsg(ep, 1, []byte{0x01})
	msg[keySubpackets] = map[string]interface{}{
		keySnr: []interface{}{simSNR, simSNR}, keyRssi: []interface{}{simRSSI}, keyFrequency: []interface{}{868100000},
	}
	requireProtocolError(t, bs.request(t, msg))
}

// B7 BSSCI §3.10.1 l.522: rxTime is a 64-bit ns Unix time; 0 is not one.
func TestConformanceB7_RxTimeRequired(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	msg := ulDataMsg(ep, 1, []byte{0x01})
	msg[keyRxTime] = 0
	requireProtocolError(t, bs.request(t, msg))
}

// B8 BSSCI §3.10.1 l.528, RADIO §3.6.5.3: packetCnt is a 32-bit counter; values
// from 2^31 on are valid and stored.
func TestConformanceB8H3_PacketCounter32Bit(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	requireRsp(t, bs.request(t, ulDataMsg(ep, firstSignedInt32, []byte{0x01})), cmdULData)
	require.NotNil(t, messageWithCounter(storedMessages(t, tenantPrimary, ep.eui), firstSignedInt32),
		"uplink with counter 2^31 not stored")
}

// B10 BSSCI §3.10, SCACI §3.8.1 l.417: one telegram received by two stations is
// one message with two receptions and one SCACI ulData.
func TestConformanceB10_MultiStationDuplicate(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bsA := connectStation(t, stationA)
	bsB := connectStation(t, stationB)
	from := ac.mark()
	requireRsp(t, bsA.request(t, ulDataMsg(ep, 1, []byte{0x42})), cmdULData)
	time.Sleep(duplicateReceiveGap)
	requireRsp(t, bsB.request(t, ulDataMsg(ep, 1, []byte{0x42})), cmdULData)

	msgs := storedMessages(t, tenantPrimary, ep.eui)
	require.Len(t, msgs, 1, "one message row")
	require.Len(t, msgs[0].GetBaseStations(), 2, "two receptions on the message")

	ac.awaitCommand(t, from, cmdULData, forEndpoint(ep.eui))
	time.Sleep(quietWindow)
	require.Len(t, ac.framesFrom(from, func(f wireFrame) bool { return f.command == cmdULData && forEndpoint(ep.eui)(f) }), 1,
		"exactly one SCACI ulData")
}

// B12 RADIO §3.6.5.5 l.802 (H2): a UL MAC payload is at most 200 bytes, so a
// longer userData cannot be genuine and is refused.
func TestConformanceB12H2_ULPayloadLimit(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	bs := connectStation(t, stationA)
	requireProtocolError(t, bs.request(t, ulDataMsg(ep, 1, make([]byte, oversizedRadioLen))))
	require.Empty(t, storedMessages(t, tenantPrimary, ep.eui))
}

// B13 roaming (CLAUDE.md), RADIO §3.7.1: an uplink of a tenant-1 endpoint at a
// tenant-4 station is stored under tenant 1, and the tenant-1 downlink goes out
// through that station in the window.
func TestConformanceB13I10_RoamingUplinkAndDownlink(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	queID := queueDownlink(t, ep, dlRequest([]byte{0x01}))
	roamer := connectStation(t, stationRoamer)
	from := roamer.mark()
	requireRsp(t, roamer.request(t, ulDataOpen(ep, 1, []byte{0x01})), cmdULData)
	roamer.awaitCommand(t, from, cmdDLDataQue, forQueue(ep.eui, queID))
	require.NotNil(t, messageWithCounter(storedMessages(t, tenantPrimary, ep.eui), 1), "uplink not stored under the owner")
}
