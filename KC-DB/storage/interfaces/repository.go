// Package interfaces defines the storage interfaces for KiloCenter
package interfaces

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/models"
)

// EndpointRepository defines the interface for endpoint storage operations
type EndpointRepository interface {
	// Create creates a new endpoint
	Create(ctx context.Context, endpoint *models.EndPoint) error

	// Get retrieves an endpoint by EUI
	Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error)

	// GetByEUI retrieves an endpoint by EUI for a specific tenant
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error)

	// GetByTenant retrieves all endpoints for a tenant
	GetByTenant(ctx context.Context, tenantID int64) ([]*models.EndPoint, error)

	// CountByTenant returns the total count of endpoints for a tenant
	CountByTenant(ctx context.Context, tenantID int64) (int64, error)

	// ListByTenantPaginated retrieves paginated endpoints for a tenant with LIMIT/OFFSET
	ListByTenantPaginated(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error)

	// GetByID retrieves an endpoint by ID with tenant isolation
	GetByID(ctx context.Context, id int64, tenantID int64) (*models.EndPoint, error)

	// Update updates an endpoint
	Update(ctx context.Context, endpoint *models.EndPoint) error

	// UpdateWithEUI atomically cascades an EUI change and updates all endpoint fields
	UpdateWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error)

	// CheckEUIUnique verifies no endpoint with the given EUI exists (global uniqueness)
	CheckEUIUnique(ctx context.Context, eui []byte) error

	// DeleteByTenant deletes an endpoint by EUI with tenant isolation
	// Returns interfaces.ErrRecordNotFound when 0 rows affected
	DeleteByTenant(ctx context.Context, tenantID int64, eui []byte) error

	// UpdateLastSeen updates the last seen timestamp and frame count with tenant isolation
	UpdateLastSeen(ctx context.Context, tenantID int64, eui models.EUI, frameCount uint32) error

	// UpdateRadioMetrics updates the radio metrics from attach or uplink operations
	UpdateRadioMetrics(ctx context.Context, tenantID int64, eui models.EUI, snr, rssi, eqSnr float64, rxTime, rxDuration int64, profile string) error

	// UpdateRadioMetricsSelective updates radio metrics with optional field support
	UpdateRadioMetricsSelective(ctx context.Context, tenantID int64, eui models.EUI, update RadioMetricsUpdate) error

	// UpdateDetachMetrics updates detach-specific telemetry (last_detach_*) plus shared radio metrics.
	// Does NOT update attach columns (last_attach_rx_time, last_attach_rx_duration).
	UpdateDetachMetrics(ctx context.Context, tenantID int64, eui models.EUI, update DetachMetricsUpdate) error

	// UpdateFields updates endpoint fields using a map of field names to values
	UpdateFields(ctx context.Context, tenantID int64, endpointID int64, updates map[string]interface{}) error

	// StreamAllForPropagation fetches propagated endpoints across all tenants for cross-tenant roaming.
	// Returns endpoints ordered by propagated_at NULLS FIRST, id for stable pagination.
	// Each endpoint includes tenant_id for owner resolution in reconciliation.
	StreamAllForPropagation(ctx context.Context, cursorID int64, limit int) ([]*models.EndPoint, error)

	// HasEndpointsSince checks if any propagated endpoints exist after the given timestamp.
	// Used by reconciliation decision logic to detect new endpoints since last sync.
	HasEndpointsSince(ctx context.Context, since time.Time) (bool, error)

	// GetEndpointWithKeysForDetachValidation retrieves an endpoint with all crypto material for detach signature validation
	// Returns complete endpoint record including Sign, NwkSnKey, and PresharedKey fields
	GetEndpointWithKeysForDetachValidation(ctx context.Context, eui models.EUI) (*models.EndPoint, error)

	// GetPreferredBsEui returns the last_attached_bs_eui for an endpoint
	// Returns (nil, false, nil) if endpoint not found or column is NULL
	// Used by UL Data Transmit (SCACI §3.9.1) for "Service Center preferred BS" selection
	GetPreferredBsEui(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error)
}

// RadioMetricsUpdate contains radio metrics with optional field support.
// Nil pointers indicate "do not update this field" (preserve DB value).
type RadioMetricsUpdate struct {
	SNR        float64
	RSSI       float64
	EqSNR      float64
	RxTime     int64
	RxDuration *int64  // nil = preserve existing value
	Profile    *string // nil = preserve existing value
}

