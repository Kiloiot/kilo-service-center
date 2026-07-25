// Package queries provides SQL query templates for operation status tracking
package queries

// SQL query templates for operation status repository

const (
	// SQLOperationEventsByID retrieves events for a specific operation ID
	// Supports optional event type filtering via fmt.Sprintf format specifier
	// Parameters: $1=categories array, $2=interval, $3=operation_id, $4=limit
	// Usage: fmt.Sprintf(SQLOperationEventsByID, eventTypeFilter)
	SQLOperationEventsByID = `
		SELECT
			se.event_type,
			se.title,
			se.occurred_at,
			se.data
		FROM system_events se
		WHERE se.event_category = ANY($1::text[])
		AND se.occurred_at > NOW() - $2::interval
		AND se.data ->> 'operation_id' = $3
		%s
		ORDER BY se.occurred_at DESC
		LIMIT $4
	`

	// SQLEndpointOperationEvents retrieves events for a specific endpoint EUI
	// Searches across multiple fields (data, title, description) for endpoint reference
	// Parameters: $1=categories array, $2=interval, $3=endpoint_eui, $4=limit, $5=offset
	// Supports OFFSET for pagination
	SQLEndpointOperationEvents = `
		SELECT
			se.event_type,
			se.title,
			COALESCE(se.description, '') AS description,
			se.occurred_at,
			se.data
		FROM system_events se
		WHERE se.event_category = ANY($1::text[])
		AND se.occurred_at > NOW() - $2::interval
		AND (
			se.data::text LIKE '%' || $3 || '%'
			OR se.title LIKE '%' || $3 || '%'
			OR se.description LIKE '%' || $3 || '%'
		)
		ORDER BY se.occurred_at DESC
		LIMIT $4
		OFFSET $5
	`

	// SQLBSSCIStatusSummary retrieves base station status with recent event counts and session metadata
	// Joins base stations with event counts and most recent session for operational monitoring
	// Parameters: $1=tenant_id, $2=categories array, $3=interval (24h)
	// Session metadata LEFT JOIN shows ALL stations regardless of connection state
	SQLBSSCIStatusSummary = `
		SELECT
			bs.bs_eui,
			bs.name,
			bs.is_online,
			bs.last_seen_at,
			bs.bidi,
			bs.latitude,
			bs.longitude,
			bs.altitude,
			(
				SELECT COUNT(DISTINCT se.id)
				FROM system_events se
				WHERE se.event_category = ANY($2::text[])
				AND se.data::text LIKE '%' || encode(bs.bs_eui, 'hex') || '%'
				AND se.occurred_at > NOW() - $3::interval
			) as recent_events,
			sess.sn_sc_uuid,
			sess.sn_bs_uuid,
			sess.sn_bs_op_id,
			sess.sn_sc_op_id,
			sess.can_resume,
			sess.encoding,
			sess.protocol_version,
			sess.connect_info,
			sess.last_ping_at
		FROM basestations bs
		LEFT JOIN LATERAL (
			SELECT encode(sn_sc_uuid, 'hex') as sn_sc_uuid,
			       encode(sn_bs_uuid, 'hex') as sn_bs_uuid,
			       sn_bs_op_id, sn_sc_op_id,
			       can_resume, encoding, protocol_version, connect_info,
			       last_ping_at
			FROM basestation_sessions
			WHERE basestation_id = bs.id
			  AND status IN ('active', 'disconnected', 'terminated')
			ORDER BY started_at DESC
			LIMIT 1
		) sess ON true
		WHERE bs.tenant_id = $1
		ORDER BY bs.last_seen_at DESC
	`

	// SQLEventSummary aggregates event types within time window
	// Parameters: $1=categories array, $2=interval
	SQLEventSummary = `
		SELECT
			se.event_type,
			COUNT(*) AS event_count
		FROM system_events se
		WHERE se.event_category = ANY($1::text[])
			AND se.occurred_at > NOW() - $2::interval
		GROUP BY se.event_type
		ORDER BY event_count DESC
	`
)

// Event type filter clauses for operation-specific queries
// These are SQL fragments used with fmt.Sprintf() to customize SQLOperationEventsByID
const (
	// EventFilterAttachOps filters for attach-related operation events
	// Exact matches for all attach lifecycle events
	EventFilterAttachOps = `AND se.event_type IN (
		'attach_propagate_initiated',
		'attach_propagate_complete',
		'attach_propagate_failed',
		'endpoint_attached',
		'endpoint_attach_failed',
		'endpoint_operation_failed',
		'bssci_error_received',
		'bssci_error_sent',
		'bssci_error_ack'
	)`

	// EventFilterDetachOps filters for detach-related operation events
	// Exact matches for all detach lifecycle events
	EventFilterDetachOps = `AND se.event_type IN (
		'detach_propagate_initiated',
		'detach_propagate_complete',
		'detach_propagate_failed',
		'endpoint_detached',
		'endpoint_detach_failed',
		'endpoint_operation_failed',
		'bssci_error_received',
		'bssci_error_sent',
		'bssci_error_ack'
	)`

	// EventFilterAllOps filters for all attach/detach operation events
	// Exact matches combining both attach and detach lifecycle events
	EventFilterAllOps = `AND se.event_type IN (
		'attach_propagate_initiated',
		'attach_propagate_complete',
		'attach_propagate_failed',
		'detach_propagate_initiated',
		'detach_propagate_complete',
		'detach_propagate_failed',
		'endpoint_attached',
		'endpoint_detached',
		'endpoint_attach_failed',
		'endpoint_detach_failed',
		'endpoint_operation_failed',
		'bssci_error_received',
		'bssci_error_sent',
		'bssci_error_ack'
	)`
)
