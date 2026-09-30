package adapters

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// txLifecycle records the lifecycle calls an adapter makes. Repository
// accessors are added per handle type by the embedding fakes; anything an
// adapter did not declare is simply not there to call.
type txLifecycle struct {
	committed   bool
	rolledBack  bool
	commitErr   error
	rollbackErr error
}

func (f *txLifecycle) Commit() error {
	if f.commitErr != nil {
		return f.commitErr
	}
	f.committed = true
	return nil
}

func (f *txLifecycle) Rollback() error {
	f.rolledBack = true
	return f.rollbackErr
}

// fakeEndpointSessionTx satisfies endpointSessionTxHandle with nil
// repositories; the lifecycle tests never dereference them.
type fakeEndpointSessionTx struct {
	txLifecycle
}

func (f *fakeEndpointSessionTx) EndPoints() interfaces.EndpointRepository { return nil }

func (f *fakeEndpointSessionTx) UplinkClassifier() interfaces.UplinkClassifierReset { return nil }

func (f *fakeEndpointSessionTx) EndPointSessions() interfaces.EndPointSessionRepository { return nil }

// fakeBlueprintTx satisfies blueprintTxHandle the same way.
type fakeBlueprintTx struct {
	txLifecycle
	savepoints int
}

func (f *fakeBlueprintTx) DeviceModels() interfaces.DeviceModelRepository { return nil }
func (f *fakeBlueprintTx) Blueprints() interfaces.BlueprintRepository     { return nil }
func (f *fakeBlueprintTx) WithSavepoint(_ context.Context, fn func() error) error {
	f.savepoints++
	return fn()
}

// TestEndpointSessionTransactionAdapter_Run covers the lifecycle guarantees the
// service layer no longer expresses itself: commit on success, rollback on
// failure, and a rollback that still happens when the callback panics.
func TestEndpointSessionTransactionAdapter_Run(t *testing.T) {
	ctx := testutil.TestContext()

	newAdapter := func(tx *fakeEndpointSessionTx, beginErr error) *EndpointSessionTransactionAdapter {
		return &EndpointSessionTransactionAdapter{begin: func(context.Context) (endpointSessionTxHandle, error) {
			if beginErr != nil {
				return nil, beginErr
			}
			return tx, nil
		}}
	}

	t.Run("commits when the callback succeeds", func(t *testing.T) {
		tx := &fakeEndpointSessionTx{}
		err := newAdapter(tx, nil).Run(ctx, func(EndpointSessionOps) error { return nil })

		require.NoError(t, err)
		assert.True(t, tx.committed, "a successful callback must commit")
		assert.False(t, tx.rolledBack)
	})

	t.Run("rolls back and returns the callback error unchanged", func(t *testing.T) {
		tx := &fakeEndpointSessionTx{}
		err := newAdapter(tx, nil).Run(ctx, func(EndpointSessionOps) error { return storage.ErrNotFound })

		require.ErrorIs(t, err, storage.ErrNotFound,
			"the caller's sentinel must survive the adapter")
		assert.True(t, tx.rolledBack, "a failed callback must roll back")
		assert.False(t, tx.committed)
	})

	t.Run("reports a begin failure without running the callback", func(t *testing.T) {
		called := false
		err := newAdapter(nil, storage.ErrNotFound).Run(ctx, func(EndpointSessionOps) error {
			called = true
			return nil
		})

		require.ErrorIs(t, err, ErrTxBegin)
		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.False(t, called, "the callback must not run when the transaction never opened")
	})

	t.Run("reports a commit failure and rolls back", func(t *testing.T) {
		tx := &fakeEndpointSessionTx{txLifecycle: txLifecycle{commitErr: storage.ErrNotFound}}
		err := newAdapter(tx, nil).Run(ctx, func(EndpointSessionOps) error { return nil })

		require.ErrorIs(t, err, ErrTxCommit)
		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.True(t, tx.rolledBack, "a failed commit must still release the transaction")
	})

	t.Run("joins a rollback failure onto the callback error", func(t *testing.T) {
		tx := &fakeEndpointSessionTx{txLifecycle: txLifecycle{rollbackErr: storage.ErrInvalidInput}}
		err := newAdapter(tx, nil).Run(ctx, func(EndpointSessionOps) error { return storage.ErrNotFound })

		require.ErrorIs(t, err, storage.ErrNotFound, "the callback error must survive")
		require.ErrorIs(t, err, storage.ErrInvalidInput, "the rollback failure must be reported too")
	})

	t.Run("rolls back when the callback panics", func(t *testing.T) {
		tx := &fakeEndpointSessionTx{}
		adapter := newAdapter(tx, nil)

		assert.Panics(t, func() {
			_ = adapter.Run(ctx, func(EndpointSessionOps) error { panic("boom") })
		})

		assert.True(t, tx.rolledBack, "a panicking callback must not abandon the transaction")
		assert.False(t, tx.committed)
	})
}

