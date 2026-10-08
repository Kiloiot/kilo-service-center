package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// ceOnboardingTime is the completion time the tests record.
var ceOnboardingTime = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

func TestCEInstallationRepository_CompleteOnboardingCreatesTheInstallation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	repo := NewCEInstallationRepository(db)
	ctx := testutil.TestContext()

	ceID, err := repo.CompleteOnboarding(ctx, "Acme", ceOnboardingTime)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, ceID)

	inst, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, ceID, inst.CEID)
	assert.Equal(t, "Acme", inst.CompanyName)
	require.NotNil(t, inst.OnboardingCompletedAt)
	assert.True(t, inst.OnboardingCompletedAt.Equal(ceOnboardingTime))
}

func TestCEInstallationRepository_CompleteOnboardingKeepsTheCEIDAndRunsOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	repo := NewCEInstallationRepository(db)
	ctx := testutil.TestContext()
	existing := uuid.New()
	_, err := db.ExecContext(ctx, `INSERT INTO ce_installation (id, ce_id, company_name) VALUES (1, $1, 'Draft')`, existing)
	require.NoError(t, err)

	ceID, err := repo.CompleteOnboarding(ctx, "Acme", ceOnboardingTime)
	require.NoError(t, err)
	assert.Equal(t, existing, ceID, "an unfinished installation keeps its CE ID")

	_, err = repo.CompleteOnboarding(ctx, "Someone Else", ceOnboardingTime.Add(time.Hour))
	require.ErrorIs(t, err, storage.ErrInstallationOnboarded)
	inst, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Acme", inst.CompanyName, "a completed installation is never rewritten")
	assert.True(t, inst.OnboardingCompletedAt.Equal(ceOnboardingTime))
}
