package bssciservices

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixture errors returned by the persistence test doubles.
var (
	errTestUpdateFailed  = errors.New("update failed")
	errTestPersistDBDown = errors.New("db down")
)

// persistEndpointRepo records the transactional endpoint field update and
// the packet counter restart.
type persistEndpointRepo struct {
	interfaces.EndpointRepository
	attachmentParams models.EndpointAttachmentStateParams
	updateErr        error
	restarts         []persistRestart
	restartErr       error
	storedAttachCnt  *uint32
	lockErr          error
}

func (r *persistEndpointRepo) LockAttachCounter(context.Context, int64, int64) (*uint32, error) {
	return r.storedAttachCnt, r.lockErr
}

// persistRestart is one recorded packet counter restart.
type persistRestart struct {
	tenantID   int64
	endpointID int64
}

func (r *persistEndpointRepo) RestartPacketCounter(_ context.Context, tenantID, endpointID int64) error {
	r.restarts = append(r.restarts, persistRestart{tenantID: tenantID, endpointID: endpointID})
	return r.restartErr
}

func (r *persistEndpointRepo) EndpointAttachmentStateUpdate(_ context.Context, _ int64, _ int64, p models.EndpointAttachmentStateParams) error {
	r.attachmentParams = p
	return r.updateErr
}

func (r *persistEndpointRepo) EndpointAttachSessionUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointAttachSessionParams) error {
	return r.updateErr
}

// persistSessionRepo records the endpoint-session upsert.
type persistSessionRepo struct {
	interfaces.EndPointSessionRepository
	active  *models.EndPointSession
	created *models.EndPointSession
	updated *models.EndPointSession
}

func (r *persistSessionRepo) GetActive(_ context.Context, _ string) (*models.EndPointSession, error) {
	return r.active, nil
}

func (r *persistSessionRepo) Create(_ context.Context, s *models.EndPointSession) error {
	r.created = s
	return nil
}

func (r *persistSessionRepo) Update(_ context.Context, s *models.EndPointSession) error {
	r.updated = s
	return nil
}

// persistTx exposes the two recording repositories the attach operation may
// reach. It is deliberately not a full transaction: the service no longer sees
// one.
type persistTx struct {
	endpoints *persistEndpointRepo
	sessions  *persistSessionRepo
}

func (t *persistTx) EndpointAttachmentStateUpdate(ctx context.Context, tenantID, endpointID int64,
	p models.EndpointAttachmentStateParams) error {
	return t.endpoints.EndpointAttachmentStateUpdate(ctx, tenantID, endpointID, p)
}

func (t *persistTx) EndpointAttachSessionUpdate(ctx context.Context, tenantID, endpointID int64,
	p models.EndpointAttachSessionParams) error {
	return t.endpoints.EndpointAttachSessionUpdate(ctx, tenantID, endpointID, p)
}

func (t *persistTx) RestartPacketCounter(ctx context.Context, tenantID, endpointID int64) error {
	return t.endpoints.RestartPacketCounter(ctx, tenantID, endpointID)
}

func (t *persistTx) LockAttachCounter(ctx context.Context, tenantID, endpointID int64) (*uint32, error) {
	return t.endpoints.LockAttachCounter(ctx, tenantID, endpointID)
}

func (t *persistTx) GetActiveSession(ctx context.Context, endpointID string) (*models.EndPointSession, error) {
	return t.sessions.GetActive(ctx, endpointID)
}

func (t *persistTx) UpdateSession(ctx context.Context, session *models.EndPointSession) error {
	return t.sessions.Update(ctx, session)
}

func (t *persistTx) CreateSession(ctx context.Context, session *models.EndPointSession) error {
	return t.sessions.Create(ctx, session)
}

// persistRunner stands in for the storage transaction runner. Commit and
// rollback are the adapter's concern and are covered by its own tests; here we
// only record whether the callback reported success.
type persistRunner struct {
	tx          *persistTx
	runErr      error
	callbackErr error
}

func (r *persistRunner) Run(_ context.Context, fn func(EndpointSessionTx) error) error {
	if r.runErr != nil {
		return r.runErr
	}
	r.callbackErr = fn(r.tx)
	return r.callbackErr
}

// noPrimaryStation knows no base station, so a session names none as primary.
type noPrimaryStation struct{}

func (noPrimaryStation) GetByEUI(context.Context, int64, []byte) (*models.BaseStation, error) {
	return nil, storage.ErrNotFound
}

