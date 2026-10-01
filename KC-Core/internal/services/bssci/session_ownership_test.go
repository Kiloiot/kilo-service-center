package bssciservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	repodoubles "github.com/Kiloiot/kilo-service-center/KC-Core/internal/testsupport/repodoubles"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// ownerCapturingSessionRepo records the owning service center every session
// write carries.
type ownerCapturingSessionRepo struct {
	*repodoubles.BaseStationSessionRepo
	createdFor   []models.EUI
	activatedFor []models.EUI
}

func (r *ownerCapturingSessionRepo) CreateSession(ctx context.Context, req *models.BaseStationSessionCreateRequest) (*models.BaseStationSession, error) {
	r.createdFor = append(r.createdFor, req.ScEui)
	return r.BaseStationSessionRepo.CreateSession(ctx, req)
}

func (r *ownerCapturingSessionRepo) ActivateSessionIfResumable(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) (bool, error) {
	if req.ScEui != nil {
		r.activatedFor = append(r.activatedFor, *req.ScEui)
	}
	return r.BaseStationSessionRepo.ActivateSessionIfResumable(ctx, tenantID, sessionID, req)
}

// Session rows record the service center that created or resumed them, the
// ownership the startup reconciliation is scoped by.
func TestPersistSessionRecordsOwningServiceCenter(t *testing.T) {
	const tenantID = int64(950)
	owner := models.EUI(mioty.EUI64(bssci.TestScEui01).ToBytes())
	repo := &ownerCapturingSessionRepo{BaseStationSessionRepo: repodoubles.NewBaseStationSessionRepo()}
	svc := NewSessionService(repo, newMockPendingOpsStore(),
		&repodoubles.SystemEventStore{}, tenantID, bssci.TestScEui01, logger.NewNop())

	bsUUID := [16]byte{0x51, 0x52}
	scUUID := [16]byte{0x61, 0x62}
	seedResumableSession(repo.BaseStationSessionRepo, 96, tenantID, bsUUID, scUUID, 4, -2)
	resumed := newResumeClaimConnection("connection-resume", tenantID, bsUUID, scUUID)
	require.NoError(t, svc.PersistSession(testutil.TestContext(), resumed, nil, true, nil))
	assert.Equal(t, []models.EUI{owner}, repo.activatedFor, "a resumed session is claimed for this service center")

	fresh := newResumeClaimConnection("connection-fresh", tenantID, [16]byte{0x71}, [16]byte{0x81})
	fresh.IsResumed = false
	require.NoError(t, svc.PersistSession(testutil.TestContext(), fresh, &basestation.BaseStation{ID: 2, TenantID: tenantID}, false, nil))
	assert.Equal(t, []models.EUI{owner}, repo.createdFor, "a new session is created for this service center")
}
