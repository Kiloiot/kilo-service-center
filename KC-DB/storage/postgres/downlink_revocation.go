package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// DownlinkRevocations ends downlinks revoked where they wait: in the service
// center's queue, or at the base station that answered a dlDataRev.
type DownlinkRevocations struct {
	db    sqlx.ExtContext
	clock clock.Clock
}

// RevokeDownlink ends the tenant's downlink revoked and reports whether it
// did. A revoke in the queue ends only a pending downlink, which no base
// station holds yet, so a dispatcher that reserved it first wins. A base
// station's answer ends only the in-flight downlink that station holds,
// including one whose dlDataQue it had not confirmed yet; a downlink another
// station holds since, one it was asked to drop when its lifetime ended
// (DownlinkRevoking ends that one) and one that already finished keep their
// state. The result column holds the dlDataRes outcome and is left alone.
func (r *DownlinkRevocations) RevokeDownlink(ctx context.Context, revocation storage.DownlinkRevocation) (bool, error) {
	scope := newSQLScope(mioty.DLQueueStatusRevoked, r.clock.Now())
	scopeRevocation(scope, revocation)
	if revocation.Station == nil {
		scope.equals(colStatus, mioty.DLQueueStatusPending)
	} else {
		scope.anyOf(colStatus, statusArray(revocableAtStation))
		scope.equals(colBsEUI, mioty.EUI64Bytes(*revocation.Station))
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE downlink_queue SET status = $1, updated_at = $2 WHERE `+scope.where(), scope.args...)
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapRevokeDownlink, err)
	}
	revoked, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapGetAffectedRows, err)
	}
	return revoked > 0, nil
}

// revocableAtStation are the held states a station's revoke answer ends revoked.
var revocableAtStation = []mioty.DLQueueStatus{mioty.DLQueueStatusReserved, mioty.DLQueueStatusQueued}

// scopeRevocation narrows a scope to the downlink a revocation names: the
// tenant's queue id, of the organization and endpoint when it names them.
func scopeRevocation(scope *sqlScope, revocation storage.DownlinkRevocation) {
	scope.equals(colQueID, revocation.QueID)
	scope.equals(colTenantID, revocation.TenantID)
	scope.organization(revocation.OrganizationID)
	if revocation.EpEUI != nil {
		scope.equals(colEpEUI, mioty.EUI64Bytes(*revocation.EpEUI))
	}
}
