package postgres

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// fixtureCommandRef is the ref of the MQTT command that queued the fixture downlinks.
const fixtureCommandRef = "order-17"

// TestDownlinkCommandRef_TravelsWithEveryResult: the ref of the MQTT command
// that queued a downlink comes back from every change that reports the
// downlink's result - a station's result, a station's refusal, the expiry
// sweep and the endpoint's acknowledgement - and a downlink queued without
// one comes back with none.
func TestDownlinkCommandRef_TravelsWithEveryResult(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	const sentRow, refusedRow, expiredRow, withoutRefRow = 887001, 887002, 887003, 887004
	for _, queID := range []int64{sentRow, refusedRow, expiredRow, withoutRefRow} {
		downlink := applicationDownlink(321, orgs[321], queID, nil)
		if queID != withoutRefRow {
			downlink.Ref = fixtureCommandRef
		}
		_, err := downlinks.EnqueueDownlink(ctx, downlink, testDownlinkLifetime)
		require.NoError(t, err)
	}
	_, err := db.Exec(`UPDATE downlink_queue SET status = $1, bs_eui = $2 WHERE que_id IN ($3, $4)`,
		mioty.DLQueueStatusQueued, mioty.EUI64Bytes(releaseStation), sentRow, refusedRow)
	require.NoError(t, err)

	packetCnt := uint32(41)
	sent, err := downlinks.UpdateDownlinkResult(ctx, 321, releaseStation,
		&mioty.DLDataResult{EpEui: fixtureEndpointEUI, QueId: sentRow, Result: mioty.DLDataResultSent, PacketCnt: &packetCnt})
	require.NoError(t, err)
	refused, err := downlinks.FailQueuedDownlink(ctx, refusedRow, 321, releaseStation, "refused")
	require.NoError(t, err)
	acknowledged, marked, err := downlinks.MarkEndpointAcknowledged(ctx, 321, fixtureEndpointEUI, int64(packetCnt))
	require.NoError(t, err)
	require.True(t, marked)
	_, err = db.Exec(`UPDATE downlink_queue SET earliest_at = NOW() - INTERVAL '1 hour', latest_at = NOW() - INTERVAL '1 minute' WHERE que_id IN ($1, $2)`, expiredRow, withoutRefRow)
	require.NoError(t, err)
	expired, err := downlinks.ExpireOverdueDownlinks(ctx, 10)
	require.NoError(t, err)
	require.Len(t, expired, 2)

	reported := map[string]*storage.DownlinkMessage{"station result": sent, "station refusal": refused, "acknowledgement": acknowledged}
	for _, downlink := range expired {
		reported["expiry of "+strconv.FormatInt(downlink.QueID, 10)] = downlink
	}
	for name, downlink := range reported {
		want := fixtureCommandRef
		if downlink.QueID == withoutRefRow {
			want = ""
		}
		assert.Equal(t, want, downlink.Ref, name)
	}
	assert.Equal(t, int64(sentRow), acknowledged.QueID, "the acknowledgement names the downlink sent in the window")
}
