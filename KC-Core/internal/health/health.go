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

	"github.com/kilocenter/KC-Core/pkg/logger"
)

// Status represents the health status of a component
type Status string

// Health status constants indicate component state.
const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Health check message constants (centralized per governance rules).
const (
	MsgHealthy         = "healthy"
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

// NewService creates a new health check service
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
		if response.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if response.Status == StatusDegraded {
			statusCode = http.StatusOK // Still return 200 for degraded
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(response) // Error handled by HTTP layer
	}
}

// PostgreSQLChecker checks PostgreSQL health
type PostgreSQLChecker struct {
	db     *sql.DB
	logger logger.Logger
}

// NewPostgreSQLChecker creates a new PostgreSQL health checker
func NewPostgreSQLChecker(db *sql.DB, logger logger.Logger) *PostgreSQLChecker {
	return &PostgreSQLChecker{
		db:     db,
		logger: logger,
	}
}

// Check performs the PostgreSQL health check
func (c *PostgreSQLChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      "postgresql",
		Timestamp: start,
		Metadata:  make(map[string]interface{}),
	}

	// Check connection
	err := c.db.PingContext(ctx)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("Failed to ping database: %v", err)
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
	if float64(stats.InUse)/float64(stats.MaxOpenConnections) > 0.9 {
		check.Status = StatusDegraded
		check.Message = "High connection usage"
	} else {
		check.Status = StatusHealthy
		check.Message = "Database is healthy"
	}

	// Perform a simple query to verify functionality
	var result int
	err = c.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("Failed to execute test query: %v", err)
	}

	check.Duration = time.Since(start)
	return check
}

// MQTTChecker checks MQTT broker connectivity
type MQTTChecker struct {
	isConnected func() bool
	logger      logger.Logger
}

// NewMQTTChecker creates a new MQTT health checker
func NewMQTTChecker(isConnected func() bool, logger logger.Logger) *MQTTChecker {
	return &MQTTChecker{
		isConnected: isConnected,
		logger:      logger,
	}
}

// Check performs the MQTT health check
func (c *MQTTChecker) Check(_ context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      "mqtt",
		Timestamp: start,
		Metadata:  make(map[string]interface{}),
	}

	if c.isConnected() {
		check.Status = StatusHealthy
		check.Message = "MQTT broker is connected"
	} else {
		check.Status = StatusUnhealthy
		check.Message = "MQTT broker is disconnected"
	}

	check.Duration = time.Since(start)
	return check
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
		Name:      "http",
		Timestamp: start,
	}

	client := &http.Client{Timeout: c.timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("%s: %v", MsgFailedToCreate, err)
		check.Duration = time.Since(start)
		return check
	}

	resp, err := client.Do(req)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("%s: %v", MsgFailedToConnect, err)
		check.Duration = time.Since(start)
		return check
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
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
		Name:      "tcp",
		Timestamp: start,
	}

	d := net.Dialer{Timeout: c.timeout}
	conn, err := d.DialContext(ctx, "tcp", c.address)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("%s: %v", MsgFailedToConnect, err)
	} else {
		_ = conn.Close()
		check.Status = StatusHealthy
		check.Message = MsgOK
	}
	check.Duration = time.Since(start)
	return check
}
