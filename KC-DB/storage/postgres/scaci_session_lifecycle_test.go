package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	lifecycleTenant      int64 = 301
	lifecycleOtherTenant int64 = 302
)

var (
	lifecycleAcEUI      = [8]byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x01}
	lifecycleOtherAcEUI = [8]byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x02}
)

// lifecycleFixture is a session repository over a migrated per-test database.
type lifecycleFixture struct {
	t    *testing.T
	ctx  context.Context
	db   *sqlx.DB
	repo *SCACISessionRepository
}

func newLifecycleFixture(t *testing.T) lifecycleFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupSCACISessionTestDB(t)
	createSCACITestTenant(t, db, lifecycleTenant, "LifecycleTenant")
	createSCACITestTenant(t, db, lifecycleOtherTenant, "LifecycleOtherTenant")
	ctx, cancel := context.WithTimeout(testutil.TestContext(), testContextTimeout)
	t.Cleanup(cancel)
	return lifecycleFixture{t: t, ctx: ctx, db: db, repo: NewSCACISessionRepository(db, clock.SystemClock{}, logger.Get())}
}

func lifecycleUUID(seed byte) [16]byte {
	var id [16]byte
	for i := range id {
		id[i] = seed + byte(i)
	}
	return id
}

func lifecycleRequest(tenantID int64, acEUI [8]byte, seed byte) *models.SCACISessionCreateRequest {
	return &models.SCACISessionCreateRequest{
		TenantID:          tenantID,
		AcEUI:             acEUI,
		SnAcUUID:          lifecycleUUID(seed),
		SnScUUID:          lifecycleUUID(seed + 0x40),
		CanResume:         true,
		NegotiatedVersion: "1.0.0",
	}
}

// lifecycleAC is the application center of acEUI in the organization, nil
// for none.
func lifecycleAC(tenantID int64, orgID *uuid.UUID, acEUI [8]byte) models.SCACIApplicationCenter {
	return models.SCACIApplicationCenter{TenantID: tenantID, OrganizationID: orgID, AcEUI: acEUI}
}

// create opens a session without an organization the way the service center
// does.
func (f lifecycleFixture) create(tenantID int64, acEUI [8]byte, seed byte) *models.SCACISession {
	f.t.Helper()
	return f.createIn(tenantID, nil, acEUI, seed)
}

// createIn opens a session of the organization the way the service center
// does: the application center's earlier sessions are retired in the
// transaction that creates it.
func (f lifecycleFixture) createIn(tenantID int64, orgID *uuid.UUID, acEUI [8]byte, seed byte) *models.SCACISession {
	f.t.Helper()
	var session *models.SCACISession
	f.inTx(func(inTx *SCACISessionRepository) {
		require.NoError(f.t, inTx.RetirePriorSessions(f.ctx, lifecycleAC(tenantID, orgID, acEUI)))
		req := lifecycleRequest(tenantID, acEUI, seed)
		req.OrganizationID = orgID
		created, err := inTx.CreateSession(f.ctx, req)
		require.NoError(f.t, err)
		session = created
	})
	return session
}

// inTx runs fn against the repository inside a committed transaction.
func (f lifecycleFixture) inTx(fn func(inTx *SCACISessionRepository)) {
	f.t.Helper()
	tx, err := f.db.BeginTxx(f.ctx, nil)
	require.NoError(f.t, err)
	defer func() { _ = tx.Rollback() }()
	fn(&SCACISessionRepository{db: tx, clock: f.repo.clock, log: f.repo.log})
	require.NoError(f.t, tx.Commit())
}

