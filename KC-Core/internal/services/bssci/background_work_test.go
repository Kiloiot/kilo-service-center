package bssciservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// TestBackgroundWork_StopGivesUpWhenTheContextEnds: Stop waits for running
// work only as long as its context allows.
func TestBackgroundWork_StopGivesUpWhenTheContextEnds(t *testing.T) {
	work := NewBackgroundWork()
	release := make(chan struct{})
	work.Go(testutil.TestContext(), func(context.Context) { <-release })
	ended, cancel := context.WithCancel(testutil.TestContext())
	cancel()

	require.ErrorIs(t, work.Stop(ended), context.Canceled)

	close(release)
	require.NoError(t, work.Stop(testutil.TestContext()))
}
