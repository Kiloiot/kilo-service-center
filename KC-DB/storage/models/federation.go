package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	// FederationOutboxStatusPending means the relay record has not been sent yet.
	FederationOutboxStatusPending = "pending"
	// FederationOutboxStatusSent means the relay record was sent but not yet acknowledged.
	FederationOutboxStatusSent = "sent"
	// FederationOutboxStatusAcked means ECE acknowledged receipt successfully.
	FederationOutboxStatusAcked = "acked"
	// FederationOutboxStatusRejected means ECE rejected the uplink (terminal; no retry).
	FederationOutboxStatusRejected = "rejected"
)

// FederationOutboxRecord represents a single uplink pending relay to ECE.
type FederationOutboxRecord struct {
	RelayID      uuid.UUID  `db:"relay_id"`
	EpEUI        int64      `db:"ep_eui"`
	BsEUI        int64      `db:"bs_eui"`
	RawFrame     []byte     `db:"raw_frame"`
	ReceivedAtNs int64      `db:"received_at_ns"`
	Status       string     `db:"status"`
	LastError    *string    `db:"last_error"`
	SentAt       *time.Time `db:"sent_at"`
	AckedAt      *time.Time `db:"acked_at"`
	CreatedAt    time.Time  `db:"created_at"`
}
