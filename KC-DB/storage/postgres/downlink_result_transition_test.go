package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

const (
	transitionTenant   = int64(351)
	transitionEndpoint = uint64(0x70b3d59cd0000351)
	transitionStation  = uint64(0x70b3d59cd0000a51)
	transitionUnknown  = "unknown-result"
)

// TestUpdateDownlinkResult_EndsTheDownlinkInItsResultsStatus pins the queue
// status each BSSCI §3.14.1 result a station reports stores the downlink as;
// only a sent downlink records when it was transmitted.
func TestUpdateDownlinkResult_EndsTheDownlinkInItsResultsStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, sqlxDB, transitionTenant, "TransitionTenant")
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: transitionEndpoint, Name: "TransitionEndpoint", TenantID: transitionTenant})
	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	downlinks := NewRepositories(db).Downlinks

	cases := map[string]struct {
		status      mioty.DLQueueStatus
		transmitted bool
	}{
		mioty.DLDataResultSent:    {mioty.DLQueueStatusTransmitted, true},
		mioty.DLDataResultInvalid: {mioty.DLQueueStatusFailed, false},
		mioty.DLDataResultExpired: {mioty.DLQueueStatusExpired, false},
	}
	for result, want := range cases {
		queID := insertDownlink(t, db, DownlinkInsertParams{EpEUI: transitionEndpoint, TenantID: transitionTenant, Status: mioty.DLQueueStatusQueued, BsEUI: transitionStation})

		_, err := downlinks.UpdateDownlinkResult(t.Context(), transitionTenant, transitionStation,
			&mioty.DLDataResult{EpEui: transitionEndpoint, QueId: uint64(queID), Result: result})
		require.NoError(t, err, result)

		var status mioty.DLQueueStatus
		var transmittedAt sql.NullTime
		require.NoError(t, sqlxDB.QueryRow(`SELECT status, transmitted_at FROM downlink_queue WHERE que_id = $1`, queID).
			Scan(&status, &transmittedAt), result)
		assert.Equal(t, want.status, status, result)
		assert.Equal(t, want.transmitted, transmittedAt.Valid, result)
	}
}

func TestResultTransition(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	status, transmittedAt := resultTransition(mioty.DLDataResultSent, now)
	assert.Equal(t, string(mioty.DLQueueStatusTransmitted), status)
	assert.Equal(t, now, transmittedAt)

	status, transmittedAt = resultTransition(mioty.DLDataResultExpired, now)
	assert.Equal(t, string(mioty.DLQueueStatusExpired), status)
	assert.Nil(t, transmittedAt)

	status, transmittedAt = resultTransition(transitionUnknown, now)
	assert.Nil(t, status)
	assert.Nil(t, transmittedAt)
}
