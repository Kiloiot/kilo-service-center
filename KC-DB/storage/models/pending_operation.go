package models

import (
	"encoding/json"
	"time"
)

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
	// Metadata is []byte rather than json.RawMessage: the column is nullable,
	// and database/sql can scan NULL into *[]byte but not *json.RawMessage.
	Metadata  []byte    `db:"metadata"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
