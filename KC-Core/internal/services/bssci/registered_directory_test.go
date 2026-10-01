package bssciservices

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// expiryRecordingBSRepo records the certificate expiry backfills it receives.
type expiryRecordingBSRepo struct {
	certIdentityBSRepo
	tenantID, id int64
	expiresAt    time.Time
}

func (r *expiryRecordingBSRepo) UpdateTLSCertExpiryIfBlank(_ context.Context, tenantID, id int64, expiresAt time.Time) (bool, error) {
	r.tenantID, r.id, r.expiresAt = tenantID, id, expiresAt
	return true, nil
}

// The directory hands the binder the stored expiry and backfills a new one
// for the station's own tenant and row.
func TestRegisteredBaseStationDirectory_CertExpiry(t *testing.T) {
	stored := time.Date(2034, time.May, 6, 7, 8, 9, 0, time.UTC)
	repo := &expiryRecordingBSRepo{certIdentityBSRepo: certIdentityBSRepo{
		baseStation: &models.BaseStation{ID: 9, TenantID: 3, TLSCertExpiresAt: &stored},
	}}
	directory := NewRegisteredBaseStationDirectory(repo)
	ctx := testutil.TestContext()

	registered, err := directory.GetGlobal(ctx, 0x0102030405060708)
	require.NoError(t, err)
	require.NotNil(t, registered.TLSCertExpiresAt)
	assert.True(t, stored.Equal(*registered.TLSCertExpiresAt))

	next := stored.AddDate(1, 0, 0)
	recorded, err := directory.BackfillCertExpiryIfBlank(ctx, registered.TenantID, registered.ID, next)
	require.NoError(t, err)
	assert.True(t, recorded)
	assert.Equal(t, int64(3), repo.tenantID)
	assert.Equal(t, int64(9), repo.id)
	assert.True(t, next.Equal(repo.expiresAt))
}
