// Package storage provides the storage interface and common types
package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
)

// Common errors
var (
	ErrNotFound              = errors.New("not found")
	ErrAlreadyExists         = errors.New("already exists")
	ErrInvalidInput          = errors.New("invalid input")
	ErrDownlinkNotFound      = errors.New("downlink message not found")
	ErrStorageNotAvailable   = errors.New("storage not available")
	ErrOperationNotSupported = errors.New("operation not supported")
	ErrInvalidTenantID       = errors.New("tenant_id must be positive")

	// Quota enforcement errors
	ErrBaseStationsNotAllowed   = errors.New("organization cannot have base stations")
	ErrBaseStationQuotaExceeded = errors.New("base station quota exceeded")
	ErrEndpointQuotaExceeded    = errors.New("endpoint quota exceeded")

	// Database constraint errors - Blueprint feature (Migration 000102-000104)
	ErrDuplicateKey        = errors.New("duplicate key violation")
	ErrForeignKeyViolation = errors.New("foreign key violation")

	// Endpoint key CHECK constraint violations (migration 001_initial_schema)
	ErrNwkKeyLength   = errors.New("nwk_key length violation")
	ErrAppKeyLength   = errors.New("app_key length violation")
	ErrCheckViolation = errors.New("check constraint violation")
)

// Storage defines the main storage interface
type Storage interface {
	// EndPoint operations
	CreateEndPoint(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error)
	GetEndPoint(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error)
	UpdateEndPoint(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error)
	UpdateEndPointWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error)
	CheckEndPointEUIUnique(ctx context.Context, eui []byte) error
	DeleteEndPoint(ctx context.Context, eui []byte, tenantID int64) error
	ListEndPoints(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error)

	// BaseStation operations (uses canonical models.BaseStation)
	CreateBaseStation(ctx context.Context, baseStation *models.BaseStation) (*models.BaseStation, error)
	GetBaseStation(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error)
	UpdateBaseStation(ctx context.Context, baseStation *models.BaseStation) (*models.BaseStation, error)
	UpdateBaseStationEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error)
	DeleteBaseStation(ctx context.Context, eui []byte, tenantID int64) error
	ListBaseStations(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error)
	ListAllBaseStationLocations(ctx context.Context) ([]*models.BaseStation, error)

	// Message operations removed - use MessageRepository methods from mioty package instead:
	// - CreateULDataMessage, GetULDataMessage, ListULDataMessages

	// Downlink operations
	// orgID filters by organization; nil = no org filter (backward compatible)
	EnqueueDownlink(ctx context.Context, downlink *DownlinkMessage) (*DownlinkMessage, error)
	GetDownlinkQueue(ctx context.Context, endpointEUI string, tenantID string) ([]*DownlinkMessage, error)
	GetDownlinkResults(ctx context.Context, endpointEUI string, tenantID string, orgID *uuid.UUID, statusFilter string, timeFrom, timeTo *time.Time, limit, offset int) ([]*DownlinkMessage, int, error)
	UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error
	UpdateDownlinkBaseStation(ctx context.Context, queId uint64, tenantID string, bsEUI uint64) error
	GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*DownlinkMessage, error)
	GetDownlinkByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) (*DownlinkMessage, error)
	RevokeDownlink(ctx context.Context, queId int64, tenantID string) error
	UpdateDownlinkResult(ctx context.Context, queId int64, result string, txTime *int64, packetCnt *uint32, bsEUI []byte, epEUI []byte, tenantID string, orgID *uuid.UUID) error

	// DL RX Status operations (BSSCI §3.15-3.16)
	CreateDLRXStatus(ctx context.Context, status *mioty.DLRXStatus) error
	GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error)
	GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error)

	// DL RX Status Query Tracking (BSSCI §5.15 audit trail)
	// orgUUID is nullable for backward compatibility (nil if not available from context)
	// MarkDLRXStatusReceived correlates by tenant+endpoint only (BSSCI §5.15)
	// bsEui and bsOpID stored for audit, NOT used in WHERE clause (different namespaces)
	CreateDLRXStatusQuery(ctx context.Context, tenantID int64, orgUUID *uuid.UUID, epEui, bsEui []byte, opId int64) error
	MarkDLRXStatusReceived(ctx context.Context, tenantID int64, epEui []byte, bsEui []byte, bsOpID int64) (found bool, err error)
	ExpireDLRXStatusQuery(ctx context.Context, cutoff time.Time) (int64, error)

	// DL RX Status Query Telemetry (BSSCI §5.15 query history and stats)
	GetDLRXStatusQueryHistory(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatusQuery, int, error)
	GetDLRXStatusQueryStats(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (pending, received, timeout int64, err error)

	// Helper methods
	GetEndpointBaseStation(ctx context.Context, epEui string, tenantID string) (bsEui string, err error)

	// Base Station Message Statistics
	GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte,
		startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error)
	GetBaseStationEndpointCounts(ctx context.Context, tenantID int64, bsEui []byte,
		startTime, endTime *time.Time) (map[string]int64, error)
	GetBaseStationLastSeen(ctx context.Context, tenantID int64, bsEui []byte) (*time.Time, error)

	// Roaming Support
	GetEndpointOwner(ctx context.Context, epEui []byte) (ownerTenantID int64, err error)
	GetEndpointWithOwnership(ctx context.Context, epEui []byte, servingTenantID int64) (*models.EndPoint, error)
	IsRoamingEnabled(ctx context.Context, tenantID int64) (bool, error)
	AreTenantsPartners(ctx context.Context, tenant1, tenant2 int64) (bool, error)
	RecordRoamingEvent(ctx context.Context, event *models.RoamingEvent) error
	GetRoamingStatistics(ctx context.Context, tenantID int64) (*models.RoamingStatistics, error)
	UpdateEndpointRoamingStatus(ctx context.Context, epEui []byte, servingTenantID int64) error
	GetRoamingEndpointsInSession(ctx context.Context, sessionID int64) ([]models.RoamingEndpointInfo, error)
	AddRoamingEndpointToSession(ctx context.Context, sessionID int64, epEui string, ownerTenantID int64) error
	RemoveRoamingEndpointFromSession(ctx context.Context, sessionID int64, epEui string) error
	GetTenantRoamingConfig(ctx context.Context, tenantID int64) (*models.TenantRoamingConfig, error)
	UpdateTenantRoamingConfig(ctx context.Context, tenantID int64, config *models.TenantRoamingConfig) error

	// Health check
	Ping(ctx context.Context) error

	// Close closes the storage connection
	Close() error
}

