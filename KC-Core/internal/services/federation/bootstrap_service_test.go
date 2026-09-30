package federation

import (
	"context"
	"errors"
	"testing"

	"time"

	pkgfederation "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOnboardingTime is the instant the injected clock reports.
var testOnboardingTime = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// fakeInstallationRepo is an in-memory installation store that completes
// onboarding once, like the PostgreSQL store.
type fakeInstallationRepo struct {
	inst        *models.CEInstallation
	getErr      error
	completeErr error
	completions int
}

func (f *fakeInstallationRepo) Get(context.Context) (*models.CEInstallation, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.inst == nil {
		return nil, storage.ErrNotFound
	}
	return f.inst, nil
}

func (f *fakeInstallationRepo) CompleteOnboarding(_ context.Context, companyName string, at time.Time) (uuid.UUID, error) {
	f.completions++
	if f.completeErr != nil {
		return uuid.Nil, f.completeErr
	}
	if f.inst != nil && f.inst.OnboardingCompletedAt != nil {
		return uuid.Nil, storage.ErrInstallationOnboarded
	}
	if f.inst == nil {
		f.inst = &models.CEInstallation{CEID: uuid.New()}
	}
	f.inst.CompanyName = companyName
	f.inst.OnboardingCompletedAt = &at
	return f.inst.CEID, nil
}

// fakeRelayController records EnsureStarted calls.
type fakeRelayController struct {
	started int
	err     error
}

func (f *fakeRelayController) EnsureStarted(context.Context) error { f.started++; return f.err }
func (f *fakeRelayController) IsConnected() bool                   { return false }

// fakeRelayGate records SetRelayEnabled calls.
type fakeRelayGate struct{ enabled []bool }

func (f *fakeRelayGate) SetRelayEnabled(v bool) { f.enabled = append(f.enabled, v) }

const testEditionCE = "ce"

// Injected failures the bootstrap and disposition tests assert on.
var (
	errDBDown       = errors.New("db down")
	errUpdateFailed = errors.New("update failed")
	errRelayDown    = errors.New("relay down")
)

func newBootstrapService(t *testing.T, repo CEInstallationStore, edition string) *CEBootstrapService {
	t.Helper()
	svc, err := NewCEBootstrapService(repo, logger.NewNop(), testutil.NewFakeClock(testOnboardingTime), edition)
	require.NoError(t, err)
	return svc
}

func TestNewCEBootstrapService_RefusesMissingDependencies(t *testing.T) {
	clk := testutil.NewFakeClock(testOnboardingTime)
	_, err := NewCEBootstrapService(nil, logger.NewNop(), clk, testEditionCE)
	assert.ErrorIs(t, err, ErrBootstrapDependencyMissing)
	_, err = NewCEBootstrapService(&fakeInstallationRepo{}, nil, clk, testEditionCE)
	assert.ErrorIs(t, err, ErrBootstrapDependencyMissing)
	_, err = NewCEBootstrapService(&fakeInstallationRepo{}, logger.NewNop(), nil, testEditionCE)
	assert.ErrorIs(t, err, ErrBootstrapDependencyMissing)
}

func TestStatusNonCE(t *testing.T) {
	svc := newBootstrapService(t, &fakeInstallationRepo{}, "ece")
	_, err := svc.Status(testutil.TestContext())
	assert.ErrorIs(t, err, pkgfederation.ErrNotCommunityEdition)
}

func TestStatusRepoFailure(t *testing.T) {
	cause := errDBDown
	svc := newBootstrapService(t, &fakeInstallationRepo{getErr: cause}, testEditionCE)
	_, err := svc.Status(testutil.TestContext())
	assert.ErrorIs(t, err, pkgfederation.ErrInstallationRead)
	assert.ErrorIs(t, err, cause)
}

func TestStatusOnboardingRequired(t *testing.T) {
	svc := newBootstrapService(t, &fakeInstallationRepo{}, testEditionCE)
	st, err := svc.Status(testutil.TestContext())
	require.NoError(t, err)
	assert.True(t, st.OnboardingRequired)
	assert.Empty(t, st.CEID)
}

func TestStatusOnboarded(t *testing.T) {
	now := time.Now()
	ceID := uuid.New()
	repo := &fakeInstallationRepo{inst: &models.CEInstallation{
		CEID: ceID, CompanyName: "Acme", OnboardingCompletedAt: &now,
	}}
	svc := newBootstrapService(t, repo, testEditionCE)
	svc.WithConnectedFn(func() bool { return true })
	st, err := svc.Status(testutil.TestContext())
	require.NoError(t, err)
	assert.False(t, st.OnboardingRequired)
	assert.Equal(t, ceID.String(), st.CEID)
	assert.Equal(t, "Acme", st.CompanyName)
	assert.True(t, st.FederationConnected)
}

func TestCompleteOnboardingNonCE(t *testing.T) {
	svc := newBootstrapService(t, &fakeInstallationRepo{}, "ece")
	_, err := svc.CompleteOnboarding(testutil.TestContext(), "Acme")
	assert.ErrorIs(t, err, pkgfederation.ErrNotCommunityEdition)
}

func TestCompleteOnboardingEmptyCompanyName(t *testing.T) {
	repo := &fakeInstallationRepo{}
	svc := newBootstrapService(t, repo, testEditionCE)
	_, err := svc.CompleteOnboarding(testutil.TestContext(), "")
	assert.ErrorIs(t, err, pkgfederation.ErrCompanyNameRequired)
	assert.Zero(t, repo.completions)
}

func TestCompleteOnboardingFirstInstall(t *testing.T) {
	repo := &fakeInstallationRepo{}
	rc := &fakeRelayController{}
	rg := &fakeRelayGate{}
	svc := newBootstrapService(t, repo, testEditionCE)
	svc.WithRelayController(rc)
	svc.WithRelayGate(rg)

	result, err := svc.CompleteOnboarding(testutil.TestContext(), "Acme")
	require.NoError(t, err)
	require.NotNil(t, repo.inst)
	assert.Equal(t, "Acme", repo.inst.CompanyName)
	require.NotNil(t, repo.inst.OnboardingCompletedAt)
	assert.True(t, repo.inst.OnboardingCompletedAt.Equal(testOnboardingTime), "completed at the injected clock")
	assert.Equal(t, repo.inst.CEID.String(), result.CEID)
	assert.Equal(t, []bool{true}, rg.enabled)
	assert.Equal(t, 1, rc.started)
}

// CompleteCEOnboarding is unauthenticated: once an installation is onboarded,
// no later caller may rename it or re-run the bootstrap.
func TestCompleteOnboardingRefusedOnceCompleted(t *testing.T) {
	now := time.Now()
	for _, name := range []string{"Old", "Someone Else"} {
		repo := &fakeInstallationRepo{inst: &models.CEInstallation{
			CEID: uuid.New(), CompanyName: "Old", OnboardingCompletedAt: &now,
		}}
		rc := &fakeRelayController{}
		svc := newBootstrapService(t, repo, testEditionCE)
		svc.WithRelayController(rc)

		_, err := svc.CompleteOnboarding(testutil.TestContext(), name)
		require.ErrorIs(t, err, pkgfederation.ErrOnboardingAlreadyCompleted, "company %q", name)
		assert.Equal(t, "Old", repo.inst.CompanyName, "a completed installation is never rewritten")
		assert.Zero(t, rc.started)
	}
}

// An installation record that exists without completed onboarding is
// completed by the first onboarding call.
func TestCompleteOnboardingCompletesAnUnfinishedInstallation(t *testing.T) {
	ceID := uuid.New()
	repo := &fakeInstallationRepo{inst: &models.CEInstallation{CEID: ceID, CompanyName: "Draft"}}
	rc := &fakeRelayController{}
	svc := newBootstrapService(t, repo, testEditionCE)
	svc.WithRelayController(rc)

	result, err := svc.CompleteOnboarding(testutil.TestContext(), "Acme")
	require.NoError(t, err)
	assert.Equal(t, "Acme", repo.inst.CompanyName)
	require.NotNil(t, repo.inst.OnboardingCompletedAt)
	assert.True(t, repo.inst.OnboardingCompletedAt.Equal(testOnboardingTime))
	assert.Equal(t, ceID.String(), result.CEID)
	assert.Equal(t, 1, rc.started)
}

func TestCompleteOnboardingStoreFailure(t *testing.T) {
	rc := &fakeRelayController{}
	repo := &fakeInstallationRepo{
		inst:        &models.CEInstallation{CEID: uuid.New(), CompanyName: "Old"},
		completeErr: errUpdateFailed,
	}
	svc := newBootstrapService(t, repo, testEditionCE)
	svc.WithRelayController(rc)
	_, err := svc.CompleteOnboarding(testutil.TestContext(), "New")
	assert.ErrorIs(t, err, pkgfederation.ErrInstallationWrite)
	assert.ErrorIs(t, err, errUpdateFailed)
	assert.Zero(t, rc.started, "relay must not start after a failed write")
}

func TestCompleteOnboardingRelayStartFailureIsNotFatal(t *testing.T) {
	rc := &fakeRelayController{err: errRelayDown}
	svc := newBootstrapService(t, &fakeInstallationRepo{}, testEditionCE)
	svc.WithRelayController(rc)
	_, err := svc.CompleteOnboarding(testutil.TestContext(), "Acme")
	assert.NoError(t, err, "onboarding succeeds even when relay startup fails")
	assert.Equal(t, 1, rc.started)
}

func TestIsCEOnboardingRequired(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name    string
		repo    *fakeInstallationRepo
		edition string
		want    bool
		wantErr bool
	}{
		{name: "non-ce is never required", repo: &fakeInstallationRepo{}, edition: "ece", want: false},
		{name: "no record", repo: &fakeInstallationRepo{}, edition: testEditionCE, want: true},
		{name: "incomplete record", repo: &fakeInstallationRepo{inst: &models.CEInstallation{CEID: uuid.New()}}, edition: testEditionCE, want: true},
		{name: "onboarded", repo: &fakeInstallationRepo{inst: &models.CEInstallation{CEID: uuid.New(), OnboardingCompletedAt: &now}}, edition: testEditionCE, want: false},
		{name: "repo failure fails closed", repo: &fakeInstallationRepo{getErr: errDBDown}, edition: testEditionCE, want: true, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newBootstrapService(t, tc.repo, tc.edition)
			got, err := svc.IsCEOnboardingRequired(testutil.TestContext())
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
