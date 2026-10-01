package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var (
	ownershipThisSC  = models.EUI{0x4B, 0x43, 0x00, 0x00, 0x00, 0x00, 0x01, 0x78}
	ownershipOtherSC = models.EUI{0x4B, 0x43, 0x00, 0x00, 0x00, 0x00, 0x02, 0x78}
)

// createOwned opens a session owned by the service center scEUI.
func (f lifecycleFixture) createOwned(acEUI [8]byte, seed byte, scEUI models.EUI) *models.SCACISession {
	f.t.Helper()
	req := lifecycleRequest(lifecycleTenant, acEUI, seed)
	req.ScEui = scEUI
	created, err := f.repo.CreateSession(f.ctx, req)
	require.NoError(f.t, err)
	return created
}

// A service center restarting after a crash finds the sessions its previous
// process left live and hands exactly those (and ownerless rows written
// before ownership tracking) back resumable and disconnected; another
// service center's live session and an ended session stay as they are.
func TestSCACISessionRepository_DisconnectAbandonedSessions(t *testing.T) {
	f := newLifecycleFixture(t)

	own := f.createOwned(lifecycleAcEUI, 0x10, ownershipThisSC)
	resumed := f.createOwned(lifecycleOtherAcEUI, 0x20, ownershipThisSC)
	_, err := f.db.ExecContext(f.ctx, `UPDATE scaci_sessions SET status = $1 WHERE id = $2`, models.SCACISessionStatusResumed, resumed.ID)
	require.NoError(t, err)
	legacy := f.createOwned([8]byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x03}, 0x30, ownershipThisSC)
	_, err = f.db.ExecContext(f.ctx, `UPDATE scaci_sessions SET sc_eui = NULL WHERE id = $1`, legacy.ID)
	require.NoError(t, err)
	foreign := f.createOwned([8]byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x04}, 0x40, ownershipOtherSC)
	ended := f.createOwned([8]byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x05}, 0x50, ownershipThisSC)
	require.NoError(t, f.repo.TerminateSession(f.ctx, lifecycleTenant, ended.ID))

	reconciled, err := f.repo.DisconnectAbandonedSessions(f.ctx, ownershipThisSC)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{own.ID, resumed.ID, legacy.ID}, reconciled)

	for _, id := range []int64{own.ID, resumed.ID, legacy.ID} {
		row := f.reload(lifecycleTenant, id)
		assert.Equal(t, models.SCACISessionStatusDisconnected, row.Status, "no live connection holds the session after a restart")
		assert.True(t, row.CanResume, "the Application Center may still resume it (SCACI §1)")
		assert.NotNil(t, row.DisconnectedAt)
	}
	var legacyOwner []byte
	require.NoError(t, f.db.GetContext(f.ctx, &legacyOwner, `SELECT sc_eui FROM scaci_sessions WHERE id = $1`, legacy.ID))
	assert.Equal(t, ownershipThisSC[:], legacyOwner, "an ownerless row is claimed by the reconciling service center")

	assertUntouched(t, f.reload(lifecycleTenant, foreign.ID))
	assertRetired(t, f.reload(lifecycleTenant, ended.ID))

	again, err := f.repo.DisconnectAbandonedSessions(f.ctx, ownershipThisSC)
	require.NoError(t, err)
	assert.Empty(t, again, "a repeated reconciliation finds nothing")
}

// A resume moves the session to the service center that accepted it.
func TestSCACISessionRepository_ResumeSessionClaimsTheSession(t *testing.T) {
	f := newLifecycleFixture(t)
	held := f.createOwned(lifecycleAcEUI, 0x10, ownershipOtherSC)
	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, held.ID))

	require.NoError(t, f.repo.ResumeSession(f.ctx, lifecycleTenant, held.ID, &models.SCACISessionResume{ScEui: ownershipThisSC}))

	var owner []byte
	require.NoError(t, f.db.GetContext(f.ctx, &owner, `SELECT sc_eui FROM scaci_sessions WHERE id = $1`, held.ID))
	assert.Equal(t, ownershipThisSC[:], owner)
	assert.Equal(t, models.SCACISessionStatusActive, f.reload(lifecycleTenant, held.ID).Status)
}
