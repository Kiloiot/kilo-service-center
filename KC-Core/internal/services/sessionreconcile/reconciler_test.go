package sessionreconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitestutil "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	testServiceCenterEUI = uint64(0x4B43000000000001)
	testReconciledLog    = "test sessions reconciled"
)

var errTestDatabaseUnavailable = errors.New("database unavailable")

// abandonedSessionStoreFake records the service center it reconciled for.
type abandonedSessionStoreFake struct {
	reconciledFor []models.EUI
	ids           []int64
	err           error
}

func (f *abandonedSessionStoreFake) DisconnectAbandonedSessions(_ context.Context, scEUI models.EUI) ([]int64, error) {
	f.reconciledFor = append(f.reconciledFor, scEUI)
	return f.ids, f.err
}

func TestNewRejectsMissingCollaborators(t *testing.T) {
	_, err := New(nil, testServiceCenterEUI, testReconciledLog, logger.NewNop())
	require.ErrorIs(t, err, ErrNilAbandonedSessionStore)
	_, err = New(&abandonedSessionStoreFake{}, testServiceCenterEUI, testReconciledLog, nil)
	require.ErrorIs(t, err, ErrNilReconcilerLogger)
}

// The reconciler acts for its own service center only and reports how many
// sessions became resumable again under its protocol's message.
func TestReconcilerReconcilesOwnSessions(t *testing.T) {
	store := &abandonedSessionStoreFake{ids: []int64{3, 5}}
	recorder := bsscitestutil.NewRecordingLogger()
	reconciler, err := New(store, testServiceCenterEUI, testReconciledLog, recorder)
	require.NoError(t, err)

	require.NoError(t, reconciler.ReconcileAbandonedSessions(testutil.TestContext()))

	assert.Equal(t, []models.EUI{models.EUI(mioty.EUI64(testServiceCenterEUI).ToBytes())}, store.reconciledFor)
	reported := recorder.FilterMessage(testReconciledLog)
	require.Len(t, reported, 1)
	assert.Equal(t, 2, reported[0].FieldMap()[logger.FieldCount])
}

// Nothing to reconcile logs nothing.
func TestReconcilerIsQuietWithoutAbandonedSessions(t *testing.T) {
	recorder := bsscitestutil.NewRecordingLogger()
	reconciler, err := New(&abandonedSessionStoreFake{}, testServiceCenterEUI, testReconciledLog, recorder)
	require.NoError(t, err)

	require.NoError(t, reconciler.ReconcileAbandonedSessions(testutil.TestContext()))
	assert.Empty(t, recorder.FilterMessage(testReconciledLog))
}

func TestReconcilerSurfacesStoreFailure(t *testing.T) {
	store := &abandonedSessionStoreFake{err: errTestDatabaseUnavailable}
	reconciler, err := New(store, testServiceCenterEUI, testReconciledLog, logger.NewNop())
	require.NoError(t, err)

	err = reconciler.ReconcileAbandonedSessions(testutil.TestContext())
	require.ErrorIs(t, err, errTestDatabaseUnavailable)
	assert.ErrorIs(t, err, errReconcileAbandonedSessions)
}
