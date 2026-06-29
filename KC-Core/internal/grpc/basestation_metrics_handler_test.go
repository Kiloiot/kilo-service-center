package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

type stubAvailabilityReader struct{}

func (stubAvailabilityReader) GetBaseStationOnlineIntervals(_ context.Context, _, _ int64, _, _ time.Time,
) ([]mioty.BaseStationOnlineInterval, error) {
	return nil, nil
}

type stubMessageBucketReader struct{}

func (stubMessageBucketReader) CountBaseStationMessagesByBucket(_ context.Context, _ int64, _ []byte, _, _ time.Time,
	_ int64,
) (map[int64]int64, error) {
	return nil, nil
}

func metricsTestWindow() (startUnix, endUnix int64) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return start.Unix(), start.Add(24 * time.Hour).Unix()
}

// A cross-tenant or non-existent base station must surface NotFound, never fake zero data.
func TestGetBaseStationAvailability_NotFound(t *testing.T) {
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			return nil, storage.ErrNotFound
		},
	}
	svc := &CoreService{
		basestationSvc:     mockBsSvc,
		availabilityReader: stubAvailabilityReader{},
		log:                &mockLogger{},
	}

	startUnix, endUnix := metricsTestWindow()
	ctx := testutil.TestContextWithTenant(100)
	resp, err := svc.GetBaseStationAvailability(ctx, &pb.GetBaseStationAvailabilityRequest{
		BsEui:           "AAAAAAAAAAAAAAAA",
		StartTime:       timestamppb.New(time.Unix(startUnix, 0)),
		EndTime:         timestamppb.New(time.Unix(endUnix, 0)),
		IntervalSeconds: 3600,
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound), st.Message())
}

func TestGetBaseStationMessagesReceived_NotFound(t *testing.T) {
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			return nil, storage.ErrNotFound
		},
	}
	svc := &CoreService{
		basestationSvc:      mockBsSvc,
		messageBucketReader: stubMessageBucketReader{},
		log:                 &mockLogger{},
	}

	startUnix, endUnix := metricsTestWindow()
	ctx := testutil.TestContextWithTenant(100)
	resp, err := svc.GetBaseStationMessagesReceived(ctx, &pb.GetBaseStationMessagesReceivedRequest{
		BsEui:           "AAAAAAAAAAAAAAAA",
		StartTime:       timestamppb.New(time.Unix(startUnix, 0)),
		EndTime:         timestamppb.New(time.Unix(endUnix, 0)),
		IntervalSeconds: 3600,
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound), st.Message())
}

// An invalid window (non-positive interval / bad bounds) must be InvalidArgument.
func TestGetBaseStationAvailability_InvalidRequest(t *testing.T) {
	svc := &CoreService{
		basestationSvc:     &mockBasestationSvc{},
		availabilityReader: stubAvailabilityReader{},
		log:                &mockLogger{},
	}

	startUnix, endUnix := metricsTestWindow()
	ctx := testutil.TestContextWithTenant(100)
	resp, err := svc.GetBaseStationAvailability(ctx, &pb.GetBaseStationAvailabilityRequest{
		BsEui:           "AAAAAAAAAAAAAAAA",
		StartTime:       timestamppb.New(time.Unix(startUnix, 0)),
		EndTime:         timestamppb.New(time.Unix(endUnix, 0)),
		IntervalSeconds: 0, // invalid
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}
