package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	modeTenant    = int64(1)
	foreignOwner  = int64(2)
	modeTestEUI   = "1122334455667788"
	modeOtherEUI  = "8877665544332211"
	modeAbsentEUI = "0101010101010101"
)

var errModeReadFailed = errors.New("mode read failed")

// memModes is a tenant switch store shared by every handler of one test, as the database is.
type memModes struct {
	mu      sync.Mutex
	enabled map[int64]bool
	err     error
}

func newMemModes(on ...int64) *memModes {
	m := &memModes{enabled: map[int64]bool{}}
	for _, tenant := range on {
		m.enabled[tenant] = true
	}
	return m
}

func (m *memModes) GetSnapshotOnlyDecoding(_ context.Context, tenantID int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return false, m.err
	}
	return m.enabled[tenantID], nil
}

func (m *memModes) SetSnapshotOnlyDecoding(_ context.Context, tenantID int64, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.enabled[tenantID] = enabled
	return nil
}

// catalog is a tenant's device catalog: two models, the first with a default blueprint.
type catalog struct {
	modelWithDefault, modelWithout, otherModel uuid.UUID
	defaultBp, otherModelBp                    *models.Blueprint
}

func newCatalog() catalog {
	c := catalog{modelWithDefault: uuid.New(), modelWithout: uuid.New(), otherModel: uuid.New()}
	c.defaultBp = &models.Blueprint{ID: uuid.New(), DeviceModelID: c.modelWithDefault, Version: "1",
		TypeEUI: []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 0x11}, SpecJSON: []byte(`{"d":1}`), IsDefault: true}
	c.otherModelBp = &models.Blueprint{ID: uuid.New(), DeviceModelID: c.otherModel, Version: "7",
		TypeEUI: []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 0x22}, SpecJSON: []byte(`{"o":1}`), IsDefault: true}
	return c
}

// catalogBlueprints answers the catalog reads of the endpoint and bulk paths from a catalog.
type catalogBlueprints struct {
	*mockBlueprintSvcForEndpoint
	defaults map[uuid.UUID]*models.Blueprint
}

func (c *catalogBlueprints) GetDefaultForModel(_ context.Context, _ int64, modelID uuid.UUID) (*models.Blueprint, error) {
	return c.defaults[modelID], nil
}

func (c catalog) service() *catalogBlueprints {
	byID := map[uuid.UUID]*models.Blueprint{c.defaultBp.ID: c.defaultBp, c.otherModelBp.ID: c.otherModelBp}
	defaults := map[uuid.UUID]*models.Blueprint{c.modelWithDefault: c.defaultBp, c.otherModel: c.otherModelBp}
	return &catalogBlueprints{defaults: defaults, mockBlueprintSvcForEndpoint: &mockBlueprintSvcForEndpoint{
		resolveTypeEUIFn: func(_ context.Context, _ int64, modelID uuid.UUID) (*models.EUI, error) {
			if bp := defaults[modelID]; bp != nil {
				var e models.EUI
				copy(e[:], bp.TypeEUI)
				return &e, nil
			}
			return nil, nil
		},
		getBlueprintFn: func(id uuid.UUID) (*models.Blueprint, error) {
			if bp := byID[id]; bp != nil {
				return bp, nil
			}
			return nil, storage.ErrNotFound
		},
	}}
}

func modeService(modes *memModes, endpoints *mockEndpointSvcForUpdate, blueprints *catalogBlueprints) (*CoreService, *createCapturingEndpoints) {
	created := &createCapturingEndpoints{mockEndpointSvcForUpdate: endpoints}
	svc := testCoreService(coreFields{endpointSvc: created, blueprintSvc: blueprints, snapshotModes: modes, log: &mockLogger{}})
	return svc, created
}

func newEndpointRequest(modelID, blueprintID string) *pb.CreateEndPointRequest {
	return &pb.CreateEndPointRequest{Endpoint: &pb.EndPoint{
		EpEui: modeTestEUI, Name: "mode", EpClass: mioty.EndpointClassBidirectional, NwkSnKey: testNetworkKey(),
		DeviceModelId: modelID, BlueprintId: blueprintID,
	}}
}

func requireCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, want, status.Code(err), "%v", err)
}

// TC-CM-107: creating an endpoint while its tenant decodes only from snapshots.
func TestCreateEndPoint_SnapshotOnlyModeEndsWithASnapshotOrARefusal(t *testing.T) {
	c := newCatalog()
	tests := []struct {
		name       string
		model      string
		blueprint  string
		wantCode   codes.Code
		wantSource string
	}{
		{name: "model with a default blueprint pins it", model: c.modelWithDefault.String(), wantSource: c.defaultBp.ID.String()},
		{name: "no model and no blueprint", wantCode: codes.FailedPrecondition},
		{name: "model without a default blueprint", model: c.modelWithout.String(), wantCode: codes.FailedPrecondition},
		{name: "blueprint of the requested model", model: c.otherModel.String(), blueprint: c.otherModelBp.ID.String(), wantSource: c.otherModelBp.ID.String()},
		{name: "blueprint of another model", model: c.modelWithDefault.String(), blueprint: c.otherModelBp.ID.String(), wantCode: codes.InvalidArgument},
		{name: "blueprint without a requested model", blueprint: c.otherModelBp.ID.String(), wantCode: codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, created := modeService(newMemModes(modeTenant), &mockEndpointSvcForUpdate{}, c.service())

			_, err := svc.CreateEndPoint(testutil.TestContextWithTenant(modeTenant), newEndpointRequest(tt.model, tt.blueprint))

			if tt.wantCode != codes.OK {
				requireCode(t, err, tt.wantCode)
				assert.Nil(t, created.created, "nothing is stored")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, created.created)
			assert.Equal(t, tt.wantSource, snapshotSourceID(created.created.BlueprintSnapshot))
		})
	}
}

// TC-CM-106: with the mode off, creation follows the catalog default as before.
func TestCreateEndPoint_ModeOffKeepsFollowingTheCatalog(t *testing.T) {
	c := newCatalog()
	for name, model := range map[string]string{"no model": "", "model without default": c.modelWithout.String(), "model with default": c.modelWithDefault.String()} {
		t.Run(name, func(t *testing.T) {
			svc, created := modeService(newMemModes(), &mockEndpointSvcForUpdate{}, c.service())

			_, err := svc.CreateEndPoint(testutil.TestContextWithTenant(modeTenant), newEndpointRequest(model, ""))

			require.NoError(t, err)
			require.NotNil(t, created.created)
			assert.Empty(t, created.created.BlueprintSnapshot, "without the mode an endpoint follows its catalog default")
		})
	}
}

func TestCreateEndPoint_ModeReadFailureRefuses(t *testing.T) {
	c := newCatalog()
	modes := newMemModes()
	modes.err = errModeReadFailed
	svc, created := modeService(modes, &mockEndpointSvcForUpdate{}, c.service())

	_, err := svc.CreateEndPoint(testutil.TestContextWithTenant(modeTenant), newEndpointRequest(c.modelWithDefault.String(), ""))

	requireCode(t, err, codes.Internal)
	assert.Nil(t, created.created)
}

// TC-CM-107: the switch is read from the store on every request, never from a cache.
func TestCreateEndPoint_SeesTheModeRightAfterItIsSwitchedOn(t *testing.T) {
	c := newCatalog()
	modes := newMemModes()
	svc, created := modeService(modes, &mockEndpointSvcForUpdate{}, c.service())
	ctx := testutil.TestContextWithTenant(modeTenant)

	_, err := svc.SetSnapshotOnlyDecoding(ctx, &pb.SetSnapshotOnlyDecodingRequest{Enabled: true})
	require.NoError(t, err)
	_, err = svc.CreateEndPoint(ctx, newEndpointRequest("", ""))

	requireCode(t, err, codes.FailedPrecondition)
	assert.Nil(t, created.created)
}

