package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/jmoiron/sqlx"
)

// Savepoint statements for WithSavepoint. One fixed name suffices: savepoints
// are transaction-scoped, WithSavepoint releases or rolls back before it
// returns, and re-establishing the same name is defined behavior.
const (
	sqlSavepoint           = "SAVEPOINT tx_step"
	sqlRollbackToSavepoint = "ROLLBACK TO SAVEPOINT tx_step"
	sqlReleaseSavepoint    = "RELEASE SAVEPOINT tx_step"
)

// Transaction represents a database transaction. It exposes only the
// repositories that participate in transactional operations: endpoint attach
// (endpoints + endpoint sessions), blueprint creation (device models +
// blueprints), downlink reservation (mioty downlinks) and SCACI session
// creation (SCACI sessions).
type Transaction struct {
	tx  *sqlx.Tx
	db  *DB
	log logger.Logger

	endpointRepo        interfaces.EndpointRepository
	endpointSessionRepo interfaces.EndPointSessionRepository
	miotyDownlinkRepo   interfaces.MIOTYDownlinkTxRepository
	deviceModelRepo     interfaces.DeviceModelRepository
	blueprintRepo       interfaces.BlueprintRepository
}

// BeginTx starts a new database transaction. It returns the concrete type:
// transactions are consumed by the storage adapters in this module, which pass
// a narrow, operation-specific view to their callers.
func (db *DB) BeginTx(ctx context.Context) (*Transaction, error) {
	tx, err := db.sqlxDB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapFailedToBeginTransaction, err)
	}

	return &Transaction{
		tx:  tx,
		db:  db,
		log: db.log.WithField("transaction", true),
	}, nil
}

// WithSavepoint runs fn inside a private savepoint. When fn fails, the
// transaction is rolled back to the savepoint - recovering it from an aborted
// statement (for example a unique violation) so the caller can retry inside
// the same transaction - and fn's error is returned. On success the savepoint
// is released.
func (t *Transaction) WithSavepoint(ctx context.Context, fn func() error) error {
	if _, err := t.tx.ExecContext(ctx, sqlSavepoint); err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateSavepoint, err)
	}
	if err := fn(); err != nil {
		if _, rbErr := t.tx.ExecContext(ctx, sqlRollbackToSavepoint); rbErr != nil {
			return errors.Join(err, fmt.Errorf("%s: %w", errWrapRollbackToSavepoint, rbErr))
		}
		return err
	}
	if _, err := t.tx.ExecContext(ctx, sqlReleaseSavepoint); err != nil {
		return fmt.Errorf("%s: %w", errWrapReleaseSavepoint, err)
	}
	return nil
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	err := t.tx.Commit()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToCommitTransaction, err)
	}
	t.log.Debug(logMsgTransactionCommitted)
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	err := t.tx.Rollback()
	if err != nil && err != sql.ErrTxDone {
		return fmt.Errorf("%s: %w", errWrapRollbackTransaction, err)
	}
	t.log.Debug(logMsgTransactionRolledBack)
	return nil
}

// EndPoints returns the endpoint repository for this transaction
func (t *Transaction) EndPoints() interfaces.EndpointRepository {
	if t.endpointRepo == nil {
		t.endpointRepo = &EndPointRepository{db: t.tx, cipher: t.db.cipher, clock: t.db.clock}
	}
	return t.endpointRepo
}

// UplinkClassifier returns the uplink classifier reset for this transaction.
func (t *Transaction) UplinkClassifier() interfaces.UplinkClassifierReset {
	return &UplinkClassifierReset{db: t.tx}
}

// EndPointSessions returns the endpoint session repository for this transaction
func (t *Transaction) EndPointSessions() interfaces.EndPointSessionRepository {
	if t.endpointSessionRepo == nil {
		t.endpointSessionRepo = &EndPointSessionRepository{db: t.tx, cipher: t.db.cipher, logger: t.log}
	}
	return t.endpointSessionRepo
}

// MIOTYDownlinks returns the transactional downlink repository, the only path
// to the FOR UPDATE SKIP LOCKED reservation queries.
func (t *Transaction) MIOTYDownlinks() interfaces.MIOTYDownlinkTxRepository {
	if t.miotyDownlinkRepo == nil {
		t.miotyDownlinkRepo = &DownlinkReservations{db: t.tx, clock: t.db.clock}
	}
	return t.miotyDownlinkRepo
}

// DeviceModels returns device model repository for this transaction
func (t *Transaction) DeviceModels() interfaces.DeviceModelRepository {
	if t.deviceModelRepo == nil {
		t.deviceModelRepo = NewDeviceModelRepository(t.tx, t.db.log)
	}
	return t.deviceModelRepo
}

// SCACISessions returns the SCACI session repository for this transaction.
func (t *Transaction) SCACISessions() interfaces.SCACISessionRepository {
	return &SCACISessionRepository{db: t.tx, clock: t.db.clock, log: t.log}
}

// Blueprints returns blueprint repository for this transaction
func (t *Transaction) Blueprints() interfaces.BlueprintRepository {
	if t.blueprintRepo == nil {
		t.blueprintRepo = NewBlueprintRepository(t.tx)
	}
	return t.blueprintRepo
}
