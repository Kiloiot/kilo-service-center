//go:build integration

package probe

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// MQTT command fields and events of the downlink deadline (KC-MQTT; the MQTT
// ingress is SC-specific, not part of the MIOTY specifications).
const (
	mqttKeyRef            = "ref"
	mqttKeyExpiresAt      = "expiresAt"
	mqttKeyCode           = "code"
	mqttKeyResult         = "result"
	mqttQueuedSuffix      = "/event/downlink_queued"
	mqttRejectedSuffix    = "/event/downlink_rejected"
	deadlinePassedCode    = "mqtt.command.expired"
	sqlExpireCommand      = `UPDATE downlink_queue SET earliest_at = NOW() - INTERVAL '2 hours', latest_at = NOW() - INTERVAL '1 hour' WHERE ref = $1`
	sqlCommandDownlinks   = `SELECT count(*) FROM downlink_queue WHERE ref = $1`
	commandDeadlineMargin = time.Hour
)

// publishCommand publishes an MQTT command/down body for the endpoint.
func (m *mqttTap) publishCommand(t *testing.T, eui uint64, command map[string]interface{}) {
	t.Helper()
	body, err := json.Marshal(command)
	require.NoError(t, err)
	topic := fmt.Sprintf(mqttCommandTopicFmt, stackEnv(t, envMQTTPrefix), tenantOrg(t, tenantPrimary), eui)
	tok := m.client.Publish(topic, mqttQoS, false, body)
	require.True(t, tok.WaitTimeout(dialTimeout), "MQTT publish timed out")
	require.NoError(t, tok.Error(), "MQTT publish")
}

// outcomes lists the events of the kind (topic suffix) that carry the ref.
func (m *mqttTap) outcomes(suffix, ref string) []mqttMessage {
	var matched []mqttMessage
	for _, msg := range m.messages() {
		if strings.HasSuffix(msg.topic, suffix) && msg.payload[mqttKeyRef] == ref {
			matched = append(matched, msg)
		}
	}
	return matched
}

// deadlineCommand is a command/down carrying one byte, the ref and the deadline.
func deadlineCommand(ref string, expiresAt time.Time) map[string]interface{} {
	return map[string]interface{}{mqttKeyData: "AQ==", mqttKeyRef: ref, mqttKeyExpiresAt: expiresAt.UTC().Format(time.RFC3339Nano)}
}

// stackDB opens the conformance stack's database.
func stackDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", stackEnv(t, envDBDSN))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// J1 SC-specific MQTT ingress: a command's ref is accepted once. A repeat
// while its downlink waits, and a repeat after the downlink expired whose own
// deadline has passed by then, queue nothing and publish neither
// downlink_queued nor downlink_rejected.
func TestConformanceJ1_MQTTRefAcceptedOnce(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	tap := newMQTTTap(t, ep.eui)
	db := stackDB(t)
	ref := fmt.Sprintf("j1-%016x", ep.eui)
	command := deadlineCommand(ref, time.Now().Add(commandDeadlineMargin))

	tap.publishCommand(t, ep.eui, command)
	require.True(t, eventually(func() bool { return len(tap.outcomes(mqttQueuedSuffix, ref)) == 1 }), "the first command is not queued")
	tap.publishCommand(t, ep.eui, command)

	_, err := db.Exec(sqlExpireCommand, ref)
	require.NoError(t, err, "age the downlink past its deadline")
	require.True(t, eventually(func() bool { return len(tap.outcomes(mqttResultSuffix, ref)) > 0 }), "the overdue downlink is not reported")
	tap.publishCommand(t, ep.eui, deadlineCommand(ref, time.Now().Add(-commandDeadlineMargin)))
	time.Sleep(quietWindow)

	require.Len(t, tap.outcomes(mqttQueuedSuffix, ref), 1, "downlink_queued events for one ref")
	require.Empty(t, tap.outcomes(mqttRejectedSuffix, ref), "a repeat is never refused")
	var queued int
	require.NoError(t, db.QueryRow(sqlCommandDownlinks, ref).Scan(&queued))
	require.Equal(t, 1, queued, "downlinks queued for one ref")
	require.Equal(t, resultExpired, tap.outcomes(mqttResultSuffix, ref)[0].payload[mqttKeyResult])
}

