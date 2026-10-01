package downlinks

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Revoke request outcomes: the downlink was revoked in the queue, and so
// carries the revoked queue status, or the base station holding it was asked
// to revoke it and the row ends revoked once it confirms.
const (
	RevokeStatusRevoked   = string(mioty.DLQueueStatusRevoked)
	RevokeStatusInitiated = "revoke_initiated"
)

// RevokeResult tells where a revoke went: Station is zero when the downlink
// was revoked in the queue, else the base station asked to revoke it; Status
// is the outcome of the request.
type RevokeResult struct {
	Station uint64
	Status  string
}

// revokeResult is the outcome of a revoke the station, if any, was asked for.
func revokeResult(station uint64) RevokeResult {
	if station == 0 {
		return RevokeResult{Status: RevokeStatusRevoked}
	}
	return RevokeResult{Station: station, Status: RevokeStatusInitiated}
}

// recordQueued audits a queued downlink.
func (s *Service) recordQueued(ctx context.Context, owner Owner, result *scaci.DLDataQueueResult) {
	epEUI := mioty.FormatEUI64(owner.EpEUI)
	details := map[string]any{
		bssci.EventKeyEpEui:         epEUI,
		bssci.EventKeyQueID:         result.QueID,
		models.EventDetailKeyOpID:   result.OpID,
		models.EventDetailKeyStatus: string(result.Status),
	}
	if result.BsEui != 0 {
		details[bssci.EventKeyBsEui] = mioty.FormatEUI64(result.BsEui)
	}
	s.audit.Record(ctx, audit.Event{
		TenantID:    owner.TenantID,
		EventType:   models.EventTypeDownlinkQueued,
		Title:       models.EventTitleDownlinkQueued,
		Description: fmt.Sprintf(models.EventDescriptionDownlinkQueued, result.QueID, epEUI),
		Details:     details,
	})
}

// recordRevoke audits a revoke request and where it went.
func (s *Service) recordRevoke(ctx context.Context, target Target, result RevokeResult) {
	epEUI := mioty.FormatEUI64(target.EpEUI)
	details := map[string]any{
		bssci.EventKeyEpEui:         epEUI,
		bssci.EventKeyQueID:         target.QueID,
		models.EventDetailKeyStatus: result.Status,
	}
	if result.Station != 0 {
		details[bssci.EventKeyBsEui] = mioty.FormatEUI64(result.Station)
	}
	s.audit.Record(ctx, audit.Event{
		TenantID:    target.TenantID,
		EventType:   models.EventTypeDownlinkRevokeRequested,
		Title:       models.EventTitleDownlinkRevokeRequested,
		Description: fmt.Sprintf(models.EventDescriptionDownlinkRevokeRequested, target.QueID, epEUI),
		Details:     details,
	})
}
