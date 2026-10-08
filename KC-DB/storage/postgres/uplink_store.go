package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// UplinkStore is the PostgreSQL uplink classifier and writer. One
// transaction claims the (owner tenant, endpoint, packet counter) row, then
// either stores a new message with its outbox rows or merges the reception
// into the first message of the window; no in-memory state takes part.
// The request's tenant is the endpoint owner and keys every row written.
type UplinkStore struct {
	log   logger.Logger
	clock clock.Clock
	db    *sqlx.DB
}

// NewUplinkStore creates the uplink store over the shared connection pool.
func NewUplinkStore(db *sqlx.DB, clk clock.Clock, log logger.Logger) *UplinkStore {
	return &UplinkStore{
		log: log, clock: clk, db: db}
}

// classifierRow mirrors one mioty_message_deduplication row.
type classifierRow struct {
	MessageHash    []byte `db:"message_hash"`
	FirstMessageID string `db:"first_message_id"`
	FirstRxTime    int64  `db:"first_rx_time"`
	DuplicateCount int    `db:"duplicate_count"`
}

// sameTelegram windows on radio time (BSSCI §3.10.1 rxTime) so a late-arriving copy of the telegram still merges.
func (r classifierRow) sameTelegram(rxTime int64, window time.Duration) bool {
	spread := rxTime - r.FirstRxTime
	if spread < 0 {
		spread = -spread
	}
	return spread <= window.Nanoseconds()
}

const (
	sqlClaimClassifierRow = `
		INSERT INTO mioty_message_deduplication (
			owner_tenant_id, ep_eui, packet_cnt, message_hash,
			first_message_id, first_bs_eui, first_received_at, last_received_at, first_rx_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8)
		ON CONFLICT (owner_tenant_id, ep_eui, packet_cnt) DO NOTHING`
	sqlLockClassifierRow = `
		SELECT message_hash, first_message_id, first_rx_time, duplicate_count
		FROM mioty_message_deduplication
		WHERE owner_tenant_id = $1 AND ep_eui = $2 AND packet_cnt = $3
		FOR UPDATE`
	sqlResetClassifierRow = `
		UPDATE mioty_message_deduplication
		SET message_hash = $4, first_message_id = $5, first_bs_eui = $6,
		    first_received_at = $7, last_received_at = $7, first_rx_time = $8, duplicate_count = 0
		WHERE owner_tenant_id = $1 AND ep_eui = $2 AND packet_cnt = $3`
	sqlCountDuplicate = `
		UPDATE mioty_message_deduplication
		SET duplicate_count = duplicate_count + 1, last_received_at = $4
		WHERE owner_tenant_id = $1 AND ep_eui = $2 AND packet_cnt = $3`
	sqlLockMessageReceptions = `
		SELECT base_stations FROM messages WHERE id = $1 AND owner_tenant_id = $2 FOR UPDATE`
	sqlMergeMessageReceptions = `
		UPDATE messages SET base_stations = $2, duplicate = true, updated_at = NOW() WHERE id = $1`
)

// Persist classifies the reception in req and stores it accordingly.
func (s *UplinkStore) Persist(ctx context.Context, req models.UplinkPersistRequest) (_ models.UplinkPersistOutcome, err error) {
	if err := validateUplinkPersistRequest(req); err != nil {
		return models.UplinkPersistOutcome{}, err
	}
	msg := req.Message
	reception := msg.BaseStations[0]
	hash := uplinkMessageHash(msg.EpEui, msg.PacketCnt, msg.UserData)
	epEui := mioty.EUI64Bytes(msg.EpEui)
	bsEui := mioty.EUI64Bytes(reception.BsEui)
	now := s.clock.Now()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapFailedToBeginTransaction, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	if _, err = tx.ExecContext(ctx, sqlClaimClassifierRow,
		msg.TenantID, epEui, int64(msg.PacketCnt), hash, msg.ID, bsEui, now, reception.RxTime); err != nil {
		return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapClaimUplinkClassifier, err)
	}
	var row classifierRow
	if err = tx.GetContext(ctx, &row, sqlLockClassifierRow, msg.TenantID, epEui, int64(msg.PacketCnt)); err != nil {
		return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapLockUplinkClassifier, err)
	}

	var outcome models.UplinkPersistOutcome
	switch {
	case row.FirstMessageID == msg.ID:
		outcome, err = s.persistNew(ctx, tx, req, false)
	case !row.sameTelegram(reception.RxTime, req.Window):
		if _, err = tx.ExecContext(ctx, sqlResetClassifierRow,
			msg.TenantID, epEui, int64(msg.PacketCnt), hash, msg.ID, bsEui, now, reception.RxTime); err != nil {
			return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapResetUplinkClassifier, err)
		}
		outcome, err = s.persistNew(ctx, tx, req, true)
	case !bytes.Equal(row.MessageHash, hash):
		return models.UplinkPersistOutcome{}, storage.ErrPacketCounterCollision
	default:
		outcome, err = s.mergeDuplicate(ctx, tx, req, row, now)
	}
	if err != nil {
		return models.UplinkPersistOutcome{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapCommitUplinkStore, err)
	}
	return outcome, nil
}

