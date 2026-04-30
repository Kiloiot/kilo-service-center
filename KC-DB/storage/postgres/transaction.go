package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-Core/pkg/crypto"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// Transaction represents a database transaction
type Transaction struct {
	tx           *sql.Tx
	db           *DB
	log          logger.Logger
	keyEncryptor *crypto.KeyEncryptor // Optional - for session key lazy migration

	// Repository instances for this transaction
	endpointRepo             interfaces.EndpointRepository
	baseStationRepo          interfaces.BaseStationRepository
	downlinkQueueRepo        interfaces.DownlinkQueueRepository
	baseStationReceptionRepo interfaces.BaseStationReceptionRepository
	endpointSessionRepo      interfaces.EndPointSessionRepository
	endpointKeyRepo          interfaces.EndPointKeyRepository
	roamingRepo              interfaces.RoamingAgreementRepository
	basestationSessionRepo   interfaces.BaseStationSessionRepository
	pendingOpRepo            interfaces.PendingOperationRepository
	miotyDownlinkRepo        interfaces.MIOTYDownlinkRepository // transactional dispatch
	userRepo                 interfaces.UserRepository          // User management
	apiKeyRepo               interfaces.APIKeyRepository        // API key management
	integrationRepo          interfaces.IntegrationRepository   // Integration management

	// Blueprint device catalog repositories (Migration 000102-000104)
	manufacturerRepo interfaces.ManufacturerRepository
	deviceModelRepo  interfaces.DeviceModelRepository
	blueprintRepo    interfaces.BlueprintRepository
}

// Ensure Transaction implements interfaces.Transaction
var _ interfaces.Transaction = (*Transaction)(nil)

// BeginTx starts a new database transaction
func (db *DB) BeginTx(ctx context.Context) (interfaces.Transaction, error) {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &Transaction{
		tx:           tx,
		db:           db,
		log:          db.log.WithField("transaction", true),
		keyEncryptor: db.keyEncryptor, // Pass through for repository lazy migration
	}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	err := t.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	t.log.Debug("Transaction committed")
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	err := t.tx.Rollback()
	if err != nil && err != sql.ErrTxDone {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	t.log.Debug("Transaction rolled back")
	return nil
}

// EndPoints returns the endpoint repository for this transaction
func (t *Transaction) EndPoints() interfaces.EndpointRepository {
	if t.endpointRepo == nil {
		t.endpointRepo = &transactionalEndPointRepository{tx: t.tx, db: t.db}
	}
	return t.endpointRepo
}

// BaseStations returns the base station repository for this transaction
func (t *Transaction) BaseStations() interfaces.BaseStationRepository {
	if t.baseStationRepo == nil {
		t.log.Warn("BaseStation repository not supported in transactions")
	}
	return t.baseStationRepo
}

// DownlinkQueue returns the downlink queue repository for this transaction
func (t *Transaction) DownlinkQueue() interfaces.DownlinkQueueRepository {
	if t.downlinkQueueRepo == nil {
		t.log.Warn("Downlink queue repository not supported in transactions")
	}
	return t.downlinkQueueRepo
}

// BaseStationReceptions returns the base station reception repository for this transaction
func (t *Transaction) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	if t.baseStationReceptionRepo == nil {
		t.baseStationReceptionRepo = &transactionalBaseStationReceptionRepository{tx: t.tx, db: t.db}
	}
	return t.baseStationReceptionRepo
}

// EndPointSessions returns the endpoint session repository for this transaction
func (t *Transaction) EndPointSessions() interfaces.EndPointSessionRepository {
	if t.endpointSessionRepo == nil {
		t.endpointSessionRepo = &transactionalEndPointSessionRepository{
			tx:           t.tx,
			db:           t.db,
			keyEncryptor: t.keyEncryptor, // Enable lazy migration
			logger:       t.log,          // For migration logging
		}
	}
	return t.endpointSessionRepo
}

// EndPointKeys returns the endpoint key repository for this transaction
func (t *Transaction) EndPointKeys() interfaces.EndPointKeyRepository {
	if t.endpointKeyRepo == nil {
		t.endpointKeyRepo = &transactionalEndPointKeyRepository{tx: t.tx, db: t.db}
	}
	return t.endpointKeyRepo
}

// RoamingAgreements returns the roaming agreement repository for this transaction
func (t *Transaction) RoamingAgreements() interfaces.RoamingAgreementRepository {
	if t.roamingRepo == nil {
		t.log.Warn("Roaming agreement repository not supported in transactions")
	}
	return t.roamingRepo
}

// BaseStationSessions returns the base station session repository for this transaction
func (t *Transaction) BaseStationSessions() interfaces.BaseStationSessionRepository {
	if t.basestationSessionRepo == nil {
		// Falls back to non-transactional version (BSSCI sessions don't participate in transactions)
		t.log.Debug("BaseStationSession repository using non-transactional version within transaction")
		return t.db.BaseStationSessions()
	}
	return t.basestationSessionRepo
}

// PendingOperations returns the pending operation repository for this transaction
func (t *Transaction) PendingOperations() interfaces.PendingOperationRepository {
	if t.pendingOpRepo == nil {
		// Falls back to non-transactional version (pending ops don't participate in transactions)
		t.log.Debug("PendingOperation repository using non-transactional version within transaction")
		return t.db.PendingOperations()
	}
	return t.pendingOpRepo
}

