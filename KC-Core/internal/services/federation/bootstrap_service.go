// Package federation implements CE↔ECE cooperative roaming services.
package federation

import (
	"context"
	"errors"
	"fmt"
	"time"

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	pkgfederation "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/google/uuid"
)

// CEInstallationStore reads the singleton CE installation record and
// completes its onboarding once.
type CEInstallationStore interface {
	Get(ctx context.Context) (*models.CEInstallation, error)
	CompleteOnboarding(ctx context.Context, companyName string, at time.Time) (uuid.UUID, error)
}

// CEBootstrapService handles CE installation status and onboarding completion.
// It is intended to run in CE mode; in ECE mode these RPCs are no-ops that return an
// appropriate error.
type CEBootstrapService struct {
	installationRepo CEInstallationStore
	logger           logger.Logger
	clock            clock.Clock
	edition          string

	relayController RelayController
	relayGate       RelayGate
	connectedFn     func() bool
}

// NewCEBootstrapService creates a new CEBootstrapService; the installation
// store, logger and clock are required.
func NewCEBootstrapService(
	installationRepo CEInstallationStore,
	log logger.Logger,
	clk clock.Clock,
	edition string,
) (*CEBootstrapService, error) {
	if installationRepo == nil || log == nil || clk == nil {
		return nil, ErrBootstrapDependencyMissing
	}
	return &CEBootstrapService{
		installationRepo: installationRepo,
		logger:           log,
		clock:            clk,
		edition:          edition,
	}, nil
}

// WithRelayController injects the relay controller for post-onboarding activation.
func (s *CEBootstrapService) WithRelayController(rc RelayController) *CEBootstrapService {
	s.relayController = rc
	return s
}

// WithRelayGate injects the relay gate for enabling routing after onboarding.
func (s *CEBootstrapService) WithRelayGate(rg RelayGate) *CEBootstrapService {
	s.relayGate = rg
	return s
}

// WithConnectedFn injects a function that reports relay stream connectivity for GetCEStatus.
func (s *CEBootstrapService) WithConnectedFn(fn func() bool) *CEBootstrapService {
	s.connectedFn = fn
	return s
}

// Status returns the current CE installation state.
func (s *CEBootstrapService) Status(ctx context.Context) (pkgfederation.CEStatus, error) {
	if s.edition != pkgconfig.EditionCommunity {
		return pkgfederation.CEStatus{}, pkgfederation.ErrNotCommunityEdition
	}

	inst, err := s.installationRepo.Get(ctx)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		s.logger.ErrorContext(ctx, LogInstallationFetchFailed, logger.FieldError, err)
		return pkgfederation.CEStatus{}, fmt.Errorf("%w: %w", pkgfederation.ErrInstallationRead, err)
	}

	if inst == nil || inst.OnboardingCompletedAt == nil {
		return pkgfederation.CEStatus{OnboardingRequired: true}, nil
	}

	federationConnected := false
	if s.connectedFn != nil {
		federationConnected = s.connectedFn()
	}

	return pkgfederation.CEStatus{
		OnboardingRequired:  false,
		CEID:                inst.CEID.String(),
		CompanyName:         inst.CompanyName,
		FederationConnected: federationConnected,
	}, nil
}

// CompleteOnboarding records the company name, creating the installation and
// its CE ID when there is none. The call is unauthenticated, so the store
// completes onboarding at most once, also for concurrent calls.
func (s *CEBootstrapService) CompleteOnboarding(ctx context.Context, companyName string) (pkgfederation.OnboardingResult, error) {
	if s.edition != pkgconfig.EditionCommunity {
		return pkgfederation.OnboardingResult{}, pkgfederation.ErrNotCommunityEdition
	}
	if companyName == "" {
		return pkgfederation.OnboardingResult{}, pkgfederation.ErrCompanyNameRequired
	}

	ceID, err := s.installationRepo.CompleteOnboarding(ctx, companyName, s.clock.Now())
	if errors.Is(err, storage.ErrInstallationOnboarded) {
		return pkgfederation.OnboardingResult{}, pkgfederation.ErrOnboardingAlreadyCompleted
	}
	if err != nil {
		return pkgfederation.OnboardingResult{}, fmt.Errorf("%w: %w", pkgfederation.ErrInstallationWrite, err)
	}

	s.logger.InfoContext(ctx, LogOnboardingCompleted, logger.FieldCeID, ceID, logger.FieldCompany, companyName)
	s.activateRelay(ctx)
	return pkgfederation.OnboardingResult{CEID: ceID.String(), CompanyName: companyName}, nil
}

// activateRelay enables relay routing and starts the relay client after onboarding.
func (s *CEBootstrapService) activateRelay(ctx context.Context) {
	if s.relayGate != nil {
		s.relayGate.SetRelayEnabled(true)
	}
	if s.relayController != nil {
		if err := s.relayController.EnsureStarted(ctx); err != nil {
			if errors.Is(err, pkgfederation.ErrRelayOnboardingIncomplete) {
				s.logger.InfoContext(ctx, LogRelayStartDeferredOnboarding)
			} else {
				s.logger.WarnContext(ctx, LogRelayStartFailed, logger.FieldError, err)
			}
		}
	}
}

// IsCEOnboardingRequired returns true if the CE has not completed onboarding.
// Used by the disposition resolver gate to block relay until onboarding is done.
func (s *CEBootstrapService) IsCEOnboardingRequired(ctx context.Context) (bool, error) {
	if s.edition != pkgconfig.EditionCommunity {
		return false, nil
	}
	inst, err := s.installationRepo.Get(ctx)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return true, fmt.Errorf("%w: %w", ErrInstallationCheck, err)
	}
	return inst == nil || inst.OnboardingCompletedAt == nil, nil
}
