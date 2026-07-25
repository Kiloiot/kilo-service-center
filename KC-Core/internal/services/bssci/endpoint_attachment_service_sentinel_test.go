package bssciservices

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
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
		st.beginErr = storage.ErrNotFound
		p := NewEndpointAttachmentPersistence(st, nil, logger.NewNop())

		err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

		require.ErrorIs(t, err, storage.ErrNotFound,
			"the storage sentinel must survive the wrapper")
		assert.Contains(t, err.Error(), bssci.LogBSSCIFailedToBeginTransaction,
			"the wrapper must carry the catalog log-message prefix")
	})

	t.Run("endpoint update failure", func(t *testing.T) {
		st, tx := newPersistFixture()
		tx.endpoints.updateErr = storage.ErrNotFound
		p := NewEndpointAttachmentPersistence(st, nil, logger.NewNop())

		err := p.PersistAttachSession(testutil.TestContext(), bssci.AttachSessionRecord{TenantID: 1, EndpointID: 1})

		require.ErrorIs(t, err, storage.ErrNotFound)
		assert.Contains(t, err.Error(), bssci.LogBSSCIFailedToUpdateEndpointAttachMetadata)
		assert.True(t, tx.rolledBack, "failed transaction must roll back")
	})
}