func ownedEndpoint(t *testing.T, owner int64, model *uuid.UUID, pinned *models.Blueprint) *models.EndPoint {
	t.Helper()
	ep := &models.EndPoint{EUI: models.EUIFromString(modeTestEUI), TenantID: modeTenant, OwnerTenantID: owner, DeviceModelID: model}
	if pinned != nil {
		snap, err := buildBlueprintSnapshot(pinned, testServiceStart)
		require.NoError(t, err)
		ep.BlueprintSnapshot = snap
	}
	return ep
}

func updateRequest(ep *pb.EndPoint, paths ...string) *pb.UpdateEndPointRequest {
	ep.EpEui = modeTestEUI
	return &pb.UpdateEndPointRequest{Endpoint: ep, UpdateMask: &fieldmaskpb.FieldMask{Paths: paths}}
}

// TC-CM-107: updating an endpoint whose owner decodes only from snapshots.
func TestUpdateEndPoint_SnapshotOnlyModeOfTheOwner(t *testing.T) {
	c := newCatalog()
	pinnedOther := c.otherModelBp
	tests := []struct {
		name           string
		owner          int64
		modesOn        []int64
		existing       func(t *testing.T) *models.EndPoint
		req            *pb.UpdateEndPointRequest
		wantCode       codes.Code
		wantSource     string
		wantNoSnapshot bool
		wantModel      *uuid.UUID
	}{
		{
			name: "model change pins the new model's default", owner: modeTenant, modesOn: []int64{modeTenant},
			existing:   func(t *testing.T) *models.EndPoint { return ownedEndpoint(t, modeTenant, &c.otherModel, pinnedOther) },
			req:        updateRequest(&pb.EndPoint{DeviceModelId: c.modelWithDefault.String()}, fieldMaskDeviceModelID),
			wantSource: c.defaultBp.ID.String(),
		},
		{
			name: "model change to a model without default is refused", owner: modeTenant, modesOn: []int64{modeTenant},
			existing: func(t *testing.T) *models.EndPoint { return ownedEndpoint(t, modeTenant, &c.otherModel, pinnedOther) },
			req:      updateRequest(&pb.EndPoint{DeviceModelId: c.modelWithout.String()}, fieldMaskDeviceModelID),
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "clearing the model is refused", owner: modeTenant, modesOn: []int64{modeTenant},
			existing: func(t *testing.T) *models.EndPoint { return ownedEndpoint(t, modeTenant, &c.otherModel, pinnedOther) },
			req:      updateRequest(&pb.EndPoint{}, fieldMaskDeviceModelID),
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "blueprint of another model is refused", owner: modeTenant, modesOn: []int64{modeTenant},
			existing: func(t *testing.T) *models.EndPoint {
				return ownedEndpoint(t, modeTenant, &c.modelWithDefault, c.defaultBp)
			},
			req: updateRequest(&pb.EndPoint{DeviceModelId: c.modelWithDefault.String(), BlueprintId: c.otherModelBp.ID.String()},
				fieldMaskDeviceModelID, fieldMaskBlueprintID),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "blueprint with an empty requested model is refused", owner: modeTenant, modesOn: []int64{modeTenant},
			existing: func(t *testing.T) *models.EndPoint {
				return ownedEndpoint(t, modeTenant, &c.modelWithDefault, nil)
			},
			req:      updateRequest(&pb.EndPoint{BlueprintId: c.defaultBp.ID.String()}, fieldMaskBlueprintID),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "blueprint of the requested model, which is the endpoint's own, is pinned", owner: modeTenant, modesOn: []int64{modeTenant},
			existing: func(t *testing.T) *models.EndPoint {
				return ownedEndpoint(t, modeTenant, &c.modelWithDefault, nil)
			},
			req:        updateRequest(&pb.EndPoint{DeviceModelId: c.modelWithDefault.String(), BlueprintId: c.defaultBp.ID.String()}, fieldMaskBlueprintID),
			wantSource: c.defaultBp.ID.String(),
			wantModel:  &c.modelWithDefault,
		},
		{
			name: "blueprint of the requested model is pinned", owner: modeTenant, modesOn: []int64{modeTenant},
			existing: func(t *testing.T) *models.EndPoint {
				return ownedEndpoint(t, modeTenant, &c.modelWithDefault, c.defaultBp)
			},
			req: updateRequest(&pb.EndPoint{DeviceModelId: c.otherModel.String(), BlueprintId: c.otherModelBp.ID.String()},
				fieldMaskDeviceModelID, fieldMaskBlueprintID),
			wantSource: c.otherModelBp.ID.String(),
		},
		{
			name: "an edit that changes neither model nor blueprint passes without a snapshot", owner: modeTenant, modesOn: []int64{modeTenant},
			existing:       func(t *testing.T) *models.EndPoint { return ownedEndpoint(t, modeTenant, &c.modelWithDefault, nil) },
			req:            updateRequest(&pb.EndPoint{Name: "renamed"}, fieldMaskName),
			wantNoSnapshot: true,
		},
		{
			name: "an unmasked device_model_id does not stand in for the endpoint's model", owner: modeTenant, modesOn: []int64{modeTenant},
			existing: func(t *testing.T) *models.EndPoint {
				return ownedEndpoint(t, modeTenant, &c.modelWithDefault, c.defaultBp)
			},
			req:      updateRequest(&pb.EndPoint{DeviceModelId: c.otherModel.String(), BlueprintId: c.otherModelBp.ID.String()}, fieldMaskBlueprintID),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "roaming: the owner's mode applies, not the serving tenant's", owner: foreignOwner, modesOn: []int64{foreignOwner},
			existing: func(t *testing.T) *models.EndPoint { return ownedEndpoint(t, foreignOwner, &c.otherModel, pinnedOther) },
			req:      updateRequest(&pb.EndPoint{DeviceModelId: c.modelWithout.String()}, fieldMaskDeviceModelID),
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "roaming: a serving tenant's mode does not bind another owner", owner: foreignOwner, modesOn: []int64{modeTenant},
			existing:   func(t *testing.T) *models.EndPoint { return ownedEndpoint(t, foreignOwner, &c.otherModel, pinnedOther) },
			req:        updateRequest(&pb.EndPoint{DeviceModelId: c.modelWithout.String()}, fieldMaskDeviceModelID),
			wantSource: pinnedOther.ID.String(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints := &mockEndpointSvcForUpdate{getByEUIResult: tt.existing(t)}
			svc, _ := modeService(newMemModes(tt.modesOn...), endpoints, c.service())

			_, err := svc.UpdateEndPoint(testutil.TestContextWithTenant(modeTenant), tt.req)

			if tt.wantCode != codes.OK {
				requireCode(t, err, tt.wantCode)
				assert.Nil(t, endpoints.updateCapture, "nothing is stored")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, endpoints.updateCapture)
			assert.Equal(t, tt.wantSource, snapshotSourceID(endpoints.updateCapture.BlueprintSnapshot))
			if tt.wantModel != nil {
				require.NotNil(t, endpoints.updateCapture.DeviceModelID)
				assert.Equal(t, *tt.wantModel, *endpoints.updateCapture.DeviceModelID, "the endpoint keeps its model")
			}
			if tt.wantNoSnapshot {
				assert.Empty(t, endpoints.updateCapture.BlueprintSnapshot, "no snapshot is pinned")
			}
		})
	}
}

