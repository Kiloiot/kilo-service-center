// Package endpoint provides shared endpoint state management helpers
// used by both BSSCI and SCACI protocol handlers to maintain consistent
// endpoint lifecycle operations.
package endpoint

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// errDetachState reports a failed write of an endpoint's detach state.
var errDetachState = errors.New("endpoint detach state")

// DetachStateUpdater is the repository capability DetachEndpoint needs; both
// the full endpoint repository and narrower protocol-server views satisfy it.
type DetachStateUpdater interface {
	EndpointDetachStateUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointDetachStateParams) error
	TransitionEndpointStatus(ctx context.Context, tenantID int64, endpointID int64, status string) (bool, error)
}

// DetachTelemetry carries the optional detach-message fields recorded alongside
// the detach state transition. A nil *DetachTelemetry records no telemetry.
type DetachTelemetry struct {
	// Sign is the 4-byte endpoint detach signature; ignored unless it is
	// exactly 4 bytes.
	Sign []byte
	// PacketCnt is the endpoint packet counter from the detach message; nil
	// means it was not provided.
	PacketCnt *uint32
}

// DetachEndpoint marks an endpoint detached without modifying identity fields
// (nwk_key, app_key, etc.). It clears the base-station attachment and the
// propagation state, optionally recording detach-message telemetry, and
// reports whether this call is the one that moved the endpoint to detached;
// the status transition records the decision time.
//
// The attachment decider calls it for every detachment, with the det's
// telemetry for one heard over the air.
func DetachEndpoint(
	ctx context.Context,
	repo DetachStateUpdater,
	tenantID int64,
	endpointID int64,
	telemetry *DetachTelemetry,
) (bool, error) {
	propagateStatus := PropagateStatusDetached
	propagated := false

	params := models.EndpointDetachStateParams{
		LastAttachedBsEui: models.OptionalBytes{Set: true}, // clear attachment
		PropagateStatus:   &propagateStatus,
		Propagated:        &propagated,
		PropagatedAt:      models.OptionalNullTime{Set: true}, // clear timestamp
	}

	if telemetry != nil {
		if len(telemetry.Sign) == 4 {
			params.LastDetachSign = telemetry.Sign
		}
		if telemetry.PacketCnt != nil {
			params.LastDetachPacketCnt = telemetry.PacketCnt
		}
	}

	if err := repo.EndpointDetachStateUpdate(ctx, tenantID, endpointID, params); err != nil {
		return false, fmt.Errorf("%w: %w", errDetachState, err)
	}
	detached, err := repo.TransitionEndpointStatus(ctx, tenantID, endpointID, EndpointStatusDetached)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errDetachState, err)
	}
	return detached, nil
}
