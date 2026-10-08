package events

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const scopedEndpoint = "70B3D56770111505"

var endpointRegistered = time.Date(2026, 9, 28, 12, 47, 16, 0, time.UTC)

// registrationAt answers one registration time, or none.
type registrationAt struct {
	at         time.Time
	registered bool
}

func (r registrationAt) Start(_ context.Context, _ int64, _ []byte, requested *time.Time) (*time.Time, bool, error) {
	if !r.registered {
		return nil, false, nil
	}
	if requested != nil && requested.After(r.at) {
		return requested, true, nil
	}
	return &r.at, true, nil
}

func scopedEvents(window RegistrationWindow) *Service {
	store := newMockEventStore()
	store.add(
		&models.SystemEvent{ID: "1", CreatedAt: endpointRegistered.Add(-time.Hour)},
		&models.SystemEvent{ID: "2", CreatedAt: endpointRegistered.Add(time.Minute)},
	)
	return New(store, &mockResolver{}, window, time.Second, testStreamOverlap, streamwake.NewSignal(), 100, &mockLogger{})
}

// An endpoint's events start at its registration, in the Logs views
// (ListEvents filtered by the endpoint) and on its activity timeline.
func TestEndpointEvents_StartAtItsRegistration(t *testing.T) {
	svc := scopedEvents(registrationAt{at: endpointRegistered, registered: true})
	ctx := testutil.TestContext()

	listed, total, err := svc.List(ctx, 1, &grpcservices.EventFilters{EpEUI: scopedEndpoint}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, listed, 1)

	timeline, total, err := svc.ListByEndPoint(ctx, 1, []byte{0x70}, &grpcservices.EventFilters{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, timeline, 1)
}

// An EUI the tenant has not registered has no events in either view.
func TestEndpointEvents_OfAnUnregisteredEndpointAreEmpty(t *testing.T) {
	svc := scopedEvents(registrationAt{})
	ctx := testutil.TestContext()

	listed, total, err := svc.List(ctx, 1, &grpcservices.EventFilters{EpEUI: scopedEndpoint}, 10, 0)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, listed)

	timeline, _, err := svc.ListByEndPoint(ctx, 1, []byte{0x70}, nil, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, timeline)
}

// Tenant-wide event listings are not narrowed by any registration.
func TestEvents_WithoutAnEndpointListEveryEvent(t *testing.T) {
	svc := scopedEvents(registrationAt{})

	_, total, err := svc.List(testutil.TestContext(), 1, &grpcservices.EventFilters{}, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}
