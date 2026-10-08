package scaciservices

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SessionResumeReader reads what a resume looks up about the application
// center's stored session (SCACI §1, §3.3.1).
type SessionResumeReader interface {
	GetSessionByAcUUID(ctx context.Context, ac models.SCACIApplicationCenter, snAcUuid [16]byte) (*models.SCACISession, error)                  //nolint:revive // snAcUuid matches SCACI spec naming
	CheckSessionResumable(ctx context.Context, ac models.SCACIApplicationCenter, snAcUuid [16]byte) (*models.SCACISessionResumptionInfo, error) //nolint:revive // snAcUuid matches SCACI spec naming
}

// SessionLifecycleRows updates a session's row as its connection moves: the
// resume, the loss of the connection and the end of its resumability (SCACI
// §1).
type SessionLifecycleRows interface {
	ResumeSession(ctx context.Context, tenantID, sessionID int64, req *models.SCACISessionResume) error
	MarkSessionDisconnected(ctx context.Context, tenantID, sessionID int64) error
	TerminateSession(ctx context.Context, tenantID, sessionID int64) error
}

// SessionActivityRows updates a session's heartbeat (SCACI §3.4) and its
// operation ID counters (§3.2).
type SessionActivityRows interface {
	UpdateHeartbeat(ctx context.Context, tenantID, sessionID int64) error
	UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, acOpId, scOpId int64) error
}

// SessionCreationTx is what the creation of a fresh session does in one
// transaction: retire the application center's earlier sessions, then insert
// the new one (SCACI §1: a new session discards the previous one's state).
type SessionCreationTx interface {
	RetirePriorSessions(ctx context.Context, ac models.SCACIApplicationCenter) error
	CreateSession(ctx context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error)
}

// SessionCreationRunner runs a session creation in one transaction; storage
// owns begin, commit and rollback.
type SessionCreationRunner interface {
	Run(ctx context.Context, fn func(SessionCreationTx) error) error
}

// SCACIOperationStore records SCACI operations and their error dispositions;
// the ulData operation of one stored uplink is recorded once per session
// (SCACI §3.2).
type SCACIOperationStore interface {
	RecordOperation(ctx context.Context, req *models.SCACIOperationRequest) (*models.SCACIOperation, error)
	EnsureUplinkOperation(ctx context.Context, req *models.SCACIOperationRequest, sourceMessageID string) (*models.SCACIOperation, bool, error)
	UpdateOperationStateWithError(ctx context.Context, sessionID int64, opId int64,
		state models.OperationState, errorCode int, errorToken string, errorMessage string,
		responseData map[string]interface{}) error
	CompleteFailedOperation(ctx context.Context, sessionID int64, opId int64, responseData map[string]interface{}) error
}

// ResumableSessionLister lists the sessions an Application Center may still
// resume (SCACI §1).
type ResumableSessionLister interface {
	ListResumableSessions(ctx context.Context) ([]*models.SCACISession, error)
}

// HeldSessionRows persists what the holder changes on a held session: its
// operation ID counters (SCACI §3.2) and the end of its resumability (§1).
type HeldSessionRows interface {
	PersistOpIDs(ctx context.Context, session *scaci.Session, ids scaci.OpIDPair) error
	EndResumability(ctx context.Context, session *scaci.Session) error
}

// PendingOperationCounter counts the service center operations of a session
// still awaiting completion: what its resume reissues (SCACI §1).
type PendingOperationCounter interface {
	CountPendingSCOperations(ctx context.Context, sessionID int64) (int, error)
}

// AttachmentDecider records an application center's attach or detach of an
// endpoint and announces a change once (SCACI §3.13).
type AttachmentDecider interface {
	Decide(ctx context.Context, decision bssci.AttachmentDecision) (bool, error)
}

// EndpointStore reads and registers endpoints for SCACI operations.
type EndpointStore interface {
	Create(ctx context.Context, endpoint *models.EndPoint) error
	Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error)
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error)
	EndpointRegistrationUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointRegistrationParams) error
	GetPreferredBsEui(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error)
}

// BaseStationStore reads base stations for SCACI status and preference lookups.
type BaseStationStore interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)
	List(ctx context.Context, filter *models.BaseStationFilter) ([]*models.BaseStation, int64, error)
}
