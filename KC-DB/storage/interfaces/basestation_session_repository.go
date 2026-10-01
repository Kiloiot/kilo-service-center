package interfaces

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// BaseStationSessionRepository defines the interface for Base Station session management
// This implements MIOTY BSSCI v1.0.0 Section 3.3 session management requirements
type BaseStationSessionRepository interface {
	// CreateSession creates a new Base Station session per MIOTY BSSCI 3.3
	CreateSession(ctx context.Context, req *models.BaseStationSessionCreateRequest) (*models.BaseStationSession, error)

	// GetSessionByID retrieves a session by ID
	GetSessionByID(ctx context.Context, tenantID, sessionID int64) (*models.BaseStationSession, error)

	// GetActiveSessionByBaseStation retrieves the active session for a Base Station
	GetActiveSessionByBaseStation(ctx context.Context, tenantID, baseStationID int64) (*models.BaseStationSession, error)

	// GetSessionByScUUID retrieves a session by Service Center session UUID
	GetSessionByScUUID(ctx context.Context, tenantID int64, snScUUID [16]byte) (*models.BaseStationSession, error)

	// UpdateSession updates session fields (operation IDs, status, timing)
	UpdateSession(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) error

	// UpdateOperationIDs updates both Base Station and Service Center operation IDs atomically
	UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, bsOpId, scOpId int64) error

	// UpdateEncoding updates the message encoding for a session (json or msgpack)
	// This is called when encoding is negotiated on first message per BSSCI Section 1
	UpdateEncoding(ctx context.Context, tenantID, sessionID int64, encoding string) error

	// TerminateSession marks a session as terminated
	TerminateSession(ctx context.Context, tenantID, sessionID int64) error

	// ListSessions retrieves sessions based on filter criteria
	ListSessions(ctx context.Context, filter *models.BaseStationSessionFilter) ([]*models.BaseStationSession, int64, error)

	// GetSessionStatistics retrieves session statistics
	GetSessionStatistics(ctx context.Context, tenantID int64) (*models.SessionStatistics, error)

	// CheckSessionResumable determines if a session can be resumed per MIOTY spec
	CheckSessionResumable(ctx context.Context, tenantID int64, snBsUuid [16]byte, bsOpId int64) (*models.SessionResumptionInfo, error) //nolint:revive // snBsUuid, bsOpId match BSSCI spec naming

	// MarkDisconnected marks a session disconnected and resumable, but only
	// while it is still active and the stored connection ID matches - a newer
	// connection that replaced this one, or a session already terminated, is
	// left untouched (zero rows is not an error)
	MarkDisconnected(ctx context.Context, tenantID, sessionID int64, connectionID string, endedAt time.Time) error

	// ActivateSessionIfResumable applies the resume activation while the row is
	// still disconnected and resumable, reporting whether it was claimed: a
	// second connection resuming the same row matches zero rows and gets false
	ActivateSessionIfResumable(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) (bool, error)

	// FindResumableSession finds the resumable session for a base station:
	// scoped by tenant, base station EUI, and snBsUuid, with
	// status=disconnected and can_resume=true (BSSCI §5.3.1)
	FindResumableSession(ctx context.Context, tenantID int64, bsEUI []byte, snBsUUID [16]byte) (*models.BaseStationSession, error)

	// TerminateResumableSessions retires a base station's leftover resumable sessions
	// and returns their ids so their pending operations can be discarded too; zero
	// matches is not an error
	TerminateResumableSessions(ctx context.Context, tenantID, baseStationID int64) ([]int64, error)
}
