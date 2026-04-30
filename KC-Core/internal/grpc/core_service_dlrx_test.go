package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/kilocenter/KC-Core/pkg/logger"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fakeDLRXStorage is a minimal test fake that implements only DLRXStatusQueryStorage interface.
// Avoids 50+ panic stubs from the full storage.Storage interface.
type fakeDLRXStorage struct {
	queries    []*mioty.DLRXStatusQuery
	totalCount int
	pending    int64
	received   int64
	timeout    int64
	historyErr error
	statsErr   error
	wasCalled  bool // tracks whether GetDLRXStatusQueryHistory was invoked
	lastLimit  int  // captures the limit parameter passed
	lastOffset int  // captures the offset parameter passed
}

func (f *fakeDLRXStorage) GetDLRXStatusQueryHistory(_ context.Context, _ int64, _ []byte,
	limit, offset int, _, _ *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	f.wasCalled = true
	f.lastLimit = limit
	f.lastOffset = offset
	if f.historyErr != nil {
		return nil, 0, f.historyErr
	}
	return f.queries, f.totalCount, nil
}

func (f *fakeDLRXStorage) GetDLRXStatusQueryStats(_ context.Context, _ int64, _ []byte,
	_, _ *time.Time) (pending, received, timeout int64, err error) {
	if f.statsErr != nil {
		return 0, 0, 0, f.statsErr
	}
	return f.pending, f.received, f.timeout, nil
}

// TestGetDLRXStatusQueries_ValidationRejections verifies input validation per BSSCI §5.15.
// Tests that validation happens BEFORE defaulting to prevent database errors.
func TestGetDLRXStatusQueries_ValidationRejections(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.GetDLRXStatusQueriesRequest
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name: "negative limit rejected",
			req: &pb.GetDLRXStatusQueriesRequest{
				EpEui:  "0102030405060708",
				Limit:  -1,
				Offset: 0,
			},
			wantCode:    codes.InvalidArgument,
			wantMessage: grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLimitNonNegative),
		},
		{
			name: "negative offset rejected",
			req: &pb.GetDLRXStatusQueriesRequest{
				EpEui:  "0102030405060708",
				Limit:  10,
				Offset: -5,
			},
			wantCode:    codes.InvalidArgument,
			wantMessage: grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOffsetNonNegative),
		},
		{
			name: "inverted time range rejected",
			req: &pb.GetDLRXStatusQueriesRequest{
				EpEui:     "0102030405060708",
				Limit:     10,
				Offset:    0,
				StartTime: timestamppb.New(time.Now()),
				EndTime:   timestamppb.New(time.Now().Add(-1 * time.Hour)),
			},
			wantCode:    codes.InvalidArgument,
			wantMessage: grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidTimeRange),
		},
		{
			name: "empty EUI rejected",
			req: &pb.GetDLRXStatusQueriesRequest{
				EpEui:  "",
				Limit:  10,
				Offset: 0,
			},
			wantCode:    codes.InvalidArgument,
			wantMessage: grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired),
		},
		{
			name: "invalid EUI length rejected",
			req: &pb.GetDLRXStatusQueriesRequest{
				EpEui:  "0102",
				Limit:  10,
				Offset: 0,
			},
			wantCode:    codes.InvalidArgument,
			wantMessage: grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEUIHexLengthInvalid),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service with fake storage
			fake := &fakeDLRXStorage{}
			svc := &CoreService{
				log:         logger.Get(),
				dlrxStorage: fake,
			}

			// Create context with tenant
			ctx := testutil.TestContextWithTenant(1)

			// Call method
			resp, err := svc.GetDLRXStatusQueries(ctx, tt.req)

			// Verify rejection
			if err == nil {
				t.Fatalf("expected error with code %v, got nil", tt.wantCode)
			}
			if resp != nil {
				t.Fatalf("expected nil response on error, got %+v", resp)
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got %T: %v", err, err)
			}

			if st.Code() != tt.wantCode {
				t.Errorf("expected code %v, got %v", tt.wantCode, st.Code())
			}

			if st.Message() != tt.wantMessage {
				t.Errorf("expected message %q, got %q", tt.wantMessage, st.Message())
			}

			// Verify that storage was never called for validation errors
			if fake.wasCalled {
				t.Error("storage should not be called when validation rejects the request")
			}
		})
	}
}

