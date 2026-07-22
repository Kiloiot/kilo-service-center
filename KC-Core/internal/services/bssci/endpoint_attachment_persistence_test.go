package bssciservices

import (
	"context"
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// persistEndpointRepo records the transactional endpoint field update.
type persistEndpointRepo struct {
	interfaces.EndpointRepository
	updates   map[string]interface{}
	updateErr error
}

func (r *persistEndpointRepo) UpdateFields(_ context.Context, _ int64, _ int64, updates map[string]interface{}) error {
	r.updates = updates
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

// persistTx implements interfaces.Transaction over the two recording repos.
type persistTx struct {
	interfaces.Transaction
	endpoints  *persistEndpointRepo
	sessions   *persistSessionRepo
	committed  bool
	rolledBack bool
}

func (t *persistTx) EndPoints() interfaces.EndpointRepository               { return t.endpoints }
func (t *persistTx) EndPointSessions() interfaces.EndPointSessionRepository { return t.sessions }
func (t *persistTx) Commit() error                                          { t.committed = true; return nil }
func (t *persistTx) Rollback() error                                        { t.rolledBack = true; return nil }

// persistStorage hands out the single transaction under test.
type persistStorage struct {
	interfaces.Storage
	tx       *persistTx
	beginErr error
}

func (s *persistStorage) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	if s.beginErr != nil {
		return nil, s.beginErr
	}
	return s.tx, nil
}

func newPersistFixture() (*persistStorage, *persistTx) {
	tx := &persistTx{endpoints: &persistEndpointRepo{}, sessions: &persistSessionRepo{}}
	return &persistStorage{tx: tx}, tx
}

// TestPersistAttachSession_CreatesSessionAndCommits: with no active session a
// new endpoint-session row is created carrying the attach counter and short
// address, and the transaction commits.
func TestPersistAttachSession_CreatesSessionAndCommits(t *testing.T) {
	st, tx := newPersistFixture()
	p := NewEndpointAttachmentPersistence(st, nil, logger.NewNop())

	err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{
		TenantID:        7,
		EndpointID:      42,
		EndpointUpdates: map[string]interface{}{"attached": true},
		EncryptedKey:    []byte{1, 2, 3},
		AttachCnt:       9,
		ShAddr:          0x1505,
	})

	require.NoError(t, err)
	assert.True(t, tx.committed, "transaction must commit")
	assert.False(t, tx.rolledBack)
	require.NotNil(t, tx.sessions.created)
	assert.Equal(t, int64(7), tx.sessions.created.TenantID)
	assert.Equal(t, int32(9), tx.sessions.created.AttachCnt)
	require.NotNil(t, tx.sessions.created.ShAddr)
	assert.Equal(t, int32(0x1505), *tx.sessions.created.ShAddr)
	assert.Equal(t, map[string]interface{}{"attached": true}, tx.endpoints.updates)
}

// TestPersistAttachSession_UpdateFailureRollsBack: an endpoint update failure
// rolls the transaction back and propagates the raw error.
func TestPersistAttachSession_UpdateFailureRollsBack(t *testing.T) {
	st, tx := newPersistFixture()
	updateErr := errors.New("update failed")
	tx.endpoints.updateErr = updateErr
	p := NewEndpointAttachmentPersistence(st, nil, logger.NewNop())

	err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

	require.ErrorIs(t, err, updateErr)
	assert.True(t, tx.rolledBack, "failed transaction must roll back")
	assert.False(t, tx.committed)
	assert.Nil(t, tx.sessions.created, "no session row after a failed endpoint update")
}

// TestPersistAttachPropagateSession_UpdatesActiveSession: an existing active
// session is updated (never duplicated) and the wrapped error strings of the
// propagate path stay intact on begin failure.
func TestPersistAttachPropagateSession_UpdatesActiveSession(t *testing.T) {
	st, tx := newPersistFixture()
	tx.sessions.active = &models.EndPointSession{ID: 5, TenantID: 3}
	p := NewEndpointAttachmentPersistence(st, nil, logger.NewNop())

	err := p.PersistAttachPropagateSession(testutil.TestContext(), bssci.AttachPropagateSessionRecord{
		TenantID:     3,
		EndpointID:   42,
		EncryptedKey: []byte{9},
		ShAddr:       0x22,
	})

	require.NoError(t, err)
	assert.True(t, tx.committed)
	require.NotNil(t, tx.sessions.updated, "active session must be updated in place")
	assert.Nil(t, tx.sessions.created)
	assert.Equal(t, []byte{9}, tx.sessions.updated.SessionKey)
}

// TestPersistAttachPropagateSession_BeginFailureKeepsErrorString: the caller
// depends on the exact wrapped error text of the propagate path.
func TestPersistAttachPropagateSession_BeginFailureKeepsErrorString(t *testing.T) {
	st, _ := newPersistFixture()
	st.beginErr = errors.New("db down")
	p := NewEndpointAttachmentPersistence(st, nil, logger.NewNop())

	err := p.PersistAttachPropagateSession(testutil.TestContext(), bssci.AttachPropagateSessionRecord{})

	require.Error(t, err)
	assert.Equal(t, "failed to begin attach propagate transaction: db down", err.Error())
}
