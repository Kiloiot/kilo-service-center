package messages

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// statsWindow records the window the service asks the store for.
type statsWindow struct {
	MessageStore
	start, end *time.Time
}

func (w *statsWindow) GetBaseStationStats(_ context.Context, _ int64, _ []byte, start, end *time.Time) (*mioty.BaseStationMessageStats, error) {
	w.start, w.end = start, end
	return &mioty.BaseStationMessageStats{}, nil
}

// Without a window the station summary covers every uplink, so its total is
// never below this week's or this month's.
func TestGetBaseStationMessageStats_WithoutAWindowCoversEveryUplink(t *testing.T) {
	store := &statsWindow{}
	svc := New(store, nil, testPollInterval, testOverlap, streamwake.NewSignal(), testBatchSize, logger.NewNop())

	_, err := svc.GetBaseStationMessageStats(testutil.TestContext(), testTenant, testStationEUI, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, store.start, "no lower bound")
	assert.Nil(t, store.end, "no upper bound")
}