func newTestAttachmentPersistence(t *testing.T, tx EndpointSessionTransactionRunner) bssci.EndpointAttachmentPersistence {
	t.Helper()
	p, err := NewEndpointAttachmentPersistence(tx, noPrimaryStation{}, clock.SystemClock{}, logger.NewNop())
	require.NoError(t, err)
	return p
}

// The attachment persister refuses a missing collaborator when it is built.
func TestNewEndpointAttachmentPersistenceRefusesNilCollaborators(t *testing.T) {
	runner, _ := newPersistFixture()
	cases := map[string]struct {
		build func() (bssci.EndpointAttachmentPersistence, error)
		want  error
	}{
		"transaction runner": {func() (bssci.EndpointAttachmentPersistence, error) {
			return NewEndpointAttachmentPersistence(nil, noPrimaryStation{}, clock.SystemClock{}, logger.NewNop())
		}, errNilAttachTransactionRunner},
		"station lookup": {func() (bssci.EndpointAttachmentPersistence, error) {
			return NewEndpointAttachmentPersistence(runner, nil, clock.SystemClock{}, logger.NewNop())
		}, errNilPrimaryStationLookup},
		"clock": {func() (bssci.EndpointAttachmentPersistence, error) {
			return NewEndpointAttachmentPersistence(runner, noPrimaryStation{}, nil, logger.NewNop())
		}, errNilAttachPersistenceClock},
		"logger": {func() (bssci.EndpointAttachmentPersistence, error) {
			return NewEndpointAttachmentPersistence(runner, noPrimaryStation{}, clock.SystemClock{}, nil)
		}, errNilAttachPersistenceLogger},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tc.build()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func newPersistFixture() (*persistRunner, *persistTx) {
	tx := &persistTx{endpoints: &persistEndpointRepo{}, sessions: &persistSessionRepo{}}
	return &persistRunner{tx: tx}, tx
}

// TestPersistAttachSession_CreatesSessionAndCommits: with no active session a
// new endpoint-session row is created carrying the attach counter and short
// address, and the transaction commits.
func TestPersistAttachSession_CreatesSessionAndCommits(t *testing.T) {
	st, tx := newPersistFixture()
	p := newTestAttachmentPersistence(t, st)

	attached := true
	err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{
		TenantID:        7,
		EndpointID:      42,
		EndpointUpdates: models.EndpointAttachmentStateParams{Propagated: &attached},
		EncryptedKey:    []byte{1, 2, 3},
		AttachCnt:       9,
		ShAddr:          0x1505,
	})

	require.NoError(t, err)
	assert.NoError(t, st.callbackErr, "the transactional work must succeed so the adapter commits")
	require.NotNil(t, tx.sessions.created)
	assert.Equal(t, int64(7), tx.sessions.created.TenantID)
	assert.Equal(t, int32(9), tx.sessions.created.AttachCnt)
	require.NotNil(t, tx.sessions.created.ShAddr)
	assert.Equal(t, int32(0x1505), *tx.sessions.created.ShAddr)
	assert.Equal(t, models.EndpointAttachmentStateParams{Propagated: &attached}, tx.endpoints.attachmentParams)
}

// TestPersistAttachSession_RestartsThePacketCounter: an over-the-air attach
// restarts the endpoint packet counter inside the attach transaction (radio
// protocol §3.6.5.3), so the restarted sequence is classified as new data.
func TestPersistAttachSession_RestartsThePacketCounter(t *testing.T) {
	st, tx := newPersistFixture()
	p := newTestAttachmentPersistence(t, st)

	err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 7, EndpointID: 42})

	require.NoError(t, err)
	assert.Equal(t, []persistRestart{{tenantID: 7, endpointID: 42}}, tx.endpoints.restarts,
		"the attach transaction restarts the counter of the attached endpoint")
}

// TestPersistAttachSession_RestartFailureRollsBack: without the restart the
// attach must not commit, or the old counters would outlive the new session.
func TestPersistAttachSession_RestartFailureRollsBack(t *testing.T) {
	st, tx := newPersistFixture()
	tx.endpoints.restartErr = errTestUpdateFailed
	p := newTestAttachmentPersistence(t, st)

	err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 7, EndpointID: 42})

	require.ErrorIs(t, err, errTestUpdateFailed)
	assert.Error(t, st.callbackErr, "the transactional work must fail so the adapter rolls back")
	assert.Nil(t, tx.sessions.created, "no session row after a failed restart")
}