// organization inserts an organization of the tenant.
func (f lifecycleFixture) organization(tenantID int64) uuid.UUID {
	f.t.Helper()
	orgID := uuid.New()
	_, err := f.db.ExecContext(f.ctx, `INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, tenantID, orgID.String())
	require.NoError(f.t, err)
	return orgID
}

func (f lifecycleFixture) reload(tenantID, id int64) *models.SCACISession {
	f.t.Helper()
	session, err := f.repo.GetSessionByID(f.ctx, tenantID, id)
	require.NoError(f.t, err)
	return session
}

func assertRetired(t *testing.T, session *models.SCACISession) {
	t.Helper()
	assert.Equal(t, models.SCACISessionStatusTerminated, session.Status, "a new session discards the previous one (SCACI §1)")
	assert.False(t, session.CanResume, "a discarded session can no longer be resumed")
	assert.NotNil(t, session.DisconnectedAt)
}

func assertUntouched(t *testing.T, session *models.SCACISession) {
	t.Helper()
	assert.Equal(t, models.SCACISessionStatusActive, session.Status)
	assert.True(t, session.CanResume)
	assert.Nil(t, session.DisconnectedAt)
}

// A fresh connect of an application center whose previous session never saw
// a teardown (process crash, half-open socket) must still open a session.
func TestSCACISessionRepository_RetirePriorSessions_RetiresActivePriorSession(t *testing.T) {
	f := newLifecycleFixture(t)

	prior := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)
	otherAC := f.create(lifecycleTenant, lifecycleOtherAcEUI, 0x20)
	otherTenant := f.create(lifecycleOtherTenant, lifecycleAcEUI, 0x30)

	fresh := f.create(lifecycleTenant, lifecycleAcEUI, 0x50)

	assert.Equal(t, models.SCACISessionStatusActive, f.reload(lifecycleTenant, fresh.ID).Status)
	assertRetired(t, f.reload(lifecycleTenant, prior.ID))
	assertUntouched(t, f.reload(lifecycleTenant, otherAC.ID))
	assertUntouched(t, f.reload(lifecycleOtherTenant, otherTenant.ID))
}

// A disconnected session stays resumable until the application center starts
// a new session instead of resuming it.
func TestSCACISessionRepository_RetirePriorSessions_RetiresDisconnectedPriorSession(t *testing.T) {
	f := newLifecycleFixture(t)

	prior := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)
	_, err := f.db.ExecContext(f.ctx, `UPDATE scaci_sessions SET status = $1, disconnected_at = NOW() WHERE id = $2`,
		models.SCACISessionStatusDisconnected, prior.ID)
	require.NoError(t, err)

	f.create(lifecycleTenant, lifecycleAcEUI, 0x50)

	assertRetired(t, f.reload(lifecycleTenant, prior.ID))
}

// An Application Center of another organization of the tenant that claims
// the acEui retires none of the owner's sessions; the owner's own new
// session still retires its previous one.
func TestSCACISessionRepository_RetirePriorSessions_KeepsAnotherOrganizationsSessions(t *testing.T) {
	f := newLifecycleFixture(t)
	owner, claimant := f.organization(lifecycleTenant), f.organization(lifecycleTenant)
	held := f.createIn(lifecycleTenant, &owner, lifecycleAcEUI, 0x10)
	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, held.ID))
	live := f.createIn(lifecycleTenant, &owner, lifecycleOtherAcEUI, 0x20)

	claimed := f.createIn(lifecycleTenant, &claimant, lifecycleAcEUI, 0x50)
	f.inTx(func(inTx *SCACISessionRepository) {
		require.NoError(t, inTx.RetirePriorSessions(f.ctx, lifecycleAC(lifecycleTenant, &claimant, lifecycleOtherAcEUI)))
	})

	stillHeld := f.reload(lifecycleTenant, held.ID)
	assert.Equal(t, models.SCACISessionStatusDisconnected, stillHeld.Status, "the owner's held session is not retired")
	assert.True(t, stillHeld.CanResume, "the owner can still resume its session")
	assertUntouched(t, f.reload(lifecycleTenant, live.ID))
	assert.Equal(t, models.SCACISessionStatusActive, f.reload(lifecycleTenant, claimed.ID).Status, "the claimant has its own session")

	f.createIn(lifecycleTenant, &owner, lifecycleOtherAcEUI, 0x60)
	assertRetired(t, f.reload(lifecycleTenant, live.ID))
}

// Two organizations of one tenant each hold a live session of an Application
// Center with the same acEui, and the owner resumes its session while the
// other organization's is live; within one organization a new session still
// replaces the live one.
func TestSCACISessionRepository_OrganizationsHoldTheirOwnLiveSession(t *testing.T) {
	f := newLifecycleFixture(t)
	owner, other := f.organization(lifecycleTenant), f.organization(lifecycleTenant)
	owned := f.createIn(lifecycleTenant, &owner, lifecycleAcEUI, 0x10)

	claimed := f.createIn(lifecycleTenant, &other, lifecycleAcEUI, 0x50)
	assertUntouched(t, f.reload(lifecycleTenant, owned.ID))
	assertUntouched(t, f.reload(lifecycleTenant, claimed.ID))

	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, owned.ID))
	require.NoError(t, f.repo.ResumeSession(f.ctx, lifecycleTenant, owned.ID, &models.SCACISessionResume{Metadata: map[string]interface{}{}}))
	assert.Equal(t, models.SCACISessionStatusActive, f.reload(lifecycleTenant, owned.ID).Status, "the owner resumes while the other organization is live")
	assertUntouched(t, f.reload(lifecycleTenant, claimed.ID))

	replacement := f.createIn(lifecycleTenant, &owner, lifecycleAcEUI, 0x60)
	assertRetired(t, f.reload(lifecycleTenant, owned.ID))
	assert.Equal(t, models.SCACISessionStatusActive, f.reload(lifecycleTenant, replacement.ID).Status)
	assertUntouched(t, f.reload(lifecycleTenant, claimed.ID))
}

// The resume lookups read the application center's own session of an
// snAcUuid: another organization's newer session of the same snAcUuid and
// acEui is never the owner's, nor is a session of another acEui.
func TestSCACISessionRepository_ResumeLookupsReadTheApplicationCentersOwnSession(t *testing.T) {
	f := newLifecycleFixture(t)
	owner, other := f.organization(lifecycleTenant), f.organization(lifecycleTenant)
	owned := f.createIn(lifecycleTenant, &owner, lifecycleAcEUI, 0x10)
	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, owned.ID))
	f.createIn(lifecycleTenant, &other, lifecycleAcEUI, 0x10)
	ownerAC := lifecycleAC(lifecycleTenant, &owner, lifecycleAcEUI)

	found, err := f.repo.GetSessionByAcUUID(f.ctx, ownerAC, owned.SnAcUUID)
	require.NoError(t, err)
	assert.Equal(t, owned.ID, found.ID)
	info, err := f.repo.CheckSessionResumable(f.ctx, ownerAC, owned.SnAcUUID)
	require.NoError(t, err)
	assert.Equal(t, owned.ID, info.SessionID)
	assert.True(t, info.CanResume, info.ReasonIfNotResumable)
	_, err = f.repo.GetSessionByAcUUID(f.ctx, lifecycleAC(lifecycleTenant, &owner, lifecycleOtherAcEUI), owned.SnAcUUID)
	assert.ErrorIs(t, err, storage.ErrNotFound, "another acEui's session is not the application center's")
}

// Retiring the earlier sessions is its own step of the creating
// transaction: inserting a session changes no other session.
func TestSCACISessionRepository_CreateSession_RetiresNothing(t *testing.T) {
	f := newLifecycleFixture(t)
	prior := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)
	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, prior.ID))

	_, err := f.repo.CreateSession(f.ctx, lifecycleRequest(lifecycleTenant, lifecycleAcEUI, 0x50))
	require.NoError(t, err)

	untouched := f.reload(lifecycleTenant, prior.ID)
	assert.Equal(t, models.SCACISessionStatusDisconnected, untouched.Status, "CreateSession alone retires no session")
	assert.True(t, untouched.CanResume)
}

func TestSCACISessionRepository_MarkSessionDisconnected(t *testing.T) {
	f := newLifecycleFixture(t)

	session := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)

	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, session.ID))

	disconnected := f.reload(lifecycleTenant, session.ID)
	assert.Equal(t, models.SCACISessionStatusDisconnected, disconnected.Status)
	assert.True(t, disconnected.CanResume, "a lost connection keeps the session resumable (SCACI §1)")
	assert.NotNil(t, disconnected.DisconnectedAt)

	info, err := f.repo.CheckSessionResumable(f.ctx, lifecycleAC(lifecycleTenant, nil, lifecycleAcEUI), session.SnAcUUID)
	require.NoError(t, err)
	assert.True(t, info.CanResume, "the application center can resume the disconnected session")
}

// Tearing down a connection whose session a newer session already replaced
// must not revive the discarded session.
func TestSCACISessionRepository_MarkSessionDisconnected_LeavesRetiredSessionTerminated(t *testing.T) {
	f := newLifecycleFixture(t)

	prior := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)
	f.create(lifecycleTenant, lifecycleAcEUI, 0x50)

	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, prior.ID))

	assertRetired(t, f.reload(lifecycleTenant, prior.ID))
}

func TestSCACISessionRepository_MarkSessionDisconnected_IsTenantScoped(t *testing.T) {
	f := newLifecycleFixture(t)

	session := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)

	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleOtherTenant, session.ID))

	assertUntouched(t, f.reload(lifecycleTenant, session.ID))
}

// A resume makes its session active again with the new connection's TLS
// evidence; a session that is no longer resumable is not revived (SCACI §1).
func TestSCACISessionRepository_ResumeSession(t *testing.T) {
	f := newLifecycleFixture(t)
	tlsVersion, cipherSuite := "TLS 1.3", "TLS_AES_128_GCM_SHA256"
	resume := &models.SCACISessionResume{TLSVersion: &tlsVersion, CipherSuite: &cipherSuite, Metadata: map[string]interface{}{"name": "ac"}}

	disconnected := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)
	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, disconnected.ID))
	require.NoError(t, f.repo.ResumeSession(f.ctx, lifecycleTenant, disconnected.ID, resume))
	resumed := f.reload(lifecycleTenant, disconnected.ID)
	assert.Equal(t, models.SCACISessionStatusActive, resumed.Status)
	assert.NotNil(t, resumed.LastHeartbeat)
	require.NotNil(t, resumed.TLSVersion)
	assert.Equal(t, tlsVersion, *resumed.TLSVersion)

	ended := f.create(lifecycleTenant, lifecycleOtherAcEUI, 0x20)
	require.NoError(t, f.repo.TerminateSession(f.ctx, lifecycleTenant, ended.ID))
	err := f.repo.ResumeSession(f.ctx, lifecycleTenant, ended.ID, resume)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assertRetired(t, f.reload(lifecycleTenant, ended.ID))

	err = f.repo.ResumeSession(f.ctx, lifecycleOtherTenant, disconnected.ID, resume)
	assert.ErrorIs(t, err, storage.ErrNotFound, "another tenant's session is not resumed")
}

// The service center keeps recording operations for every session an
// Application Center may still resume, across tenants (SCACI §1); a
// discarded or non-resumable session is not one of them.
func TestSCACISessionRepository_ListResumableSessions(t *testing.T) {
	f := newLifecycleFixture(t)

	retired := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)
	current := f.create(lifecycleTenant, lifecycleAcEUI, 0x50)
	require.NoError(t, f.repo.MarkSessionDisconnected(f.ctx, lifecycleTenant, current.ID))
	connected := f.create(lifecycleTenant, lifecycleOtherAcEUI, 0x20)
	otherTenant := f.create(lifecycleOtherTenant, lifecycleAcEUI, 0x30)
	notResumable := f.create(lifecycleOtherTenant, lifecycleOtherAcEUI, 0x60)
	_, err := f.db.ExecContext(f.ctx, `UPDATE scaci_sessions SET can_resume = false WHERE id = $1`, notResumable.ID)
	require.NoError(t, err)

	sessions, err := f.repo.ListResumableSessions(f.ctx)
	require.NoError(t, err)

	listed := make(map[int64]*models.SCACISession, len(sessions))
	for _, session := range sessions {
		listed[session.ID] = session
	}
	assert.Len(t, listed, 3)
	assert.Contains(t, listed, current.ID, "a disconnected session stays resumable")
	assert.Contains(t, listed, connected.ID, "a session left active by a lost process stays resumable")
	assert.Contains(t, listed, otherTenant.ID, "every tenant's sessions are listed")
	assert.NotContains(t, listed, retired.ID, "a session a newer one replaced is discarded")
	assert.NotContains(t, listed, notResumable.ID)
	assert.Equal(t, lifecycleOtherTenant, listed[otherTenant.ID].TenantID)
	assert.Equal(t, lifecycleAcEUI, listed[current.ID].AcEUI)
}

// Only the operations the service center initiated for the session that still
// await completion count (SCACI §3.2): not its completed ones, not the
// Application Center's, not another session's.
func TestSCACIOperationRepository_CountPendingSCOperations(t *testing.T) {
	f := newLifecycleFixture(t)
	operations := NewSCACIOperationRepository(f.db, logger.Get(), clock.SystemClock{})
	session := f.create(lifecycleTenant, lifecycleAcEUI, 0x10)
	other := f.create(lifecycleTenant, lifecycleOtherAcEUI, 0x20)
	record := func(s *models.SCACISession, opID int64, direction models.OperationDirection) {
		t.Helper()
		_, err := operations.RecordOperation(f.ctx, &models.SCACIOperationRequest{
			SessionID: s.ID, TenantID: s.TenantID, OpId: opID, Command: "ulData", Direction: string(direction),
		})
		require.NoError(t, err)
	}
	record(session, -1, models.OperationDirectionOutbound)
	record(session, -2, models.OperationDirectionOutbound)
	record(session, -3, models.OperationDirectionOutbound)
	record(session, 1, models.OperationDirectionInbound)
	record(other, -1, models.OperationDirectionOutbound)
	require.NoError(t, operations.UpdateOperationState(f.ctx, session.ID, -2, models.OperationStateAcknowledged, nil))
	require.NoError(t, operations.UpdateOperationState(f.ctx, session.ID, -3, models.OperationStateCompleted, nil))

	count, err := operations.CountPendingSCOperations(f.ctx, session.ID)

	require.NoError(t, err)
	assert.Equal(t, 2, count, "the pending and the acknowledged service center operation of the session")
}
