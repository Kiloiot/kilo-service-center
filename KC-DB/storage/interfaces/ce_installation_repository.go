package interfaces

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// CEInstallationRepository manages the singleton CE installation record.
// It enforces the table's singleton constraint (id = 1).
type CEInstallationRepository interface {
	// Get retrieves the singleton installation record, or storage.ErrNotFound
	// before one exists.
	Get(ctx context.Context) (*models.CEInstallation, error)

	// Update applies a partial update using the provided field map.
	// Caller must only include fields that should change.
	Update(ctx context.Context, updates map[string]interface{}) error
}
