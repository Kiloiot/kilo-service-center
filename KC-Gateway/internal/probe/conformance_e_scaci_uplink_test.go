//go:build integration

package probe

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	classZPayload        = 0x42
	secondReceptionDelay = 300 * time.Millisecond
	// deliveryPoll is the SC's uplink delivery poll (1 s by default).
	deliveryPoll = time.Second
)

// receptionWindow is how long the SC holds a new uplink for the receptions
// of the other stations before it is delivered.
func receptionWindow(t *testing.T) time.Duration {
	t.Helper()
	window, err := time.ParseDuration(stackEnv(t, envReceptionWindow))
	require.NoError(t, err)
	return window
}

// outboxSettle covers a new uplink's reception window and the delivery poll.
func outboxSettle(t *testing.T) time.Duration {
	t.Helper()
	return receptionWindow(t) + 2*deliveryPoll
}

func stationEntries(t *testing.T, ul wireFrame) []map[string]interface{} {
	t.Helper()
	raw, ok := ul.fields[keyBaseStations].([]interface{})
	require.True(t, ok, "ulData.baseStations is not an array: %v", ul.fields)
	out := make([]map[string]interface{}, 0, len(raw))
	for _, entry := range raw {
		m, isMap := entry.(map[string]interface{})
		require.True(t, isMap, "baseStations entry %v", entry)
		out = append(out, m)
	}
	return out
}

// E1 SCACI §3.8.1 l.403-417: ulData carries the mandatory fields, the
// receiving station and only spec fields.
func TestConformanceE1_ULDataFields(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	from := ac.mark()
	requireRsp(t, bs.request(t, ulDataMsg(ep, uplinkCounter, []byte{classZPayload})), cmdULData)
	ul := ac.awaitCommand(t, from, cmdULData, forEndpoint(ep.eui))
	for _, key := range []string{keyEpEui, keyBaseStations, keyPacketCnt, keyUserData, keyDlOpen, keyResponseExp, keyDlAck} {
		require.True(t, ul.has(key), "ulData lacks %s: %v", key, ul.fields)
	}
	require.Subset(t, scaciULDataFields, keysOf(ul.fields))
	cnt, _ := ul.int(keyPacketCnt)
	require.EqualValues(t, uplinkCounter, cnt)
	data, _ := anyBytes(ul.fields[keyUserData])
	require.Equal(t, []byte{classZPayload}, data)
	open, _ := ul.bool(keyDlOpen)
	require.False(t, open)
	entries := stationEntries(t, ul)
	require.Len(t, entries, 1)
	require.Subset(t, scaciULDataStationFields, keysOf(entries[0]))
	bsEui, _ := toUint64(entries[0][keyBsEui])
	require.Equal(t, stationA.eui, bsEui)
}

// E2 SCACI §3.8.1 l.410: baseStations lists every receiving station; a copy a
// second station reports 300 ms after the first, inside the SC's reception
// window, is part of the same ulData whenever the delivery poll runs.
func TestConformanceE2_ReceptionsMerged(t *testing.T) {
	window := receptionWindow(t)
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bsA := connectStation(t, stationA)
	bsB := connectStation(t, stationB)
	from := ac.mark()
	firstReported := time.Now()
	requireRsp(t, bsA.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01})), cmdULData)
	time.Sleep(secondReceptionDelay)
	requireRsp(t, bsB.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01})), cmdULData)
	secondStored := time.Since(firstReported)
	require.Less(t, secondStored, window,
		"the second reception was acknowledged %s after the first was reported, outside the %s reception window: the scenario did not run", secondStored, window)
	ul := ac.awaitCommand(t, from, cmdULData, forEndpoint(ep.eui))
	require.Len(t, stationEntries(t, ul), 2, "one ulData with both receptions")
}

