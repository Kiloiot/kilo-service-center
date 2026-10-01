package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	scacimonitoring "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci_monitoring"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// orgScopedQueue answers like the downlink queue store: a filter with an
// organization lists only the rows that organization queued.
type orgScopedQueue struct{ rows []*storage.DownlinkMessage }

func (q *orgScopedQueue) matching(filter storage.DownlinkQueueFilter) []*storage.DownlinkMessage {
	var listed []*storage.DownlinkMessage
	for _, row := range q.rows {
		if filter.OrganizationID == nil || *row.OrganizationID == *filter.OrganizationID {
			listed = append(listed, row)
		}
	}
	return listed
}

func (q *orgScopedQueue) ListTenantQueue(_ context.Context, _ int64, filter storage.DownlinkQueueFilter, _, _ int) ([]*storage.DownlinkMessage, error) {
	return q.matching(filter), nil
}

func (q *orgScopedQueue) CountTenantQueue(_ context.Context, _ int64, filter storage.DownlinkQueueFilter) (int64, error) {
	return int64(len(q.matching(filter))), nil
}

// The SCACI queue listing is scoped like ListDownlinkQueue: another
// organization of the same tenant never sees the caller's downlinks.
func TestListScaciQueues_ListsOnlyTheRequestOrganization(t *testing.T) {
	const ownerRow, siblingRow = int64(850001), int64(850002)
	siblingOrg := uuid.MustParse("0d7b3a61-58c1-4f7e-9f5a-2b6de1c1a3aa")
	svc := newRetainedService(t, &retainedFakes{})
	svc.scaciMonitorSvc = scacimonitoring.New(scacimonitoring.Deps{
		Queue: &orgScopedQueue{rows: []*storage.DownlinkMessage{
			{ID: ownerRow, EPEUI: retainedBsEUIHex, Status: "pending", OrganizationID: &retainedOwnerOrg},
			{ID: siblingRow, EPEUI: retainedBsEUIHex, Status: "pending", OrganizationID: &siblingOrg},
		}},
		Log: logger.NewNop(),
	})

	resp, err := svc.ListScaciQueues(ownerCtx(), &pb.ListScaciQueuesRequest{})

	require.NoError(t, err)
	require.Len(t, resp.QueueEntries, 1, "listed a sibling organization's downlink")
	assert.Equal(t, "850001", resp.QueueEntries[0].Id)
	assert.Equal(t, int32(1), resp.TotalCount)

	_, err = svc.ListScaciQueues(testutil.TestContextWithTenant(retainedOwnerTenant), &pb.ListScaciQueuesRequest{})
	assertCode(t, err, grpcerrors.ErrTokenMissingOrgContext)
}

// idleSessions and idleOperations answer a status request for a tenant with
// no SCACI activity.
type idleSessions struct{}

func (idleSessions) GetSessionByID(context.Context, int64, int64) (*models.SCACISession, error) {
	return nil, storage.ErrNotFound
}

func (idleSessions) GetSessionStatistics(context.Context, int64) (*models.SCACISessionStatistics, error) {
	return &models.SCACISessionStatistics{}, nil
}

func (idleSessions) ListSessions(context.Context, *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	return nil, 0, nil
}

type idleOperations struct{}

func (idleOperations) GetTenantOperationSummaryBetween(context.Context, int64, time.Time, time.Time) (*models.SCACIOperationSummary, error) {
	return &models.SCACIOperationSummary{}, nil
}

func (idleOperations) CountOperationsBySession(context.Context, int64, []int64) (map[int64]int64, error) {
	return nil, nil
}

func (idleOperations) ListFailedOperationGroups(context.Context, int64, time.Time, time.Time, int, int) ([]*models.SCACIOperationErrorGroup, int64, error) {
	return nil, 0, nil
}

func (idleOperations) GetPingSummary(context.Context, int64, string, time.Time, time.Time, time.Duration) (*models.SCACIPingSummary, error) {
	return &models.SCACIPingSummary{}, nil
}

func (idleOperations) GetLatestOperation(context.Context, int64, string) (*models.SCACIOperation, error) {
	return nil, nil
}

// The SCACI status counts the pending operations of the request organization
// only, as the queue listing lists them: another organization of the same
// tenant never learns how many downlinks the caller has queued.
func TestGetScaciStatus_CountsOnlyTheRequestOrganization(t *testing.T) {
	siblingOrg := uuid.MustParse("0d7b3a61-58c1-4f7e-9f5a-2b6de1c1a3aa")
	svc := newRetainedService(t, &retainedFakes{})
	svc.scaciMonitorSvc = scacimonitoring.New(scacimonitoring.Deps{
		Sessions:   idleSessions{},
		Operations: idleOperations{},
		Queue: &orgScopedQueue{rows: []*storage.DownlinkMessage{
			{ID: 850101, EPEUI: retainedBsEUIHex, Status: "pending", OrganizationID: &retainedOwnerOrg},
			{ID: 850102, EPEUI: retainedBsEUIHex, Status: "pending", OrganizationID: &siblingOrg},
			{ID: 850103, EPEUI: retainedBsEUIHex, Status: "pending", OrganizationID: &siblingOrg},
		}},
		Clock: clock.SystemClock{},
		Log:   logger.NewNop(),
	})

	resp, err := svc.GetScaciStatus(ownerCtx(), &pb.GetScaciStatusRequest{})

	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.Status.PendingOperations, "counted a sibling organization's downlinks")

	_, err = svc.GetScaciStatus(testutil.TestContextWithTenant(retainedOwnerTenant), &pb.GetScaciStatusRequest{})
	assertCode(t, err, grpcerrors.ErrTokenMissingOrgContext)
}