// TestBlueprintTransactionAdapter_Run mirrors the endpoint-session lifecycle:
// both adapters must behave identically so neither becomes the odd one out.
func TestBlueprintTransactionAdapter_Run(t *testing.T) {
	ctx := testutil.TestContext()

	newAdapter := func(tx *fakeBlueprintTx, beginErr error) *BlueprintTransactionAdapter {
		return &BlueprintTransactionAdapter{begin: func(context.Context) (blueprintTxHandle, error) {
			if beginErr != nil {
				return nil, beginErr
			}
			return tx, nil
		}}
	}

	t.Run("commits when the callback succeeds", func(t *testing.T) {
		tx := &fakeBlueprintTx{}
		require.NoError(t, newAdapter(tx, nil).Run(ctx, func(BlueprintTx) error { return nil }))
		assert.True(t, tx.committed)
	})

	t.Run("rolls back on callback failure", func(t *testing.T) {
		tx := &fakeBlueprintTx{}
		err := newAdapter(tx, nil).Run(ctx, func(BlueprintTx) error { return storage.ErrNotFound })

		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.True(t, tx.rolledBack)
	})

	t.Run("reports a begin failure without running the callback", func(t *testing.T) {
		called := false
		err := newAdapter(nil, storage.ErrNotFound).Run(ctx, func(BlueprintTx) error {
			called = true
			return nil
		})

		require.ErrorIs(t, err, ErrTxBegin)
		assert.False(t, called)
	})

	t.Run("reports a commit failure and rolls back", func(t *testing.T) {
		tx := &fakeBlueprintTx{txLifecycle: txLifecycle{commitErr: storage.ErrNotFound}}
		err := newAdapter(tx, nil).Run(ctx, func(BlueprintTx) error { return nil })

		require.ErrorIs(t, err, ErrTxCommit)
		assert.True(t, tx.rolledBack)
	})

	t.Run("joins a rollback failure onto the callback error", func(t *testing.T) {
		tx := &fakeBlueprintTx{txLifecycle: txLifecycle{rollbackErr: storage.ErrInvalidInput}}
		err := newAdapter(tx, nil).Run(ctx, func(BlueprintTx) error { return storage.ErrNotFound })

		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, storage.ErrInvalidInput)
	})

	t.Run("rolls back when the callback panics", func(t *testing.T) {
		tx := &fakeBlueprintTx{}
		adapter := newAdapter(tx, nil)

		assert.Panics(t, func() {
			_ = adapter.Run(ctx, func(BlueprintTx) error { panic("boom") })
		})
		assert.True(t, tx.rolledBack)
	})
}

// reservingTx returns a downlink repository stub from the transaction so the
// reservation adapter can be driven without a database.
type reservingTx struct {
	txLifecycle
	repo *stubDownlinkRepo
}

func (r *reservingTx) MIOTYDownlinks() interfaces.MIOTYDownlinkTxRepository { return r.repo }

type stubDownlinkRepo struct {
	result *storage.DownlinkMessage
	err    error

	nextCalls   int
	byQueueIDs  []uint64
	queueOrgIDs []uuid.UUID
}

func (s *stubDownlinkRepo) ReserveNextPendingDownlink(_ context.Context, _ int64, _ []byte,
	_ uint64,
) (*storage.DownlinkMessage, error) {
	s.nextCalls++
	if s.err == nil && s.result == nil {
		return nil, storage.ErrNotFound
	}
	return s.result, s.err
}

func (s *stubDownlinkRepo) ReservePendingDownlinkByQueueID(_ context.Context, _ int64, orgID uuid.UUID,
	queueID uint64, _ []byte, _ uint64,
) (*storage.DownlinkMessage, error) {
	s.byQueueIDs = append(s.byQueueIDs, queueID)
	s.queueOrgIDs = append(s.queueOrgIDs, orgID)
	if s.err == nil && s.result == nil {
		return nil, storage.ErrNotFound
	}
	return s.result, s.err
}

func beginReserving(tx *reservingTx, beginErr error) beginDownlinkTx {
	return func(context.Context) (downlinkTxHandle, error) {
		if beginErr != nil {
			return nil, beginErr
		}
		return tx, nil
	}
}

