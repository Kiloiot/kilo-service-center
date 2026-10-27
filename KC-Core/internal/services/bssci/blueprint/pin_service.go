package blueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// PinBatchSize is how many endpoints one pass reads at a time.
const PinBatchSize = 500

// Outcomes of pinning one endpoint.
const (
	PinOutcomePinned       = "pinned"
	PinOutcomeNothingToPin = "nothing_to_pin"
	PinOutcomeError        = "error"
)

// Failure texts of a pin result; the cause goes to the log only.
const (
	pinErrResolve  = "blueprint lookup failed"
	pinErrSnapshot = "snapshot build failed"
	pinErrWrite    = "snapshot write failed"
)

// PinResult is the outcome for one endpoint that had no snapshot.
type PinResult struct {
	EpEUI       string
	Outcome     string
	BlueprintID string
	Error       string
}

// ModeIgnoringResolver finds what an endpoint decodes with regardless of the tenant's switch.
type ModeIgnoringResolver interface {
	ResolveIgnoringMode(ctx context.Context, tenantID int64, endpoint *models.EndPoint) (*models.Blueprint, error)
}

// UnpinnedEndpoints pages an owner's endpoints without a snapshot and pins one.
type UnpinnedEndpoints interface {
	ListUnpinnedByOwner(ctx context.Context, ownerTenantID, afterID int64, limit int) ([]*models.EndPoint, error)
	PinSnapshotIfUnpinned(ctx context.Context, ownerTenantID, endpointID int64, snapshot json.RawMessage) (bool, error)
}

// PinService pins a tenant's endpoints without a snapshot to what they decode with now.
type PinService struct {
	resolver  ModeIgnoringResolver
	endpoints UnpinnedEndpoints
	now       func() time.Time
	log       logger.Logger
}

// NewPinService builds the service; now stamps the snapshots it pins.
func NewPinService(resolver ModeIgnoringResolver, endpoints UnpinnedEndpoints, now func() time.Time, log logger.Logger) (*PinService, error) {
	if resolver == nil || endpoints == nil || now == nil {
		return nil, errNilPinDependency
	}
	return &PinService{resolver: resolver, endpoints: endpoints, now: now, log: log}, nil
}

// Pin leaves out an endpoint another write pinned meanwhile: it no longer lacks a snapshot.
func (p *PinService) Pin(ctx context.Context, tenantID int64, dryRun bool) ([]PinResult, error) {
	var results []PinResult
	var after int64
	for {
		page, err := p.endpoints.ListUnpinnedByOwner(ctx, tenantID, after, PinBatchSize)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapListUnpinned, err)
		}
		for _, ep := range page {
			if result, reported := p.pinOne(ctx, tenantID, ep, dryRun); reported {
				results = append(results, result)
			}
		}
		if len(page) < PinBatchSize {
			return results, nil
		}
		after = page[len(page)-1].ID
	}
}

func (p *PinService) pinOne(ctx context.Context, tenantID int64, ep *models.EndPoint, dryRun bool) (PinResult, bool) {
	result := PinResult{EpEUI: ep.EUI.String()}
	bp, err := p.resolver.ResolveIgnoringMode(ctx, tenantID, ep)
	if err != nil {
		return p.failed(ctx, result, pinErrResolve, err), true
	}
	if bp == nil {
		result.Outcome = PinOutcomeNothingToPin
		return result, true
	}
	result.BlueprintID = bp.ID.String()
	snapshot, err := json.Marshal(models.NewBlueprintSnapshot(bp, p.now().UTC()))
	if err != nil {
		return p.failed(ctx, result, pinErrSnapshot, err), true
	}
	if !dryRun {
		written, err := p.endpoints.PinSnapshotIfUnpinned(ctx, tenantID, ep.ID, snapshot)
		if err != nil {
			return p.failed(ctx, result, pinErrWrite, err), true
		}
		if !written {
			return result, false
		}
	}
	result.Outcome = PinOutcomePinned
	return result, true
}

func (p *PinService) failed(ctx context.Context, result PinResult, text string, cause error) PinResult {
	p.log.WarnContext(ctx, LogBlueprintPinFailed, logger.FieldEndpointEui, result.EpEUI, logger.FieldError, cause)
	result.Outcome = PinOutcomeError
	result.Error = text
	result.BlueprintID = ""
	return result
}
