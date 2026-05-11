package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// baseStationReceptionRepository implements interfaces.BaseStationReceptionRepository
type baseStationReceptionRepository struct {
	db *DB
}

// NewBaseStationReceptionRepository creates a new base station reception repository
func NewBaseStationReceptionRepository(db *DB) interfaces.BaseStationReceptionRepository {
	return &baseStationReceptionRepository{db: db}
}

// Create creates a new base station reception record
func (r *baseStationReceptionRepository) Create(ctx context.Context, reception *models.BaseStationReception) error {
	query := `
		INSERT INTO basestation_receptions (
			message_id, basestation_id, tenant_id,
			rssi, snr, eq_snr, frequency, channel,
			received_at, time_on_air_ms,
			basestation_latitude, basestation_longitude, basestation_altitude,
			antenna_id, fine_timestamp, context
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		reception.MessageID,
		reception.BaseStationID,
		reception.TenantID,
		reception.RSSI,
		reception.SNR,
		reception.EqSNR,
		reception.Frequency,
		reception.Channel,
		reception.ReceivedAt,
		reception.TimeOnAirMs,
		reception.BaseStationLatitude,
		reception.BaseStationLongitude,
		reception.BaseStationAltitude,
		reception.AntennaID,
		reception.FineTimestamp,
		reception.Context,
	).Scan(&reception.ID, &reception.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create base station reception: %w", err)
	}

	return nil
}

// GetByMessage retrieves all base station receptions for a specific message
func (r *baseStationReceptionRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.BaseStationReception, error) {
	query := `
		SELECT 
			id, message_id, basestation_id, tenant_id,
			rssi, snr, eq_snr, frequency, channel,
			received_at, time_on_air_ms,
			basestation_latitude, basestation_longitude, basestation_altitude,
			antenna_id, fine_timestamp, context,
			created_at
		FROM basestation_receptions
		WHERE message_id = $1
		ORDER BY rssi DESC`

	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query base station receptions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("reception rows close failed", "error", err)
		}
	}()

	var receptions []*models.BaseStationReception
	for rows.Next() {
		reception := &models.BaseStationReception{}
		err := rows.Scan(
			&reception.ID,
			&reception.MessageID,
			&reception.BaseStationID,
			&reception.TenantID,
			&reception.RSSI,
			&reception.SNR,
			&reception.EqSNR,
			&reception.Frequency,
			&reception.Channel,
			&reception.ReceivedAt,
			&reception.TimeOnAirMs,
			&reception.BaseStationLatitude,
			&reception.BaseStationLongitude,
			&reception.BaseStationAltitude,
			&reception.AntennaID,
			&reception.FineTimestamp,
			&reception.Context,
			&reception.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan base station reception: %w", err)
		}
		receptions = append(receptions, reception)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating base station receptions for message: %w", err)
	}

	return receptions, nil
}

// GetByBaseStation retrieves base station receptions for a specific base station
func (r *baseStationReceptionRepository) GetByBaseStation(ctx context.Context, baseStationID string, limit int, offset int) ([]*models.BaseStationReception, error) {
	query := `
		SELECT 
			id, message_id, basestation_id, tenant_id,
			rssi, snr, eq_snr, frequency, channel,
			received_at, time_on_air_ms,
			basestation_latitude, basestation_longitude, basestation_altitude,
			antenna_id, fine_timestamp, context,
			created_at
		FROM basestation_receptions
		WHERE basestation_id = $1
		ORDER BY received_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, baseStationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query base station receptions for base station: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("reception agg rows close failed", "error", err)
		}
	}()

	var receptions []*models.BaseStationReception
	for rows.Next() {
		reception := &models.BaseStationReception{}
		err := rows.Scan(
			&reception.ID,
			&reception.MessageID,
			&reception.BaseStationID,
			&reception.TenantID,
			&reception.RSSI,
			&reception.SNR,
			&reception.EqSNR,
			&reception.Frequency,
			&reception.Channel,
			&reception.ReceivedAt,
			&reception.TimeOnAirMs,
			&reception.BaseStationLatitude,
			&reception.BaseStationLongitude,
			&reception.BaseStationAltitude,
			&reception.AntennaID,
			&reception.FineTimestamp,
			&reception.Context,
			&reception.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan base station reception: %w", err)
		}
		receptions = append(receptions, reception)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating base station receptions for base station: %w", err)
	}

	return receptions, nil
}

// GetBestReception retrieves the best reception (highest RSSI) for a message
func (r *baseStationReceptionRepository) GetBestReception(ctx context.Context, messageID string) (*models.BaseStationReception, error) {
	query := `
		SELECT 
			id, message_id, basestation_id, tenant_id,
			rssi, snr, eq_snr, frequency, channel,
			received_at, time_on_air_ms,
			basestation_latitude, basestation_longitude, basestation_altitude,
			antenna_id, fine_timestamp, context,
			created_at
		FROM basestation_receptions
		WHERE message_id = $1
		ORDER BY rssi DESC
		LIMIT 1`

	reception := &models.BaseStationReception{}
	err := r.db.QueryRow(ctx, query, messageID).Scan(
		&reception.ID,
		&reception.MessageID,
		&reception.BaseStationID,
		&reception.TenantID,
		&reception.RSSI,
		&reception.SNR,
		&reception.EqSNR,
		&reception.Frequency,
		&reception.Channel,
		&reception.ReceivedAt,
		&reception.TimeOnAirMs,
		&reception.BaseStationLatitude,
		&reception.BaseStationLongitude,
		&reception.BaseStationAltitude,
		&reception.AntennaID,
		&reception.FineTimestamp,
		&reception.Context,
		&reception.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get best base station reception for message: %w", err)
	}

	return reception, nil
}

// GetReceptionStats gets reception statistics for a base station
func (r *baseStationReceptionRepository) GetReceptionStats(ctx context.Context, baseStationID string, since time.Time) (*models.BaseStationReceptionStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_receptions,
			AVG(rssi) as avg_rssi,
			MIN(rssi) as min_rssi,
			MAX(rssi) as max_rssi,
			AVG(snr) as avg_snr,
			MIN(snr) as min_snr,
			MAX(snr) as max_snr,
			COUNT(DISTINCT message_id) as unique_messages
		FROM basestation_receptions
		WHERE basestation_id = $1 AND received_at >= $2`

	stats := &models.BaseStationReceptionStats{
		BaseStationID: baseStationID,
		Since:         since,
	}

	err := r.db.QueryRow(ctx, query, baseStationID, since).Scan(
		&stats.TotalReceptions,
		&stats.AvgRSSI,
		&stats.MinRSSI,
		&stats.MaxRSSI,
		&stats.AvgSNR,
		&stats.MinSNR,
		&stats.MaxSNR,
		&stats.UniqueMessages,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get reception stats for base station: %w", err)
	}

	return stats, nil
}

// DeleteByMessage deletes all base station receptions for a message
func (r *baseStationReceptionRepository) DeleteByMessage(ctx context.Context, messageID string) error {
	query := `DELETE FROM basestation_receptions WHERE message_id = $1`

	result, err := r.db.Exec(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete base station receptions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
