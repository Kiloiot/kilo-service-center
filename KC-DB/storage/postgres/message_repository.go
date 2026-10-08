package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// MessageRepository implements interfaces.MIOTYMessageRepository: the
// stored uplinks and propagate messages of the messages table.
type MessageRepository struct {
	log   logger.Logger
	clock clock.Clock
	db    *sqlx.DB
}

// bsContainmentFmt renders the JSONB @> containment probe matching a base
// station inside the base_stations reception array.
const bsContainmentFmt = `[{"bsEui":%d}]`

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *sqlx.DB, clk clock.Clock, log logger.Logger) *MessageRepository {
	return &MessageRepository{
		log: log, clock: clk, db: db}
}

// uplinkExecutor is the query surface the uplink persistence helpers need,
// satisfied by *sqlx.DB and *sqlx.Tx alike so the same statements run inside
// the uplink store transaction.
type uplinkExecutor interface {
	sqlx.ExtContext
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// ClaimDownlinkWindow claims the downlink window of the owner's telegram for
// one dispatch and reports whether this caller got it (radio §3.6.1).
func (r *MessageRepository) ClaimDownlinkWindow(ctx context.Context, ownerTenantID int64, messageID string) (bool, error) {
	result, err := r.db.ExecContext(ctx, sqlClaimDownlinkWindow, messageID, ownerTenantID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapClaimDownlinkWindow, err)
	}
	claimed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapClaimDownlinkWindow, err)
	}
	return claimed == 1, nil
}

// ReleaseDownlinkWindow gives back the claim of a telegram whose dispatch sent nothing.
func (r *MessageRepository) ReleaseDownlinkWindow(ctx context.Context, ownerTenantID int64, messageID string) error {
	if _, err := r.db.ExecContext(ctx, sqlReleaseDownlinkWindow, messageID, ownerTenantID); err != nil {
		return fmt.Errorf("%s: %w", errWrapReleaseDownlinkWindow, err)
	}
	return nil
}

const (
	sqlClaimDownlinkWindow = `
		UPDATE messages SET dl_window_claimed = true
		WHERE id = $1 AND owner_tenant_id = $2 AND NOT dl_window_claimed`
	sqlReleaseDownlinkWindow = `
		UPDATE messages SET dl_window_claimed = false
		WHERE id = $1 AND owner_tenant_id = $2`
)
