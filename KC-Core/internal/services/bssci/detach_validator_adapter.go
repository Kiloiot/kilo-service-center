package bssciservices

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// detachValidatorDirectAdapter implements bssci.DetachSignatureValidator via direct repository calls.
type detachValidatorDirectAdapter struct {
	endpointRepo interfaces.EndpointRepository
	logger       logger.Logger
}

// NewDetachValidatorDirectAdapter creates a direct repository adapter for detach signature validation.
func NewDetachValidatorDirectAdapter(endpointRepo interfaces.EndpointRepository, log logger.Logger) bssci.DetachSignatureValidator {
	return &detachValidatorDirectAdapter{
		endpointRepo: endpointRepo,
		logger:       log,
	}
}

// ValidateDetachSignature validates an unknown endpoint's detach signature via direct database lookup.
//
// Validation chain:
//  1. Retrieve endpoint with crypto material (Sign, NwkSnKey, PresharedKey) and tenant metadata
//  2. Call ValidateDetachSignatureCMAC with fallback chain (equality check until CMAC IV defined by spec)
//  3. Return validation status with tenant/org metadata for roaming scenarios
//
// Error sentinel translation:
//   - storage.ErrNotFound → bssci.ErrDetachValidationEndpointNotFound (for errors.Is detection in server.go)
//   - signature mismatch → bssci.ErrDetachSignatureInvalid
func (a *detachValidatorDirectAdapter) ValidateDetachSignature(ctx context.Context, epEUI uint64, detachSign []byte) (*bssci.DetachValidationResult, error) {
	// Convert uint64 EUI to models.EUI [8]byte
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], epEUI)

	// Retrieve endpoint with all crypto fields and tenant metadata
	endpoint, err := a.endpointRepo.GetEndpointWithKeysForDetachValidation(ctx, eui)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			a.logger.WarnContext(ctx, "Detach validation failed: endpoint not found",
				"ep_eui", fmt.Sprintf("%016x", epEUI))
			// Return sentinel error for errors.Is detection in server.go
			return nil, bssci.ErrDetachValidationEndpointNotFound
		}
		a.logger.ErrorContext(ctx, "Detach validation database error",
			"ep_eui", fmt.Sprintf("%016x", epEUI),
			"error", err.Error())
		return nil, fmt.Errorf("failed to retrieve endpoint: %w", err)
	}

	// Validate signature using hybrid CMAC/fallback approach
	// TODO: Uses signature equality until MIOTY spec defines detach CMAC IV format
	err = bssci.ValidateDetachSignatureCMAC(detachSign, endpoint.Sign, endpoint.NwkSnKey, endpoint.PresharedKey)
	if err != nil {
		a.logger.WarnContext(ctx, "Detach validation failed: signature mismatch",
			"ep_eui", fmt.Sprintf("%016x", epEUI),
			"tenant_id", endpoint.TenantID,
			"error", err.Error())
		// Return tenant metadata + typed sentinel (roaming handoff may need context)
		return &bssci.DetachValidationResult{
			Valid:            false,
			TenantID:         endpoint.TenantID,
			OwnerTenantID:    endpoint.OwnerTenantID,
			ValidationStatus: bssci.ValidationStatusInvalidSignature,
		}, bssci.ErrDetachSignatureInvalid
	}

	// Signature valid - return full metadata
	a.logger.InfoContext(ctx, "Detach validation succeeded via direct repository",
		"ep_eui", fmt.Sprintf("%016x", epEUI),
		"tenant_id", endpoint.TenantID,
		"owner_tenant_id", endpoint.OwnerTenantID,
		"validation_status", bssci.ValidationStatusValidated)

	return &bssci.DetachValidationResult{
		Valid:            true,
		TenantID:         endpoint.TenantID,
		OwnerTenantID:    endpoint.OwnerTenantID,
		ValidationStatus: bssci.ValidationStatusValidated,
	}, nil
}
