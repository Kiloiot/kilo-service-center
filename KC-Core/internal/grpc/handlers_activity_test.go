package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	activityTestTenant  = int64(1)
	activityTestStation = "70B3D59CD00009E6"
	activityTestDevice  = "70B3D56770111505"
	activityTestData    = `{"bsEui":"70B3D59CD00009E6","queId":118}`
)

// oneEventActivity answers both activity feeds with one downlink event.
type oneEventActivity struct{}

func (oneEventActivity) result() *grpcservices.ActivityListResult {
	event := &grpcservices.Event{
		ID: "evt-1", EventType: "dl_data_sent", Category: "message", Severity: "info",
		Title: "Downlink sent", SourceName: activityTestDevice, Timestamp: time.Unix(0, 0), Data: []byte(activityTestData),
	}
	items := []*grpcservices.ActivityItem{{Type: grpcservices.ActivityItemTypeEvent, OccurredAt: event.Timestamp, Event: event}}
	return &grpcservices.ActivityListResult{Items: items, TotalCount: int64(len(items))}
}

func (a oneEventActivity) ListBaseStationActivity(context.Context, int64, []byte, *grpcservices.ActivityFilters, int, string) (*grpcservices.ActivityListResult, error) {
	return a.result(), nil
}

func (a oneEventActivity) ListEndpointActivity(context.Context, int64, []byte, *grpcservices.ActivityFilters, int, string) (*grpcservices.ActivityListResult, error) {
	return a.result(), nil
}

func assertEventCarriesSourceAndData(t *testing.T, event *pb.Event) {
	t.Helper()
	require.NotNil(t, event)
	assert.Equal(t, activityTestDevice, event.SourceName, "the timeline names the event's source")
	assert.JSONEq(t, activityTestData, string(event.Data), "the event's details reach the timeline")
}

// Both activity feeds render an event exactly as the event log does.
func TestActivityFeeds_CarryTheEventSourceAndDetails(t *testing.T) {
	handlers := NewMessageHandlers(MessageHandlerDeps{Activity: oneEventActivity{}}, logger.NewNop())
	ctx := testutil.TestContextWithTenant(activityTestTenant)

	station, err := handlers.ListBaseStationActivity(ctx, &pb.ListBaseStationActivityRequest{BsEui: activityTestStation})
	require.NoError(t, err)
	require.Len(t, station.Items, 1)
	assertEventCarriesSourceAndData(t, station.Items[0].GetEvent())

	device, err := handlers.ListEndpointActivity(ctx, &pb.ListEndpointActivityRequest{EpEui: activityTestDevice})
	require.NoError(t, err)
	require.Len(t, device.Items, 1)
	assertEventCarriesSourceAndData(t, device.Items[0].GetEvent())
}