// bulkEndpoints holds the endpoints a bulk assignment loads by EUI and records the writes.
type bulkEndpoints struct {
	*mockEndpointSvcForUpdate
	byEUI   map[string]*models.EndPoint
	readErr error
	written []*models.EndPoint
}

func (b *bulkEndpoints) GetByEUI(_ context.Context, eui []byte, _ int64) (*models.EndPoint, error) {
	if b.readErr != nil {
		return nil, b.readErr
	}
	var e models.EUI
	copy(e[:], eui)
	if ep, ok := b.byEUI[e.String()]; ok {
		return ep, nil
	}
	return nil, storage.ErrNotFound
}

func (b *bulkEndpoints) Update(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	b.written = append(b.written, ep)
	return ep, nil
}

func pinnedAt(t *testing.T, eui string, owner int64, model *uuid.UUID, pinned json.RawMessage) *models.EndPoint {
	t.Helper()
	return &models.EndPoint{EUI: models.EUIFromString(eui), TenantID: modeTenant, OwnerTenantID: owner, DeviceModelID: model, BlueprintSnapshot: pinned}
}

// TC-CM-107: BulkAssignBlueprint while an owner decodes only from snapshots.
func TestBulkAssignBlueprint_SnapshotOnlyModeOfEachOwner(t *testing.T) {
	c := newCatalog()
	pinned, err := buildBlueprintSnapshot(c.defaultBp, testServiceStart)
	require.NoError(t, err)
	inputPinned, err := buildSpecSnapshot(json.RawMessage(`{"i":1}`), "1", []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 0x33}, testServiceStart)
	require.NoError(t, err)
	tests := []struct {
		name        string
		modesOn     []int64
		endpoints   func(t *testing.T) []*models.EndPoint
		euis        []string
		wantCode    codes.Code
		wantWritten int
	}{
		{
			name: "every endpoint of the blueprint's model is written", modesOn: []int64{modeTenant},
			endpoints: func(t *testing.T) []*models.EndPoint {
				return []*models.EndPoint{pinnedAt(t, modeTestEUI, modeTenant, &c.modelWithDefault, pinned)}
			},
			euis: []string{modeTestEUI}, wantWritten: 1,
		},
		{
			name: "one endpoint of another model refuses the whole request", modesOn: []int64{modeTenant},
			endpoints: func(t *testing.T) []*models.EndPoint {
				return []*models.EndPoint{
					pinnedAt(t, modeTestEUI, modeTenant, &c.modelWithDefault, pinned),
					pinnedAt(t, modeOtherEUI, modeTenant, &c.otherModel, pinned),
				}
			},
			euis: []string{modeTestEUI, modeOtherEUI}, wantCode: codes.InvalidArgument,
		},
		{
			name: "an endpoint pinned from blueprint_input has no model and refuses the request", modesOn: []int64{modeTenant},
			endpoints: func(t *testing.T) []*models.EndPoint {
				return []*models.EndPoint{
					pinnedAt(t, modeTestEUI, modeTenant, &c.modelWithDefault, pinned),
					pinnedAt(t, modeOtherEUI, modeTenant, nil, inputPinned),
				}
			},
			euis: []string{modeTestEUI, modeOtherEUI}, wantCode: codes.InvalidArgument,
		},
		{
			name: "roaming: a serving tenant without the mode cannot repin an owner's endpoint of another model", modesOn: []int64{foreignOwner},
			endpoints: func(t *testing.T) []*models.EndPoint {
				return []*models.EndPoint{pinnedAt(t, modeTestEUI, foreignOwner, &c.otherModel, pinned)}
			},
			euis: []string{modeTestEUI}, wantCode: codes.InvalidArgument,
		},
		{
			name: "an unknown EUI is skipped, the others are written", modesOn: []int64{modeTenant},
			endpoints: func(t *testing.T) []*models.EndPoint {
				return []*models.EndPoint{pinnedAt(t, modeTestEUI, modeTenant, &c.modelWithDefault, pinned)}
			},
			euis: []string{modeAbsentEUI, modeTestEUI}, wantWritten: 1,
		},
		{
			name: "mode off: an endpoint of another model is repinned as before",
			endpoints: func(t *testing.T) []*models.EndPoint {
				return []*models.EndPoint{pinnedAt(t, modeTestEUI, modeTenant, &c.otherModel, pinned)}
			},
			euis: []string{modeTestEUI}, wantWritten: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints := &bulkEndpoints{mockEndpointSvcForUpdate: &mockEndpointSvcForUpdate{}, byEUI: map[string]*models.EndPoint{}}
			for _, ep := range tt.endpoints(t) {
				endpoints.byEUI[ep.EUI.String()] = ep
			}
			svc := testCoreService(coreFields{endpointSvc: endpoints, blueprintSvc: c.service(),
				snapshotModes: newMemModes(tt.modesOn...), log: &mockLogger{}})

			resp, err := svc.BulkAssignBlueprint(testutil.TestContextWithTenant(modeTenant),
				&pb.BulkAssignBlueprintRequest{BlueprintId: c.defaultBp.ID.String(), EpEuis: tt.euis})

			if tt.wantCode != codes.OK {
				requireCode(t, err, tt.wantCode)
				assert.Empty(t, endpoints.written, "nothing is written")
				return
			}
			require.NoError(t, err)
			assert.Len(t, endpoints.written, tt.wantWritten)
			assert.Equal(t, int32(tt.wantWritten), resp.GetAffectedCount())
		})
	}
}