// J2 SC-specific MQTT ingress: a new command whose deadline already passed
// queues nothing and is refused as expired.
func TestConformanceJ2_MQTTPassedDeadlineRefused(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	tap := newMQTTTap(t, ep.eui)
	ref := fmt.Sprintf("j2-%016x", ep.eui)

	tap.publishCommand(t, ep.eui, deadlineCommand(ref, time.Now().Add(-time.Second)))

	require.True(t, eventually(func() bool { return len(tap.outcomes(mqttRejectedSuffix, ref)) > 0 }), "the command is not refused")
	require.Equal(t, deadlinePassedCode, tap.outcomes(mqttRejectedSuffix, ref)[0].payload[mqttKeyCode])
	require.Empty(t, tap.outcomes(mqttQueuedSuffix, ref))
	var queued int
	require.NoError(t, stackDB(t).QueryRow(sqlCommandDownlinks, ref).Scan(&queued))
	require.Zero(t, queued, "nothing is queued")
}

// Revoke of an overdue downlink a station holds (BSSCI §3.13, §3.14, §3.17).
const (
	revokingStatus      = "revoking"
	cmdDLDataRevRsp     = "dlDataRevRsp"
	posixENOENT         = 2
	posixEIO            = 5
	heldSentCounter     = uint32(77)
	sqlDownlinkStatusOf = `SELECT status FROM downlink_queue WHERE que_id = $1`
)

// heldOverdue queues a downlink at the station serving the endpoint, has the
// station confirm it, and ages it past its lifetime.
func heldOverdue(t *testing.T, bs *simBS, ep testEndpoint) int64 {
	t.Helper()
	from := bs.mark()
	queID := queueDownlink(t, ep, dlRequest([]byte{0x01}))
	bs.dlDataQueFor(t, from, ep, queID)
	bs.awaitCommand(t, from, cmdDLDataQueCmp, nil)
	_, err := stackDB(t).Exec(sqlExpireDownlink, queID)
	require.NoError(t, err, "age the downlink past its lifetime")
	return queID
}

// queueStatus reads the downlink's queue status from the database.
func queueStatus(t *testing.T, queID int64) string {
	t.Helper()
	var status string
	require.NoError(t, stackDB(t).QueryRow(sqlDownlinkStatusOf, queID).Scan(&status))
	return status
}

// confirmRevoke answers the dlDataRev with dlDataRevRsp and awaits the SC's dlDataRevCmp.
func (bs *simBS) confirmRevoke(t *testing.T, rev wireFrame) {
	t.Helper()
	from := bs.mark()
	require.NoError(t, bs.write(map[string]interface{}{keyCommand: cmdDLDataRevRsp, keyOpID: rev.opID}))
	bs.awaitCommand(t, from, cmdDLDataRevCmp, func(f wireFrame) bool { return f.opID == rev.opID })
}

// J3 BSSCI §3.13, §3.14.1: a downlink the station holds when its lifetime
// ends is revoked there and reported expired only after the station confirms
// it dropped it; until then it is revoking and has no result.
func TestConformanceJ3_OverdueHeldDownlinkExpiresOnTheStationsAnswer(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	bs.onRequest(cmdDLDataRev, replyRule{action: replySilently, match: forEndpoint(ep.eui)})
	from := bs.mark()
	queID := heldOverdue(t, bs, ep)

	rev := bs.awaitCommand(t, from, cmdDLDataRev, forQueue(ep.eui, queID))
	time.Sleep(quietWindow)
	require.Equal(t, revokingStatus, queueStatus(t, queID))
	require.Nil(t, downlinkResult(t, ep, queID), "no result before the station answers")
	bs.assertQuiet(t, from, cmdDLDataRev, func(f wireFrame) bool { return forQueue(ep.eui, queID)(f) && f.opID != rev.opID })

	bs.confirmRevoke(t, rev)
	res := awaitDownlinkResult(t, ep, queID)
	require.True(t, res != nil && res.GetResult() == resultExpired, "confirmed revoke not reported expired: %v", res)
}

