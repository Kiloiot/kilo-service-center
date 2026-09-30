package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

var errResolverDown = errors.New("resolver down")

// failingResolver fails every device lookup.
type failingResolver struct{}

func (failingResolver) ResolveBaseStationID(context.Context, int64, []byte) (*int64, error) {
	return nil, errResolverDown
}

func (failingResolver) ResolveEndpointID(context.Context, int64, []byte) (*int64, error) {
	return nil, errResolverDown
}

var scopeTestEUI = []byte{0x70, 0xb3, 0xd5, 0x67, 0x70, 0x11, 0x15, 0x05}

func TestListByDevice_FailedLookupFailsTheListing(t *testing.T) {
	store := newMockEventStore()
	svc := New(store, failingResolver{}, registrationAt{registered: true}, time.Second, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})
	ctx := testutil.TestContext()

	_, _, err := svc.ListByBaseStation(ctx, 1, scopeTestEUI, nil, 10, 0)
	require.ErrorIs(t, err, errResolverDown, "a failed lookup must not silently widen the base station scope to the EUI alone")

	_, _, err = svc.ListByEndPoint(ctx, 1, scopeTestEUI, nil, 10, 0)
	require.ErrorIs(t, err, errResolverDown)
	assert.Zero(t, store.listCallCount, "no events are read under an unresolved scope")
}

func TestListByDevice_UnknownDeviceFallsBackToTheEUI(t *testing.T) {
	store := newMockEventStore()
	svc := New(store, &mockResolver{}, registrationAt{registered: true}, time.Second, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})

	_, _, err := svc.ListByBaseStation(testutil.TestContext(), 1, scopeTestEUI, nil, 10, 0)

	require.NoError(t, err)
	require.NotNil(t, store.lastFilter)
	assert.Nil(t, store.lastFilter.BaseStationID)
	assert.NotEmpty(t, store.lastFilter.BaseStationEUI)
}
