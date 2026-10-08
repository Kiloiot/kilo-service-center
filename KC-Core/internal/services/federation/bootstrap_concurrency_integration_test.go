package federation

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgfederation "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
)

// Company names two concurrent onboarding calls race to record.
var concurrentOnboardingCompanies = []string{"First Company", "Second Company"}

// readBarrierStore holds every installation read until each racing caller has
// read, so all of them see the installation before any of them writes.
type readBarrierStore struct {
	CEInstallationStore
	reads *sync.WaitGroup
}

func (s readBarrierStore) Get(ctx context.Context) (*models.CEInstallation, error) {
	inst, err := s.CEInstallationStore.Get(ctx)
	s.reads.Done()
	s.reads.Wait()
	return inst, err
}

// TestCompleteOnboarding_ConcurrentCallsCompleteOnce races two unauthenticated
// onboarding calls on an installation that has not completed onboarding:
// exactly one may record its company name.
func TestCompleteOnboarding_ConcurrentCallsCompleteOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	store, cleanup := teststore.Setup(t)
	t.Cleanup(cleanup)
	ctx := testutil.TestContext()
	_, err := store.Sqlx().ExecContext(ctx,
		`INSERT INTO ce_installation (id, ce_id, company_name) VALUES (1, gen_random_uuid(), 'Draft')`)
	require.NoError(t, err)

	reads := &sync.WaitGroup{}
	reads.Add(len(concurrentOnboardingCompanies))
	svc := newBootstrapService(t, readBarrierStore{
		CEInstallationStore: postgres.NewRepositories(store).CEInstallations,
		reads:               reads,
	}, testEditionCE)

	errs := make([]error, len(concurrentOnboardingCompanies))
	var wg sync.WaitGroup
	for i, company := range concurrentOnboardingCompanies {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = svc.CompleteOnboarding(ctx, company)
		}()
	}
	wg.Wait()

	winners := 0
	winner := ""
	for i, err := range errs {
		if err == nil {
			winners++
			winner = concurrentOnboardingCompanies[i]
			continue
		}
		assert.ErrorIs(t, err, pkgfederation.ErrOnboardingAlreadyCompleted)
	}
	require.Equal(t, 1, winners, "exactly one concurrent onboarding call may succeed: %v", errs)

	inst, err := postgres.NewRepositories(store).CEInstallations.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, winner, inst.CompanyName)
}
