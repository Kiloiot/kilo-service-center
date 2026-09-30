package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	endpointpkg "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	kcerrors "github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	preAttachTestTenant = int64(42)
	preAttachTestEUI    = "70B3D56770111505"
)

var errPreAttachRefused = errors.New("the endpoint cannot be attached")

// preAttachTable is the endpoints table CreateEndPoint and the attachment
// service share: one endpoint per EUI, stored detached unless it is stored
// with a status.
type preAttachTable struct {
	mockEndpointSvcForDuplicate
	stored map[models.EUI]*models.EndPoint
}

func newPreAttachTable() *preAttachTable {
	return &preAttachTable{stored: make(map[models.EUI]*models.EndPoint)}
}

func (t *preAttachTable) Create(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	return t.store(ep, endpointpkg.EndpointStatusDetached)
}

func (t *preAttachTable) CreateWithStatus(_ context.Context, ep *models.EndPoint, status string) (*models.EndPoint, error) {
	return t.store(ep, status)
}

func (t *preAttachTable) store(ep *models.EndPoint, status string) (*models.EndPoint, error) {
	if _, exists := t.stored[ep.EUI]; exists {
		return nil, kcerrors.ErrDuplicate
	}
	stored := *ep
	stored.ID = int64(len(t.stored) + 1)
	stored.EpStatus = status
	t.stored[ep.EUI] = &stored
	clone := stored
	return &clone, nil
}

func (t *preAttachTable) GetByEUI(_ context.Context, eui []byte, _ int64) (*models.EndPoint, error) {
	var key models.EUI
	copy(key[:], eui)
	stored, ok := t.stored[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	clone := *stored
	return &clone, nil
}

// preAttachAttachments is the attachment service over the table: it records
// the endpoints it attached and, while refusing, attaches none.
type preAttachAttachments struct {
	table    *preAttachTable
	refuse   bool
	attached []string
}

func (a *preAttachAttachments) AttachEndPoint(_ context.Context, epEui string, _ int64) (*grpcservices.EndpointOperationResult, error) {
	if a.refuse {
		return nil, errPreAttachRefused
	}
	a.table.stored[models.EUIFromString(epEui)].EpStatus = endpointpkg.EndpointStatusAttached
	a.attached = append(a.attached, epEui)
	return &grpcservices.EndpointOperationResult{Status: bssci.OperationStatusInitiated}, nil
}

func (a *preAttachAttachments) DetachEndPoint(context.Context, string, int64) (*grpcservices.EndpointOperationResult, error) {
	return &grpcservices.EndpointOperationResult{Status: bssci.OperationStatusInitiated}, nil
}

func (a *preAttachAttachments) CreateAttached(ctx context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	if a.refuse {
		return nil, errPreAttachRefused
	}
	created, err := a.table.CreateWithStatus(ctx, ep, endpointpkg.EndpointStatusAttached)
	if err != nil {
		return nil, err
	}
	a.attached = append(a.attached, created.EUI.String())
	return created, nil
}

func preAttachService(table *preAttachTable, attachments *preAttachAttachments) *CoreService {
	return testCoreService(coreFields{endpointSvc: table, endpointAttachmentSvc: attachments, log: &mockLogger{}})
}

func createPreAttachEndpoint(svc *CoreService, preAttach bool) (*pb.EndPoint, error) {
	return svc.CreateEndPoint(testutil.TestContextWithTenant(preAttachTestTenant), &pb.CreateEndPointRequest{Endpoint: &pb.EndPoint{
		EpClass:      mioty.EndpointClassBidirectional,
		EpEui:        preAttachTestEUI,
		Name:         "pre-attached",
		NwkSnKey:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		PreAttach:    preAttach,
		Status:       endpointpkg.EndpointStatusAttached,
		AttachStatus: endpointpkg.EndpointStatusAttached,
	}})
}

// An endpoint created with pre-attachment is attached through the attachment
// service when it is created (BSSCI §3.8 offline pre-attachment); one
// created without it stays detached until an explicit or over-the-air
// attach, whatever status the request names.
func TestCreateEndPoint_PreAttachmentAttachesTheNewEndpoint(t *testing.T) {
	table := newPreAttachTable()
	attachments := &preAttachAttachments{table: table}
	created, err := createPreAttachEndpoint(preAttachService(table, attachments), true)
	require.NoError(t, err)
	assert.Equal(t, []string{preAttachTestEUI}, attachments.attached, "the new endpoint is attached once")
	assert.Equal(t, endpointpkg.EndpointStatusAttached, created.AttachStatus)

	table = newPreAttachTable()
	attachments = &preAttachAttachments{table: table}
	created, err = createPreAttachEndpoint(preAttachService(table, attachments), false)
	require.NoError(t, err)
	assert.Empty(t, attachments.attached, "without pre-attachment nothing is attached")
	assert.Equal(t, endpointpkg.EndpointStatusDetached, created.AttachStatus)
}

// A pre-attached create whose attachment is refused stores nothing, so the
// client can send the same create again once the endpoint can be attached.
func TestCreateEndPoint_RefusedPreAttachmentStoresNothing(t *testing.T) {
	table := newPreAttachTable()
	attachments := &preAttachAttachments{table: table, refuse: true}
	svc := preAttachService(table, attachments)

	_, err := createPreAttachEndpoint(svc, true)
	require.Error(t, err)
	assert.Empty(t, table.stored, "a refused pre-attached create stores no endpoint")

	attachments.refuse = false
	created, err := createPreAttachEndpoint(svc, true)
	require.NoError(t, err, "the refused create left no endpoint behind, so it can be sent again")
	assert.Equal(t, endpointpkg.EndpointStatusAttached, created.AttachStatus)
}

// A zero-filled network key can never be sent to a base station, so a create
// that names one is refused before anything is stored.
func TestCreateEndPoint_ZeroFilledNetworkKeyStoresNothing(t *testing.T) {
	table := newPreAttachTable()
	svc := preAttachService(table, &preAttachAttachments{table: table})

	_, err := svc.CreateEndPoint(testutil.TestContextWithTenant(preAttachTestTenant), &pb.CreateEndPointRequest{Endpoint: &pb.EndPoint{
		EpClass:  mioty.EndpointClassBidirectional,
		EpEui:    preAttachTestEUI,
		Name:     "zero key",
		NwkSnKey: make([]byte, 16),
	}})

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyZero), st.Message())
	assert.Empty(t, table.stored, "no endpoint is left behind")
}