// TestDownlinkReservationAdapter_ReserveNextPending covers the reason this
// adapter exists: the reservation must be committed before it is handed back,
// because the caller performs network I/O with it.
func TestDownlinkReservationAdapter_ReserveNextPending(t *testing.T) {
	ctx := testutil.TestContext()
	reserved := &storage.DownlinkMessage{QueID: 1}

	t.Run("commits a successful reservation before returning", func(t *testing.T) {
		tx := &reservingTx{repo: &stubDownlinkRepo{result: reserved}}
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		dl, err := adapter.ReserveNextPending(ctx, 1, []byte{1}, 2)

		require.NoError(t, err)
		require.NotNil(t, dl)
		assert.True(t, tx.committed, "the reservation must be durable before the caller sends")
	})

	t.Run("rolls back when nothing is pending", func(t *testing.T) {
		tx := &reservingTx{repo: &stubDownlinkRepo{}}
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		dl, err := adapter.ReserveNextPending(ctx, 1, []byte{1}, 2)

		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.Nil(t, dl)
		assert.False(t, tx.committed, "an empty reservation must not commit")
		assert.True(t, tx.rolledBack)
	})

	t.Run("rolls back and wraps a reservation failure", func(t *testing.T) {
		tx := &reservingTx{repo: &stubDownlinkRepo{err: storage.ErrNotFound}}
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		_, err := adapter.ReserveNextPending(ctx, 1, []byte{1}, 2)

		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.True(t, tx.rolledBack)
	})

	t.Run("reports a begin failure", func(t *testing.T) {
		adapter := &DownlinkReservationAdapter{begin: beginReserving(nil, storage.ErrNotFound)}

		_, err := adapter.ReserveNextPending(ctx, 1, []byte{1}, 2)

		require.ErrorIs(t, err, ErrTxBegin)
		require.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("reports a commit failure", func(t *testing.T) {
		tx := &reservingTx{repo: &stubDownlinkRepo{result: reserved}}
		tx.commitErr = storage.ErrNotFound
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		_, err := adapter.ReserveNextPending(ctx, 1, []byte{1}, 2)

		require.ErrorIs(t, err, ErrTxCommit)
		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.False(t, tx.rolledBack,
			"commit finishes the transaction; no rollback may run against it")
	})
}

// TestDownlinkReservationAdapter_ReserveByQueueID pins the exact-row
// reservation path: the queue ID and organization reach the query, and the
// lifecycle matches ReserveNextPending.
func TestDownlinkReservationAdapter_ReserveByQueueID(t *testing.T) {
	ctx := testutil.TestContext()
	reserved := &storage.DownlinkMessage{QueID: 7}

	t.Run("commits and passes queue ID and organization through", func(t *testing.T) {
		org := uuid.New()
		tx := &reservingTx{repo: &stubDownlinkRepo{result: reserved}}
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		dl, err := adapter.ReserveByQueueID(ctx, 1, org, 7, []byte{1}, 2)

		require.NoError(t, err)
		require.NotNil(t, dl)
		assert.True(t, tx.committed)
		require.Len(t, tx.repo.byQueueIDs, 1)
		assert.Equal(t, uint64(7), tx.repo.byQueueIDs[0])
		assert.Equal(t, org, tx.repo.queueOrgIDs[0])
	})

	t.Run("rolls back on reservation failure", func(t *testing.T) {
		tx := &reservingTx{repo: &stubDownlinkRepo{err: storage.ErrNotFound}}
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		_, err := adapter.ReserveByQueueID(ctx, 1, uuid.New(), 7, []byte{1}, 2)

		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.True(t, tx.rolledBack)
	})
}

// TestDownlinkReservationAdapter_RollbackFailures proves a rollback failure is
// surfaced on every non-committed path, including after a successful
// reservation with nothing pending, and that both the operation cause and the
// rollback cause survive errors.Is when the two coincide.
func TestDownlinkReservationAdapter_RollbackFailures(t *testing.T) {
	ctx := testutil.TestContext()

	t.Run("surfaces a rollback failure when nothing is pending", func(t *testing.T) {
		tx := &reservingTx{repo: &stubDownlinkRepo{}}
		tx.rollbackErr = storage.ErrInvalidInput
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		dl, err := adapter.ReserveNextPending(ctx, 1, []byte{1}, 2)

		require.Nil(t, dl)
		require.ErrorIs(t, err, storage.ErrNotFound)
		require.ErrorIs(t, err, ErrTxRollback)
		require.ErrorIs(t, err, storage.ErrInvalidInput,
			"a rollback failure after an empty reservation must not be swallowed")
		assert.True(t, tx.rolledBack)
		assert.False(t, tx.committed)
	})

	t.Run("joins the reservation cause and the rollback cause", func(t *testing.T) {
		tx := &reservingTx{repo: &stubDownlinkRepo{err: storage.ErrNotFound}}
		tx.rollbackErr = storage.ErrInvalidInput
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		_, err := adapter.ReserveNextPending(ctx, 1, []byte{1}, 2)

		require.ErrorIs(t, err, storage.ErrNotFound, "the reservation cause must survive")
		require.ErrorIs(t, err, ErrTxRollback)
		require.ErrorIs(t, err, storage.ErrInvalidInput, "the rollback cause must survive")
		assert.True(t, tx.rolledBack)
	})

	t.Run("does not roll back after a failed commit", func(t *testing.T) {
		tx := &reservingTx{repo: &stubDownlinkRepo{result: &storage.DownlinkMessage{QueID: 3}}}
		tx.commitErr = storage.ErrNotFound
		tx.rollbackErr = storage.ErrInvalidInput
		adapter := &DownlinkReservationAdapter{begin: beginReserving(tx, nil)}

		_, err := adapter.ReserveByQueueID(ctx, 1, uuid.New(), 3, []byte{1}, 2)

		require.ErrorIs(t, err, ErrTxCommit)
		require.ErrorIs(t, err, storage.ErrNotFound, "the commit cause must survive")
		assert.NotErrorIs(t, err, storage.ErrInvalidInput,
			"commit finishes the transaction; no rollback may run against it")
		assert.False(t, tx.rolledBack)
		assert.False(t, tx.committed)
	})
}
