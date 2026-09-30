package grpcservices

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Persistence failures the store stub returns per operation.
var (
	errInsertFailed = errors.New("insert failed")
	errUpdateFailed = errors.New("update failed")
	errDeleteFailed = errors.New("delete failed")
)

// indexRecorder records every index synchronization call.
type indexRecorder struct {
	added   []models.EUI
	removed []models.EUI
}

func (r *indexRecorder) Add(_ context.Context, eui models.EUI)    { r.added = append(r.added, eui) }
func (r *indexRecorder) Remove(_ context.Context, eui models.EUI) { r.removed = append(r.removed, eui) }

// endpointStoreStub answers each persistence call with a canned result.
type endpointStoreStub struct {
	createErr error
	updateErr error
	deleteErr error
	removedID int64
	persisted *models.EndPoint
}

func (s *endpointStoreStub) Create(_ context.Context, ep *models.EndPoint) error {
	if s.createErr != nil {
		return s.createErr
	}
	if s.persisted != nil {
		*ep = *s.persisted
	}
	return nil
}

func (s *endpointStoreStub) GetByEUI(context.Context, int64, []byte) (*models.EndPoint, error) {
	return nil, nil
}

func (s *endpointStoreStub) Update(_ context.Context, _ *models.EndPoint) error {
	return s.updateErr
}

func (s *endpointStoreStub) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return ep, nil
}

func (s *endpointStoreStub) CheckEUIUnique(context.Context, []byte) error { return nil }

func (s *endpointStoreStub) CreateWithStatus(ctx context.Context, ep *models.EndPoint, status string) error {
	if err := s.Create(ctx, ep); err != nil {
		return err
	}
	ep.EpStatus = status
	return nil
}

func (s *endpointStoreStub) DeleteByTenant(context.Context, int64, []byte) (int64, error) {
	return s.removedID, s.deleteErr
}

func (s *endpointStoreStub) ListByTenantPaginated(context.Context, int64, int, int) ([]*models.EndPoint, error) {
	return nil, nil
}

func (s *endpointStoreStub) ListByModelWithSnapshot(context.Context, int64, uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func testEUI(v uint64) models.EUI {
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], v)
	return eui
}

// attachedStatus is the status a pre-attached endpoint is stored with.
const attachedStatus = "attached"

func TestEndpointService_CreateSyncsIndex(t *testing.T) {
	idx := &indexRecorder{}
	persisted := &models.EndPoint{EUI: testEUI(0x11)}
	svc := NewEndpointService(&endpointStoreStub{persisted: persisted}, idx)

	_, err := svc.Create(testutil.TestContext(), &models.EndPoint{EUI: testEUI(0x11)})
	require.NoError(t, err)
	assert.Equal(t, []models.EUI{testEUI(0x11)}, idx.added, "the persisted EUI is added after a successful create")
	assert.Empty(t, idx.removed)
}

func TestEndpointService_FailedCreateLeavesIndexUntouched(t *testing.T) {
	idx := &indexRecorder{}
	svc := NewEndpointService(&endpointStoreStub{createErr: errInsertFailed}, idx)

	_, err := svc.Create(testutil.TestContext(), &models.EndPoint{EUI: testEUI(0x11)})
	require.Error(t, err)
	assert.Empty(t, idx.added)
	assert.Empty(t, idx.removed)
}

// A pre-attached endpoint is indexed like any other once it is stored, and
// not at all when its creation fails.
func TestEndpointService_CreateWithStatusSyncsIndex(t *testing.T) {
	idx := &indexRecorder{}
	svc := NewEndpointService(&endpointStoreStub{persisted: &models.EndPoint{EUI: testEUI(0x11)}}, idx)

	created, err := svc.CreateWithStatus(testutil.TestContext(), &models.EndPoint{EUI: testEUI(0x11)}, attachedStatus)
	require.NoError(t, err)
	assert.Equal(t, attachedStatus, created.EpStatus)
	assert.Equal(t, []models.EUI{testEUI(0x11)}, idx.added)

	failing := &indexRecorder{}
	_, err = NewEndpointService(&endpointStoreStub{createErr: errInsertFailed}, failing).
		CreateWithStatus(testutil.TestContext(), &models.EndPoint{EUI: testEUI(0x11)}, attachedStatus)
	require.Error(t, err)
	assert.Empty(t, failing.added)
}

