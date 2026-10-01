package scaciservices

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Sessions of the holder tests: two Application Centers of holderTenant and
// one of another tenant.
const (
	holderTenant      = int64(71)
	holderOtherTenant = int64(72)
	holderSessionID   = int64(710)
	holderOtherACID   = int64(711)
	holderForeignID   = int64(720)
	holderAcEui       = uint64(0x70B3D59CD0000C01)
	holderOtherAcEui  = uint64(0x70B3D59CD0000C02)
	holderLastIssued  = int64(-4)
	holderLimit       = 3
	holderCommand     = scaci.CmdULData
)

var errHolderTestStore = errors.New("store unavailable")

// counterWrite is one persisted pair of operation ID counters.
type counterWrite struct {
	sessionID int64
	sc        int64
}

// holderRecord is one operation recorded for a session.
type holderRecord struct {
	sessionID int64
	opID      int64
}

// holderStores fakes every store the holder writes to or reads from.
type holderStores struct {
	rows        []*models.SCACISession
	pending     map[int64]int
	counters    []counterWrite
	records     []holderRecord
	terminated  []int64
	counterFail error
	recordFail  error
}

func (f *holderStores) ListResumableSessions(context.Context) ([]*models.SCACISession, error) {
	return f.rows, nil
}

func (f *holderStores) PersistOpIDs(_ context.Context, session *scaci.Session, ids scaci.OpIDPair) error {
	if f.counterFail != nil {
		return f.counterFail
	}
	f.counters = append(f.counters, counterWrite{sessionID: session.ID, sc: ids.SC})
	return nil
}

func (f *holderStores) EndResumability(_ context.Context, session *scaci.Session) error {
	f.terminated = append(f.terminated, session.ID)
	return nil
}

func (f *holderStores) CountPendingSCOperations(_ context.Context, sessionID int64) (int, error) {
	return f.pending[sessionID], nil
}

// record records a new operation for the session.
func (f *holderStores) record(session *scaci.Session, opId int64) (scaci.Recording, error) {
	if f.recordFail != nil {
		return scaci.NotRecorded, f.recordFail
	}
	f.records = append(f.records, holderRecord{sessionID: session.ID, opID: opId})
	return scaci.RecordedNew, nil
}

// recordUplink records one stored uplink: a session that has it keeps its
// first record, as the operation log's uplink identity does (SCACI §3.2).
func (f *holderStores) recordUplink(session *scaci.Session, opId int64) (scaci.Recording, error) {
	if len(f.recordedFor(session.ID)) > 0 {
		return scaci.RecordedBefore, nil
	}
	return f.record(session, opId)
}

func (f *holderStores) recordedFor(sessionID int64) []holderRecord {
	var records []holderRecord
	for _, r := range f.records {
		if r.sessionID == sessionID {
			records = append(records, r)
		}
	}
	return records
}

func newTestHolder(t *testing.T, stores *holderStores) *ResumeHolder {
	t.Helper()
	holder, err := NewResumeHolder(stores, stores, holderLimit, logger.NewNop())
	require.NoError(t, err)
	return holder
}

func holderSession(id, tenantID int64, acEui uint64) *scaci.Session {
	row := &models.SCACISession{ID: id, TenantID: tenantID, LastOpIDSc: holderLastIssued}
	binary.BigEndian.PutUint64(row.AcEUI[:], acEui)
	return scaci.RestoreSession(row)
}

func everyone(*scaci.Session) bool { return true }

func recordOnce(t *testing.T, holder *ResumeHolder, stores *holderStores, reaches func(*scaci.Session) bool) error {
	t.Helper()
	return holder.Record(testutil.TestContext(), holderTenant, reaches, holderCommand, stores.record)
}

// Each operation is recorded under the session's next SC opId, and the
// counters are persisted before it, so the resume and a restart continue
// from them (SCACI §3.2).
func TestResumeHolder_RecordsUnderTheNextSCOpID(t *testing.T) {
	stores := &holderStores{}
	holder := newTestHolder(t, stores)
	holder.Hold(testutil.TestContext(), holderSession(holderSessionID, holderTenant, holderAcEui))

	require.NoError(t, recordOnce(t, holder, stores, everyone))
	require.NoError(t, recordOnce(t, holder, stores, everyone))

	assert.Equal(t, []holderRecord{{holderSessionID, holderLastIssued - 1}, {holderSessionID, holderLastIssued - 2}}, stores.records)
	assert.Equal(t, []counterWrite{{holderSessionID, holderLastIssued - 1}, {holderSessionID, holderLastIssued - 2}}, stores.counters)
}

