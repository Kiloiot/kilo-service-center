package interfaces

import (
	"context"

	"github.com/kilocenter/KC-DB/storage/mioty"
)

// MIOTYBaseStationStatusRepository provides base station status history storage operations
type MIOTYBaseStationStatusRepository interface {
	// Create stores a new base station status record
	Create(ctx context.Context, status *mioty.BaseStationStatusRecord) error
}
