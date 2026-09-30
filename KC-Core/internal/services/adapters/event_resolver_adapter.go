package adapters

import (
	"context"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// baseStationByEUI is the single base station lookup the resolver performs.
// Satisfied structurally by the KC-DB base station repository.
type baseStationByEUI interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)
}

// endpointByEUI is the single endpoint lookup the resolver performs.
// Satisfied structurally by the KC-DB endpoint repository.
type endpointByEUI interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error)
}

// EUIResolverAdapter resolves device EUIs to internal IDs for event scoping.
// An unknown device yields a nil ID, so callers fall back to the source_name
// EUI filter; a failed lookup is returned.
type EUIResolverAdapter struct {
	bsRepo baseStationByEUI
	epRepo endpointByEUI
}

// NewEUIResolver creates a resolver backed by the base station and endpoint repositories.
func NewEUIResolver(bsRepo baseStationByEUI, epRepo endpointByEUI) *EUIResolverAdapter {
	return &EUIResolverAdapter{bsRepo: bsRepo, epRepo: epRepo}
}

// ResolveBaseStationID returns the base station ID for an EUI, or nil if unknown.
func (r *EUIResolverAdapter) ResolveBaseStationID(ctx context.Context, tenantID int64, bsEui []byte) (*int64, error) {
	if len(bsEui) < 8 || r.bsRepo == nil {
		return nil, nil
	}
	bs, err := r.bsRepo.GetByEUI(ctx, tenantID, bsEui)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
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
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil || ep == nil {
		return nil, err
	}
	return &ep.ID, nil
}
