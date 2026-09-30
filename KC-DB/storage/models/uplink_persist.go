package models

import (
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// DeliveryChannel names a downstream consumer of a persisted uplink.
type DeliveryChannel string

const (
	// DeliveryChannelSCACI fans the uplink out to Application Centers.
	DeliveryChannelSCACI DeliveryChannel = "scaci"
	// DeliveryChannelMQTT publishes the uplink as a device event.
	DeliveryChannelMQTT DeliveryChannel = "mqtt"
)

// Delivery outbox row states.
const (
	DeliveryStatusPending   = "pending"
	DeliveryStatusDelivered = "delivered"
	DeliveryStatusParked    = "parked"
)

// UplinkClassification is the deduplication verdict for one reception.
type UplinkClassification int

const (
	// UplinkNew means the reception created a new message.
	UplinkNew UplinkClassification = iota
	// UplinkDuplicate means the reception was merged into an existing message.
	UplinkDuplicate
)

// UplinkPersistRequest carries one base station reception together with the
// message it becomes when it is the first reception of its packet.
type UplinkPersistRequest struct {
	// Message has the owner tenant, a preallocated ID and exactly one entry in
	// BaseStations: the reception being persisted.
	Message *mioty.ULDataMessage
	// Window is how long a packet counter stays a duplicate of its first reception.
	Window time.Duration
	// ReceptionWindow is how long the delivery of a new message waits for the
	// receptions of the other base stations to merge into it, so its ulData
	// lists every receiving station (SCACI §3.8.1 baseStations).
	ReceptionWindow time.Duration
	// Channels are the outbox rows a new message is queued for.
	Channels []DeliveryChannel
}

// UplinkPersistOutcome reports what the store did with the reception.
type UplinkPersistOutcome struct {
	Classification UplinkClassification
	// MessageID is the stored message: the new one, or the first reception's
	// message when the reception was a duplicate.
	MessageID string
	// BaseStations are all receptions merged into the message so far.
	BaseStations []mioty.BaseStationReception
	// DuplicateCount is the number of receptions merged after the first.
	DuplicateCount int
}

// MessageDeliveryRecord is one row of the delivery outbox.
type MessageDeliveryRecord struct {
	MessageID     uuid.UUID       `db:"message_id"`
	Channel       DeliveryChannel `db:"channel"`
	OwnerTenantID int64           `db:"owner_tenant_id"`
	Status        string          `db:"status"`
	Attempts      int             `db:"attempts"`
	NextAttemptAt time.Time       `db:"next_attempt_at"`
	LastError     *string         `db:"last_error"`
	CreatedAt     time.Time       `db:"created_at"`
	DeliveredAt   *time.Time      `db:"delivered_at"`
}