func TestBulkAssignBlueprint_AReadFailureWritesNothing(t *testing.T) {
	c := newCatalog()
	endpoints := &bulkEndpoints{mockEndpointSvcForUpdate: &mockEndpointSvcForUpdate{}, readErr: errDbDown}
	svc := testCoreService(coreFields{endpointSvc: endpoints, blueprintSvc: c.service(), snapshotModes: newMemModes(), log: &mockLogger{}})

	_, err := svc.BulkAssignBlueprint(testutil.TestContextWithTenant(modeTenant),
		&pb.BulkAssignBlueprintRequest{BlueprintId: c.defaultBp.ID.String(), EpEuis: []string{modeTestEUI}})

	requireCode(t, err, codes.Internal)
	assert.Empty(t, endpoints.written)
}

// TC-CM-101: the switch is the caller's tenant's, and switching it is recorded.
func TestSnapshotOnlyDecoding_SwitchesTheCallersTenantAndIsRecorded(t *testing.T) {
	modes := newMemModes()
	recorder := &captureAuditRecorder{}
	svc := testCoreService(coreFields{snapshotModes: modes, audit: recorder, log: &mockLogger{}})
	ctx := testutil.TestContextWithTenant(modeTenant)

	resp, err := svc.SetSnapshotOnlyDecoding(ctx, &pb.SetSnapshotOnlyDecodingRequest{Enabled: true})
	require.NoError(t, err)
	assert.True(t, resp.GetEnabled())

	own, err := svc.GetSnapshotOnlyDecoding(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	assert.True(t, own.GetEnabled())
	other, err := svc.GetSnapshotOnlyDecoding(testutil.TestContextWithTenant(foreignOwner), &emptypb.Empty{})
	require.NoError(t, err)
	assert.False(t, other.GetEnabled(), "another tenant is not switched")

	require.Len(t, recorder.events, 1, "only the change is recorded")
	ev := recorder.events[0]
	assert.Equal(t, modeTenant, ev.TenantID)
	assert.Equal(t, models.EventCategoryEndpoint, ev.Category)
	assert.Equal(t, models.EventTypeSnapshotOnlyDecodingChanged, ev.EventType)
	assert.Equal(t, true, ev.Details[models.EventDetailKeySnapshotOnlyDecoding])
}

func TestSnapshotOnlyDecoding_AFailedSwitchIsNotRecorded(t *testing.T) {
	modes := newMemModes()
	modes.err = errModeReadFailed
	recorder := &captureAuditRecorder{}
	svc := testCoreService(coreFields{snapshotModes: modes, audit: recorder, log: &mockLogger{}})

	_, err := svc.SetSnapshotOnlyDecoding(testutil.TestContextWithTenant(modeTenant), &pb.SetSnapshotOnlyDecodingRequest{Enabled: true})

	requireCode(t, err, codes.Internal)
	assert.Empty(t, recorder.events)
}

// TC-CM-101: an organization's key cannot switch its tenant's decoding; an administrator can, and every role reads it.
func TestSnapshotOnlyDecoding_OnlyAnAdministratorSwitchesIt(t *testing.T) {
	set := pb.CoreService_SetSnapshotOnlyDecoding_FullMethodName
	get := pb.CoreService_GetSnapshotOnlyDecoding_FullMethodName

	assert.False(t, admitted(t, matrixInterceptor(authz.ServiceAccountRoles), set, false), "organization key")
	assert.False(t, admitted(t, matrixInterceptor(authz.Roles{TenantManager: true}), set, false), "tenant manager")
	assert.True(t, admitted(t, matrixInterceptor(authz.AllRoles), set, false), "administrator")
	assert.True(t, admitted(t, matrixInterceptor(authz.ServiceAccountRoles), get, false), "organization key reads")
}

func TestUpdateEndPoint_ModeReadFailureRefuses(t *testing.T) {
	c := newCatalog()
	modes := newMemModes()
	modes.err = errModeReadFailed
	endpoints := &mockEndpointSvcForUpdate{getByEUIResult: ownedEndpoint(t, modeTenant, &c.otherModel, c.otherModelBp)}
	svc, _ := modeService(modes, endpoints, c.service())

	_, err := svc.UpdateEndPoint(testutil.TestContextWithTenant(modeTenant),
		updateRequest(&pb.EndPoint{DeviceModelId: c.modelWithDefault.String()}, fieldMaskDeviceModelID))

	requireCode(t, err, codes.Internal)
	assert.Nil(t, endpoints.updateCapture)
}

func TestBulkAssignBlueprint_ModeReadFailureWritesNothing(t *testing.T) {
	c := newCatalog()
	pinned, err := buildBlueprintSnapshot(c.defaultBp, testServiceStart)
	require.NoError(t, err)
	endpoints := &bulkEndpoints{mockEndpointSvcForUpdate: &mockEndpointSvcForUpdate{}, byEUI: map[string]*models.EndPoint{
		models.EUIFromString(modeTestEUI).String(): pinnedAt(t, modeTestEUI, modeTenant, &c.modelWithDefault, pinned),
	}}
	modes := newMemModes()
	modes.err = errModeReadFailed
	svc := testCoreService(coreFields{endpointSvc: endpoints, blueprintSvc: c.service(), snapshotModes: modes, log: &mockLogger{}})

	_, err = svc.BulkAssignBlueprint(testutil.TestContextWithTenant(modeTenant),
		&pb.BulkAssignBlueprintRequest{BlueprintId: c.defaultBp.ID.String(), EpEuis: []string{modeTestEUI}})

	requireCode(t, err, codes.Internal)
	assert.Empty(t, endpoints.written)
}

func TestConstructors_RequireTheSnapshotOnlyDependencies(t *testing.T) {
	_, err := NewEndpointHandlers(EndpointHandlerDeps{Endpoints: &fakeEndpointSvc{}, Attachment: &mockEndpointAttachmentSvc{},
		Clock: clock.SystemClock{}, KeyReveals: &captureAuditRecorder{}}, &captureAuditRecorder{}, &mockLogger{})
	require.EqualError(t, err, errMsgSnapshotOnlyModesCannotBeNil)

	_, err = NewBlueprintHandlers(BlueprintHandlerDeps{Clock: clock.SystemClock{}}, &captureAuditRecorder{}, &mockLogger{})
	require.EqualError(t, err, errMsgSnapshotOnlyModesCannotBeNil)
	_, err = NewBlueprintHandlers(BlueprintHandlerDeps{Modes: switchOff{}}, &captureAuditRecorder{}, &mockLogger{})
	require.EqualError(t, err, errMsgBlueprintClockCannotBeNil)

	_, err = NewSnapshotOnlyHandlers(SnapshotOnlyHandlerDeps{Pinner: nothingToPin{}}, &captureAuditRecorder{}, &mockLogger{})
	require.EqualError(t, err, errMsgSnapshotOnlyModesCannotBeNil)
	_, err = NewSnapshotOnlyHandlers(SnapshotOnlyHandlerDeps{Modes: switchOff{}}, &captureAuditRecorder{}, &mockLogger{})
	require.EqualError(t, err, errMsgBlueprintPinnerCannotBeNil)
}
