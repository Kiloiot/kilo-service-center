package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// sqlForgetEndpointClassifier deletes an endpoint's classifier rows, which the
// uplink store keys by the endpoint's owner tenant and EUI.
const sqlForgetEndpointClassifier = `
	DELETE FROM mioty_message_deduplication d
	USING endpoints e
	WHERE e.tenant_id = $1 AND e.id = $2
	  AND d.owner_tenant_id = e.owner_tenant_id AND d.ep_eui = e.ep_eui`

// UplinkClassifierReset forgets the uplink store's duplicate-classifier rows
// of one endpoint inside the caller's transaction.
type UplinkClassifierReset struct {
	db sqlx.ExtContext
}

// ForgetEndpoint deletes the endpoint's classifier rows, so the packet
// counters an over-the-air attach restarted read as new data (radio protocol
// §3.6.5.3).
func (r *UplinkClassifierReset) ForgetEndpoint(ctx context.Context, tenantID int64, endpointID int64) error {
	if _, err := r.db.ExecContext(ctx, sqlForgetEndpointClassifier, tenantID, endpointID); err != nil {
		return fmt.Errorf("%s: %w", errWrapForgetUplinkClassifier, err)
	}
	return nil
}
