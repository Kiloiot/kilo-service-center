package bssciservices

import (
	"context"

	"github.com/kilocenter/KC-Core/pkg/bssci"
)

// NoopRoamingService is a no-operation implementation of RoamingService
// used when roaming is disabled in configuration
type NoopRoamingService struct{}

// NewNoopRoamingService creates a new no-op roaming service
func NewNoopRoamingService() bssci.RoamingService {
	return &NoopRoamingService{}
}

// DetectAndValidateRoaming always returns false (not roaming) when roaming is disabled
func (n *NoopRoamingService) DetectAndValidateRoaming(_ context.Context, _ []byte, servingTenantID int64) (bool, int64, error) {
	// When roaming is disabled, all endpoints are served by their current tenant
	return false, servingTenantID, nil
}

// RecordAttach does nothing when roaming is disabled
func (n *NoopRoamingService) RecordAttach(_ context.Context, _ []byte, _ []byte, _ int64) error {
	// No-op: roaming events are not recorded when disabled
	return nil
}

// RecordDetach does nothing when roaming is disabled
func (n *NoopRoamingService) RecordDetach(_ context.Context, _ []byte, _ []byte, _ int64) error {
	// No-op: roaming events are not recorded when disabled
	return nil
}

// UpdateSessionRoaming does nothing when roaming is disabled
func (n *NoopRoamingService) UpdateSessionRoaming(_ context.Context, _ int64, _ []byte, _ bool, _ int64) error {
	// No-op: session roaming tracking is not needed when disabled
	return nil
}