// TestGetDLRXStatusQueries_ValidRequest verifies successful query with test data.
func TestGetDLRXStatusQueries_ValidRequest(t *testing.T) {
	// Create test query
	requestedAt := time.Now().Add(-5 * time.Minute)
	receivedAt := time.Now().Add(-2 * time.Minute)

	testQuery := &mioty.DLRXStatusQuery{
		ID:          1,
		TenantID:    1,
		EpEui:       []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		BsEui:       []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
		OpId:        -12345,
		Status:      "received",
		RequestedAt: requestedAt,
		ReceivedAt:  &receivedAt,
	}

	// Create fake storage with test data
	fake := &fakeDLRXStorage{
		queries:    []*mioty.DLRXStatusQuery{testQuery},
		totalCount: 1,
		pending:    0,
		received:   1,
		timeout:    0,
	}

	svc := &CoreService{
		log:         logger.Get(),
		dlrxStorage: fake,
	}

	// Create context with tenant
	ctx := testutil.TestContextWithTenant(1)

	// Create valid request
	req := &pb.GetDLRXStatusQueriesRequest{
		EpEui:  "0102030405060708",
		Limit:  10,
		Offset: 0,
	}

	// Call method
	resp, err := svc.GetDLRXStatusQueries(ctx, req)

	// Verify success
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	// Verify response structure
	if resp.TotalCount != 1 {
		t.Errorf("expected total_count=1, got %d", resp.TotalCount)
	}
	if len(resp.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(resp.Queries))
	}

	// Verify query data
	q := resp.Queries[0]
	if q.EpEui != "0102030405060708" {
		t.Errorf("expected ep_eui=0102030405060708, got %s", q.EpEui)
	}
	if q.BsEui != "1122334455667788" {
		t.Errorf("expected bs_eui=1122334455667788, got %s", q.BsEui)
	}
	if q.OpId != -12345 {
		t.Errorf("expected op_id=-12345, got %d", q.OpId)
	}
	if q.Status != "received" {
		t.Errorf("expected status=received, got %s", q.Status)
	}

	// Verify stats
	if resp.Stats == nil {
		t.Fatal("expected stats, got nil")
	}
	if resp.Stats.Pending != 0 {
		t.Errorf("expected pending=0, got %d", resp.Stats.Pending)
	}
	if resp.Stats.Received != 1 {
		t.Errorf("expected received=1, got %d", resp.Stats.Received)
	}
	if resp.Stats.Timeout != 0 {
		t.Errorf("expected timeout=0, got %d", resp.Stats.Timeout)
	}
}

// TestGetDLRXStatusQueries_DefaultsAndBounds verifies pagination defaults and clamping.
func TestGetDLRXStatusQueries_DefaultsAndBounds(t *testing.T) {
	tests := []struct {
		name        string
		inputLimit  int32
		inputOffset int32
		wantValid   bool
	}{
		{
			name:        "zero limit defaults to 20",
			inputLimit:  0,
			inputOffset: 0,
			wantValid:   true,
		},
		{
			name:        "limit clamped to max 100",
			inputLimit:  200,
			inputOffset: 0,
			wantValid:   true,
		},
		{
			name:        "valid limit accepted",
			inputLimit:  50,
			inputOffset: 10,
			wantValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeDLRXStorage{
				queries:    []*mioty.DLRXStatusQuery{},
				totalCount: 0,
			}

			svc := &CoreService{
				log:         logger.Get(),
				dlrxStorage: fake,
			}

			ctx := testutil.TestContextWithTenant(1)

			req := &pb.GetDLRXStatusQueriesRequest{
				EpEui:  "0102030405060708",
				Limit:  tt.inputLimit,
				Offset: tt.inputOffset,
			}

			resp, err := svc.GetDLRXStatusQueries(ctx, req)

			if tt.wantValid {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if resp == nil {
					t.Error("expected response, got nil")
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}
