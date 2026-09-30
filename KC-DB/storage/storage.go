// Package storage provides the storage interface and common types
package storage

import (
	"errors"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// Common errors
var (
	ErrNotFound = errors.New("not found")

	// ErrRecordNotFound indicates a query returned no matching rows. It is a
	// distinct per-layer sentinel from ErrNotFound: repository implementations
	// wrap this sentinel for errors.Is checks, while ErrNotFound is the
	// storage-interface not-found result. The two are intentionally not merged.
	ErrRecordNotFound = errors.New("record not found")

	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	// ErrPacketCounterCollision reports a second uplink inside the duplicate
	// window with the same packet counter but different content.
	ErrPacketCounterCollision = errors.New("packet counter collision")
	ErrDownlinkNotFound       = errors.New("downlink message not found")
	ErrDownlinkNotPending     = errors.New("downlink is no longer pending")
	// ErrDownlinkFinished reports a downlink that already reached its final
	// state, so a later answer about it changes nothing.
	ErrDownlinkFinished = errors.New("downlink already finished")
	// ErrDownlinkQueueIDTaken reports that the service center's own queue id
	// is already assigned to another downlink; a fresh id can be drawn.
	ErrDownlinkQueueIDTaken = errors.New("downlink queue id already assigned")
	ErrInvalidTenantID      = errors.New("tenant_id must be positive")
	// ErrInstallationOnboarded reports an onboarding of a CE installation
	// that has already completed onboarding.
	ErrInstallationOnboarded = errors.New("installation already onboarded")

	// Database constraint errors - Blueprint feature (Migration 000102-000104)
	ErrDuplicateKey        = errors.New("duplicate key violation")
	ErrForeignKeyViolation = errors.New("foreign key violation")

	// Endpoint key CHECK constraint violations (migration 001_initial_schema)
	ErrNwkKeyLength   = errors.New("nwk_key length violation")
	ErrAppKeyLength   = errors.New("app_key length violation")
	ErrCheckViolation = errors.New("check constraint violation")
)

// Use canonical models.EndPoint and models.BaseStation from KC-DB/storage/models/ instead.

// Message struct removed - use mioty.ULDataMessage from KC-DB/storage/mioty/types.go instead

// MaxDownlinkRefBytes bounds the ref an MQTT downlink command carries; the
// downlink_queue.ref CHECK constraint (migration 000187) holds the same bound.
const MaxDownlinkRefBytes = 128

// DownlinkMessage represents a downlink message
type DownlinkMessage struct {
	ID                    int64
	EPEUI                 string // End Point EUI
	TenantID              string
	OrganizationID        *uuid.UUID // Organization UUID from SCACI session for audit trail
	Payload               []byte
	Priority              float32 // MIOTY BSSCI §3.12.1: single precision float (>= 0.0)
	Status                mioty.DLQueueStatus
	Attempts              int32
	MaxAttempts           int32
	CreatedAt             time.Time
	ScheduledAt           *time.Time
	SentAt                *time.Time
	QueID                 int64   // Service center queue ID, the one base stations see (BSSCI 3.12.1)
	ACQueID               *uint64 // Queue ID the Application Center assigned (SCACI 3.10.1); nil when enqueued without one
	ACEUI                 *uint64 // EUI of the Application Center that queued it, told its result (SCACI 3.12); nil when none did
	Ref                   string  // Correlation ref of the MQTT command that queued it, echoed with its results; empty when none
	CntDepend             bool    // True if userData is counter dependent
	PacketCntArray        []int64 // End Point packet counters for which userData is valid
	Format                uint8   // User data format identifier (8 bit)
	ResponseExp           bool    // True to request End Point response
	ResponsePrio          bool    // True to request priority End Point response
	DlWindReq             bool    // True to request further End Point DL window
	ExpOnly               bool    // True to send downlink only if End Point expects response
	DlRxStatQry           bool    // SCACI §3.10.1: True to query DL RX status from endpoint
	Result                string  // "sent", "expired", "invalid" per MIOTY spec
	BsEui                 uint64  // Base Station holding the downlink or reporting its result; zero when none
	TxTime                int64   // Unix UTC time of transmission (BSSCI §3.14.1)
	TransmissionPacketCnt int64   // End Point packet counter when transmitted (BSSCI §3.14.1)
	UpdatedAt             time.Time
	UserData              [][]byte // Canonical MIOTY DLDataQueue.UserData field for counter-dependent messages
	// EndpointAckedAt is when an uplink with dlAck acknowledged the transmitted downlink (BSSCI §3.10.1).
	EndpointAckedAt *time.Time
	// AcceptedAt is when BsEui accepted the downlink with dlDataQueRsp (BSSCI §3.12); nil until then.
	AcceptedAt *time.Time
}

// Outcome is the downlink's result as its originators see it: the result a
// base station reported (BSSCI §3.14.1), or revoked, which no station
// reports; empty while the downlink is in flight.
func (m *DownlinkMessage) Outcome() string {
	if m.Status != mioty.DLQueueStatusRevoked {
		return m.Result
	}
	revoked, _ := mioty.ResultForQueueStatus(m.Status)
	return revoked
}

// WireQueueID is the service center queue id as the BSSCI and SCACI wire
// carries it, unsigned; a stored id that is not positive reports false.
func (m *DownlinkMessage) WireQueueID() (uint64, bool) {
	if m.QueID <= 0 {
		return 0, false
	}
	return uint64(m.QueID), true
}

// EndpointLocation is what the service center knows about where a tenant's
// endpoint can be reached: the base stations that received its latest uplink
// and the one its active session was last attached or propagated through
// (zero when none). Deciding which station serves the endpoint is left to the
// caller.
type EndpointLocation struct {
	LatestReceptions []mioty.BaseStationReception
	AttachedThrough  uint64
}

// PendingDownlink names a downlink waiting for a base station: its queue row,
// the tenant that owns its endpoint, the organization that enqueued it and the
// endpoint.
type PendingDownlink struct {
	QueID          uint64
	TenantID       int64
	OrganizationID uuid.UUID
	EpEUI          uint64
}

// DownlinkRevocation revokes the tenant's downlink named by its service center
// queue id where it waits. Without a station it is a revoke in the service
// center's queue, which ends only a downlink no base station holds yet. With
// one it is that base station's answer to dlDataRev (BSSCI §3.13, §3.17),
// which ends only the in-flight downlink the station holds. A set
// organization and endpoint narrow it to the downlink that organization
// queued for that endpoint, so another owner's queue id revokes nothing.
type DownlinkRevocation struct {
	QueID          int64
	TenantID       int64
	OrganizationID *uuid.UUID
	EpEUI          *uint64
	Station        *uint64
}

// PacketCounterDownlinks names the in-flight counter-dependent downlinks of
// an endpoint scheduled for one packet counter (SCACI §3.11.1): the tenant's,
// narrowed to the organization when it is set.
type PacketCounterDownlinks struct {
	TenantID       int64
	OrganizationID *uuid.UUID
	EpEUI          uint64
	PacketCnt      uint32
}

// DownlinkQueueFilter narrows the in-flight downlink queue listing; nil fields
// are not applied and a nil Status means every in-flight state.
type DownlinkQueueFilter struct {
	EpEUI *[8]byte
	// BsEUI limits the listing to the downlinks this base station holds.
	BsEUI    *[8]byte
	Status   *mioty.DLQueueStatus
	Priority *float32
	QueID    *int64
	// OrganizationID limits the listing to the downlinks the organization queued.
	OrganizationID *uuid.UUID
	// QueuedFrom limits the listing to downlinks queued at or after it: an
	// endpoint's registration, so a deleted registration's downlinks stay with it.
	QueuedFrom *time.Time
}

// DownlinkPatch replaces the SCACI §3.10.1 fields of a pending downlink.
type DownlinkPatch struct {
	Payloads     [][]byte
	Priority     float32
	CntDepend    bool
	PacketCnt    []int64
	Format       uint8
	ResponseExp  bool
	ResponsePrio bool
	DlWindReq    bool
	ExpOnly      bool
	DlRxStatQry  bool
}

// DownlinkResultFilter narrows the terminal downlink results listing; an
// empty Status means every terminal state.
type DownlinkResultFilter struct {
	EpEUI  []byte
	BsEUI  []byte
	QueID  *int64
	Status string
	From   *time.Time
	To     *time.Time
	// QueuedFrom limits the results to downlinks queued at or after it (see DownlinkQueueFilter).
	QueuedFrom *time.Time
}