// Tenant isolation: a held session of another tenant gets no record.
func TestResumeHolder_RecordsForTheTenantOnly(t *testing.T) {
	stores := &holderStores{}
	holder := newTestHolder(t, stores)
	holder.Hold(testutil.TestContext(), holderSession(holderForeignID, holderOtherTenant, holderAcEui))

	require.NoError(t, recordOnce(t, holder, stores, everyone))

	assert.Empty(t, stores.records)
}

// Only the sessions the caller's audience selects get the record.
func TestResumeHolder_RecordsForTheSelectedSessionsOnly(t *testing.T) {
	stores := &holderStores{}
	holder := newTestHolder(t, stores)
	holder.Hold(testutil.TestContext(), holderSession(holderSessionID, holderTenant, holderAcEui))
	holder.Hold(testutil.TestContext(), holderSession(holderOtherACID, holderTenant, holderOtherAcEui))

	require.NoError(t, recordOnce(t, holder, stores, func(s *scaci.Session) bool { return s.AcEui == holderAcEui }))

	assert.Len(t, stores.recordedFor(holderSessionID), 1)
	assert.Empty(t, stores.recordedFor(holderOtherACID))
}

// A session holds at most the limit, counting what it held when its
// connection was lost: the operation that exceeds it ends the session's
// resumability, so nothing it holds is reissued, and nothing more is recorded
// for it.
func TestResumeHolder_LimitEndsResumability(t *testing.T) {
	stores := &holderStores{pending: map[int64]int{holderSessionID: holderLimit - 1}}
	holder := newTestHolder(t, stores)
	holder.Hold(testutil.TestContext(), holderSession(holderSessionID, holderTenant, holderAcEui))

	for range holderLimit {
		require.NoError(t, recordOnce(t, holder, stores, everyone))
	}

	assert.Len(t, stores.recordedFor(holderSessionID), 2, "one operation fits under the limit, the next one ends the session")
	assert.Equal(t, []int64{holderSessionID}, stores.terminated, "the session past the limit is no longer resumable")
	_, held := holder.Release(testutil.TestContext(), holderSession(holderSessionID, holderTenant, holderAcEui))
	assert.False(t, held, "the session is no longer held")
}

// A retried uplink the session already holds is neither recorded nor counted
// again: it counts once toward the limit, so a repeat at the limit does not
// end the session's resumability (SCACI §1, §3.2).
func TestResumeHolder_RepeatedUplinkCountsOnce(t *testing.T) {
	stores := &holderStores{pending: map[int64]int{holderSessionID: holderLimit - 1}}
	holder := newTestHolder(t, stores)
	holder.Hold(testutil.TestContext(), holderSession(holderSessionID, holderTenant, holderAcEui))

	for range holderLimit {
		require.NoError(t, holder.Record(testutil.TestContext(), holderTenant, everyone, holderCommand, stores.recordUplink))
	}

	assert.Len(t, stores.recordedFor(holderSessionID), 1, "one operation per session and uplink")
	assert.Empty(t, stores.terminated, "a repeat adds nothing the session holds")

	require.NoError(t, recordOnce(t, holder, stores, everyone))
	assert.Equal(t, []int64{holderSessionID}, stores.terminated, "the uplink counted once: the next operation is past the limit")
}

// A restarted service center holds every session the earlier run left
// resumable and continues its SC opIds (SCACI §1, §3.2).
func TestResumeHolder_LoadHoldsTheResumableSessions(t *testing.T) {
	stores := &holderStores{rows: []*models.SCACISession{
		{ID: holderSessionID, TenantID: holderTenant, LastOpIDSc: holderLastIssued},
		{ID: holderForeignID, TenantID: holderOtherTenant},
	}}
	holder := newTestHolder(t, stores)

	require.NoError(t, holder.Load(testutil.TestContext(), stores))
	require.NoError(t, recordOnce(t, holder, stores, everyone))

	assert.Equal(t, []holderRecord{{holderSessionID, holderLastIssued - 1}}, stores.records)
	_, held := holder.Release(testutil.TestContext(), &scaci.Session{ID: holderForeignID, TenantID: holderOtherTenant})
	assert.True(t, held, "every tenant's resumable sessions are held")
}