// DLRXStatus returns DL RX status repository (currently non-transactional)
func (t *Transaction) DLRXStatus() interfaces.DLRXStatusRepository {
	// Returns non-transactional version (DL RX status doesn't participate in transactions)
	t.log.Debug("DLRXStatus repository using non-transactional version within transaction")
	return t.db.DLRXStatus()
}

// MIOTYMessages returns MIOTY message repository (currently non-transactional)
func (t *Transaction) MIOTYMessages() interfaces.MIOTYMessageRepository {
	// Returns non-transactional version
	t.log.Debug("MIOTY message repository using non-transactional version within transaction")
	return t.db.MIOTYMessages()
}

// MIOTYDownlinks returns MIOTY downlink repository with transaction support
// Field is cached (like other transactional repos) for reuse within transaction
func (t *Transaction) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository {
	if t.miotyDownlinkRepo == nil {
		t.miotyDownlinkRepo = &transactionalMIOTYDownlinkRepository{tx: t.tx, db: t.db}
	}
	return t.miotyDownlinkRepo
}

// MIOTYBaseStationStatus returns MIOTY base station status repository (currently non-transactional)
func (t *Transaction) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	// Returns non-transactional version
	t.log.Debug("MIOTY base station status repository using non-transactional version within transaction")
	return t.db.MIOTYBaseStationStatus()
}

// Users returns user repository for this transaction (non-transactional)
func (t *Transaction) Users() interfaces.UserRepository {
	if t.userRepo == nil {
		// Falls back to non-transactional version
		t.log.Debug("User repository using non-transactional version within transaction")
		return t.db.Users()
	}
	return t.userRepo
}

// APIKeys returns API key repository for this transaction (non-transactional)
func (t *Transaction) APIKeys() interfaces.APIKeyRepository {
	if t.apiKeyRepo == nil {
		// Falls back to non-transactional version
		t.log.Debug("API key repository using non-transactional version within transaction")
		return t.db.APIKeys()
	}
	return t.apiKeyRepo
}

// Integrations returns integration repository for this transaction (non-transactional)
func (t *Transaction) Integrations() interfaces.IntegrationRepository {
	if t.integrationRepo == nil {
		// Falls back to non-transactional version
		t.log.Debug("Integration repository using non-transactional version within transaction")
		return t.db.Integrations()
	}
	return t.integrationRepo
}

// Manufacturers returns manufacturer repository for this transaction
// Blueprint feature (Migration 000102)
func (t *Transaction) Manufacturers() interfaces.ManufacturerRepository {
	if t.manufacturerRepo == nil {
		t.manufacturerRepo = &transactionalManufacturerRepository{tx: t.tx, db: t.db}
	}
	return t.manufacturerRepo
}

// DeviceModels returns device model repository for this transaction
// Blueprint feature (Migration 000103)
func (t *Transaction) DeviceModels() interfaces.DeviceModelRepository {
	if t.deviceModelRepo == nil {
		t.deviceModelRepo = &transactionalDeviceModelRepository{tx: t.tx, db: t.db}
	}
	return t.deviceModelRepo
}

// Blueprints returns blueprint repository for this transaction
// Blueprint feature (Migration 000104)
func (t *Transaction) Blueprints() interfaces.BlueprintRepository {
	if t.blueprintRepo == nil {
		t.blueprintRepo = &transactionalBlueprintRepository{tx: t.tx, db: t.db}
	}
	return t.blueprintRepo
}

// Organizations returns organization repository for this transaction (non-transactional)
func (t *Transaction) Organizations() interfaces.OrganizationRepository {
	// Returns non-transactional version
	t.log.Debug("Organization repository using non-transactional version within transaction")
	return t.db.Organizations()
}

// GetSqlxDB returns the sqlx database connection for adapters
func (t *Transaction) GetSqlxDB() *sqlx.DB {
	return t.db.GetSqlxDB()
}

// SystemEvents returns the system event store (non-transactional)
func (t *Transaction) SystemEvents() interfaces.SystemEventStore {
	// Returns non-transactional version
	t.log.Debug("SystemEvents using non-transactional version within transaction")
	return t.db.SystemEvents()
}

// SCACISessions returns the SCACI session repository (non-transactional)
func (t *Transaction) SCACISessions() interfaces.SCACISessionRepository {
	// Returns non-transactional version
	t.log.Debug("SCACISessions using non-transactional version within transaction")
	return t.db.SCACISessions()
}

// SCACIOperations returns the SCACI operation repository (non-transactional)
func (t *Transaction) SCACIOperations() interfaces.SCACIOperationRepository {
	// Returns non-transactional version
	t.log.Debug("SCACIOperations using non-transactional version within transaction")
	return t.db.SCACIOperations()
}

// DownlinkQueueReader returns the downlink queue reader (non-transactional)
func (t *Transaction) DownlinkQueueReader() interfaces.DownlinkQueueReader {
	// Returns non-transactional version
	t.log.Debug("DownlinkQueueReader using non-transactional version within transaction")
	return t.db.DownlinkQueueReader()
}