// Use canonical models.EndPoint and models.BaseStation from KC-DB/storage/models/ instead.

// Message struct removed - use mioty.ULDataMessage from KC-DB/storage/mioty/types.go instead

// DownlinkMessage represents a downlink message
type DownlinkMessage struct {
	ID                    int64
	EPEUI                 string // End Point EUI
	TenantID              string
	OrganizationID        *uuid.UUID // Organization UUID from SCACI session for audit trail
	Payload               []byte
	Priority              float32 // MIOTY BSSCI §3.12.1: single precision float (>= 0.0)
	Status                string
	Attempts              int32
	MaxAttempts           int32
	CreatedAt             time.Time
	ScheduledAt           *time.Time
	SentAt                *time.Time
	QueID                 int64   // MIOTY queue ID for reference (BSSCI 3.12.1)
	CntDepend             bool    // True if userData is counter dependent
	PacketCntArray        []int64 // End Point packet counters for which userData is valid
	Format                uint8   // User data format identifier (8 bit)
	ResponseExp           bool    // True to request End Point response
	ResponsePrio          bool    // True to request priority End Point response
	DlWindReq             bool    // True to request further End Point DL window
	ExpOnly               bool    // True to send downlink only if End Point expects response
	DlRxStatQry           bool    // SCACI §3.10.1: True to query DL RX status from endpoint
	Result                string  // "sent", "expired", "invalid" per MIOTY spec
	BsEui                 uint64  // Base Station EUI that owns this queue item
	TxTime                int64   // Unix UTC time of transmission (BSSCI §3.14.1)
	TransmissionPacketCnt int64   // End Point packet counter when transmitted (BSSCI §3.14.1)
	TxBSEUI               string  // EUI of transmitting Base Station
	UpdatedAt             time.Time
	UserData              [][]byte // Canonical MIOTY DLDataQueue.UserData field for counter-dependent messages
}