// E3 SCACI §3.8.1 l.429-430: dlRxSnr/dlRxRssi report the endpoint's previous
// downlink reception, so only the first uplink after a DL RX status carries it.
func TestConformanceE3_DLRxReportOnce(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	requireRsp(t, bs.request(t, dlRxStatMsg(ep, uplinkCounter)), cmdDLRxStat)
	from := ac.mark()
	for i := uint32(1); i <= 3; i++ {
		requireRsp(t, bs.request(t, ulDataMsg(ep, uplinkCounter+i, []byte{0x01})), cmdULData)
	}
	for i := uint32(1); i <= 3; i++ {
		ul := ac.awaitCommand(t, from, cmdULData, func(f wireFrame) bool {
			cnt, _ := f.int(keyPacketCnt)
			return forEndpoint(ep.eui)(f) && cnt == int64(uplinkCounter+i)
		})
		_, carries := stationEntries(t, ul)[0][keyDlRxSnr]
		require.Equal(t, i == 1, carries, "ulData %d: dlRxSnr present=%v", i, carries)
	}
}

// E5 SCACI §3.8.1 l.417: a reused packet counter is flagged duplicate (or the
// reuse refused), never presented to the AC as fresh data.
func TestConformanceE5_CounterReuseFlagged(t *testing.T) {
	window, err := strconv.Atoi(stackEnv(t, envDuplicateWindowSec))
	require.NoError(t, err)
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	requireRsp(t, bs.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01})), cmdULData)
	time.Sleep(time.Duration(window)*time.Second + time.Second)
	from := ac.mark()
	answer := bs.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01}))
	if answer.command == cmdError {
		return
	}
	ul := ac.awaitCommand(t, from, cmdULData, forEndpoint(ep.eui))
	dup, _ := ul.bool(keyDuplicate)
	require.True(t, dup, "reused counter delivered as fresh data: %v", ul.fields)
}

// E6 SCACI §3.2 l.203-209: an SC operation the AC did not answer is reissued
// with the same opId when the session is resumed.
func TestConformanceE6_ResumeReissuesULData(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	first := newAC(t)
	first.onRequest(cmdULData, replyRule{action: replySilently})
	bs := connectStation(t, stationA)
	requireRsp(t, bs.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01})), cmdULData)
	ul := first.awaitCommand(t, 0, cmdULData, forEndpoint(ep.eui))
	first.close()

	resumed := connectAC(t, first.eui, first)
	again := resumed.awaitCommand(t, 0, cmdULData, forEndpoint(ep.eui))
	require.Equal(t, ul.opID, again.opID, "reissued ulData must keep its opId")
}

// E8 SCACI §1 l.118-123, §3.8: an uplink received while the AC was offline
// reaches it when it resumes the session, after the opIds it already used;
// a new session starts from discarded state and gets nothing.
func TestConformanceE8_ULDataWhileACOffline(t *testing.T) {
	t.Run("resumed session", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary)
		first := newAC(t)
		bs := connectStation(t, stationA)
		requireRsp(t, bs.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x08})), cmdULData)
		seen := first.awaitCommand(t, 0, cmdULData, forEndpoint(ep.eui))
		first.close()
		requireRsp(t, bs.request(t, ulDataMsg(ep, uplinkCounter+1, []byte{0x09})), cmdULData)
		time.Sleep(outboxSettle(t))

		resumed := connectAC(t, first.eui, first)
		missed := resumed.awaitCommand(t, 0, cmdULData, func(f wireFrame) bool {
			cnt, _ := f.int(keyPacketCnt)
			return forEndpoint(ep.eui)(f) && cnt == int64(uplinkCounter+1)
		})
		require.Less(t, missed.opID, seen.opID, "service center opIds keep decrementing across the resume")
	})
	t.Run("new session", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary)
		first := newAC(t)
		bs := connectStation(t, stationA)
		first.close()
		requireRsp(t, bs.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x08})), cmdULData)
		// The uplink is processed while the AC is offline; a connected session is served live.
		time.Sleep(outboxSettle(t))

		fresh := connectAC(t, first.eui, nil)
		fresh.assertQuiet(t, 0, cmdULData, forEndpoint(ep.eui))
	})
}

// E7 tenant isolation (CLAUDE.md): uplinks reach only the owner tenant's ACs;
// the CE default tenant's AC gets nothing for a tenant-4 endpoint.
func TestConformanceE7_UplinkTenantIsolation(t *testing.T) {
	ep := newEndpoint(t, tenantSecondary)
	ac := newAC(t)
	roamer := connectStation(t, stationRoamer)
	from := ac.mark()
	requireRsp(t, roamer.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01})), cmdULData)
	time.Sleep(outboxSettle(t))
	ac.assertQuiet(t, from, cmdULData, forEndpoint(ep.eui))
}
