package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// transactionalBaseStationReceptionRepository implements interfaces.BaseStationReceptionRepository within a transaction
type transactionalBaseStationReceptionRepository struct {
	tx *sql.Tx
	db *DB
}

// Create creates a new gateway reception within the transaction
func (r *transactionalBaseStationReceptionRepository) Create(ctx context.Context, reception *models.BaseStationReception) error {
	query := `
		INSERT INTO basestation_receptions (
			message_id, basestation_id, tenant_id, rssi, snr, eq_snr,
			frequency, channel, received_at, time_on_air_ms,
			basestation_latitude, basestation_longitude, basestation_altitude,
			antenna_id, fine_timestamp, context
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at`

	err := r.tx.QueryRowContext(ctx, query,
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

// GetByMessage retrieves all receptions for a message within the transaction
func (r *transactionalBaseStationReceptionRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.BaseStationReception, error) {
	query := `
		SELECT 
			id, message_id, basestation_id, tenant_id, rssi, snr, eq_snr,
			frequency, channel, received_at, time_on_air_ms,
			basestation_latitude, basestation_longitude, basestation_altitude,
			antenna_id, fine_timestamp, context, created_at
		FROM basestation_receptions
		WHERE message_id = $1
		ORDER BY rssi DESC`

	rows, err := r.tx.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query base station receptions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "GetByMessage")
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
		return nil, fmt.Errorf("error iterating base station receptions: %w", err)
	}

	return receptions, nil
}

// GetByBaseStation retrieves all receptions by a gateway within the transaction
func (r *transactionalBaseStationReceptionRepository) GetByBaseStation(ctx context.Context, baseStationID string, limit int, offset int) ([]*models.BaseStationReception, error) {
	query := `
		SELECT 
			id, message_id, basestation_id, tenant_id, rssi, snr, eq_snr,
			frequency, channel, received_at, time_on_air_ms,
			basestation_latitude, basestation_longitude, basestation_altitude,
			antenna_id, fine_timestamp, context, created_at
		FROM basestation_receptions
		WHERE basestation_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.tx.QueryContext(ctx, query, baseStationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query base station receptions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "GetByBaseStation")
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
		return nil, fmt.Errorf("error iterating base station receptions: %w", err)
	}

	return receptions, nil
}

// GetBestReception retrieves the reception with the best RSSI for a message within the transaction
func (r *transactionalBaseStationReceptionRepository) GetBestReception(ctx context.Context, messageID string) (*models.BaseStationReception, error) {
	query := `
		SELECT 
			id, message_id, basestation_id, tenant_id, rssi, snr, eq_snr,
			frequency, channel, received_at, time_on_air_ms,
			basestation_latitude, basestation_longitude, basestation_altitude,
			antenna_id, fine_timestamp, context, created_at
		FROM basestation_receptions
		WHERE message_id = $1
		ORDER BY rssi DESC
		LIMIT 1`

	reception := &models.BaseStationReception{}
	err := r.tx.QueryRowContext(ctx, query, messageID).Scan(
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
		return nil, fmt.Errorf("failed to get best base station reception: %w", err)
	}

	return reception, nil
}

// GetStats retrieves reception statistics for a gateway within the transaction
func (r *transactionalBaseStationReceptionRepository) GetStats(ctx context.Context, baseStationID string, hours int) (*models.BaseStationReceptionStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_receptions,
			AVG(rssi) as avg_rssi,
			MIN(rssi) as min_rssi,
			MAX(rssi) as max_rssi,
			AVG(snr) as avg_snr,
			MIN(snr) as min_snr,
			MAX(snr) as max_snr
		FROM basestation_receptions
		WHERE basestation_id = $1
		AND created_at >= NOW() - INTERVAL '%d hours'`

	query = fmt.Sprintf(query, hours)

	stats := &models.BaseStationReceptionStats{
		BaseStationID: baseStationID,
	}

	err := r.tx.QueryRowContext(ctx, query, baseStationID).Scan(
		&stats.TotalReceptions,
		&stats.AvgRSSI,
		&stats.MinRSSI,
		&stats.MaxRSSI,
		&stats.AvgSNR,
		&stats.MinSNR,
		&stats.MaxSNR,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get base station reception stats: %w", err)
	}

	return stats, nil
}

// GetReceptionStats gets reception statistics for a gateway within the transaction
func (r *transactionalBaseStationReceptionRepository) GetReceptionStats(ctx context.Context, baseStationID string, since time.Time) (*models.BaseStationReceptionStats, error) {
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

	err := r.tx.QueryRowContext(ctx, query, baseStationID, since).Scan(
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
		return nil, fmt.Errorf("failed to get base station reception stats: %w", err)
	}

	return stats, nil
}

// DeleteByMessage deletes all receptions for a message within the transaction
func (r *transactionalBaseStationReceptionRepository) DeleteByMessage(ctx context.Context, messageID string) error {
	query := `DELETE FROM basestation_receptions WHERE message_id = $1`

	_, err := r.tx.ExecContext(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete base station receptions: %w", err)
	}

	return nil
}
