package postgres

import (
	"time"
)

// Connection, retry, and backoff policy.
const (
	// connectTimeout bounds the initial connection phase in New.
	connectTimeout = 30 * time.Second
	// connectMaxRetries is the number of connection attempts made by Connect.
	connectMaxRetries = 5
	// initialRetryAttempt is the first attempt number in retry loops.
	initialRetryAttempt = 1
	// connectRetryDelayMax caps the exponential backoff between connection attempts.
	connectRetryDelayMax = 30 * time.Second
	// connectPingTimeout bounds the ping that validates a fresh connection.
	connectPingTimeout = 5 * time.Second
	// healthCheckPingTimeout bounds the ping issued by HealthCheck.
	healthCheckPingTimeout = 2 * time.Second
	// executeMaxRetries is the number of attempts made by ExecuteWithRetry.
	executeMaxRetries = 3
	// executeInitialRetryDelay is the first backoff delay used by ExecuteWithRetry.
	executeInitialRetryDelay = 100 * time.Millisecond
	// retryBackoffMultiplier doubles the delay between retryable failures.
	retryBackoffMultiplier = 2
	// migrationPingTimeout bounds the connectivity check before running migrations.
	migrationPingTimeout = 5 * time.Second
	// maxConnectionRetryBackoffSeconds caps the exponential base station connection retry backoff (1 hour).
	maxConnectionRetryBackoffSeconds = 3600
	// defaultDownlinkMaxAttempts is used when a queued downlink specifies no attempt limit.
	defaultDownlinkMaxAttempts = 3
)

// Query pagination policy.
const (
	// defaultSCACIEventsLimit is the page size when ListSCACIEvents gets no limit.
	defaultSCACIEventsLimit = 50
	// maxSCACIEventsLimit caps the ListSCACIEvents page size.
	maxSCACIEventsLimit = 100
)

// Stored-row notification listener policy.
const (
	// listenerReconnectFirst is the first wait before the listener reconnects; it doubles per failed attempt.
	listenerReconnectFirst = time.Second
	// listenerReconnectCap caps the wait between the listener's reconnect attempts.
	listenerReconnectCap = 30 * time.Second
	// listenerPingInterval is how often the idle listener proves its connection, so a silently dropped one reconnects.
	listenerPingInterval = 90 * time.Second
)

// Feature defaults.
const (
	// defaultArchivalEnabled enables the archival scheduler in the default configuration.
	defaultArchivalEnabled = true
	// defaultArchivalCheckInterval is how often the archival scheduler wakes up.
	defaultArchivalCheckInterval = 15 * time.Minute
)

// Repository operation names recorded as the operation log field.
const (
	opGetDownlinkResults          = "GetDownlinkResults"
	opExpireOverdueDownlinks      = "ExpireOverdueDownlinks"
	opGetDownlinksByPacketCnt     = "GetDownlinksByPacketCnt"
	opGetByTenant                 = "GetByTenant"
	opListByTenantPaginated       = "ListByTenantPaginated"
	opStreamAllForPropagation     = "StreamAllForPropagation"
	opGetActiveSessionsByEndpoint = "GetActiveSessionsByEndpoint"
	opListActiveSessions          = "ListActiveSessions"
	opList                        = "List"
	opListByDeviceModel           = "ListByDeviceModel"
	opListWithModel               = "ListWithModel"
	opListByManufacturer          = "ListByManufacturer"
	opListWithManufacturer        = "ListWithManufacturer"
)
