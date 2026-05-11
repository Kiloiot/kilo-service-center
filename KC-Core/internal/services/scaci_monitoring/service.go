// Package scacimonitoring provides SCACI session monitoring.
package scacimonitoring

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Service implements grpcservices.ScaciMonitoringService.
type Service struct {
	sessionRepo   interfaces.SCACISessionRepository
	operationRepo interfaces.SCACIOperationRepository
	queueReader   interfaces.DownlinkQueueReader
	logger        logger.Logger
}

// New creates a new SCACI monitoring service.
func New(
	sessionRepo interfaces.SCACISessionRepository,
	operationRepo interfaces.SCACIOperationRepository,
	queueReader interfaces.DownlinkQueueReader,
	log logger.Logger,
) *Service {
	return &Service{
		sessionRepo:   sessionRepo,
		operationRepo: operationRepo,
		queueReader:   queueReader,
		logger:        log,
	}
}

// ListSessions returns SCACI sessions for the given tenant.
func (s *Service) ListSessions(ctx context.Context, tenantID int64, limit, offset int) ([]*grpcservices.ScaciSession, int64, error) {
	filter := &models.SCACISessionFilter{
		TenantID: &tenantID,
		Limit:    limit,
		Offset:   offset,
	}

	sessions, total, err := s.sessionRepo.ListSessions(ctx, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list SCACI sessions", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}

	result := make([]*grpcservices.ScaciSession, len(sessions))
	for i, session := range sessions {
		result[i] = &grpcservices.ScaciSession{
			ID:              fmt.Sprintf("%d", session.ID),
			AcEUI:           hex.EncodeToString(session.AcEUI[:]),
			Status:          session.Status,
			CanResume:       session.CanResume,
			ProtocolVersion: session.NegotiatedVersion,
			ConnectedAt:     session.ConnectedAt,
			LastActivityAt:  session.LastHeartbeat,
			OperationsCount: session.LastOpIDAc + (-session.LastOpIDSc), // Estimate from op ID counters
		}
	}

	return result, total, nil
}

// GetSession returns a specific SCACI session.
func (s *Service) GetSession(ctx context.Context, tenantID int64, sessionID string) (*grpcservices.ScaciSession, error) {
	var id int64
	if _, err := fmt.Sscanf(sessionID, "%d", &id); err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	session, err := s.sessionRepo.GetSessionByID(ctx, tenantID, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get SCACI session", "tenantID", tenantID, "sessionID", sessionID, "error", err)
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	return &grpcservices.ScaciSession{
		ID:              fmt.Sprintf("%d", session.ID),
		AcEUI:           hex.EncodeToString(session.AcEUI[:]),
		Status:          session.Status,
		CanResume:       session.CanResume,
		ProtocolVersion: session.NegotiatedVersion,
		ConnectedAt:     session.ConnectedAt,
		LastActivityAt:  session.LastHeartbeat,
		OperationsCount: session.LastOpIDAc + (-session.LastOpIDSc),
	}, nil
}

// GetStatistics returns SCACI operation statistics.
func (s *Service) GetStatistics(ctx context.Context, tenantID int64) (*grpcservices.ScaciStatistics, error) {
	// Get session statistics
	sessionStats, err := s.sessionRepo.GetSessionStatistics(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get session statistics", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get session statistics: %w", err)
	}

	// Get operation summary (24-hour window)
	opSummary, err := s.operationRepo.GetTenantOperationSummary(ctx, tenantID, 24)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get operation summary", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get operation summary: %w", err)
	}

	// Extract success/failure counts from state counts map
	var successfulOps, failedOps int64
	if opSummary.StateCounts != nil {
		successfulOps = opSummary.StateCounts["completed"]
		failedOps = opSummary.StateCounts["failed"]
	}

	// Calculate success rate
	var successRate float64
	if opSummary.TotalOperations > 0 {
		successRate = float64(successfulOps) / float64(opSummary.TotalOperations) * 100
	}

	return &grpcservices.ScaciStatistics{
		TotalSessions:        sessionStats.TotalSessions,
		ActiveSessions:       sessionStats.ActiveSessions,
		TotalOperations:      opSummary.TotalOperations,
		SuccessfulOperations: successfulOps,
		FailedOperations:     failedOps,
		SuccessRate:          successRate,
		UptimeSince:          nil, // Session statistics don't track uptime
	}, nil
}

// ListErrors returns SCACI errors for the given tenant.
func (s *Service) ListErrors(ctx context.Context, tenantID int64, _ int, _ int) ([]*grpcservices.ScaciError, int64, error) {
	// This would need a dedicated error store; return empty for now
	s.logger.DebugContext(ctx, "ListErrors not fully implemented", "tenantID", tenantID)
	return []*grpcservices.ScaciError{}, 0, nil
}

// ListQueues returns SCACI queue status for the given tenant.
func (s *Service) ListQueues(ctx context.Context, tenantID int64, _ int, _ int) ([]*grpcservices.ScaciQueue, int64, error) {
	// Get queue depth
	queueDepth, err := s.queueReader.CountTenantQueue(ctx, tenantID, nil)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to count queue", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("count queue: %w", err)
	}

	// Return a summary queue status (placeholder until proper queue tracking is implemented)
	// ScaciQueue has: ID, EpEUI, OperationType, Status, Payload, QueuedAt, ProcessedAt
	result := []*grpcservices.ScaciQueue{}
	// No queue entries to return in this placeholder implementation

	return result, queueDepth, nil
}

// GetStatus returns overall SCACI status.
func (s *Service) GetStatus(ctx context.Context, tenantID int64) (*grpcservices.ScaciStatus, error) {
	// Get session statistics for active count
	sessionStats, err := s.sessionRepo.GetSessionStatistics(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get session statistics for status", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get session statistics: %w", err)
	}

	// Get pending operations count
	queueDepth, err := s.queueReader.CountTenantQueue(ctx, tenantID, nil)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get pending operations", "tenantID", tenantID, "error", err)
		queueDepth = 0 // Non-fatal, continue with zero
	}

	activeSessions := int32(0)
	if sessionStats.ActiveSessions > math.MaxInt32 {
		activeSessions = math.MaxInt32
	} else if sessionStats.ActiveSessions > 0 {
		activeSessions = int32(sessionStats.ActiveSessions)
	}

	pendingOps := int32(0)
	if queueDepth > math.MaxInt32 {
		pendingOps = math.MaxInt32
	} else if queueDepth > 0 {
		pendingOps = int32(queueDepth)
	}

	return &grpcservices.ScaciStatus{
		ServiceOnline:     true, // SCACI server is always running if this service is reachable
		ActiveSessions:    activeSessions,
		PendingOperations: pendingOps,
		UptimeSince:       nil,     // Session statistics don't track service uptime
		ProtocolVersion:   "1.0.0", // SCACI v1.0.0
	}, nil
}

// Ensure Service implements grpcservices.ScaciMonitoringService
var _ grpcservices.ScaciMonitoringService = (*Service)(nil)
