package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// revokingAnswer is the answer of station to the dlDataRev of the tenant 321
// downlink queID.
func revokingAnswer(queID int64, station uint64) storage.DownlinkRevocation {
	return storage.DownlinkRevocation{QueID: queID, TenantID: 321, Station: &station}
}

// TestExpireRevokedDownlink_EndsOnlyTheHoldersRevokingDownlink: the holder's
// answer ends a revoking downlink expired once and returns it for its
// originators; another station's answer, a repeated answer and an answer for
// a downlink no revoke is under way for change nothing.
func TestExpireRevokedDownlink_EndsOnlyTheHoldersRevokingDownlink(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	seedHeldDownlink(t, db, 321, orgs[321], 850001, mioty.DLQueueStatusRevoking, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 850002, mioty.DLQueueStatusQueued, releaseStation)
	_, err := db.Exec(`UPDATE downlink_queue SET ref = 'order-85' WHERE que_id = 850001`)
	require.NoError(t, err)

	_, ended, err := downlinks.ExpireRevokedDownlink(ctx, revokingAnswer(850001, neighborStation))
	require.NoError(t, err)
	assert.False(t, ended, "another station's answer")
	_, ended, err = downlinks.ExpireRevokedDownlink(ctx, revokingAnswer(850002, releaseStation))
	require.NoError(t, err)
	assert.False(t, ended, "a queued downlink no revoke is under way for")

	expired, ended, err := downlinks.ExpireRevokedDownlink(ctx, revokingAnswer(850001, releaseStation))
	require.NoError(t, err)
	require.True(t, ended)
	assert.Equal(t, int64(850001), expired.QueID)
	assert.Equal(t, "order-85", expired.Ref, "the result reaches the command's client")
	assert.Equal(t, releaseStation, expired.BsEui)
	assert.Equal(t, mioty.ResultExpired, expired.Result)
	status, result := downlinkStatusOf(t, db, 850001)
	assert.Equal(t, string(mioty.DLQueueStatusExpired), status)
	require.NotNil(t, result)
	assert.Equal(t, mioty.ResultExpired, *result)

	_, ended, err = downlinks.ExpireRevokedDownlink(ctx, revokingAnswer(850001, releaseStation))
	require.NoError(t, err)
	assert.False(t, ended, "a repeated answer ends nothing again")
	assert.Equal(t, string(mioty.DLQueueStatusQueued), heldDownlink(t, db, 850002).Status)
}

// TestRevokingDownlink_ASentResultWins: a dlDataRes "sent" for a downlink
// being revoked is its outcome (BSSCI §3.14.1); the station's revoke answer
// that follows ends nothing, and neither its confirmation nor a refusal
// revokes it.
func TestRevokingDownlink_ASentResultWins(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	seedHeldDownlink(t, db, 321, orgs[321], 850011, mioty.DLQueueStatusRevoking, releaseStation)
	packetCnt := uint32(7)

	sent, err := downlinks.UpdateDownlinkResult(ctx, 321, releaseStation,
		&mioty.DLDataResult{EpEui: 850011, QueId: 850011, Result: mioty.DLDataResultSent, PacketCnt: &packetCnt})
	require.NoError(t, err)
	assert.Equal(t, int64(850011), sent.QueID)

	_, ended, err := downlinks.ExpireRevokedDownlink(ctx, revokingAnswer(850011, releaseStation))
	require.NoError(t, err)
	assert.False(t, ended)
	revoked, err := downlinks.RevokeDownlink(ctx, revokingAnswer(850011, releaseStation))
	require.NoError(t, err)
	assert.False(t, revoked)
	assert.Equal(t, string(mioty.DLQueueStatusTransmitted), heldDownlink(t, db, 850011).Status)

	_, err = downlinks.UpdateDownlinkResult(ctx, 321, releaseStation,
		&mioty.DLDataResult{EpEui: 850011, QueId: 850011, Result: mioty.DLDataResultSent, PacketCnt: &packetCnt})
	assert.ErrorIs(t, err, storage.ErrDownlinkFinished, "a repeated result changes nothing")
}

// TestRevokeDownlink_StationAnswerLeavesARevokingDownlinkToTheExpiry: the
// revoke answer path that ends a downlink revoked never takes a downlink being
// revoked for its lifetime, which ends expired instead.
func TestRevokeDownlink_StationAnswerLeavesARevokingDownlinkToTheExpiry(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedHeldDownlink(t, db, 321, orgs[321], 850021, mioty.DLQueueStatusRevoking, releaseStation)

	revoked, err := downlinks.RevokeDownlink(t.Context(), revokingAnswer(850021, releaseStation))

	require.NoError(t, err)
	assert.False(t, revoked)
	assert.Equal(t, string(mioty.DLQueueStatusRevoking), heldDownlink(t, db, 850021).Status)
}

