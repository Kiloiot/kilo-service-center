package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// errWrappedNotFound is how the repositories report a row the tenant does
// not own: the storage sentinel wrapped with the failing operation.
var errWrappedNotFound = fmt.Errorf("lookup scoped to the caller's tenant: %w", storage.ErrNotFound)

// foreignEndpointSvc owns no endpoint of the caller's tenant.
type foreignEndpointSvc struct{ *fakeEndpointSvc }

func (foreignEndpointSvc) GetByEUI(context.Context, []byte, int64) (*models.EndPoint, error) {
	return nil, errWrappedNotFound
}

// foreignBasestationSvc owns no base station of the caller's tenant.
type foreignBasestationSvc struct{ *fakeBasestationSvc }

func (foreignBasestationSvc) GetByEUI(context.Context, []byte, int64) (*models.BaseStation, error) {
	return nil, errWrappedNotFound
}

// foreignEndpointMessages serves no DL RX data or serving base station for
// an endpoint the caller's tenant does not own.
type foreignEndpointMessages struct{ *fakeMessageSvc }

func (foreignEndpointMessages) ServingStation(context.Context, int64, uint64) (uint64, bool, error) {
	return 0, false, errWrappedNotFound
}

func (foreignEndpointMessages) GetDLRXStatusByEndpoint(context.Context, int64, []byte, int, int, *time.Time, *time.Time) ([]*mioty.DLRXStatus, int, error) {
	return nil, 0, errWrappedNotFound
}

// TestRevokeDownlink_EndpointOfAnotherTenantIsNotFound: an endpoint of tenant
// B requested by tenant A reads as not found, not as a lookup failure.
func TestRevokeDownlink_EndpointOfAnotherTenantIsNotFound(t *testing.T) {
	h := testCoreService(coreFields{
		endpointSvc: foreignEndpointSvc{&fakeEndpointSvc{}},
		messageSvc:  &fakeMessageSvc{},
		revoker:     &fakeDownlinkRevoker{},
		log:         logger.NewNop(),
	}).DownlinkHandlers

	_, err := h.RevokeDownlink(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), revokeRequest())

	requireRevokeCode(t, err, codes.NotFound)
}

// TestSendULTransmit_BaseStationOfAnotherTenantIsNotFound: a base station of
// tenant B named by tenant A reads as not found, not as a lookup failure.
func TestSendULTransmit_BaseStationOfAnotherTenantIsNotFound(t *testing.T) {
	h := testCoreService(coreFields{
		endpointSvc:    &fakeEndpointSvc{},
		basestationSvc: foreignBasestationSvc{&fakeBasestationSvc{}},
		log:            logger.NewNop(),
	}).ULTransmitHandlers

	_, err := h.SendULTransmit(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), &pb.SendULTransmitRequest{
		EpEui: revokeRequestEUI,
		BsEui: mioty.FormatEUI64(revokeHolderEUI),
	})

	requireRevokeCode(t, err, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound))
}

// TestQueryDLRXStatus_EndpointOfAnotherTenantIsNotAttached: the tenant-scoped
// serving base station lookup finds nothing for a foreign endpoint, which
// answers like an endpoint that is not attached.
func TestQueryDLRXStatus_EndpointOfAnotherTenantIsNotAttached(t *testing.T) {
	h := testCoreService(coreFields{
		messageSvc:  foreignEndpointMessages{&fakeMessageSvc{}},
		downlinkCmd: &fakeDownlinkCmd{},
		sessionDir:  &fakeSessionDirectory{},
		log:         logger.NewNop(),
	}).DLRXHandlers

	resp, err := h.QueryDLRXStatus(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), &pb.QueryDLRXStatusRequest{EpEui: revokeRequestEUI})

	require.NoError(t, err)
	assert.False(t, resp.QueryInitiated)
	assert.Equal(t, grpcerrors.MsgEndpointNotAttached, resp.Message)
}

// TestGetDLRXStatus_EndpointOfAnotherTenantHasNoStatus: a foreign endpoint has
// no DL RX status the caller can see.
func TestGetDLRXStatus_EndpointOfAnotherTenantHasNoStatus(t *testing.T) {
	h := testCoreService(coreFields{
		messageSvc: foreignEndpointMessages{&fakeMessageSvc{}},
		log:        logger.NewNop(),
	}).DLRXHandlers

	resp, err := h.GetDLRXStatus(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), &pb.GetDLRXStatusRequest{EpEui: revokeRequestEUI})

	require.NoError(t, err)
	assert.Empty(t, resp.Statuses)
}
