package blueprint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var errCatalogRead = errors.New("catalog unreadable")

const pinOwner = int64(9)

// catalogRepo answers default and Type EUI lookups from maps.
type catalogRepo struct {
	stubBlueprintRepo
	defaults   map[uuid.UUID]*models.Blueprint
	byTypeEUI  map[string]*models.Blueprint
	defaultErr error
	typeEUIErr error
}

func (r *catalogRepo) GetDefaultForModel(_ context.Context, _ int64, modelID uuid.UUID) (*models.Blueprint, error) {
	if r.defaultErr != nil {
		return nil, r.defaultErr
	}
	if bp := r.defaults[modelID]; bp != nil {
		return bp, nil
	}
	return nil, storage.ErrNotFound
}

func (r *catalogRepo) GetByTypeEUI(_ context.Context, _ int64, typeEUI []byte) (*models.Blueprint, error) {
	if r.typeEUIErr != nil {
		return nil, r.typeEUIErr
	}
	if bp := r.byTypeEUI[string(typeEUI)]; bp != nil {
		return bp, nil
	}
	return nil, storage.ErrRecordNotFound
}

// unpinnedStore pages endpoints without a snapshot and records the pinned snapshots.
type unpinnedStore struct {
	endpoints []*models.EndPoint
	pages     int
	pinned    map[int64]json.RawMessage
	taken     map[int64]bool
	owners    []int64
}

func (s *unpinnedStore) ListUnpinnedByOwner(_ context.Context, ownerTenantID, afterID int64, limit int) ([]*models.EndPoint, error) {
	s.pages++
	s.owners = append(s.owners, ownerTenantID)
	var page []*models.EndPoint
	for _, ep := range s.endpoints {
		if ep.ID > afterID && len(page) < limit {
			page = append(page, ep)
		}
	}
	return page, nil
}

func (s *unpinnedStore) PinSnapshotIfUnpinned(_ context.Context, ownerTenantID, endpointID int64, snapshot json.RawMessage) (bool, error) {
	s.owners = append(s.owners, ownerTenantID)
	if s.taken[endpointID] {
		return false, nil
	}
	if s.pinned == nil {
		s.pinned = map[int64]json.RawMessage{}
	}
	s.pinned[endpointID] = snapshot
	return true, nil
}

func unpinned(id int64, model *uuid.UUID, typeEUI []byte) *models.EndPoint {
	ep := &models.EndPoint{ID: id, EUI: models.EUIFromString(fmt.Sprintf("%016x", 0x7000+id)), OwnerTenantID: pinOwner, DeviceModelID: model}
	if typeEUI != nil {
		var e models.EUI
		copy(e[:], typeEUI)
		ep.TypeEUI = &e
	}
	return ep
}

func newPinService(t *testing.T, repo Reader, store *unpinnedStore) *PinService {
	t.Helper()
	resolver := newResolver(t, logger.NewNop(), repo, fixedMode{err: errModeSource})
	svc, err := NewPinService(resolver, store, func() time.Time { return time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC) }, logger.NewNop())
	require.NoError(t, err)
	return svc
}

// TC-CM-108: each endpoint without a snapshot is pinned to what decoding would use without the switch.
func TestPin_PinsWhatTheResolverWouldUseWithoutTheSwitch(t *testing.T) {
	model := uuid.New()
	typeEUI := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 0x55}
	byModel := &models.Blueprint{ID: uuid.New(), DeviceModelID: model, Version: "3", TypeEUI: []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 0x44}, SpecJSON: json.RawMessage(`{"m":1}`)}
	byType := &models.Blueprint{ID: uuid.New(), DeviceModelID: uuid.New(), Version: "4", TypeEUI: typeEUI, SpecJSON: json.RawMessage(`{"t":1}`)}
	repo := &catalogRepo{defaults: map[uuid.UUID]*models.Blueprint{model: byModel}, byTypeEUI: map[string]*models.Blueprint{string(typeEUI): byType}}
	store := &unpinnedStore{endpoints: []*models.EndPoint{
		unpinned(1, &model, nil),
		unpinned(2, nil, typeEUI),
		unpinned(3, nil, nil),
	}}

	results, err := newPinService(t, repo, store).Pin(testutil.TestContext(), pinOwner, false)

	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, PinResult{EpEUI: store.endpoints[0].EUI.String(), Outcome: PinOutcomePinned, BlueprintID: byModel.ID.String()}, results[0])
	assert.Equal(t, PinResult{EpEUI: store.endpoints[1].EUI.String(), Outcome: PinOutcomePinned, BlueprintID: byType.ID.String()}, results[1])
	assert.Equal(t, PinResult{EpEUI: store.endpoints[2].EUI.String(), Outcome: PinOutcomeNothingToPin}, results[2])
	var snap models.BlueprintSnapshot
	require.NoError(t, json.Unmarshal(store.pinned[2], &snap))
	assert.Equal(t, byType.ID.String(), snap.SourceBlueprintID)
	assert.JSONEq(t, `{"t":1}`, string(snap.SpecJSON))
	assert.Equal(t, "4", snap.Version)
	assert.NotContains(t, store.pinned, int64(3))
	for _, owner := range store.owners {
		assert.Equal(t, pinOwner, owner, "every read and write is the owner's")
	}
}

