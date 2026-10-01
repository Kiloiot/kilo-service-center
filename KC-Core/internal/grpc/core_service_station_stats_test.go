package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// stationStatsListing answers the station statistics with fixed figures.
type stationStatsListing struct {
	grpcservices.MessageListingService
	stats *mioty.BaseStationMessageStats
}

func (l stationStatsListing) GetBaseStationMessageStats(context.Context, int64, []byte, *time.Time, *time.Time) (*mioty.BaseStationMessageStats, error) {
	return l.stats, nil
}

func TestGetBaseStationMessageStats_ReportsTheFirstAndLastUplink(t *testing.T) {
	first := time.Date(2026, time.September, 27, 11, 9, 50, 0, time.UTC)
	last := time.Date(2026, time.September, 28, 23, 36, 23, 0, time.UTC)
	svc := testCoreService(coreFields{
		log: testLogger{},
		msgListingSvc: stationStatsListing{stats: &mioty.BaseStationMessageStats{
			TotalMessages: 40, FirstMessageAt: &first, LastMessageAt: &last,
		}},
	})

	resp, err := svc.GetBaseStationMessageStats(testutil.TestContextWithTenant(1),
		&pb.GetBaseStationMessageStatsRequest{BsEui: "70B3D59CD00009E6"})
	require.NoError(t, err)
	require.NotNil(t, resp.Stats.FirstMessageAt, "the first uplink reaches the Traffic summary")
	assert.True(t, first.Equal(resp.Stats.FirstMessageAt.AsTime()))
	assert.True(t, last.Equal(resp.Stats.LastMessageAt.AsTime()))
}
