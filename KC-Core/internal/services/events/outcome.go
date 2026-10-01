package events

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// outcomeSeverities maps the two request outcomes onto the severities the
// repository stores; an unknown outcome is rejected before any query.
var outcomeSeverities = map[string][]string{
	grpcservices.EventOutcomeSuccess: {models.EventSeverityInfo, models.EventSeverityWarning},
	grpcservices.EventOutcomeFailure: {models.EventSeverityError, models.EventSeverityCritical},
}

// severitiesFor narrows the requested severities to the outcome's set; with
// no outcome the request severities pass through.
func severitiesFor(outcome string, requested []string) ([]string, error) {
	if outcome == "" {
		return requested, nil
	}
	allowed, ok := outcomeSeverities[outcome]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOutcome, outcome)
	}
	if len(requested) == 0 {
		return allowed, nil
	}
	narrowed := make([]string, 0, len(requested))
	for _, severity := range requested {
		if slices.Contains(allowed, severity) {
			narrowed = append(narrowed, severity)
		}
	}
	if len(narrowed) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrOutcomeSeverityMismatch, outcome)
	}
	return narrowed, nil
}

// Request validation failures surfaced to the delivery layer.
var (
	// ErrInvalidOutcome reports an outcome outside success/failure.
	ErrInvalidOutcome = errors.New("invalid event outcome")
	// ErrOutcomeSeverityMismatch reports severities that belong to another outcome.
	ErrOutcomeSeverityMismatch = errors.New("severity does not belong to outcome")
	// ErrInvalidEndpointEUI reports an endpoint filter that is not an EUI.
	ErrInvalidEndpointEUI = errors.New("invalid endpoint EUI")
	// errResolveEventScope reports a device lookup that failed while scoping events.
	errResolveEventScope = errors.New("resolve event scope")
)