// TC-CM-108: a failed catalog read is an error, never "nothing to pin".
func TestPin_CatalogReadFailuresAreErrors(t *testing.T) {
	model := uuid.New()
	typeEUI := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 0x55}
	for name, repo := range map[string]*catalogRepo{
		"default of the model": {defaultErr: errCatalogRead},
		"by type EUI":          {typeEUIErr: errCatalogRead},
	} {
		t.Run(name, func(t *testing.T) {
			ep := unpinned(1, &model, nil)
			if repo.typeEUIErr != nil {
				ep = unpinned(1, nil, typeEUI)
			}
			store := &unpinnedStore{endpoints: []*models.EndPoint{ep}}

			results, err := newPinService(t, repo, store).Pin(testutil.TestContext(), pinOwner, false)

			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, PinOutcomeError, results[0].Outcome)
			assert.NotEmpty(t, results[0].Error)
			assert.Empty(t, store.pinned)
		})
	}
}

// TC-CM-108: one endpoint's failure does not stop the others.
func TestPin_OneFailureLeavesTheOthersPinned(t *testing.T) {
	good, bad := uuid.New(), uuid.New()
	bp := &models.Blueprint{ID: uuid.New(), DeviceModelID: good, Version: "1", SpecJSON: json.RawMessage(`{}`)}
	repo := &failingModelRepo{catalogRepo: catalogRepo{defaults: map[uuid.UUID]*models.Blueprint{good: bp}}, failing: bad}
	store := &unpinnedStore{endpoints: []*models.EndPoint{unpinned(1, &bad, nil), unpinned(2, &good, nil)}}

	results, err := newPinService(t, repo, store).Pin(testutil.TestContext(), pinOwner, false)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, PinOutcomeError, results[0].Outcome)
	assert.Equal(t, PinOutcomePinned, results[1].Outcome)
	assert.Contains(t, store.pinned, int64(2))
}

type failingModelRepo struct {
	catalogRepo
	failing uuid.UUID
}

func (r *failingModelRepo) GetDefaultForModel(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.Blueprint, error) {
	if modelID == r.failing {
		return nil, errCatalogRead
	}
	return r.catalogRepo.GetDefaultForModel(ctx, tenantID, modelID)
}

// TC-CM-108: a dry run reports the same outcomes and writes nothing.
func TestPin_DryRunWritesNothing(t *testing.T) {
	model := uuid.New()
	bp := &models.Blueprint{ID: uuid.New(), DeviceModelID: model, Version: "1", SpecJSON: json.RawMessage(`{}`)}
	store := &unpinnedStore{endpoints: []*models.EndPoint{unpinned(1, &model, nil)}}

	results, err := newPinService(t, &catalogRepo{defaults: map[uuid.UUID]*models.Blueprint{model: bp}}, store).Pin(testutil.TestContext(), pinOwner, true)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, PinResult{EpEUI: store.endpoints[0].EUI.String(), Outcome: PinOutcomePinned, BlueprintID: bp.ID.String()}, results[0])
	assert.Empty(t, store.pinned)
}

// TC-CM-108: an endpoint pinned by another write meanwhile is not reported as pinned by this call.
func TestPin_AnEndpointPinnedMeanwhileIsLeftOut(t *testing.T) {
	model := uuid.New()
	bp := &models.Blueprint{ID: uuid.New(), DeviceModelID: model, Version: "1", SpecJSON: json.RawMessage(`{}`)}
	store := &unpinnedStore{endpoints: []*models.EndPoint{unpinned(1, &model, nil), unpinned(2, &model, nil)}, taken: map[int64]bool{1: true}}

	results, err := newPinService(t, &catalogRepo{defaults: map[uuid.UUID]*models.Blueprint{model: bp}}, store).Pin(testutil.TestContext(), pinOwner, false)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, store.endpoints[1].EUI.String(), results[0].EpEUI)
}