// An attach whose counter a concurrent attach of the endpoint already
// consumed is refused under the row lock and writes nothing.
func TestPersistAttachSession_RefusesACounterConsumedUnderTheLock(t *testing.T) {
	st, tx := newPersistFixture()
	stored := uint32(9)
	tx.endpoints.storedAttachCnt = &stored
	p := newTestAttachmentPersistence(t, st)

	err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 7, EndpointID: 42, AttachCnt: 9})

	require.ErrorIs(t, err, bssci.ErrAttachCounterStale)
	assert.Error(t, st.callbackErr, "the transactional work must fail so the adapter rolls back")
	assert.Empty(t, tx.endpoints.restarts)
	assert.Nil(t, tx.sessions.created)
}

func TestPersistAttachSession_LockFailureRollsBack(t *testing.T) {
	st, tx := newPersistFixture()
	tx.endpoints.lockErr = errTestPersistDBDown
	p := newTestAttachmentPersistence(t, st)

	err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 7, EndpointID: 42})

	require.ErrorIs(t, err, errTestPersistDBDown)
	assert.Empty(t, tx.endpoints.restarts)
	assert.Nil(t, tx.sessions.created)
}

// TestPersistAttachPropagateSession_KeepsThePacketCounter: an attach
// propagate carries the origin's last packet counter; only an over-the-air
// attach restarts it.
func TestPersistAttachPropagateSession_KeepsThePacketCounter(t *testing.T) {
	st, tx := newPersistFixture()
	p := newTestAttachmentPersistence(t, st)

	require.NoError(t, p.PersistAttachPropagateSession(testutil.TestContext(), bssci.AttachPropagateSessionRecord{TenantID: 3, EndpointID: 42}))
	assert.Empty(t, tx.endpoints.restarts)
}

// TestPersistAttachSession_UpdateFailureRollsBack: an endpoint update failure
// rolls the transaction back; the wrapped error keeps the cause matchable
// via errors.Is.
func TestPersistAttachSession_UpdateFailureRollsBack(t *testing.T) {
	st, tx := newPersistFixture()
	updateErr := errTestUpdateFailed
	tx.endpoints.updateErr = updateErr
	p := newTestAttachmentPersistence(t, st)

	err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

	require.ErrorIs(t, err, updateErr)
	assert.Error(t, st.callbackErr, "the transactional work must fail so the adapter rolls back")
	assert.Nil(t, tx.sessions.created, "no session row after a failed endpoint update")
}

// TestPersistAttachPropagateSession_UpdatesActiveSession: an existing active
// session is updated (never duplicated) and the wrapped error strings of the
// propagate path stay intact on begin failure.
func TestPersistAttachPropagateSession_UpdatesActiveSession(t *testing.T) {
	st, tx := newPersistFixture()
	tx.sessions.active = &models.EndPointSession{ID: 5, TenantID: 3}
	p := newTestAttachmentPersistence(t, st)

	err := p.PersistAttachPropagateSession(testutil.TestContext(), bssci.AttachPropagateSessionRecord{
		TenantID:     3,
		EndpointID:   42,
		EncryptedKey: []byte{9},
		ShAddr:       0x22,
	})

	require.NoError(t, err)
	assert.NoError(t, st.callbackErr)
	require.NotNil(t, tx.sessions.updated, "active session must be updated in place")
	assert.Nil(t, tx.sessions.created)
	assert.Equal(t, []byte{9}, tx.sessions.updated.SessionKey)
}

// TestPersistAttachPropagateSession_BeginFailureKeepsErrorString: the caller
// depends on the exact wrapped error text of the propagate path.
func TestPersistAttachPropagateSession_BeginFailureKeepsErrorString(t *testing.T) {
	st, _ := newPersistFixture()
	cause := errTestPersistDBDown
	st.runErr = fmt.Errorf("%w: %w", ErrTxBegin, cause)
	p := newTestAttachmentPersistence(t, st)

	err := p.PersistAttachPropagateSession(testutil.TestContext(), bssci.AttachPropagateSessionRecord{})

	require.Error(t, err)
	require.ErrorIs(t, err, cause, "the underlying cause must stay matchable")
	assert.Contains(t, err.Error(), bssci.LogBSSCIFailedToBeginAttachPropagateTransaction)
}
