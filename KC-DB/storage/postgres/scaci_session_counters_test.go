package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	countersKnownAC  int64 = 10
	countersIssuedSC int64 = -5
	countersSeed     byte  = 0x10
)

func (f lifecycleFixture) sessionWithCounters(ac, sc int64) (int64, [16]byte) {
	f.t.Helper()
	session := f.create(lifecycleTenant, lifecycleAcEUI, countersSeed)
	require.NoError(f.t, f.repo.UpdateOperationIDs(f.ctx, lifecycleTenant, session.ID, ac, sc))
	return session.ID, session.SnAcUUID
}

// Counter writes arrive from concurrent writers in any order; the stored
// counters only ever move forward (SCACI §3.2: IDs strictly increment for the
// application center and strictly decrement for the service center).
func TestSCACISessionRepository_UpdateOperationIDs_NeverMovesCountersBack(t *testing.T) {
	f := newLifecycleFixture(t)
	id, _ := f.sessionWithCounters(countersKnownAC, countersIssuedSC)

	require.NoError(t, f.repo.UpdateOperationIDs(f.ctx, lifecycleTenant, id, countersKnownAC-2, countersIssuedSC+2))
	stale := f.reload(lifecycleTenant, id)
	assert.Equal(t, countersKnownAC, stale.LastOpIDAc, "a stale write must not lower the AC counter")
	assert.Equal(t, countersIssuedSC, stale.LastOpIDSc, "a stale write must not raise the SC counter")

	require.NoError(t, f.repo.UpdateOperationIDs(f.ctx, lifecycleTenant, id, countersKnownAC+1, countersIssuedSC-1))
	advanced := f.reload(lifecycleTenant, id)
	assert.Equal(t, countersKnownAC+1, advanced.LastOpIDAc)
	assert.Equal(t, countersIssuedSC-1, advanced.LastOpIDSc)
}

// The repository reports the counters it stored for the session; whether an
// application center's view agrees with them is the SCACI rule's to decide
// (scaci.ResumeOpIDConflict, §3.3.1).
func TestSCACISessionRepository_CheckSessionResumable_ReturnsTheStoredCounters(t *testing.T) {
	f := newLifecycleFixture(t)
	id, snAcUUID := f.sessionWithCounters(countersKnownAC, countersIssuedSC)

	info, err := f.repo.CheckSessionResumable(f.ctx, lifecycleAC(lifecycleTenant, nil, lifecycleAcEUI), snAcUUID)

	require.NoError(t, err)
	assert.True(t, info.CanResume, info.ReasonIfNotResumable)
	assert.Equal(t, id, info.SessionID)
	assert.Equal(t, countersKnownAC, info.LastKnownAcOpId)
	assert.Equal(t, countersIssuedSC, info.LastKnownScOpId)
}
