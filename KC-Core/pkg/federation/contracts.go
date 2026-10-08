// Package federation defines the public contracts for CE/ECE federation services.
// These interfaces are the single canonical definitions used by both
// internal gRPC handlers and external ECE builder hooks.
package federation

import (
	"context"
	"errors"
)

// Bootstrap failures, reported as domain errors so the onboarding service
// carries no transport vocabulary. The gRPC adapter maps them onto catalog
// tokens.
var (
	// ErrNotCommunityEdition reports an onboarding call in an edition that does
	// not own CE onboarding.
	ErrNotCommunityEdition = errors.New("federation: onboarding is only available in community edition")
	// ErrCompanyNameRequired reports onboarding submitted without a company name.
	ErrCompanyNameRequired = errors.New("federation: company name is required")
	// ErrOnboardingAlreadyCompleted reports an onboarding call on an
	// installation that finished onboarding.
	ErrOnboardingAlreadyCompleted = errors.New("federation: onboarding is already completed")
	// ErrInstallationRead reports a failure reading the installation record.
	ErrInstallationRead = errors.New("federation: failed to read the CE installation")
	// ErrInstallationWrite reports a failure creating or updating the record.
	ErrInstallationWrite = errors.New("federation: failed to store the CE installation")
)

// CEStatus is the onboarding and relay state of a community installation.
type CEStatus struct {
	OnboardingRequired  bool
	CEID                string
	CompanyName         string
	FederationConnected bool
}

// OnboardingResult identifies the installation after onboarding completes.
type OnboardingResult struct {
	CEID        string
	CompanyName string
}

// CEBootstrapService is the domain side of CE onboarding: it speaks installation
// records, not protobuf.
type CEBootstrapService interface {
	Status(ctx context.Context) (CEStatus, error)
	CompleteOnboarding(ctx context.Context, companyName string) (OnboardingResult, error)
}

// RelayController manages the lifecycle of the CE→ECE relay stream.
type RelayController interface {
	EnsureStarted(ctx context.Context) error
	IsConnected() bool
}

// RelayGate enables/disables relay routing at runtime.
type RelayGate interface {
	SetRelayEnabled(enabled bool)
}
