package messages

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

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

func scopedUplinks(window RegistrationWindow) (*Service, *uplinkStore) {
	store := newUplinkStore()
	store.add(&mioty.ULDataMessage{RxTime: endpointRegistered.Add(-time.Hour).UnixNano()})
	store.add(&mioty.ULDataMessage{RxTime: endpointRegistered.Add(time.Minute).UnixNano()})
	return New(store, window, testPollInterval, testOverlap, streamwake.NewSignal(), testBatchSize, logger.NewNop()), store
}

// An endpoint's uplinks start at its registration: those an earlier
// registration of the EUI received stay with it.
func TestListMessages_OfAnEndpointStartAtItsRegistration(t *testing.T) {
	svc, _ := scopedUplinks(registrationAt{at: endpointRegistered, registered: true})

	uplinks, total, err := svc.ListMessages(testutil.TestContext(), 1,
		&grpcservices.MessageFilters{EpEui: []byte{0x70}}, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, uplinks, 1)
	assert.Equal(t, endpointRegistered.Add(time.Minute).UnixNano(), uplinks[0].RxTime)
}

// An EUI the tenant has not registered lists no uplinks.
func TestListMessages_OfAnUnregisteredEndpointAreEmpty(t *testing.T) {
	svc, _ := scopedUplinks(registrationAt{})

	uplinks, total, err := svc.ListMessages(testutil.TestContext(), 1,
		&grpcservices.MessageFilters{EpEui: []byte{0x70}}, 10, 0)

	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, uplinks)
}

// Tenant-wide and station listings are not narrowed by any registration.
func TestListMessages_WithoutAnEndpointListEveryUplink(t *testing.T) {
	svc, _ := scopedUplinks(registrationAt{})

	_, total, err := svc.ListMessages(testutil.TestContext(), 1, &grpcservices.MessageFilters{}, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}