// TC-CM-108: 501 endpoints are read in two pages of PinBatchSize and all processed.
func TestPin_ProcessesEveryPage(t *testing.T) {
	model := uuid.New()
	bp := &models.Blueprint{ID: uuid.New(), DeviceModelID: model, Version: "1", SpecJSON: json.RawMessage(`{}`)}
	store := &unpinnedStore{}
	for id := int64(1); id <= PinBatchSize+1; id++ {
		store.endpoints = append(store.endpoints, unpinned(id, &model, nil))
	}

	results, err := newPinService(t, &catalogRepo{defaults: map[uuid.UUID]*models.Blueprint{model: bp}}, store).Pin(testutil.TestContext(), pinOwner, false)

	require.NoError(t, err)
	assert.Len(t, results, PinBatchSize+1)
	assert.Len(t, store.pinned, PinBatchSize+1)
	assert.Equal(t, 2, store.pages, "a short page ends the walk")
}

// ResolveIgnoringMode resolves without the switch and passes read failures up.
func TestResolveIgnoringMode_IgnoresTheSwitchAndPassesReadFailuresUp(t *testing.T) {
	model := uuid.New()
	bp := &models.Blueprint{ID: uuid.New(), DeviceModelID: model}
	svc := newResolver(t, logger.NewNop(), &catalogRepo{defaults: map[uuid.UUID]*models.Blueprint{model: bp}}, fixedMode{on: true})

	got, err := svc.ResolveIgnoringMode(testutil.TestContext(), pinOwner, unpinned(1, &model, nil))
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, bp.ID, got.ID)

	none, err := svc.ResolveIgnoringMode(testutil.TestContext(), pinOwner, unpinned(2, nil, nil))
	require.NoError(t, err)
	assert.Nil(t, none, "nothing found is not an error")

	failing := newResolver(t, logger.NewNop(), &catalogRepo{defaultErr: errCatalogRead}, fixedMode{})
	_, err = failing.ResolveIgnoringMode(testutil.TestContext(), pinOwner, unpinned(3, &model, nil))
	require.ErrorIs(t, err, errCatalogRead)
}

// Pinning takes exactly what decoding takes with the switch off (classifier rule: pin the current source).
func TestPin_AgreesWithDecodingWithTheSwitchOff(t *testing.T) {
	withDefault, withoutDefault := uuid.New(), uuid.New()
	known := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 0x55}
	unknown := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0, 0, 0x66}
	byModel := &models.Blueprint{ID: uuid.New(), DeviceModelID: withDefault, Version: "1", SpecJSON: json.RawMessage(`{}`)}
	byType := &models.Blueprint{ID: uuid.New(), DeviceModelID: uuid.New(), Version: "2", TypeEUI: known, SpecJSON: json.RawMessage(`{}`)}
	repo := &catalogRepo{
		defaults:  map[uuid.UUID]*models.Blueprint{withDefault: byModel},
		byTypeEUI: map[string]*models.Blueprint{string(known): byType},
	}
	store := &unpinnedStore{endpoints: []*models.EndPoint{
		unpinned(1, &withDefault, unknown),
		unpinned(2, &withoutDefault, known),
		unpinned(3, nil, known),
		unpinned(4, nil, unknown),
		unpinned(5, nil, nil),
	}}
	decoding := newResolver(t, logger.NewNop(), repo, fixedMode{})

	results, err := newPinService(t, repo, store).Pin(testutil.TestContext(), pinOwner, true)

	require.NoError(t, err)
	require.Len(t, results, len(store.endpoints))
	for i, ep := range store.endpoints {
		decoded, err := decoding.ResolveBlueprintForEndpoint(testutil.TestContext(), pinOwner, ep, nil)
		require.NoError(t, err)
		if decoded == nil {
			assert.Equal(t, PinOutcomeNothingToPin, results[i].Outcome, "endpoint %d", ep.ID)
			continue
		}
		assert.Equal(t, PinOutcomePinned, results[i].Outcome, "endpoint %d", ep.ID)
		assert.Equal(t, decoded.ID.String(), results[i].BlueprintID, "endpoint %d", ep.ID)
	}
}