// A new session of an Application Center releases its previous session
// (SCACI §1: its state is discarded); a resume gets the held session back.
// Nothing more is recorded for a released session.
func TestResumeHolder_ReleaseDiscardsSupersededSessionsAndReturnsTheResumedOne(t *testing.T) {
	stores := &holderStores{}
	holder := newTestHolder(t, stores)
	previous := holderSession(holderSessionID, holderTenant, holderAcEui)
	other := holderSession(holderOtherACID, holderTenant, holderOtherAcEui)
	holder.Hold(testutil.TestContext(), previous)
	holder.Hold(testutil.TestContext(), other)

	_, resumed := holder.Release(testutil.TestContext(), &scaci.Session{ID: holderSessionID + 100, TenantID: holderTenant, AcEui: holderAcEui})
	assert.False(t, resumed, "a new session resumes nothing")
	require.NoError(t, recordOnce(t, holder, stores, everyone))
	assert.Empty(t, stores.recordedFor(holderSessionID), "the superseded session is discarded")

	back, resumed := holder.Release(testutil.TestContext(), &scaci.Session{ID: holderOtherACID, TenantID: holderTenant, AcEui: holderOtherAcEui})
	require.True(t, resumed)
	assert.Same(t, other, back)
	require.NoError(t, recordOnce(t, holder, stores, everyone))
	assert.Len(t, stores.recordedFor(holderOtherACID), 1, "only what was recorded before the resume")
}

// A session of another organization that claims the Application Center's
// acEui releases nothing: what is held for the owner stays held.
func TestResumeHolder_AnotherOrganizationsSessionReleasesNothing(t *testing.T) {
	stores := &holderStores{}
	holder := newTestHolder(t, stores)
	owned := holderSession(holderSessionID, holderTenant, holderAcEui)
	owned.OrganizationID = uuid.New()
	holder.Hold(testutil.TestContext(), owned)

	_, resumed := holder.Release(testutil.TestContext(), &scaci.Session{ID: holderSessionID + 100, TenantID: holderTenant, OrganizationID: uuid.New(), AcEui: holderAcEui})

	assert.False(t, resumed)
	require.NoError(t, recordOnce(t, holder, stores, everyone))
	assert.Len(t, stores.recordedFor(holderSessionID), 1, "the owner's session is still held")
}

// A record that cannot be made fails, so the caller retries it; a counter
// that cannot be persisted records nothing.
func TestResumeHolder_FailuresAreReported(t *testing.T) {
	for name, stores := range map[string]*holderStores{
		"counters": {counterFail: errHolderTestStore},
		"record":   {recordFail: errHolderTestStore},
	} {
		t.Run(name, func(t *testing.T) {
			holder := newTestHolder(t, stores)
			holder.Hold(testutil.TestContext(), holderSession(holderSessionID, holderTenant, holderAcEui))

			err := recordOnce(t, holder, stores, everyone)

			require.ErrorIs(t, err, errHolderTestStore)
			assert.Empty(t, stores.records)
		})
	}
}

// A session without a row can be neither recorded for nor resumed.
func TestResumeHolder_HoldsPersistedSessionsOnly(t *testing.T) {
	stores := &holderStores{}
	holder := newTestHolder(t, stores)
	holder.Hold(testutil.TestContext(), &scaci.Session{TenantID: holderTenant})

	require.NoError(t, recordOnce(t, holder, stores, everyone))

	assert.Empty(t, stores.records)
}

func TestNewResumeHolder_RefusesANonPositiveLimit(t *testing.T) {
	stores := &holderStores{}
	_, err := NewResumeHolder(stores, stores, 0, logger.NewNop())

	assert.ErrorIs(t, err, errInvalidResumeLimit)
}

func TestNewResumeHolder_RefusesAMissingCollaborator(t *testing.T) {
	stores := &holderStores{}
	for name, build := range map[string]func() (*ResumeHolder, error){
		"sessions": func() (*ResumeHolder, error) {
			return NewResumeHolder(nil, stores, holderLimit, logger.NewNop())
		},
		"operations": func() (*ResumeHolder, error) {
			return NewResumeHolder(stores, nil, holderLimit, logger.NewNop())
		},
		"logger": func() (*ResumeHolder, error) { return NewResumeHolder(stores, stores, holderLimit, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := build()
			assert.ErrorIs(t, err, errMissingResumeHolderDependency)
		})
	}
}