// J4 BSSCI §3.14.1: a transmission the station reports while the downlink is
// being revoked is its outcome; the station's later revoke confirmation
// changes nothing.
func TestConformanceJ4_SentWhileRevokingIsReportedSent(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	bs.onRequest(cmdDLDataRev, replyRule{action: replySilently, match: forEndpoint(ep.eui)})
	from := bs.mark()
	queID := heldOverdue(t, bs, ep)
	rev := bs.awaitCommand(t, from, cmdDLDataRev, forQueue(ep.eui, queID))

	requireRsp(t, bs.request(t, dlDataResSent(ep.eui, queID, heldSentCounter)), cmdDLDataRes)
	bs.confirmRevoke(t, rev)
	time.Sleep(quietWindow)

	res := downlinkResult(t, ep, queID)
	require.True(t, res != nil && res.GetResult() == resultSent, "transmission not reported sent: %v", res)
}

// J5 BSSCI §3.17: only a refusal whose code says the station does not hold
// the downlink ends it expired; an I/O error proves nothing and leaves it
// revoking without a result.
func TestConformanceJ5_RevokeRefusals(t *testing.T) {
	notHeld := newEndpoint(t, tenantPrimary, preAttached)
	failed := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, notHeld, failed)
	bs.onRequest(cmdDLDataRev, replyRule{action: replyError, code: posixENOENT, match: forEndpoint(notHeld.eui)})
	notHeldQueue := heldOverdue(t, bs, notHeld)
	res := awaitDownlinkResult(t, notHeld, notHeldQueue)
	require.True(t, res != nil && res.GetResult() == resultExpired, "not-held refusal not reported expired: %v", res)

	bs.onRequest(cmdDLDataRev, replyRule{action: replyError, code: posixEIO, match: forEndpoint(failed.eui)})
	from := bs.mark()
	failedQueue := heldOverdue(t, bs, failed)
	bs.awaitCommand(t, from, cmdDLDataRev, forQueue(failed.eui, failedQueue))
	time.Sleep(quietWindow)
	require.Equal(t, revokingStatus, queueStatus(t, failedQueue), "an I/O refusal proves nothing")
	require.Nil(t, downlinkResult(t, failed, failedQueue))
}

// J6 BSSCI §1, §3.13: a station that was offline when the downlink's lifetime
// ended leaves it revoking; it is asked to drop it when it resumes its session,
// and its confirmation ends it expired.
func TestConformanceJ6_DisconnectedHolderAskedOnResume(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	queID := heldOverdue(t, bs, ep)
	dropAbortively(bs.conn)
	require.True(t, eventually(func() bool { return queueStatus(t, queID) == revokingStatus }), "the downlink does not become revoking")
	require.Nil(t, downlinkResult(t, ep, queID), "no result while the holder is offline")

	resumed := resumeStation(t, stationA, bs)
	resume, ok := resumed.awaitFrom(0, frameWait, func(f wireFrame) bool { return f.command == cmdConRsp })
	require.True(t, ok)
	snResume, _ := resume.bool(keySnResume)
	require.True(t, snResume, "the station's session is resumed")
	resumed.awaitCommand(t, 0, cmdDLDataRev, forQueue(ep.eui, queID))
	res := awaitDownlinkResult(t, ep, queID)
	require.True(t, res != nil && res.GetResult() == resultExpired, "revoke after resume not reported expired: %v", res)
}

// J7 BSSCI §1: a station that was offline when the downlink's lifetime ended
// and starts a new session discarded the downlink; it ends expired without a
// dlDataRev.
func TestConformanceJ7_DisconnectedHolderFreshSessionExpires(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	queID := heldOverdue(t, bs, ep)
	dropAbortively(bs.conn)
	require.True(t, eventually(func() bool { return queueStatus(t, queID) == revokingStatus }), "the downlink does not become revoking")

	fresh := connectStation(t, stationA)
	res := awaitDownlinkResult(t, ep, queID)
	require.True(t, res != nil && res.GetResult() == resultExpired, "discarded downlink not reported expired: %v", res)
	fresh.assertQuiet(t, 0, cmdDLDataRev, forQueue(ep.eui, queID))
}

// The sweep runs every protocol.downlink_expiry.sweep_interval (5s, the
// stack's default); a connected holder is asked again within two of them.
const askAgainWait = 3 * 5 * time.Second

