package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// clockTestOpened is far from the database server's time, so a row stamped
// by NOW() cannot pass for one stamped by the injected clock.
var clockTestOpened = time.Date(2031, time.March, 4, 5, 6, 7, 0, time.UTC)

// Every timestamp the SCACI repositories write comes from the injected clock:
// session and operation history follows the service center's time, not the
// database server's. updated_at is the update trigger's.
func TestSCACIRepositories_StampRowsFromTheInjectedClock(t *testing.T) {
	f := newLifecycleFixture(t)
	at := func(now time.Time) {
		f.repo = NewSCACISessionRepository(f.db, fixedClock{now: now}, logger.Get())
	}

	at(clockTestOpened)
	prior := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)
	row := f.reload(lifecycleTenant, prior.ID)
	assertInstant(t, clockTestOpened, row.ConnectedAt, "connected_at")
	assertInstant(t, clockTestOpened, row.CreatedAt, "created_at")

	replaced := clockTestOpened.Add(time.Minute)
	at(replaced)
	session := f.create(lifecycleTenant, lifecycleAcEUI, 0x20)
	require.NotNil(t, f.reload(lifecycleTenant, prior.ID).DisconnectedAt)
	assertInstant(t, replaced, *f.reload(lifecycleTenant, prior.ID).DisconnectedAt, "disconnected_at of the replaced session")

	beat := replaced.Add(time.Minute)
	at(beat)
	require.NoError(t, f.repo.UpdateHeartbeat(f.ctx, lifecycleTenant, session.ID))
	require.NotNil(t, f.reload(lifecycleTenant, session.ID).LastHeartbeat)
	assertInstant(t, beat, *f.reload(lifecycleTenant, session.ID).LastHeartbeat, "last_heartbeat")

	lost := beat.Add(time.Minute)
	at(lost)
	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, session.ID))
	require.NotNil(t, f.reload(lifecycleTenant, session.ID).DisconnectedAt)
	assertInstant(t, lost, *f.reload(lifecycleTenant, session.ID).DisconnectedAt, "disconnected_at")

	resumed := lost.Add(time.Minute)
	at(resumed)
	require.NoError(t, f.repo.ResumeSession(f.ctx, lifecycleTenant, session.ID, &models.SCACISessionResume{}))
	assertInstant(t, resumed, *f.reload(lifecycleTenant, session.ID).LastHeartbeat, "last_heartbeat of the resume")

	ended := resumed.Add(time.Minute)
	at(ended)
	require.NoError(t, f.repo.TerminateSession(f.ctx, lifecycleTenant, session.ID))
	assertInstant(t, ended, *f.reload(lifecycleTenant, session.ID).DisconnectedAt, "disconnected_at of the termination")

	recorded := ended.Add(time.Minute)
	operations := NewSCACIOperationRepository(f.db, logger.Get(), fixedClock{now: recorded})
	op, err := operations.RecordOperation(f.ctx, &models.SCACIOperationRequest{
		SessionID: session.ID, TenantID: lifecycleTenant, OpId: -1, Command: "ulData",
		Direction: string(models.OperationDirectionOutbound),
	})
	require.NoError(t, err)
	assertInstant(t, recorded, op.InitiatedAt, "initiated_at")
}

func assertInstant(t *testing.T, want, got time.Time, column string) {
	t.Helper()
	assert.Truef(t, want.Equal(got), "%s is %v, not the clock's %v", column, got, want)
}
