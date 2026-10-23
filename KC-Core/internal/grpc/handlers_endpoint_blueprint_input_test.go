package grpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	pkgblueprint "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	inputSpecTypeEUI = "70B3D59CD00000A1"
	inputSpec        = `{"version":"2.1","typeEui":"70b3d59cd00000a1",
		"uplink":[{"id":0,"payload":[{"name":"t","size":8,"type":"uint"}]}]}`
	inputSpecNoUplinkNoDownlink = `{"version":"2.1","typeEui":"70B3D59CD00000A1"}`
	inputSpecDuplicateFormatID  = `{"version":"2.1","typeEui":"70B3D59CD00000A1",
		"uplink":[{"id":3,"payload":[{"name":"a","size":8,"type":"uint"}]},
		          {"id":3,"payload":[{"name":"b","size":8,"type":"uint"}]}]}`
	inputSpecBadTypeEUI = `{"version":"2.1","typeEui":"zz",
		"uplink":[{"id":0,"payload":[{"name":"t","size":8,"type":"uint"}]}]}`
	inputTestEUI       = "1122334455667788"
	pathBlueprintInput = "blueprint_input"
)

// untouchedBlueprints panics on any catalog call: a blueprint_input request never resolves a model or blueprint.
type untouchedBlueprints struct{ grpcservices.BlueprintService }

// createCapturingEndpoints records the endpoint CreateEndPoint stores.
type createCapturingEndpoints struct {
	*mockEndpointSvcForUpdate
	created *models.EndPoint
}

func (m *createCapturingEndpoints) Create(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	m.created = ep
	return ep, nil
}

func inputSnapshotOf(t *testing.T, raw json.RawMessage) models.BlueprintSnapshot {
	t.Helper()
	require.NotEmpty(t, raw, "a snapshot is pinned")
	var snap models.BlueprintSnapshot
	require.NoError(t, json.Unmarshal(raw, &snap))
	return snap
}

func assertSnapshotFromInput(t *testing.T, ep *models.EndPoint) {
	t.Helper()
	snap := inputSnapshotOf(t, ep.BlueprintSnapshot)
	assert.JSONEq(t, inputSpec, string(snap.SpecJSON), "the snapshot carries the specification")
	assert.Equal(t, "2.1", snap.Version)
	assert.Equal(t, inputSpecTypeEUI, snap.TypeEUI)
	assert.Empty(t, snap.SourceBlueprintID, "no catalog blueprint is the source")
	assert.False(t, snap.IsSystem)
	assert.False(t, snap.AdoptedAt.IsZero())
	assert.Nil(t, ep.DeviceModelID, "no KC model")
	require.NotNil(t, ep.TypeEUI)
	assert.Equal(t, inputSpecTypeEUI, mioty.FormatEUIBytes(ep.TypeEUI[:]), "type_eui comes from the specification")
}

func catalogPinnedEndpoint(t *testing.T) *models.EndPoint {
	t.Helper()
	model := uuid.New()
	snap, err := buildBlueprintSnapshot(&models.Blueprint{ID: uuid.New(), DeviceModelID: model,
		TypeEUI: []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 1}, Version: "1", SpecJSON: []byte(`{"a":1}`)})
	require.NoError(t, err)
	return &models.EndPoint{EUI: models.EUIFromString(inputTestEUI), TenantID: 1, OwnerTenantID: 1,
		DeviceModelID: &model, BlueprintSnapshot: snap}
}

func createWithInput(spec string) *pb.CreateEndPointRequest {
	return &pb.CreateEndPointRequest{Endpoint: &pb.EndPoint{
		EpEui: inputTestEUI, Name: "input", EpClass: mioty.EndpointClassBidirectional, NwkSnKey: testNetworkKey(),
		TypeEui:        []byte{0xde, 0xad, 0xbe, 0xef, 0, 0, 0, 0},
		BlueprintInput: &pb.BlueprintInput{Spec: []byte(spec)},
	}}
}

// TC-CM-103
func TestCreateEndPoint_BlueprintInputPinsASnapshotWithoutAModel(t *testing.T) {
	endpoints := &createCapturingEndpoints{mockEndpointSvcForUpdate: &mockEndpointSvcForUpdate{}}
	svc := testCoreService(coreFields{endpointSvc: endpoints, blueprintSvc: untouchedBlueprints{}, log: &mockLogger{}})

	resp, err := svc.CreateEndPoint(testutil.TestContextWithTenant(1), createWithInput(inputSpec))
	require.NoError(t, err)

	require.NotNil(t, endpoints.created)
	assertSnapshotFromInput(t, endpoints.created)
	assert.Nil(t, resp.GetBlueprintInput(), "the input is never echoed")
	assert.Empty(t, resp.GetBlueprintId(), "no source blueprint to echo")
	assert.Empty(t, resp.GetDeviceModelId())
	assert.Equal(t, []byte(endpoints.created.BlueprintSnapshot), resp.GetBlueprintSnapshot(), "the built snapshot is returned whole")
}

// TC-CM-103
func TestUpdateEndPoint_BlueprintInputReplacesTheSnapshotAndClearsTheModel(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
	}{
		{name: "with device_model_id cleared in the mask", paths: []string{pathBlueprintInput, fieldMaskDeviceModelID}},
		{name: "blueprint_input alone", paths: []string{pathBlueprintInput}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints := &mockEndpointSvcForUpdate{getByEUIResult: catalogPinnedEndpoint(t)}
			svc := testCoreService(coreFields{endpointSvc: endpoints, blueprintSvc: untouchedBlueprints{}, log: &mockLogger{}})
			req := &pb.UpdateEndPointRequest{
				Endpoint:   &pb.EndPoint{EpEui: inputTestEUI, BlueprintInput: &pb.BlueprintInput{Spec: []byte(inputSpec)}},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: tt.paths},
			}

			resp, err := svc.UpdateEndPoint(testutil.TestContextWithTenant(1), req)
			require.NoError(t, err)

			require.NotNil(t, endpoints.updateCapture)
			assertSnapshotFromInput(t, endpoints.updateCapture)
			assert.Nil(t, resp.GetBlueprintInput())
			assert.Empty(t, resp.GetBlueprintId())
		})
	}
}