// DetachMetricsUpdate contains detach-specific metrics plus shared radio metrics.
// Updates detach columns (last_detach_*) without touching attach columns (last_attach_rx_time, last_attach_rx_duration).
type DetachMetricsUpdate struct {
	RxTime    int64
	PacketCnt uint32
	Signature []byte
	SNR       float64
	RSSI      float64
	EqSNR     *float64 // nil = preserve existing value
	Profile   *string  // nil = preserve existing value
}

// MessageRepository defines the interface for message storage operations
type MessageRepository interface {
	// Create stores a new message
	Create(ctx context.Context, message *models.Message) error

	// GetByEndPoint retrieves messages for an endpoint within a time range
	GetByEndPoint(ctx context.Context, epEUI models.EUI, from, to time.Time) ([]*models.Message, error)

	// GetByBaseStation retrieves messages for a basestation within a time range
	GetByBaseStation(ctx context.Context, bsEUI models.EUI, from, to time.Time) ([]*models.Message, error)

	// GetLatest retrieves the latest N messages
	GetLatest(ctx context.Context, limit int) ([]*models.Message, error)

	// DeleteOlderThan deletes messages older than the specified time
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// DownlinkQueueRepository defines operations for downlink message queue
type DownlinkQueueRepository interface {
	// Enqueue adds a new downlink message to the queue
	Enqueue(ctx context.Context, message *models.DownlinkMessage) error

	// GetPending retrieves pending messages for a device
	GetPending(ctx context.Context, deviceEUI models.EUI) ([]*models.DownlinkMessage, error)

	// GetNext retrieves the next message to transmit for a device
	GetNext(ctx context.Context, deviceEUI models.EUI) (*models.DownlinkMessage, error)

	// UpdateStatus updates the status of a downlink message
	UpdateStatus(ctx context.Context, id int64, status string, gatewayEUI *models.EUI) error

	// MarkTransmitted marks a message as transmitted
	MarkTransmitted(ctx context.Context, id int64, gatewayEUI models.EUI) error

	// MarkConfirmed marks a message as confirmed
	MarkConfirmed(ctx context.Context, id int64) error

	// MarkFailed marks a message as failed with a reason
	MarkFailed(ctx context.Context, id int64, reason string) error

	// ExpireOldMessages marks old messages as expired
	ExpireOldMessages(ctx context.Context) (int64, error)

	// GetByTenant retrieves all downlink messages for a tenant
	GetByTenant(ctx context.Context, tenantID int64, status string) ([]*models.DownlinkMessage, error)
}

// BaseStationReceptionRepository defines operations for basestation reception records
type BaseStationReceptionRepository interface {
	// Create creates a new basestation reception record
	Create(ctx context.Context, reception *models.BaseStationReception) error

	// GetByMessage retrieves all basestation receptions for a specific message
	GetByMessage(ctx context.Context, messageID string) ([]*models.BaseStationReception, error)

	// GetByBaseStation retrieves basestation receptions for a specific basestation
	GetByBaseStation(ctx context.Context, baseStationID string, limit int, offset int) ([]*models.BaseStationReception, error)

	// GetBestReception retrieves the best reception (highest RSSI) for a message
	GetBestReception(ctx context.Context, messageID string) (*models.BaseStationReception, error)

	// GetReceptionStats gets reception statistics for a basestation
	GetReceptionStats(ctx context.Context, baseStationID string, since time.Time) (*models.BaseStationReceptionStats, error)

	// DeleteByMessage deletes all gateway receptions for a message
	DeleteByMessage(ctx context.Context, messageID string) error
}

// EndPointSessionRepository defines operations for endpoint session management
type EndPointSessionRepository interface {
	// Create creates a new endpoint session
	Create(ctx context.Context, session *models.EndPointSession) error

	// GetActive retrieves the active session for an endpoint
	GetActive(ctx context.Context, endpointID string) (*models.EndPointSession, error)

	// GetByID retrieves a session by ID
	GetByID(ctx context.Context, id string) (*models.EndPointSession, error)

	// Update updates an endpoint session
	Update(ctx context.Context, session *models.EndPointSession) error

	// UpdateActivity updates the last activity timestamp and counters
	UpdateActivity(ctx context.Context, sessionID string, isUplink bool) error

	// Terminate terminates a session
	Terminate(ctx context.Context, sessionID string) error

	// ExpireOldSessions marks old sessions as expired
	ExpireOldSessions(ctx context.Context, maxAge time.Duration) (int64, error)

	// GetByEndPoint retrieves all sessions for an endpoint
	GetByEndPoint(ctx context.Context, endpointID string, limit int, offset int) ([]*models.EndPointSession, error)

	// GetStats retrieves session statistics for an endpoint
	GetStats(ctx context.Context, endpointID string) (*models.EndPointSessionStats, error)

	// UpdateSessionKey updates only the session_key field for lazy migration
	// Includes tenant scoping to prevent cross-tenant session_id collisions
	UpdateSessionKey(ctx context.Context, tenantID int64, sessionID string, encryptedKey []byte) error
}

// EndPointKeyRepository defines operations for endpoint key management
type EndPointKeyRepository interface {
	// Create creates a new endpoint key
	Create(ctx context.Context, key *models.EndPointKey) error

	// GetActive retrieves the active key of a specific type for an endpoint
	GetActive(ctx context.Context, endpointID string, keyType string) (*models.EndPointKey, error)

	// GetByID retrieves a key by ID
	GetByID(ctx context.Context, id string) (*models.EndPointKey, error)

	// GetByEndPoint retrieves all keys for an endpoint
	GetByEndPoint(ctx context.Context, endpointID string) ([]*models.EndPointKey, error)

	// Deactivate deactivates a key
	Deactivate(ctx context.Context, id string) error

	// IncrementUsage increments the usage count and updates last used timestamp
	IncrementUsage(ctx context.Context, id string) error

	// RotateKey creates a new key version and deactivates the old one
	RotateKey(ctx context.Context, oldKeyID string, newKey *models.EndPointKey, reason string) error

	// GetStats retrieves key statistics for an endpoint
	GetStats(ctx context.Context, endpointID string) (*models.EndPointKeyStats, error)

	// ExpireKeys marks keys as expired based on their validity period
	ExpireKeys(ctx context.Context) (int64, error)
}

// RoamingAgreementRepository defines operations for roaming agreements
type RoamingAgreementRepository interface {
	// Create creates a new roaming agreement
	Create(ctx context.Context, agreement *models.RoamingAgreement) error

	// Get retrieves a roaming agreement by ID
	Get(ctx context.Context, id int64) (*models.RoamingAgreement, error)

	// GetByPartner retrieves an agreement by partner network ID
	GetByPartner(ctx context.Context, tenantID int64, partnerNetworkID string) (*models.RoamingAgreement, error)

	// GetActive retrieves all active agreements for a tenant
	GetActive(ctx context.Context, tenantID int64) ([]*models.RoamingAgreement, error)

	// Update updates an existing agreement
	Update(ctx context.Context, agreement *models.RoamingAgreement) error

	// Deactivate deactivates an agreement
	Deactivate(ctx context.Context, id int64) error
}

// PendingOperationRepository manages BSSCI pending operations
type PendingOperationRepository interface {
	// Create inserts or updates a pending operation (UPSERT pattern)
	Create(ctx context.Context, req *PendingOperationRequest) error

	// UpdateMetadata updates only the metadata field
	UpdateMetadata(ctx context.Context, sessionID int64, operationID int64, metadata json.RawMessage) error

	// DeleteBySessionAndOperation removes by compound key
	DeleteBySessionAndOperation(ctx context.Context, sessionID int64, operationID int64) error

	// DeleteByOperation removes by operation ID only
	DeleteByOperation(ctx context.Context, operationID int64) error

	// DeleteBySession removes all pending operations for a session
	// Used during session termination per BSSCI §3 "discarding state" requirement
	DeleteBySession(ctx context.Context, sessionID int64) (int64, error)

	// GetBySession retrieves all pending operations for a session
	GetBySession(ctx context.Context, sessionID int64) ([]*PendingOperation, error)
}

// PendingOperationRequest for creating/updating operations
type PendingOperationRequest struct {
	SessionID     int64           // basestation_session_id (FK)
	OperationID   int64           // MIOTY operation ID
	OperationType string          // Operation type (attPrp, detPrp, dlDataQueue, etc.)
	EndpointEUI   []byte          // Endpoint EUI (nullable)
	OperationData json.RawMessage // Pre-marshaled operation message (JSONB)
	Metadata      json.RawMessage // Pre-marshaled metadata (JSONB, nullable)
}

// PendingOperation returned from queries
type PendingOperation struct {
	ID            int64           `db:"id"`
	SessionID     int64           `db:"basestation_session_id"`
	OperationID   int64           `db:"operation_id"`
	OperationType string          `db:"operation_type"`
	EndpointEUI   []byte          `db:"endpoint_eui"`
	OperationData json.RawMessage `db:"operation_data"`
	Metadata      json.RawMessage `db:"metadata"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
}

// Storage defines the main storage interface combining all repositories
type Storage interface {
	// Repositories
	EndPoints() EndpointRepository
	DownlinkQueue() DownlinkQueueRepository
	BaseStationReceptions() BaseStationReceptionRepository
	EndPointSessions() EndPointSessionRepository
	EndPointKeys() EndPointKeyRepository
	RoamingAgreements() RoamingAgreementRepository
	BaseStations() BaseStationRepository
	BaseStationSessions() BaseStationSessionRepository
	DLRXStatus() DLRXStatusRepository
	PendingOperations() PendingOperationRepository

	// MIOTY-specific repositories
	MIOTYMessages() MIOTYMessageRepository
	MIOTYDownlinks() MIOTYDownlinkRepository
	MIOTYBaseStationStatus() MIOTYBaseStationStatusRepository

	// User management
	Users() UserRepository

	// Organization management
	Organizations() OrganizationRepository

	// API key management
	APIKeys() APIKeyRepository

	// Integration management
	Integrations() IntegrationRepository

	// Blueprint device catalog (Migration 000102-000104)
	Manufacturers() ManufacturerRepository
	DeviceModels() DeviceModelRepository
	Blueprints() BlueprintRepository

	// GetSqlxDB returns the sqlx database connection for adapters
	GetSqlxDB() *sqlx.DB
	// SystemEvents returns the system event store
	SystemEvents() SystemEventStore
	// SCACISessions returns the SCACI session repository
	SCACISessions() SCACISessionRepository
	// SCACIOperations returns the SCACI operation repository
	SCACIOperations() SCACIOperationRepository
	// DownlinkQueueReader returns the downlink queue reader
	DownlinkQueueReader() DownlinkQueueReader

	// Transaction support
	BeginTx(ctx context.Context) (Transaction, error)

	// Health check
	Ping(ctx context.Context) error

	// Close closes all connections
	Close() error
}

// Transaction defines a database transaction interface
type Transaction interface {
	// Repositories within transaction
	EndPoints() EndpointRepository
	DownlinkQueue() DownlinkQueueRepository
	BaseStationReceptions() BaseStationReceptionRepository
	EndPointSessions() EndPointSessionRepository
	EndPointKeys() EndPointKeyRepository
	RoamingAgreements() RoamingAgreementRepository
	BaseStations() BaseStationRepository
	BaseStationSessions() BaseStationSessionRepository
	PendingOperations() PendingOperationRepository
	DLRXStatus() DLRXStatusRepository

	// MIOTY-specific repositories
	MIOTYMessages() MIOTYMessageRepository
	MIOTYDownlinks() MIOTYDownlinkRepository
	MIOTYBaseStationStatus() MIOTYBaseStationStatusRepository

	// User management
	Users() UserRepository

	// Organization management
	Organizations() OrganizationRepository

	// API key management
	APIKeys() APIKeyRepository

	// Integration management
	Integrations() IntegrationRepository

	// Blueprint device catalog (Migration 000102-000104)
	Manufacturers() ManufacturerRepository
	DeviceModels() DeviceModelRepository
	Blueprints() BlueprintRepository

	// Additional accessors (mirrors Storage interface)
	GetSqlxDB() *sqlx.DB
	SystemEvents() SystemEventStore
	SCACISessions() SCACISessionRepository
	SCACIOperations() SCACIOperationRepository
	DownlinkQueueReader() DownlinkQueueReader

	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error
}
