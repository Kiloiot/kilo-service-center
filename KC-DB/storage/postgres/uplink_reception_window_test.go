package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	uplinkStoreReceptionWindow = time.Second
	// uplinkStoreSecondArrival is when the second base station's report of the
	// telegram arrives, inside the reception window.
	uplinkStoreSecondArrival = 300 * time.Millisecond
	uplinkStoreWindowCounter = uint32(61)
)

// The delivery of a new uplink is due when its reception window closes, so
// the report of a second base station that arrives meanwhile is merged into
// the one ulData the Application Centers receive, which lists every
// receiving station (SCACI §3.8.1 baseStations).
func TestUplinkStore_DeliveryWaitsForTheReceptionWindow(t *testing.T) {
	f := newUplinkStoreFixture(t)
	outbox := NewMessageDeliveryOutboxRepository(f.db, f.clock)
	first := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, uplinkStoreWindowCounter, []byte{0x61}, models.DeliveryChannelSCACI)
	first.ReceptionWindow = uplinkStoreReceptionWindow
	_, err := f.store.Persist(testutil.TestContext(), first)
	require.NoError(t, err)

	due, err := outbox.ClaimDue(testutil.TestContext(), uplinkStoreTestLimit, time.Second)
	require.NoError(t, err)
	assert.Empty(t, due, "the first reception is not delivered alone")

	f.clock.Advance(uplinkStoreSecondArrival)
	second := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiTwo, uplinkStoreWindowCounter, []byte{0x61}, models.DeliveryChannelSCACI)
	second.ReceptionWindow = uplinkStoreReceptionWindow
	outcome, err := f.store.Persist(testutil.TestContext(), second)
	require.NoError(t, err)
	require.Equal(t, models.UplinkDuplicate, outcome.Classification)
	due, err = outbox.ClaimDue(testutil.TestContext(), uplinkStoreTestLimit, time.Second)
	require.NoError(t, err)
	assert.Empty(t, due, "a merged reception does not advance the delivery")

	f.clock.Advance(uplinkStoreReceptionWindow - uplinkStoreSecondArrival)
	due, err = outbox.ClaimDue(testutil.TestContext(), uplinkStoreTestLimit, time.Second)
	require.NoError(t, err)
	require.Len(t, due, 1, "the uplink is delivered once its reception window closes")
	assert.Equal(t, uuid.MustParse(first.Message.ID), due[0].MessageID)
	assert.Equal(t, int64(1), f.count(t, `SELECT count(*) FROM messages WHERE id = $1 AND jsonb_array_length(base_stations) = 2`, first.Message.ID),
		"the delivered message lists both receiving stations")
}

// A reception window must close before the duplicate window does: a
// reception that no longer merges could never join the delivery.
func TestUplinkStore_RejectsAReceptionWindowOutsideTheDuplicateWindow(t *testing.T) {
	f := newUplinkStoreFixture(t)
	for name, window := range map[string]time.Duration{"negative": -time.Millisecond, "the whole duplicate window": uplinkStoreWindow} {
		t.Run(name, func(t *testing.T) {
			req := uplinkRequest(uplinkStoreTenantA, uplinkStoreEpEui, uplinkStoreBsEuiOne, uplinkStoreWindowCounter, nil)
			req.ReceptionWindow = window

			_, err := f.store.Persist(testutil.TestContext(), req)

			assert.ErrorIs(t, err, storage.ErrInvalidInput)
		})
	}
}
