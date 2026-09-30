package audit

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const requiredTestActor = "0b0d7c1e-2f5c-4c3a-9c2e-1f8a6f0a1d2e"

// storeRecorder records through a real Emitter into store.
func storeRecorder(t *testing.T, store *captureWriter) (*Recorder, *countingDrops, *captureLogger) {
	t.Helper()
	emitter, err := NewEmitter(store, fixedClock{})
	require.NoError(t, err)
	drops := &countingDrops{byType: map[string]int{}}
	log := &captureLogger{Logger: logger.NewNop()}
	recorder, err := NewRecorder(emitter, log, drops)
	require.NoError(t, err)
	return recorder, drops, log
}

func TestRecordRequired_ReturnsTheWriteFailure(t *testing.T) {
	recorder, drops, log := storeRecorder(t, &captureWriter{err: errStorage})

	err := recorder.RecordRequired(testutil.TestContext(), Event{TenantID: testTenant, EventType: models.EventTypeEndpointKeysRevealed})

	require.ErrorIs(t, err, errStorage, "the caller learns the event was not written")
	assert.Equal(t, map[string]int{models.EventTypeEndpointKeysRevealed: 1}, drops.byType, "the unwritten event is counted like a drop")
	assert.Equal(t, []string{LogRequiredEventNotWritten}, log.errors, "and logged as a refused action, not a committed one")
}

func TestRecordRequired_NamesTheActorLikeRecord(t *testing.T) {
	store := &captureWriter{}
	recorder, drops, log := storeRecorder(t, store)
	keyID := uuid.MustParse("5b1f0c1e-3f4e-4d1a-9f2b-7c0d6e8a9b10")

	require.NoError(t, recorder.RecordRequired(pkgcontext.WithUserID(testutil.TestContext(), requiredTestActor),
		Event{TenantID: testTenant, EventType: models.EventTypeEndpointKeysRevealed}))
	require.NoError(t, recorder.RecordRequired(pkgcontext.WithServiceAccountID(testutil.TestContext(), keyID),
		Event{TenantID: testTenant, EventType: models.EventTypeEndpointKeysRevealed}))

	require.Len(t, store.events, 2)
	assert.Equal(t, requiredTestActor, store.events[0].UserID, "the acting user is recorded")
	assert.Contains(t, string(store.events[1].Details), keyID.String(), "a service-account key is recorded as the actor")
	assert.Empty(t, drops.byType)
	assert.Empty(t, log.errors)
}

func TestRecordRequired_RefusesAnEventWithoutATenant(t *testing.T) {
	recorder, drops, _ := storeRecorder(t, &captureWriter{})

	err := recorder.RecordRequired(testutil.TestContext(), Event{EventType: models.EventTypeEndpointKeysRevealed})

	require.ErrorIs(t, err, ErrTenantRequired)
	assert.Equal(t, 1, drops.byType[models.EventTypeEndpointKeysRevealed])
}
