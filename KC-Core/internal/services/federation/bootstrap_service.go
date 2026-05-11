// Package federation implements CE↔ECE cooperative roaming services.
package federation

import (
	"context"
	"fmt"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CEBootstrapService handles CE installation status and onboarding completion.
// It is intended to run in CE mode; in ECE mode these RPCs are no-ops that return an
// appropriate error.
type CEBootstrapService struct {
	installationRepo interfaces.CEInstallationRepository
	logger           logger.Logger
	edition          string

	relayController RelayController
	relayGate       RelayGate
	connectedFn     func() bool
}

// NewCEBootstrapService creates a new CEBootstrapService.
func NewCEBootstrapService(
	installationRepo interfaces.CEInstallationRepository,
	log logger.Logger,
	edition string,
) *CEBootstrapService {
	return &CEBootstrapService{
		installationRepo: installationRepo,
		logger:           log,
		edition:          edition,
	}
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

// GetCEStatus returns the current CE installation status.
func (s *CEBootstrapService) GetCEStatus(ctx context.Context, _ *pb.GetCEStatusRequest) (*pb.GetCEStatusResponse, error) {
	if s.edition != "ce" {
		return nil, status.Error(codes.Unimplemented, "GetCEStatus is only available in CE mode")
	}

	inst, err := s.installationRepo.Get(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to fetch CE installation", "error", err)
		return nil, status.Error(codes.Internal, "failed to read CE installation")
	}

	if inst == nil || inst.OnboardingCompletedAt == nil {
		return &pb.GetCEStatusResponse{
			OnboardingRequired:  true,
			FederationConnected: false,
		}, nil
	}

	federationConnected := false
	if s.connectedFn != nil {
		federationConnected = s.connectedFn()
	}

	return &pb.GetCEStatusResponse{
		OnboardingRequired:  false,
		CeId:                inst.CEID.String(),
		CompanyName:         inst.CompanyName,
		FederationConnected: federationConnected,
	}, nil
}

// CompleteCEOnboarding stores the company name and generates a CE ID if none exists.
func (s *CEBootstrapService) CompleteCEOnboarding(ctx context.Context, req *pb.CompleteCEOnboardingRequest) (*pb.CompleteCEOnboardingResponse, error) {
	if s.edition != "ce" {
		return nil, status.Error(codes.Unimplemented, "CompleteCEOnboarding is only available in CE mode")
	}
	if req.CompanyName == "" {
		return nil, status.Error(codes.InvalidArgument, "company_name is required")
	}

	inst, err := s.installationRepo.Get(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read CE installation: %v", err)
	}

	now := time.Now()
	if inst == nil {
		ceID := uuid.New()
		newInst := &interfaces.CEInstallation{
			CEID:                  ceID,
			CompanyName:           req.CompanyName,
			OnboardingCompletedAt: &now,
		}
		if createErr := s.installationRepo.Create(ctx, newInst); createErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to create CE installation: %v", createErr)
		}
		s.logger.InfoContext(ctx, "CE onboarding completed", "ce_id", ceID, "company", req.CompanyName)
		s.activateRelay(ctx)
		return &pb.CompleteCEOnboardingResponse{
			CeId:        ceID.String(),
			CompanyName: req.CompanyName,
		}, nil
	}

	// Already onboarded: update company name if changed
	if inst.CompanyName != req.CompanyName || inst.OnboardingCompletedAt == nil {
		updates := map[string]interface{}{
			"company_name":            req.CompanyName,
			"onboarding_completed_at": now,
		}
		if updateErr := s.installationRepo.Update(ctx, updates); updateErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to update CE installation: %v", updateErr)
		}
	}

	s.activateRelay(ctx)
	return &pb.CompleteCEOnboardingResponse{
		CeId:        inst.CEID.String(),
		CompanyName: req.CompanyName,
	}, nil
}

// activateRelay enables relay routing and starts the relay client after onboarding.
func (s *CEBootstrapService) activateRelay(ctx context.Context) {
	if s.relayGate != nil {
		s.relayGate.SetRelayEnabled(true)
	}
	if s.relayController != nil {
		if err := s.relayController.EnsureStarted(ctx); err != nil {
			s.logger.WarnContext(ctx, "Failed to start federation relay after onboarding", "error", err)
		}
	}
}

// IsCEOnboardingRequired returns true if the CE has not completed onboarding.
// Used by the disposition resolver gate to block relay until onboarding is done.
func (s *CEBootstrapService) IsCEOnboardingRequired(ctx context.Context) (bool, error) {
	if s.edition != "ce" {
		return false, nil
	}
	inst, err := s.installationRepo.Get(ctx)
	if err != nil {
		return true, fmt.Errorf("ce installation check: %w", err)
	}
	return inst == nil || inst.OnboardingCompletedAt == nil, nil
}
