package activity

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const activityTenant = int64(1)

var (
	activityEndpoint = []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x05}
	registeredAt     = time.Date(2026, 9, 28, 12, 47, 16, 0, time.UTC)
	beforeDeletion   = registeredAt.Add(-24 * time.Hour)
	afterRegistering = registeredAt.Add(time.Hour)
)

// deviceHistory holds a device's events and uplinks, and applies the start
// of the window the way the stores do.
type deviceHistory struct {
	events   []*grpcservices.Event
	uplinks  []*mioty.ULDataMessage
	eventsAt *time.Time
}

func (h *deviceHistory) ListByBaseStation(context.Context, int64, []byte, *grpcservices.EventFilters, int, int) ([]*grpcservices.Event, int64, error) {
	return h.events, int64(len(h.events)), nil
}

func (h *deviceHistory) ListByEndPoint(_ context.Context, _ int64, _ []byte, f *grpcservices.EventFilters, _, _ int) ([]*grpcservices.Event, int64, error) {
	h.eventsAt = f.StartTime
	var kept []*grpcservices.Event
	for _, e := range h.events {
		if f.StartTime == nil || !e.Timestamp.Before(*f.StartTime) {
			kept = append(kept, e)
		}
	}
	return kept, int64(len(kept)), nil
}

func (h *deviceHistory) ListBaseStationMessages(context.Context, int64, []byte, *grpcservices.MessageFilters, int, int) ([]*mioty.ULDataMessage, int64, error) {
	return h.uplinks, int64(len(h.uplinks)), nil
}

func (h *deviceHistory) ListMessages(_ context.Context, _ int64, f *grpcservices.MessageFilters, _, _ int) ([]*mioty.ULDataMessage, int64, error) {
	var kept []*mioty.ULDataMessage
	for _, m := range h.uplinks {
		if f.StartTime == nil || !time.Unix(0, m.RxTime).Before(*f.StartTime) {
			kept = append(kept, m)
		}
	}
	return kept, int64(len(kept)), nil
}

func history() *deviceHistory {
	return &deviceHistory{
		events: []*grpcservices.Event{
			{ID: "deleted-registration", Timestamp: beforeDeletion},
			{ID: "this-registration", Timestamp: afterRegistering},
		},
		uplinks: []*mioty.ULDataMessage{
			{ID: "old-uplink", RxTime: beforeDeletion.UnixNano()},
			{ID: "new-uplink", RxTime: afterRegistering.UnixNano()},
		},
	}
}

func ids(result *grpcservices.ActivityListResult) []string {
	var got []string
	for _, item := range result.Items {
		if item.Event != nil {
			got = append(got, item.Event.ID)
		} else {
			got = append(got, item.Message.ID)
		}
	}
	return got
}

// The endpoint feed reads the operator's window from both readers, which
// start the endpoint's history at its registration themselves.
func TestListEndpointActivity_PassesTheWindowToBothReaders(t *testing.T) {
	events := history()
	svc := New(events, history(), logger.NewNop())
	later := afterRegistering.Add(-time.Minute)

	result, err := svc.ListEndpointActivity(authz.WithRoles(testutil.TestContext(), authz.AllRoles), activityTenant, activityEndpoint,
		&grpcservices.ActivityFilters{StartTime: &later}, 0, "")

	require.NoError(t, err)
	require.NotNil(t, events.eventsAt)
	assert.Equal(t, later, *events.eventsAt)
	assert.ElementsMatch(t, []string{"this-registration", "new-uplink"}, ids(result))
}
