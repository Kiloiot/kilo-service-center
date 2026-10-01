package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// fixtureApplicationCenterEUI is the Application Center the origin tests queue from.
const fixtureApplicationCenterEUI uint64 = 0x70B3D59CD00A0321

// queuedByApplicationCenter is a downlink the fixture Application Center
// queued under acQueID.
func queuedByApplicationCenter(orgID uuid.UUID, queID int64, acQueID uint64) *storage.DownlinkMessage {
	downlink := applicationDownlink(321, orgID, queID, &acQueID)
	acEUI := fixtureApplicationCenterEUI
	downlink.ACEUI = &acEUI
	return downlink
}

// TestDownlinkOrigin_TravelsWithTheRow: the Application Center that queued a
// downlink comes back from the lookups and from every change that reports the
// downlink's result - a station's result, a station's refusal and the
// expiry sweep - and a downlink enqueued without one comes back with none.
func TestDownlinkOrigin_TravelsWithTheRow(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	org := orgs[321]
	const sentRow, refusedRow, expiredRow, internalRow = 884001, 884002, 884003, 884004

	scheduled := queuedByApplicationCenter(org, sentRow, 1)
	scheduled.CntDepend = true
	scheduled.PacketCntArray = []int64{5}
	scheduled.UserData = [][]byte{{0x01}}
	for _, downlink := range []*storage.DownlinkMessage{
		scheduled, queuedByApplicationCenter(org, refusedRow, 2), queuedByApplicationCenter(org, expiredRow, 3),
		applicationDownlink(321, org, internalRow, nil),
	} {
		_, err := downlinks.EnqueueDownlink(ctx, downlink, testDownlinkLifetime)
		require.NoError(t, err)
	}
	assertOrigin := func(name string, downlink *storage.DownlinkMessage) {
		t.Helper()
		require.NotNil(t, downlink.ACEUI, name)
		assert.Equal(t, fixtureApplicationCenterEUI, *downlink.ACEUI, name)
	}

	byQueue, err := downlinks.GetDownlinkByQueueID(ctx, sentRow, "321")
	require.NoError(t, err)
	assertOrigin("lookup by queue id", byQueue)
	byPacketCnt, err := downlinks.GetDownlinksByPacketCnt(ctx, "321", "70B3D59CD0000321", 5)
	require.NoError(t, err)
	require.Len(t, byPacketCnt, 1)
	assertOrigin("lookup by packet counter", byPacketCnt[0])

	_, err = db.Exec(`UPDATE downlink_queue SET status = $1, bs_eui = $2 WHERE que_id IN ($3, $4)`,
		mioty.DLQueueStatusQueued, mioty.EUI64Bytes(releaseStation), sentRow, refusedRow)
	require.NoError(t, err)
	sent, err := downlinks.UpdateDownlinkResult(ctx, 321, releaseStation,
		&mioty.DLDataResult{EpEui: fixtureEndpointEUI, QueId: sentRow, Result: mioty.DLDataResultSent})
	require.NoError(t, err)
	assertOrigin("station result", sent)

	refused, err := downlinks.FailQueuedDownlink(ctx, refusedRow, 321, releaseStation, "refused")
	require.NoError(t, err)
	assertOrigin("station refusal", refused)

	_, err = db.Exec(`UPDATE downlink_queue SET earliest_at = NOW() - INTERVAL '1 hour', latest_at = NOW() - INTERVAL '1 minute' WHERE que_id IN ($1, $2)`, expiredRow, internalRow)
	require.NoError(t, err)
	expired, err := downlinks.ExpireOverdueUnheld(ctx, 10)
	require.NoError(t, err)
	require.Len(t, expired, 2)
	for _, downlink := range expired {
		if downlink.QueID == expiredRow {
			assertOrigin("expiry", downlink)
			continue
		}
		assert.Nil(t, downlink.ACEUI, "a downlink enqueued without an Application Center expires without one")
	}

	internal, err := downlinks.GetDownlinkByQueueID(ctx, internalRow, "321")
	require.NoError(t, err)
	assert.Nil(t, internal.ACEUI, "a downlink enqueued without an Application Center has none")
}