// J8 BSSCI §3.13, §3.17: a connected holder that refuses the revoke with a
// code that proves nothing is asked again once a sweep interval passed, with
// a new operation, and the downlink stays revoking without a result.
func TestConformanceJ8_UnansweredRevokeIsAskedAgain(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	bs.onRequest(cmdDLDataRev, replyRule{action: replyError, code: posixEIO, match: forEndpoint(ep.eui)})
	from := bs.mark()
	queID := heldOverdue(t, bs, ep)
	first := bs.awaitCommand(t, from, cmdDLDataRev, forQueue(ep.eui, queID))

	again, ok := bs.awaitFrom(from, askAgainWait, func(f wireFrame) bool {
		return f.command == cmdDLDataRev && forQueue(ep.eui, queID)(f) && f.opID != first.opID
	})

	require.True(t, ok, "the holder is not asked again after it refused")
	require.NotEqual(t, first.opID, again.opID)
	require.Equal(t, revokingStatus, queueStatus(t, queID), "a refusal that proves nothing never ends the downlink")
	require.Nil(t, downlinkResult(t, ep, queID))
}

// J9 BSSCI §1, §3.13: deleting the base station that holds a downlink being
// revoked closes its session, and the downlink, which it can no longer
// transmit, ends expired.
func TestConformanceJ9_DeletedHolderEndsTheRevokeExpired(t *testing.T) {
	_, err := coreClient(t).CreateBaseStation(apiCtx(t, stationDeleted.tenant), &pb.CreateBaseStationRequest{
		Basestation: &pb.BaseStation{BsEui: euiHex(stationDeleted.eui), Name: fmt.Sprintf("conformance-%s", stationDeleted.cert)},
	})
	if status.Code(err) != codes.AlreadyExists {
		require.NoError(t, err, "register station %s", stationDeleted.cert)
	}
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationDeleted, ep)
	bs.onRequest(cmdDLDataRev, replyRule{action: replySilently, match: forEndpoint(ep.eui)})
	from := bs.mark()
	queID := heldOverdue(t, bs, ep)
	bs.awaitCommand(t, from, cmdDLDataRev, forQueue(ep.eui, queID))
	require.Equal(t, revokingStatus, queueStatus(t, queID))

	_, err = coreClient(t).DeleteBaseStation(apiCtx(t, stationDeleted.tenant), &pb.DeleteBaseStationRequest{BsEui: euiHex(stationDeleted.eui)})
	require.NoError(t, err)

	select {
	case <-bs.done:
	case <-time.After(frameWait):
		t.Fatal("the deleted station's session is still open")
	}
	res := awaitDownlinkResult(t, ep, queID)
	require.True(t, res != nil && res.GetResult() == resultExpired, "the deleted holder's downlink is not reported expired: %v", res)
}

// sqlEndpointBidi sets whether an endpoint opens downlink windows.
const sqlEndpointBidi = `UPDATE endpoints SET bidi = $1 WHERE ep_eui = $2`

// J10 SC-specific MQTT ingress: a repeat of an accepted command publishes
// nothing, neither after its endpoint lost its downlink capability nor after
// the endpoint was deleted, although a new command would be refused then.
func TestConformanceJ10_MQTTRepeatIsNeverRefused(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	tap := newMQTTTap(t, ep.eui)
	db := stackDB(t)
	ref := fmt.Sprintf("j10-%016x", ep.eui)
	command := deadlineCommand(ref, time.Now().Add(commandDeadlineMargin))
	tap.publishCommand(t, ep.eui, command)
	require.True(t, eventually(func() bool { return len(tap.outcomes(mqttQueuedSuffix, ref)) == 1 }), "the first command is not queued")

	_, err := db.Exec(sqlEndpointBidi, false, mioty.EUI64Bytes(ep.eui))
	require.NoError(t, err, "the endpoint loses its downlink capability")
	tap.publishCommand(t, ep.eui, command)
	time.Sleep(quietWindow)
	_, err = coreClient(t).DeleteEndPoint(apiCtx(t, ep.tenant), &pb.DeleteEndPointRequest{EpEui: euiHex(ep.eui)})
	require.NoError(t, err)
	tap.publishCommand(t, ep.eui, command)
	time.Sleep(quietWindow)

	require.Len(t, tap.outcomes(mqttQueuedSuffix, ref), 1, "downlink_queued events for one ref")
	require.Empty(t, tap.outcomes(mqttRejectedSuffix, ref), "a repeat is never refused")
	newRef := ref + "-new"
	tap.publishCommand(t, ep.eui, deadlineCommand(newRef, time.Now().Add(commandDeadlineMargin)))
	require.True(t, eventually(func() bool { return len(tap.outcomes(mqttRejectedSuffix, newRef)) == 1 }),
		"a new command for the deleted endpoint is refused")
}
