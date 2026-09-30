package builders

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	blueprintservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/blueprints"
	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	scaciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
)

// The seams below reproduce the storage adapter's error shapes for one injected
// lifecycle failure. The adapter's own lifecycle correctness is proven in the
// adapters package; here the assertion is narrower: the bridge maps the storage
// lifecycle sentinel onto the service-owned one while every underlying cause
// stays matchable via errors.Is. The transaction value is unused because the
// callback under test never reaches an operation.

// blueprintSeam builds a blueprintTxRun that fails at the named lifecycle stage.
func blueprintSeam(stage string, cause, rollback error) blueprintTxRun {
	return func(_ context.Context, fn func(adapters.BlueprintTx) error) error {
		switch stage {
		case "begin":
			return fmt.Errorf("%w: %w", adapters.ErrTxBegin, cause)
		case "commit":
			if err := fn(nil); err != nil {
				return err
			}
			return fmt.Errorf("%w: %w", adapters.ErrTxCommit, cause)
		case "rollback":
			err := fn(nil)
			return errors.Join(err, fmt.Errorf("rollback: %w", rollback))
		default:
			return fn(nil)
		}
	}
}

// endpointSeam is the same seam for the endpoint session bridge.
func endpointSeam(stage string, cause, rollback error) endpointSessionTxRun {
	return func(_ context.Context, fn func(adapters.EndpointSessionOps) error) error {
		switch stage {
		case "begin":
			return fmt.Errorf("%w: %w", adapters.ErrTxBegin, cause)
		case "commit":
			if err := fn(adapters.EndpointSessionOps{}); err != nil {
				return err
			}
			return fmt.Errorf("%w: %w", adapters.ErrTxCommit, cause)
		case "rollback":
			err := fn(adapters.EndpointSessionOps{})
			return errors.Join(err, fmt.Errorf("rollback: %w", rollback))
		default:
			return fn(adapters.EndpointSessionOps{})
		}
	}
}

