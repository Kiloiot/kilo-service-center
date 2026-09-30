package bssciservices

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// StationPropagator sends one base station an endpoint's attachment or detachment.
type StationPropagator interface {
	SendAttachPropagateBySessionID(ctx context.Context, sessionID string, endpoint *models.EndPoint) error
	SendDetachPropagate(sessionID string, endpointEUI uint64) error
}

// SystemEventRecorder persists system events emitted by BSSCI services.
type SystemEventRecorder interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// BaseStationStore reads base stations.
type BaseStationStore interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)
	GetByEUIGlobal(ctx context.Context, eui []byte) (*models.BaseStation, error)
}

// RegisteredStationStore reads a station across tenants and backfills the
// fingerprint and expiry of the certificate it is bound to.
type RegisteredStationStore interface {
	GetByEUIGlobal(ctx context.Context, eui []byte) (*models.BaseStation, error)
	UpdateTLSFingerprintIfBlank(ctx context.Context, tenantID, id int64, fingerprint string) (bool, error)
	UpdateTLSCertExpiryIfBlank(ctx context.Context, tenantID, id int64, expiresAt time.Time) (bool, error)
}

// DownlinkQueueOwnership resolves the tenant that owns a downlink queue.
type DownlinkQueueOwnership interface {
	GetTenantIDByQueueID(ctx context.Context, queueID uint64) (tenantID int64, err error)
}

// EndpointResolver locates the owning endpoint of an uplink, across tenants
// by EUI or within the serving tenant.
type EndpointResolver interface {
	Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error)
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error)
}

// EndpointAttachmentStore reads one tenant-scoped endpoint by EUI.
type EndpointAttachmentStore interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error)
}

// EndpointDirectory enumerates a tenant's endpoints for propagation.
type EndpointDirectory interface {
	GetByID(ctx context.Context, id int64, tenantID int64) (*models.EndPoint, error)
	GetByTenant(ctx context.Context, tenantID int64) ([]*models.EndPoint, error)
	GetByAttachmentChangedSince(ctx context.Context, tenantID int64, status string, since *time.Time) ([]*models.EndPoint, error)
}

// PendingOperationPurger discards the pending operations of a retired session.
type PendingOperationPurger interface {
	DeleteBySession(ctx context.Context, sessionID int64) (int64, error)
}

// BaseStationSessionStore persists and resumes BSSCI base station sessions.
type BaseStationSessionStore interface {
	CreateSession(ctx context.Context, req *models.BaseStationSessionCreateRequest) (*models.BaseStationSession, error)
	GetActiveSessionByBaseStation(ctx context.Context, tenantID, baseStationID int64) (*models.BaseStationSession, error)
	GetSessionByScUUID(ctx context.Context, tenantID int64, snScUUID [16]byte) (*models.BaseStationSession, error)
	UpdateSession(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) error
	UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, bsOpId, scOpId int64) error
	UpdateEncoding(ctx context.Context, tenantID, sessionID int64, encoding string) error
	TerminateSession(ctx context.Context, tenantID, sessionID int64) error
	MarkDisconnected(ctx context.Context, tenantID, sessionID int64, connectionID string, endedAt time.Time) error
	ActivateSessionIfResumable(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) (bool, error)
	FindResumableSession(ctx context.Context, tenantID int64, bsEUI []byte, snBsUUID [16]byte) (*models.BaseStationSession, error)
	TerminateResumableSessions(ctx context.Context, tenantID, baseStationID int64) ([]int64, error)
}

// PendingOperationStore persists and discards BSSCI pending operations.
type PendingOperationStore interface {
	Create(ctx context.Context, req *models.PendingOperationRequest) error
	CreateBatch(ctx context.Context, reqs []*models.PendingOperationRequest) error
	UpdateMetadata(ctx context.Context, sessionID int64, operationID int64, metadata json.RawMessage) error
	DeleteBySessionAndOperation(ctx context.Context, sessionID int64, operationID int64) error
	DeleteBySession(ctx context.Context, sessionID int64) (int64, error)
	GetBySession(ctx context.Context, sessionID int64) ([]*models.PendingOperation, error)
}