func validateUplinkPersistRequest(req models.UplinkPersistRequest) error {
	switch {
	case req.Message == nil:
		return fmt.Errorf("%s: %w", errWrapUplinkPersistRequest, storage.ErrInvalidInput)
	case req.Message.TenantID <= 0:
		return fmt.Errorf("%s: %w", errWrapUplinkPersistRequest, storage.ErrInvalidTenantID)
	case req.Message.ID == "":
		return fmt.Errorf("%s: %w", errWrapUplinkPersistRequestMessageID, storage.ErrInvalidInput)
	case len(req.Message.BaseStations) != 1:
		return fmt.Errorf("%s: %w", errWrapUplinkPersistRequestReception, storage.ErrInvalidInput)
	case req.Message.BaseStations[0].RxTime <= 0:
		return fmt.Errorf("%s: %w", errWrapUplinkPersistRequestRxTime, storage.ErrInvalidInput)
	case req.Window <= 0:
		return fmt.Errorf("%s: %w", errWrapUplinkPersistRequestWindow, storage.ErrInvalidInput)
	case req.ReceptionWindow < 0 || req.ReceptionWindow >= req.Window:
		return fmt.Errorf("%s: %w", errWrapUplinkPersistRequestReceptionWindow, storage.ErrInvalidInput)
	}
	return nil
}

// persistNew stores the message, refreshes the endpoint and queues its
// delivery for when the reception window closes, so the receptions of the
// other base stations are merged by then; counterReused records a counter
// already used outside the duplicate window.
func (s *UplinkStore) persistNew(ctx context.Context, tx *sqlx.Tx, req models.UplinkPersistRequest, counterReused bool) (models.UplinkPersistOutcome, error) {
	msg := req.Message
	notDuplicate := false
	msg.Duplicate = &notDuplicate
	msg.PacketCntReused = counterReused
	storedAt := s.clock.Now()
	if err := insertULDataMessage(ctx, tx, s.log, msg, storedAt); err != nil {
		return models.UplinkPersistOutcome{}, err
	}
	if err := updateEndpointFromULData(ctx, tx, msg); err != nil {
		return models.UplinkPersistOutcome{}, err
	}
	if err := insertDeliveryRows(ctx, tx, msg.ID, msg.TenantID, req.Channels, storedAt, storedAt.Add(req.ReceptionWindow)); err != nil {
		return models.UplinkPersistOutcome{}, err
	}
	return models.UplinkPersistOutcome{
		Classification: models.UplinkNew,
		MessageID:      msg.ID,
		BaseStations:   append([]mioty.BaseStationReception(nil), msg.BaseStations...),
	}, nil
}

// mergeDuplicate folds the reception into the first message of the window,
// keyed by base station so a repeated reception from the same station
// replaces rather than duplicates its entry.
func (s *UplinkStore) mergeDuplicate(ctx context.Context, tx *sqlx.Tx, req models.UplinkPersistRequest,
	row classifierRow, now time.Time,
) (models.UplinkPersistOutcome, error) {
	msg := req.Message
	var stored []byte
	err := tx.GetContext(ctx, &stored, sqlLockMessageReceptions, row.FirstMessageID, msg.TenantID)
	if err != nil {
		return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapLockFirstUplinkMessage, err)
	}
	var receptions []mioty.BaseStationReception
	if len(stored) > 0 {
		if err = json.Unmarshal(stored, &receptions); err != nil {
			return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapDeserializeBaseStations, err)
		}
	}
	receptions = mergeReception(receptions, msg.BaseStations[0])
	merged, err := json.Marshal(receptions)
	if err != nil {
		return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapSerializeBaseStations, err)
	}
	mergedStr := string(merged)
	if _, err = tx.ExecContext(ctx, sqlMergeMessageReceptions, row.FirstMessageID, &mergedStr); err != nil {
		return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapUpdateBaseStations, err)
	}
	if _, err = tx.ExecContext(ctx, sqlCountDuplicate,
		msg.TenantID, mioty.EUI64Bytes(msg.EpEui), int64(msg.PacketCnt), now); err != nil {
		return models.UplinkPersistOutcome{}, fmt.Errorf("%s: %w", errWrapCountUplinkDuplicate, err)
	}
	return models.UplinkPersistOutcome{
		Classification: models.UplinkDuplicate,
		MessageID:      row.FirstMessageID,
		BaseStations:   receptions,
		DuplicateCount: row.DuplicateCount + 1,
	}, nil
}

// mergeReception replaces the entry for the same base station or appends.
func mergeReception(receptions []mioty.BaseStationReception, reception mioty.BaseStationReception) []mioty.BaseStationReception {
	for i := range receptions {
		if receptions[i].BsEui == reception.BsEui {
			receptions[i] = reception
			return receptions
		}
	}
	return append(receptions, reception)
}

// uplinkMessageHash fingerprints the packet content so a reused counter with
// different data is caught as a collision rather than merged.
func uplinkMessageHash(epEui uint64, packetCnt uint32, userData []byte) []byte {
	hasher := sha256.New()
	hasher.Write(mioty.EUI64Bytes(epEui))
	var counter [4]byte
	binary.BigEndian.PutUint32(counter[:], packetCnt)
	hasher.Write(counter[:])
	hasher.Write(userData)
	return hasher.Sum(nil)
}