func TestBlueprintTxBridge_Run(t *testing.T) {
	ctx := testutil.TestContext()
	noop := func(blueprintservices.BlueprintTx) error { return nil }

	t.Run("maps a begin failure onto the service sentinel and keeps the cause", func(t *testing.T) {
		b := blueprintTxBridge{run: blueprintSeam("begin", storage.ErrNotFound, nil)}

		err := b.Run(ctx, noop)

		require.ErrorIs(t, err, blueprintservices.ErrTxBegin, "the service begin sentinel must surface")
		require.ErrorIs(t, err, adapters.ErrTxBegin, "the storage begin sentinel must survive")
		require.ErrorIs(t, err, storage.ErrNotFound, "the underlying storage cause must survive")
	})

	t.Run("maps a commit failure onto the service sentinel and keeps the cause", func(t *testing.T) {
		b := blueprintTxBridge{run: blueprintSeam("commit", storage.ErrNotFound, nil)}

		err := b.Run(ctx, noop)

		require.ErrorIs(t, err, blueprintservices.ErrTxCommit, "the service commit sentinel must surface")
		require.ErrorIs(t, err, adapters.ErrTxCommit)
		require.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("passes a joined rollback failure through with both causes matchable", func(t *testing.T) {
		b := blueprintTxBridge{run: blueprintSeam("rollback", nil, storage.ErrInvalidInput)}
		fnErr := func(blueprintservices.BlueprintTx) error { return storage.ErrNotFound }

		err := b.Run(ctx, fnErr)

		require.ErrorIs(t, err, storage.ErrNotFound, "the callback cause must survive")
		require.ErrorIs(t, err, storage.ErrInvalidInput, "the rollback cause must survive")
		require.NotErrorIs(t, err, blueprintservices.ErrTxBegin, "a rollback is not a lifecycle sentinel")
		require.NotErrorIs(t, err, blueprintservices.ErrTxCommit)
	})

	t.Run("returns nil on success", func(t *testing.T) {
		b := blueprintTxBridge{run: blueprintSeam("success", nil, nil)}
		require.NoError(t, b.Run(ctx, noop))
	})
}

func TestEndpointSessionTxBridge_Run(t *testing.T) {
	ctx := testutil.TestContext()
	noop := func(bssciservices.EndpointSessionTx) error { return nil }

	t.Run("maps a begin failure onto the service sentinel and keeps the cause", func(t *testing.T) {
		b := endpointSessionTxBridge{run: endpointSeam("begin", storage.ErrNotFound, nil)}

		err := b.Run(ctx, noop)

		require.ErrorIs(t, err, bssciservices.ErrTxBegin, "the service begin sentinel must surface")
		require.ErrorIs(t, err, adapters.ErrTxBegin)
		require.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("maps a commit failure onto the service sentinel and keeps the cause", func(t *testing.T) {
		b := endpointSessionTxBridge{run: endpointSeam("commit", storage.ErrNotFound, nil)}

		err := b.Run(ctx, noop)

		require.ErrorIs(t, err, bssciservices.ErrTxCommit, "the service commit sentinel must surface")
		require.ErrorIs(t, err, adapters.ErrTxCommit)
		require.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("passes a joined rollback failure through with both causes matchable", func(t *testing.T) {
		b := endpointSessionTxBridge{run: endpointSeam("rollback", nil, storage.ErrInvalidInput)}
		fnErr := func(bssciservices.EndpointSessionTx) error { return storage.ErrNotFound }

		err := b.Run(ctx, fnErr)

		require.ErrorIs(t, err, storage.ErrNotFound, "the callback cause must survive")
		require.ErrorIs(t, err, storage.ErrInvalidInput, "the rollback cause must survive")
		require.NotErrorIs(t, err, bssciservices.ErrTxBegin)
		require.NotErrorIs(t, err, bssciservices.ErrTxCommit)
	})

	t.Run("returns nil on success", func(t *testing.T) {
		b := endpointSessionTxBridge{run: endpointSeam("success", nil, nil)}
		require.NoError(t, b.Run(ctx, noop))
	})

	t.Run("does not run the callback when begin fails", func(t *testing.T) {
		called := false
		run := func(_ context.Context, _ func(adapters.EndpointSessionOps) error) error {
			return fmt.Errorf("%w: %w", adapters.ErrTxBegin, storage.ErrNotFound)
		}
		b := endpointSessionTxBridge{run: run}

		_ = b.Run(ctx, func(bssciservices.EndpointSessionTx) error {
			called = true
			return nil
		})

		assert.False(t, called, "a begin failure must not run the transaction body")
	})
}

func TestSCACISessionTxBridge_Run(t *testing.T) {
	ctx := testutil.TestContext()
	noop := func(scaciservices.SessionCreationTx) error { return nil }
	failingAt := func(sentinel error) scaciSessionTxRun {
		return func(context.Context, func(adapters.SCACISessionTx) error) error {
			return fmt.Errorf("%w: %w", sentinel, storage.ErrNotFound)
		}
	}

	err := scaciSessionTxBridge{run: failingAt(adapters.ErrTxBegin)}.Run(ctx, noop)
	require.ErrorIs(t, err, scaciservices.ErrTxBegin, "the service begin sentinel must surface")
	require.ErrorIs(t, err, storage.ErrNotFound, "the underlying storage cause must survive")

	err = scaciSessionTxBridge{run: failingAt(adapters.ErrTxCommit)}.Run(ctx, noop)
	require.ErrorIs(t, err, scaciservices.ErrTxCommit, "the service commit sentinel must surface")
	require.ErrorIs(t, err, storage.ErrNotFound)

	ran := false
	err = scaciSessionTxBridge{run: func(_ context.Context, fn func(adapters.SCACISessionTx) error) error {
		return fn(nil)
	}}.Run(ctx, func(scaciservices.SessionCreationTx) error {
		ran = true
		return nil
	})
	require.NoError(t, err)
	assert.True(t, ran, "the creation runs inside the transaction")
}
