// Package health provides health check functionality for KiloCenter
package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Status represents the health status of a component
type Status string

// Health status constants indicate component state.
const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
	// StatusDisabled marks a component configuration turned off; it never
	// lowers the aggregate status.
	StatusDisabled Status = "disabled"
)

// Health check message constants (centralized per governance rules).
const (
	MsgOK              = "OK"
	MsgFailedToConnect = "Failed to connect"
	MsgFailedToCreate  = "Failed to create request"
	MsgHTTPErrorPrefix = "HTTP"
)

// Check represents a health check result
type Check struct {
	Name      string                 `json:"name"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Response represents the overall health check response
type Response struct {
	Status    Status            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Checks    map[string]*Check `json:"checks"`
}

// Checker defines the interface for health checks
type Checker interface {
	Check(ctx context.Context) *Check
}

// Service manages health checks
type Service struct {
	logger   logger.Logger
	checkers map[string]Checker
	mu       sync.RWMutex
	version  string
}

const (
	checkNamePostgreSQL = "postgresql"
	checkNameMQTT       = "mqtt"
	checkNameHTTP       = "http"
	checkNameTCP        = "tcp"
	checkNameListener   = "listener"
	checkNameDisabled   = "disabled"
)

// Health check messages.
const (
	msgFmtDatabasePingFailed = "Failed to ping database: %v"
	msgHighConnectionUsage   = "High connection usage"
	msgDatabaseHealthy       = "Database is healthy"
	msgFmtTestQueryFailed    = "Failed to execute test query: %v"
	msgMQTTConnected         = "MQTT broker is connected"
	msgMQTTDisconnected      = "MQTT broker is disconnected"
	msgListening             = "Listening"
	msgNotListening          = "Not listening"
)

// Log messages for failures that leave the health result itself intact.
const (
	logFailedWriteHealthResponse = "Failed to write health response"
	logFailedCloseHealthProbe    = "Failed to close health probe connection"
)

// connectionUsageDegradedRatio marks the pool as degraded when in-use
// connections exceed this share of the pool.
const connectionUsageDegradedRatio = 0.9

// NewService creates a new health check service
// Health check names.
func NewService(logger logger.Logger, version string) *Service {
	return &Service{
		logger:   logger,
		checkers: make(map[string]Checker),
		version:  version,
	}
}

// RegisterChecker registers a health checker
func (s *Service) RegisterChecker(name string, checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
}

// CheckHealth performs all health checks
func (s *Service) CheckHealth(ctx context.Context) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := &Response{
		Timestamp: time.Now(),
		Version:   s.version,
		Checks:    make(map[string]*Check),
	}

	// Perform all checks concurrently
	var wg sync.WaitGroup
	checkResults := make(chan struct {
		name  string
		check *Check
	}, len(s.checkers))

	for name, checker := range s.checkers {
		wg.Add(1)
		go func(n string, c Checker) {
			defer wg.Done()
			check := c.Check(ctx)
			checkResults <- struct {
				name  string
				check *Check
			}{name: n, check: check}
		}(name, checker)
	}

	wg.Wait()
	close(checkResults)

	// Collect results
	overallStatus := StatusHealthy
	for result := range checkResults {
		response.Checks[result.name] = result.check
		if result.check.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if result.check.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	response.Status = overallStatus
	return response
}

// HTTPHandler returns an HTTP handler for health checks
func (s *Service) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		response := s.CheckHealth(ctx)

		// Set appropriate status code
		statusCode := http.StatusOK
		switch response.Status {
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		case StatusDegraded:
			statusCode = http.StatusOK // Still return 200 for degraded
		}

		w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.logger.WarnContext(ctx, logFailedWriteHealthResponse, logger.FieldError, err)
		}
	}
}

// PostgreSQLHealthDB is the database capability the PostgreSQL checker uses:
// liveness ping, pool statistics, and one probe query.
type PostgreSQLHealthDB interface {
	PingContext(ctx context.Context) error
	Stats() sql.DBStats
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// PostgreSQLChecker checks PostgreSQL health
type PostgreSQLChecker struct {
	db     PostgreSQLHealthDB
	logger logger.Logger
}

// NewPostgreSQLChecker creates a new PostgreSQL health checker
func NewPostgreSQLChecker(db PostgreSQLHealthDB, logger logger.Logger) *PostgreSQLChecker {
	return &PostgreSQLChecker{
		db:     db,
		logger: logger,
	}
}

// Check performs the PostgreSQL health check
func (c *PostgreSQLChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      checkNamePostgreSQL,
		Timestamp: start,
		Metadata:  make(map[string]interface{}),
	}

	// Check connection
	err := c.db.PingContext(ctx)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf(msgFmtDatabasePingFailed, err)
		check.Duration = time.Since(start)
		return check
	}

	// Check database stats
	stats := c.db.Stats()
	check.Metadata["open_connections"] = stats.OpenConnections
	check.Metadata["in_use"] = stats.InUse
	check.Metadata["idle"] = stats.Idle
	check.Metadata["wait_count"] = stats.WaitCount
	check.Metadata["wait_duration"] = stats.WaitDuration.String()

	// Check if we're running low on connections
	if float64(stats.InUse)/float64(stats.MaxOpenConnections) > connectionUsageDegradedRatio {
		check.Status = StatusDegraded
		check.Message = msgHighConnectionUsage
	} else {
		check.Status = StatusHealthy
		check.Message = msgDatabaseHealthy
	}

	// Perform a simple query to verify functionality
	var result int
	err = c.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf(msgFmtTestQueryFailed, err)
	}

	check.Duration = time.Since(start)
	return check
}

// StateChecker reports a component from state this process holds, so the
// check never dials anything.
type StateChecker struct {
	name    string
	state   func() bool
	upMsg   string
	downMsg string
}

// NewMQTTChecker reports the MQTT client's connection state.
func NewMQTTChecker(isConnected func() bool) *StateChecker {
	return &StateChecker{name: checkNameMQTT, state: isConnected, upMsg: msgMQTTConnected, downMsg: msgMQTTDisconnected}
}

// ListenerState reports whether a listener this process owns is accepting.
type ListenerState interface {
	Listening() bool
}

// NewListenerChecker reports one of this process's own listeners from its
// in-process state instead of dialing its port.
func NewListenerChecker(listener ListenerState) *StateChecker {
	return &StateChecker{name: checkNameListener, state: listener.Listening, upMsg: msgListening, downMsg: msgNotListening}
}

// Check reports healthy while the state holds and unhealthy otherwise.
func (c *StateChecker) Check(_ context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      c.name,
		Timestamp: start,
		Status:    StatusUnhealthy,
		Message:   c.downMsg,
	}
	if c.state() {
		check.Status = StatusHealthy
		check.Message = c.upMsg
	}
	check.Duration = time.Since(start)
	return check
}

// DisabledChecker reports a component that configuration turned off.
type DisabledChecker struct {
	message string
}

// NewDisabledChecker reports a disabled component with the given message.
func NewDisabledChecker(message string) DisabledChecker {
	return DisabledChecker{message: message}
}

// Check reports the component as disabled.
func (c DisabledChecker) Check(_ context.Context) *Check {
	return &Check{Name: checkNameDisabled, Status: StatusDisabled, Message: c.message, Timestamp: time.Now()}
}

// HTTPChecker checks HTTP endpoint health
type HTTPChecker struct {
	url     string
	timeout time.Duration
	logger  logger.Logger
}

// NewHTTPChecker creates a new HTTP health checker
func NewHTTPChecker(url string, timeout time.Duration, logger logger.Logger) *HTTPChecker {
	return &HTTPChecker{url: url, timeout: timeout, logger: logger}
}

// Check performs the HTTP health check
func (c *HTTPChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      checkNameHTTP,
		Timestamp: start,
	}

	client := &http.Client{Timeout: c.timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil) //nolint:gosec // G704: URL comes from operator configuration, not request input
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("%s: %v", MsgFailedToCreate, err)
		check.Duration = time.Since(start)
		return check
	}

	resp, err := client.Do(req) //nolint:gosec // G704: URL comes from operator configuration, not request input
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("%s: %v", MsgFailedToConnect, err)
		check.Duration = time.Since(start)
		return check
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.WarnContext(ctx, logFailedCloseHealthProbe, logger.FieldError, err)
		}
	}()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		check.Status = StatusHealthy
		check.Message = MsgOK
	} else {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("%s %d", MsgHTTPErrorPrefix, resp.StatusCode)
	}
	check.Duration = time.Since(start)
	return check
}

// TCPChecker checks TCP endpoint health
type TCPChecker struct {
	address string
	timeout time.Duration
	logger  logger.Logger
}

// NewTCPChecker creates a new TCP health checker
func NewTCPChecker(address string, timeout time.Duration, logger logger.Logger) *TCPChecker {
	return &TCPChecker{address: address, timeout: timeout, logger: logger}
}

// Check performs the TCP health check
func (c *TCPChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      checkNameTCP,
		Timestamp: start,
	}

	d := net.Dialer{Timeout: c.timeout}
	conn, err := d.DialContext(ctx, "tcp", c.address)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("%s: %v", MsgFailedToConnect, err)
	} else {
		if err := conn.Close(); err != nil {
			c.logger.WarnContext(ctx, logFailedCloseHealthProbe, logger.FieldError, err)
		}
		check.Status = StatusHealthy
		check.Message = MsgOK
	}
	check.Duration = time.Since(start)
	return check
}
