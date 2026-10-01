package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	dlrxQueryEUI        = "0102030405060708"
	dlrxQueryNonHexEUI  = "01020304050607zz"
	dlrxQueryWindowFrom = int64(1_790_000_000)
)

// TestGetDLRXStatusQueries_OpenEndedWindowIsAccepted: a window with only a
// start, or only an end, is no inverted range.
func TestGetDLRXStatusQueries_OpenEndedWindowIsAccepted(t *testing.T) {
	fake := &fakeDLRXStorage{}
	svc := testCoreService(coreFields{log: logger.NewNop(), dlrxStorage: fake})
	at := timestamppb.New(time.Unix(dlrxQueryWindowFrom, 0))

	for name, req := range map[string]*pb.GetDLRXStatusQueriesRequest{
		"start only": {EpEui: dlrxQueryEUI, StartTime: at},
		"end only":   {EpEui: dlrxQueryEUI, EndTime: at},
	} {
		_, err := svc.GetDLRXStatusQueries(testutil.TestContextWithTenant(1), req)
		require.NoError(t, err, name)
	}
	assert.True(t, fake.wasCalled)
}

// TestDLRXQueries_ReadTheEndpointAlike: both DL RX status query RPCs refuse
// an endpoint EUI with a non-hex digit the same way.
func TestDLRXQueries_ReadTheEndpointAlike(t *testing.T) {
	svc := testCoreService(coreFields{log: logger.NewNop(), dlrxStorage: &fakeDLRXStorage{}, messageSvc: &fakeMessageSvc{}})
	want := grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEUIHexCharsInvalid)

	_, err := svc.GetDLRXStatusQueries(testutil.TestContextWithTenant(1), &pb.GetDLRXStatusQueriesRequest{EpEui: dlrxQueryNonHexEUI})
	assert.Equal(t, want, status.Convert(err).Message(), "GetDLRXStatusQueries")
	_, err = svc.QueryDLRXStatus(testutil.TestContextWithTenant(1), &pb.QueryDLRXStatusRequest{EpEui: dlrxQueryNonHexEUI})
	assert.Equal(t, want, status.Convert(err).Message(), "QueryDLRXStatus")
}
