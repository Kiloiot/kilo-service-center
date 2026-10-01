package builders

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitestutil "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const reconcileTestServiceCenterEUI = uint64(0x70B3D59CD0000001)

var errReconcileStoreDown = errors.New("session store down")

// staleSCACISessions holds the sessions a killed process left live.
type staleSCACISessions struct {
	live          []int64
	err           error
	reconciledFor []models.EUI
}

func (s *staleSCACISessions) DisconnectAbandonedSessions(_ context.Context, scEUI models.EUI) ([]int64, error) {
	s.reconciledFor = append(s.reconciledFor, scEUI)
	if s.err != nil {
		return nil, s.err
	}
	reconciled := s.live
	s.live = nil
	return reconciled, nil
}

// A start after a crash disconnects the sessions the previous process left
// live, for this service center only, so no session without a connection
// reports connected.
func TestReconcileSCACISessions_DisconnectsWhatAPreviousProcessLeftLive(t *testing.T) {
	store := &staleSCACISessions{live: []int64{1}}
	recorder := bsscitestutil.NewRecordingLogger()

	require.NoError(t, reconcileSCACISessions(testutil.TestContext(), store, reconcileTestServiceCenterEUI, recorder))

	assert.Empty(t, store.live)
	assert.Equal(t, []models.EUI{models.EUI(mioty.EUI64(reconcileTestServiceCenterEUI).ToBytes())}, store.reconciledFor)
	assert.Len(t, recorder.FilterMessage(scaci.LogSCACIReconciledAbandonedSessions), 1)
}

// A start that cannot reconcile the session rows does not start SCACI on
// them.
func TestReconcileSCACISessions_FailsTheStartWhenTheStoreFails(t *testing.T) {
	store := &staleSCACISessions{err: errReconcileStoreDown}

	err := reconcileSCACISessions(testutil.TestContext(), store, reconcileTestServiceCenterEUI, logger.NewNop())

	require.ErrorIs(t, err, errReconcileStoreDown)
}
