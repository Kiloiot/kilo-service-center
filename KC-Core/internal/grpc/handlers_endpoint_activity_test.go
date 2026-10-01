package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	testActivityWindow = time.Hour
	testActiveEpEui    = "0000000000000001"
)

var testLastSeen = time.Date(2020, 1, 15, 12, 0, 0, 0, time.UTC)

type fixedHandlerClock struct{ now time.Time }

func (c fixedHandlerClock) Now() time.Time { return c.now }

func TestGetEndPoint_ActivityFollowsTheInjectedClock(t *testing.T) {
	lastSeen := testLastSeen
	endpoints := &mockEndpointSvcIsolation{
		getByEUIFunc: func(_ context.Context, _ []byte, tenantID int64) (*models.EndPoint, error) {
			return &models.EndPoint{TenantID: tenantID, LastSeenAt: &lastSeen}, nil
		},
	}
	svc := testCoreService(coreFields{endpointSvc: endpoints, endpointActivityWindow: testActivityWindow, log: logger.NewNop()})

	svc.clock = fixedHandlerClock{now: testLastSeen.Add(testActivityWindow / 2)}
	resp, err := svc.GetEndPoint(contextForTenant(testOwnerTenant), &pb.GetEndPointRequest{EpEui: testActiveEpEui})
	require.NoError(t, err)
	assert.Equal(t, endpointActivityActive, resp.Status, "seen inside the window measured from the injected clock")

	svc.clock = fixedHandlerClock{now: testLastSeen.Add(2 * testActivityWindow)}
	resp, err = svc.GetEndPoint(contextForTenant(testOwnerTenant), &pb.GetEndPointRequest{EpEui: testActiveEpEui})
	require.NoError(t, err)
	assert.Equal(t, endpointActivityInactive, resp.Status)
}