// TestStationRevocations_SurviveAReconnectUntilSettled: no release returns a
// revoking downlink to pending; the station's revocations of every tenant
// are listed for a resumed session and expired for one that is not resumed,
// once.
func TestStationRevocations_SurviveAReconnectUntilSettled(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	seedHeldDownlink(t, db, 321, orgs[321], 850031, mioty.DLQueueStatusRevoking, releaseStation)
	seedHeldDownlink(t, db, 322, orgs[322], 850032, mioty.DLQueueStatusRevoking, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 850033, mioty.DLQueueStatusRevoking, neighborStation)

	_, err := downlinks.ReleaseStationReservations(ctx, releaseStation, nil)
	require.NoError(t, err)
	_, err = downlinks.ReleaseStationQueue(ctx, releaseStation)
	require.NoError(t, err)
	_, err = downlinks.ReleaseEndpointAtStation(ctx, 321, 850031, releaseStation, time.Now())
	require.NoError(t, err)
	for _, queID := range []int64{850031, 850032, 850033} {
		assert.Equal(t, string(mioty.DLQueueStatusRevoking), heldDownlink(t, db, queID).Status, "queue id %d", queID)
	}

	listed, err := downlinks.ListStationRevocations(ctx, releaseStation)
	require.NoError(t, err)
	require.Len(t, listed, 2)
	assert.Equal(t, []int64{850031, 850032}, []int64{listed[0].QueID, listed[1].QueID})
	assert.Equal(t, "322", listed[1].TenantID, "a roaming station drops another tenant's downlink too")
	assert.Equal(t, releaseStation, listed[0].BsEui)

	expired, err := downlinks.ExpireStationRevocations(ctx, releaseStation)
	require.NoError(t, err)
	require.Len(t, expired, 2)
	for _, downlink := range expired {
		assert.Equal(t, mioty.ResultExpired, downlink.Result)
		assert.Equal(t, string(mioty.DLQueueStatusExpired), heldDownlink(t, db, downlink.QueID).Status)
	}
	again, err := downlinks.ExpireStationRevocations(ctx, releaseStation)
	require.NoError(t, err)
	assert.Empty(t, again, "each is expired once")
	assert.Equal(t, string(mioty.DLQueueStatusRevoking), heldDownlink(t, db, 850033).Status, "another station's revocation is kept")
}

// TestRevokingDownlink_LateQueueAnswersAreAccepted: the dlDataQueRsp of a
// downlink whose revoke is under way, and its dispatcher's repeated queued
// confirmation, succeed without moving it back to queued.
func TestRevokingDownlink_LateQueueAnswersAreAccepted(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	org := orgs[321]
	seedHeldDownlink(t, db, 321, org, 850041, mioty.DLQueueStatusRevoking, releaseStation)

	require.NoError(t, downlinks.UpdateDownlinkBaseStation(ctx, 850041, 321, releaseStation))
	require.NoError(t, downlinks.MarkReservedAsQueued(ctx, 850041, 321, releaseStation, time.Now().UnixNano(), nil, &org))

	assert.Equal(t, string(mioty.DLQueueStatusRevoking), heldDownlink(t, db, 850041).Status)
}

// TestUpdateDownlinkResult_ASentResultContradictingAnExpiryIsTold: a "sent"
// for a downlink that already ended expired keeps the expiry and is told
// apart from a result for a downlink that ended otherwise.
func TestUpdateDownlinkResult_ASentResultContradictingAnExpiryIsTold(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	seedHeldDownlink(t, db, 321, orgs[321], 850051, mioty.DLQueueStatusExpired, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 850052, mioty.DLQueueStatusTransmitted, releaseStation)
	packetCnt := uint32(9)

	_, err := downlinks.UpdateDownlinkResult(ctx, 321, releaseStation,
		&mioty.DLDataResult{EpEui: 850051, QueId: 850051, Result: mioty.DLDataResultSent, PacketCnt: &packetCnt})
	assert.ErrorIs(t, err, storage.ErrDownlinkExpiredBeforeResult)
	assert.ErrorIs(t, err, storage.ErrDownlinkFinished)
	assert.Equal(t, string(mioty.DLQueueStatusExpired), heldDownlink(t, db, 850051).Status, "the expiry stands")

	_, err = downlinks.UpdateDownlinkResult(ctx, 321, releaseStation,
		&mioty.DLDataResult{EpEui: 850052, QueId: 850052, Result: mioty.DLDataResultSent, PacketCnt: &packetCnt})
	assert.ErrorIs(t, err, storage.ErrDownlinkFinished)
	assert.NotErrorIs(t, err, storage.ErrDownlinkExpiredBeforeResult)
}

// TestClaimUnansweredRevocations_AsksAConnectedHolderAgainOncePerInterval: a
// revoking downlink whose connected holder was asked at least the interval ago
// is claimed once and its ask time moves on; one asked more recently, one held
// by a station that is not connected and one that is not revoking are not.
func TestClaimUnansweredRevocations_AsksAConnectedHolderAgainOncePerInterval(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	const interval = time.Minute
	seedHeldDownlink(t, db, 322, orgs[322], 850061, mioty.DLQueueStatusRevoking, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 850062, mioty.DLQueueStatusRevoking, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 850063, mioty.DLQueueStatusRevoking, neighborStation)
	seedHeldDownlink(t, db, 321, orgs[321], 850064, mioty.DLQueueStatusQueued, releaseStation)
	_, err := db.Exec(`UPDATE downlink_queue SET revoke_asked_at = now() - interval '1 hour' WHERE que_id IN (850061, 850063, 850064)`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE downlink_queue SET revoke_asked_at = now() WHERE que_id = 850062`)
	require.NoError(t, err)

	claimed, err := downlinks.ClaimUnansweredRevocations(ctx, []uint64{releaseStation}, interval, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, int64(850061), claimed[0].QueID)
	assert.Equal(t, "322", claimed[0].TenantID, "a roaming tenant's downlink is asked for too")
	assert.Equal(t, releaseStation, claimed[0].BsEui)
	assert.Equal(t, mioty.DLQueueStatusRevoking, claimed[0].Status)

	again, err := downlinks.ClaimUnansweredRevocations(ctx, []uint64{releaseStation, neighborStation}, interval, 10)
	require.NoError(t, err)
	require.Len(t, again, 1, "the claimed one waits another interval")
	assert.Equal(t, int64(850063), again[0].QueID)
	assert.Equal(t, string(mioty.DLQueueStatusRevoking), heldDownlink(t, db, 850061).Status, "a claim changes no state")

	none, err := downlinks.ClaimUnansweredRevocations(ctx, nil, interval, 10)
	require.NoError(t, err)
	assert.Empty(t, none, "no connected station")
}
