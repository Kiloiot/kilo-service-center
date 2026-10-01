package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// counterDependentEntries are the per-counter user data a downlink queues.
var counterDependentEntries = [][]byte{{0x0A, 0x0B}, {0x0C}}

// counterDependentDocument is a user_data column holding only the entries,
// base64 as JSON renders bytes, with no protocol envelope around them.
const counterDependentDocument = `{"userData": ["Cgs=", "DA=="]}`

// TestDownlinkUserData_EveryReadKeepsTheCounterDependentEntries: the user
// data of a counter-dependent downlink (SCACI §3.10.1, one entry per packet
// counter) reaches every read of the row, the queue listing, the dispatch
// reservation and the results, whether the service center wrote the column
// or it holds only the entries.
func TestDownlinkUserData_EveryReadKeepsTheCounterDependentEntries(t *testing.T) {
	repos, db, orgs := downlinkRepositoriesFixture(t)
	ctx := t.Context()
	message := applicationDownlink(321, orgs[321], 850001, nil)
	message.CntDepend = true
	message.PacketCntArray = []int64{7, 8}
	message.UserData = counterDependentEntries
	message.Payload = counterDependentEntries[0]
	_, err := repos.Downlinks.EnqueueDownlink(ctx, message, testDownlinkLifetime)
	require.NoError(t, err)
	stored, err := repos.DownlinkQueueReader.ListTenantQueue(ctx, 321, storage.DownlinkQueueFilter{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, counterDependentEntries, stored[0].UserData, "as written by the service center")
	_, err = db.Exec(`UPDATE downlink_queue SET user_data = $1 WHERE que_id = 850001`, counterDependentDocument)
	require.NoError(t, err)

	listed, err := repos.DownlinkQueueReader.ListTenantQueue(ctx, 321, storage.DownlinkQueueFilter{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, counterDependentEntries, listed[0].UserData, "queue listing")

	reserved, err := repos.Downlinks.ReservePendingDownlinkByQueueID(ctx, 321, orgs[321], 850001, mioty.EUI64Bytes(fixtureEndpointEUI), 0x70B3D59CD0000A01)
	require.NoError(t, err)
	assert.Equal(t, counterDependentEntries, reserved.UserData, "dispatch reservation")

	_, err = db.Exec(`UPDATE downlink_queue SET status = $1 WHERE que_id = 850001`, mioty.DLQueueStatusTransmitted)
	require.NoError(t, err)
	results, _, err := repos.Downlinks.GetDownlinkResults(ctx, 321, nil, storage.DownlinkResultFilter{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, counterDependentEntries, results[0].UserData, "results")
}

// TestGetDownlinkByQueueID_ReadsEveryPayloadWithItsCounter: the lookup an
// event is built from carries the user data of each packet counter.
func TestGetDownlinkByQueueID_ReadsEveryPayloadWithItsCounter(t *testing.T) {
	repos, _, orgs := downlinkRepositoriesFixture(t)
	ctx := t.Context()
	message := applicationDownlink(321, orgs[321], 850011, nil)
	message.CntDepend = true
	message.PacketCntArray = []int64{74, 75}
	message.UserData = counterDependentEntries
	message.Payload = counterDependentEntries[0]
	_, err := repos.Downlinks.EnqueueDownlink(ctx, message, testDownlinkLifetime)
	require.NoError(t, err)

	stored, err := repos.Downlinks.GetDownlinkByQueueID(ctx, 850011, "321")

	require.NoError(t, err)
	assert.True(t, stored.CntDepend)
	assert.Equal(t, []int64{74, 75}, stored.PacketCntArray)
	assert.Equal(t, counterDependentEntries, stored.UserData)
	assert.Equal(t, counterDependentEntries[0], stored.Payload)
}

// TestDownlinkUserData_UndecodableColumnIsAnError: a user_data document the
// repository cannot read fails the read instead of dispatching the downlink
// without its entries.
func TestDownlinkUserData_UndecodableColumnIsAnError(t *testing.T) {
	repos, db, orgs := downlinkRepositoriesFixture(t)
	ctx := t.Context()
	message := applicationDownlink(321, orgs[321], 850002, nil)
	message.CntDepend = true
	message.PacketCntArray = []int64{7}
	message.UserData = [][]byte{{0x01}}
	_, err := repos.Downlinks.EnqueueDownlink(ctx, message, testDownlinkLifetime)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE downlink_queue SET user_data = '{"userData": 5}' WHERE que_id = 850002`)
	require.NoError(t, err)

	_, err = repos.DownlinkQueueReader.ListTenantQueue(ctx, 321, storage.DownlinkQueueFilter{}, 10, 0)
	assert.Error(t, err)
}
