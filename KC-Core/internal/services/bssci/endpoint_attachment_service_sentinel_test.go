package bssciservices

import (
	"fmt"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistAttachSession_WrappersPreserveSentinels: every contextual %w
// wrapper on the attach transaction keeps repository sentinels matchable via
// errors.Is while adding the catalog log-message prefix.
func TestPersistAttachSession_WrappersPreserveSentinels(t *testing.T) {
	t.Run("begin failure", func(t *testing.T) {
		st, _ := newPersistFixture()
		st.runErr = fmt.Errorf("%w: %w", ErrTxBegin, storage.ErrNotFound)
		p := newTestAttachmentPersistence(t, st)

		err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

		require.ErrorIs(t, err, storage.ErrNotFound,
			"the storage sentinel must survive the wrapper")
		assert.Contains(t, err.Error(), bssci.LogBSSCIFailedToBeginTransaction,
			"the wrapper must carry the catalog log-message prefix")
	})

	t.Run("endpoint update failure", func(t *testing.T) {
		st, tx := newPersistFixture()
		tx.endpoints.updateErr = storage.ErrNotFound
		p := newTestAttachmentPersistence(t, st)

		err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.Contains(t, err.Error(), bssci.LogBSSCIFailedToUpdateEndpointAttachMetadata)
		assert.Error(t, tx.endpoints.updateErr, "the repository failure is what aborts the transaction")
	})
}

// TestPersistAttachSession_LifecycleSentinelsRemainMatchable: each transaction
// lifecycle sentinel stays discoverable via errors.Is after the service adds
// its catalog message, and a callback's domain error passes through with its
// own identity intact.
func TestPersistAttachSession_LifecycleSentinelsRemainMatchable(t *testing.T) {
	t.Run("begin sentinel", func(t *testing.T) {
		st, _ := newPersistFixture()
		st.runErr = fmt.Errorf("%w: %w", ErrTxBegin, storage.ErrNotFound)
		p := newTestAttachmentPersistence(t, st)

		err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

		require.ErrorIs(t, err, ErrTxBegin,
			"the service-owned begin sentinel must survive the catalog wrapper")
	})

	t.Run("commit sentinel", func(t *testing.T) {
		st, _ := newPersistFixture()
		st.runErr = fmt.Errorf("%w: %w", ErrTxCommit, storage.ErrNotFound)
		p := newTestAttachmentPersistence(t, st)

		err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

		require.ErrorIs(t, err, ErrTxCommit,
			"the service-owned commit sentinel must survive the catalog wrapper")
		assert.Contains(t, err.Error(), bssci.LogBSSCIFailedToCommitAttachTransaction,
			"the wrapper must carry the catalog log-message prefix")
	})

	t.Run("callback domain error passes through", func(t *testing.T) {
		st, tx := newPersistFixture()
		tx.endpoints.updateErr = storage.ErrNotFound
		p := newTestAttachmentPersistence(t, st)

		err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

		require.ErrorIs(t, err, storage.ErrNotFound,
			"a domain failure inside the transaction keeps its identity")
		require.NotErrorIs(t, err, ErrTxBegin)
		require.NotErrorIs(t, err, ErrTxCommit)
	})
}
