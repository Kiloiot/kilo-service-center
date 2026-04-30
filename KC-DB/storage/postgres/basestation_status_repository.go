package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/mioty"
)

// BaseStationStatusRepository implements base station status history storage operations
// Implements interfaces.MIOTYBaseStationStatusRepository
type BaseStationStatusRepository struct {
	db *sqlx.DB
}

// NewBaseStationStatusRepository creates a new base station status repository
func NewBaseStationStatusRepository(db *sqlx.DB) *BaseStationStatusRepository {
	return &BaseStationStatusRepository{db: db}
}

// Create stores a new base station status record per BSSCI §3.5.2
func (r *BaseStationStatusRepository) Create(ctx context.Context, status *mioty.BaseStationStatusRecord) error {
	// Set timestamp if not already set
	if status.ReceivedAt.IsZero() {
		status.ReceivedAt = time.Now()
	}

	query := `
		INSERT INTO mioty_basestation_status (
			tenant_id, basestation_id, basestation_eui, operation_id,
			status_code, status_message, system_time,
			duty_cycle, uptime, temperature, cpu_load, memory_load,
			config, latitude, longitude, altitude,
			received_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16,
			$17
		)`

	// Convert Config []byte to *string for lib/pq JSONB text format compatibility
	var configStr *string
	if len(status.Config) > 0 && json.Valid(status.Config) {
		s := string(status.Config)
		configStr = &s
	}

	_, err := r.db.ExecContext(ctx, query,
		status.TenantID, status.BaseStationID, status.BasestationEUI, status.OperationID,
		status.StatusCode, status.StatusMessage, status.SystemTime,
		status.DutyCycle, status.UptimeSeconds, status.Temperature, status.CPULoad, status.MemoryLoad,
		configStr, status.Latitude, status.Longitude, status.Altitude,
		status.ReceivedAt,
	)

	if err != nil {
		return fmt.Errorf("insert base station status record: %w", err)
	}

	return nil
}
