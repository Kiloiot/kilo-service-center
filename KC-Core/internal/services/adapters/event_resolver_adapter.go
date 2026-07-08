package adapters

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
)

// EUIResolverAdapter resolves device EUIs to internal IDs for event scoping.
// Best-effort: an unknown device yields a nil ID (never a blocking error), so
// callers fall back to the source_name EUI filter.
type EUIResolverAdapter struct {
	bsRepo interfaces.BaseStationRepository
	epRepo interfaces.EndpointRepository
}

// NewEUIResolver creates a resolver backed by the base station and endpoint repositories.
func NewEUIResolver(bsRepo interfaces.BaseStationRepository, epRepo interfaces.EndpointRepository) *EUIResolverAdapter {
	return &EUIResolverAdapter{bsRepo: bsRepo, epRepo: epRepo}
}

// ResolveBaseStationID returns the base station ID for an EUI, or nil if unknown.
func (r *EUIResolverAdapter) ResolveBaseStationID(ctx context.Context, tenantID int64, bsEui []byte) (*int64, error) {
	if len(bsEui) < 8 || r.bsRepo == nil {
		return nil, nil
	}
	bs, err := r.bsRepo.GetByEUI(ctx, tenantID, bsEui)
	if err != nil || bs == nil {
		return nil, err
	}
	return &bs.ID, nil
}

// ResolveEndpointID returns the endpoint ID for an EUI, or nil if unknown.
func (r *EUIResolverAdapter) ResolveEndpointID(ctx context.Context, tenantID int64, epEui []byte) (*int64, error) {
	if len(epEui) < 8 || r.epRepo == nil {
		return nil, nil
	}
	ep, err := r.epRepo.GetByEUI(ctx, tenantID, epEui)
	if err != nil || ep == nil {
		return nil, err
	}
	return &ep.ID, nil
}
