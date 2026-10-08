package bssci

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	roamingTestServingTenant = int64(7)
	roamingTestOwnerTenant   = int64(3)
)

var errRoamingProbe = errors.New("roaming probe failed")

type stubRoamingService struct {
	isRoaming bool
	owner     int64
	err       error
	calls     int
}

func (r *stubRoamingService) DetectAndValidateRoaming(_ context.Context, _ []byte, _ int64) (bool, int64, error) {
	r.calls++
	return r.isRoaming, r.owner, r.err
}
func (r *stubRoamingService) RecordAttach(context.Context, []byte, []byte, int64) error { return nil }
func (r *stubRoamingService) RecordDetach(context.Context, []byte, []byte, int64) error { return nil }
func (r *stubRoamingService) UpdateSessionRoaming(context.Context, int64, []byte, bool, int64) error {
	return nil
}

func TestEvaluateRoaming(t *testing.T) {
	epEui := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	t.Run("no roaming service serves the endpoint locally", func(t *testing.T) {
		s := &Server{}
		isRoaming, err := s.evaluateRoaming(testutil.TestContext(), epEui, roamingTestServingTenant)
		require.NoError(t, err)
		assert.False(t, isRoaming)
	})

	t.Run("a roaming endpoint is reported as roaming", func(t *testing.T) {
		svc := &stubRoamingService{isRoaming: true, owner: roamingTestOwnerTenant}
		s := &Server{roamingSvc: svc}
		isRoaming, err := s.evaluateRoaming(testutil.TestContext(), epEui, roamingTestServingTenant)
		require.NoError(t, err)
		assert.True(t, isRoaming)
		assert.Equal(t, 1, svc.calls)
	})

	t.Run("a failed probe is not roaming", func(t *testing.T) {
		s := &Server{roamingSvc: &stubRoamingService{err: errRoamingProbe}}
		isRoaming, err := s.evaluateRoaming(testutil.TestContext(), epEui, roamingTestServingTenant)
		require.ErrorIs(t, err, errRoamingProbe)
		assert.False(t, isRoaming)
	})
}

func TestDependenciesMissingNamesTheFirstGap(t *testing.T) {
	assert.Equal(t, "clock", Dependencies{}.missing())
	assert.Equal(t, "session service", Dependencies{Clock: clock.SystemClock{}}.missing())
	assert.Equal(t, "session service", ProtocolServices{}.missing())
	assert.Equal(t, "event store", StorageContracts{}.missing())
	assert.Equal(t, "organization directory", IdentityResolvers{}.missing())
	assert.Equal(t, "uplink ingest service", IngestPipeline{}.missing())
}