func TestEndpointService_OrdinaryUpdateDoesNotTouchIndex(t *testing.T) {
	idx := &indexRecorder{}
	svc := NewEndpointService(&endpointStoreStub{}, idx)

	_, err := svc.Update(testutil.TestContext(), &models.EndPoint{EUI: testEUI(0x22)})
	require.NoError(t, err)
	assert.Empty(t, idx.added)
	assert.Empty(t, idx.removed)
}

func TestEndpointService_EUIChangingUpdateSwapsIndexEntries(t *testing.T) {
	idx := &indexRecorder{}
	svc := NewEndpointService(&endpointStoreStub{}, idx)
	oldEUI := testEUI(0x22)

	_, err := svc.UpdateWithEUI(testutil.TestContext(), 1, oldEUI[:], &models.EndPoint{EUI: testEUI(0x33)})
	require.NoError(t, err)
	assert.Equal(t, []models.EUI{oldEUI}, idx.removed, "the old EUI leaves the index")
	assert.Equal(t, []models.EUI{testEUI(0x33)}, idx.added, "the new EUI enters the index")
}

func TestEndpointService_SameEUIUpdateOnlyReAdds(t *testing.T) {
	idx := &indexRecorder{}
	svc := NewEndpointService(&endpointStoreStub{}, idx)
	eui := testEUI(0x44)

	_, err := svc.UpdateWithEUI(testutil.TestContext(), 1, eui[:], &models.EndPoint{EUI: eui})
	require.NoError(t, err)
	assert.Empty(t, idx.removed, "an unchanged EUI is never removed")
	assert.Equal(t, []models.EUI{eui}, idx.added)
}

func TestEndpointService_FailedUpdateLeavesIndexUntouched(t *testing.T) {
	idx := &indexRecorder{}
	svc := NewEndpointService(&endpointStoreStub{updateErr: errUpdateFailed}, idx)
	oldEUI := testEUI(0x22)

	_, err := svc.UpdateWithEUI(testutil.TestContext(), 1, oldEUI[:], &models.EndPoint{EUI: testEUI(0x33)})
	require.Error(t, err)
	assert.Empty(t, idx.added)
	assert.Empty(t, idx.removed)
}

const deletedTestEndpointID int64 = 77

func TestEndpointService_DeleteSyncsIndex(t *testing.T) {
	idx := &indexRecorder{}
	svc := NewEndpointService(&endpointStoreStub{removedID: deletedTestEndpointID}, idx)
	eui := testEUI(0x55)

	removedID, err := svc.Delete(testutil.TestContext(), eui[:], 1)
	require.NoError(t, err)
	assert.Equal(t, deletedTestEndpointID, removedID, "the delete reports the endpoint it removed")
	assert.Equal(t, []models.EUI{eui}, idx.removed)
}

func TestEndpointService_FailedDeleteLeavesIndexUntouched(t *testing.T) {
	idx := &indexRecorder{}
	svc := NewEndpointService(&endpointStoreStub{deleteErr: errDeleteFailed}, idx)
	eui := testEUI(0x55)

	_, err := svc.Delete(testutil.TestContext(), eui[:], 1)
	require.Error(t, err)
	assert.Empty(t, idx.removed)
}

func TestEndpointService_NilIndexIsSafe(t *testing.T) {
	svc := NewEndpointService(&endpointStoreStub{}, nil)
	eui := testEUI(0x66)

	_, err := svc.Create(testutil.TestContext(), &models.EndPoint{EUI: eui})
	require.NoError(t, err)
	_, err = svc.Delete(testutil.TestContext(), eui[:], 1)
	require.NoError(t, err)
}