func TestUpdateEndPoint_EmptyBlueprintInputIsNoInput(t *testing.T) {
	existing := catalogPinnedEndpoint(t)
	prior := existing.BlueprintSnapshot
	priorModel := *existing.DeviceModelID
	endpoints := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	svc := testCoreService(coreFields{endpointSvc: endpoints, blueprintSvc: &mockBlueprintSvcForEndpoint{}, log: &mockLogger{}})
	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: inputTestEUI, Name: "renamed", BlueprintInput: &pb.BlueprintInput{}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{fieldMaskName, pathBlueprintInput}},
	}

	_, err := svc.UpdateEndPoint(testutil.TestContextWithTenant(1), req)
	require.NoError(t, err)

	require.NotNil(t, endpoints.updateCapture)
	assert.Equal(t, "renamed", endpoints.updateCapture.Name)
	assert.Equal(t, prior, endpoints.updateCapture.BlueprintSnapshot, "the snapshot is untouched")
	require.NotNil(t, endpoints.updateCapture.DeviceModelID)
	assert.Equal(t, priorModel, *endpoints.updateCapture.DeviceModelID)
}

type blueprintInputRefusal struct {
	name          string
	spec          string
	blueprintID   string
	deviceModelID string
	wantToken     string
	wantDetail    string
}

func blueprintInputRefusals() []blueprintInputRefusal {
	return []blueprintInputRefusal{
		{name: "with blueprint_id", spec: inputSpec, blueprintID: uuid.NewString(), wantToken: grpcerrors.ErrTokenBlueprintInputConflict},
		{name: "with device_model_id", spec: inputSpec, deviceModelID: uuid.NewString(), wantToken: grpcerrors.ErrTokenBlueprintInputConflict},
		{name: "without uplink and downlink", spec: inputSpecNoUplinkNoDownlink, wantToken: grpcerrors.ErrTokenBlueprintInputSpecInvalid,
			wantDetail: pkgblueprint.ResolveErrorMessage(pkgblueprint.ErrMissingUplinkDefs)},
		{name: "duplicate format id", spec: inputSpecDuplicateFormatID, wantToken: grpcerrors.ErrTokenBlueprintInputSpecInvalid,
			wantDetail: pkgblueprint.ResolveErrorMessage(pkgblueprint.ErrDuplicateFormatID)},
		{name: "typeEui not 16 hex", spec: inputSpecBadTypeEUI, wantToken: grpcerrors.ErrTokenBlueprintInputTypeEUIInvalid},
	}
}

func assertBlueprintInputRefused(t *testing.T, err error, tt blueprintInputRefusal) {
	t.Helper()
	require.Error(t, err)
	st := status.Convert(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, grpcerrors.GetGRPCCode(tt.wantToken), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(tt.wantToken))
	if tt.wantDetail != "" {
		assert.Contains(t, st.Message(), tt.wantDetail, "the refusal carries the parse text")
	}
}

// TC-CM-104
func TestCreateEndPoint_BlueprintInputRefusalsCreateNothing(t *testing.T) {
	for _, tt := range blueprintInputRefusals() {
		t.Run(tt.name, func(t *testing.T) {
			endpoints := &createCapturingEndpoints{mockEndpointSvcForUpdate: &mockEndpointSvcForUpdate{}}
			svc := testCoreService(coreFields{endpointSvc: endpoints, blueprintSvc: untouchedBlueprints{}, log: &mockLogger{}})
			req := createWithInput(tt.spec)
			req.Endpoint.BlueprintId = tt.blueprintID
			req.Endpoint.DeviceModelId = tt.deviceModelID

			_, err := svc.CreateEndPoint(testutil.TestContextWithTenant(1), req)

			assertBlueprintInputRefused(t, err, tt)
			assert.Nil(t, endpoints.created, "nothing is stored")
		})
	}
}

// TC-CM-104
func TestUpdateEndPoint_BlueprintInputRefusalsChangeNothing(t *testing.T) {
	for _, tt := range blueprintInputRefusals() {
		t.Run(tt.name, func(t *testing.T) {
			endpoints := &mockEndpointSvcForUpdate{getByEUIResult: catalogPinnedEndpoint(t)}
			svc := testCoreService(coreFields{endpointSvc: endpoints, blueprintSvc: untouchedBlueprints{}, log: &mockLogger{}})
			req := &pb.UpdateEndPointRequest{
				Endpoint: &pb.EndPoint{EpEui: inputTestEUI, Name: "renamed", BlueprintId: tt.blueprintID,
					DeviceModelId: tt.deviceModelID, BlueprintInput: &pb.BlueprintInput{Spec: []byte(tt.spec)}},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{fieldMaskName, pathBlueprintInput}},
			}

			_, err := svc.UpdateEndPoint(testutil.TestContextWithTenant(1), req)

			assertBlueprintInputRefused(t, err, tt)
			assert.Nil(t, endpoints.updateCapture, "nothing is stored")
			assert.Nil(t, endpoints.updateWithEUICapture, "nothing is stored")
			assert.Empty(t, endpoints.getByEUIResult.Name, "refused before any field is applied")
		})
	}
}
