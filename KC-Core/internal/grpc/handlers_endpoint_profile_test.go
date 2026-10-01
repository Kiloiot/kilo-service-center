package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	endpointpkg "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

func propagatedAttachedEndpoint() *models.EndPoint {
	propagatedAt := time.Now().Add(-time.Hour)
	return &models.EndPoint{
		EUI:          models.EUIFromString("1122334455667788"),
		TenantID:     1,
		Name:         "profiled",
		EpStatus:     endpointpkg.EndpointStatusAttached,
		PropagatedAt: &propagatedAt,
	}
}

// An edit of a station profile parameter leaves an attached endpoint pending
// re-attach; an edit of anything else does not.
func TestUpdateEndPoint_ReportsReattachPendingForAStationProfileChange(t *testing.T) {
	cases := []struct {
		name    string
		sent    *pb.EndPoint
		path    string
		pending bool
	}{
		{"dual channel", &pb.EndPoint{EpEui: "1122334455667788", DualChan: true}, fieldMaskDualChan, true},
		{"name", &pb.EndPoint{EpEui: "1122334455667788", Name: "renamed"}, fieldMaskName, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: propagatedAttachedEndpoint()}
			service := testCoreService(coreFields{endpointSvc: mockSvc, log: &mockLogger{}})

			resp, err := service.UpdateEndPoint(testutil.TestContextWithTenant(1), &pb.UpdateEndPointRequest{
				Endpoint:   tc.sent,
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{tc.path}},
			})
			require.NoError(t, err)

			assert.Equal(t, tc.pending, resp.GetReattachPending())
			require.NotNil(t, mockSvc.updateCapture)
			assert.Equal(t, tc.pending, mockSvc.updateCapture.ProfileChangedAt != nil, "the profile change time is stored")
		})
	}
}
