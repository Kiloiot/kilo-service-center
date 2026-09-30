// Package scacimonitoring provides SCACI session monitoring.
package scacimonitoring

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/google/uuid"
)

// SCACISessionReader reads SCACI sessions and their statistics for monitoring.
type SCACISessionReader interface {
	GetSessionByID(ctx context.Context, tenantID, sessionID int64) (*models.SCACISession, error)
	GetSessionStatistics(ctx context.Context, tenantID int64) (*models.SCACISessionStatistics, error)
	ListSessions(ctx context.Context, filter *models.SCACISessionFilter) ([]*models.SCACISession, int64, error)
}

// SCACIOperationReader reads the SCACI operation log for monitoring.
type SCACIOperationReader interface {
	GetTenantOperationSummaryBetween(ctx context.Context, tenantID int64, from, to time.Time) (*models.SCACIOperationSummary, error)
	CountOperationsBySession(ctx context.Context, tenantID int64, sessionIDs []int64) (map[int64]int64, error)
	ListFailedOperationGroups(ctx context.Context, tenantID int64, from, to time.Time, limit, offset int) ([]*models.SCACIOperationErrorGroup, int64, error)
	GetPingSummary(ctx context.Context, tenantID int64, command string, from, to time.Time, staleAfter time.Duration) (*models.SCACIPingSummary, error)
	GetLatestOperation(ctx context.Context, tenantID int64, command string) (*models.SCACIOperation, error)
}

// DownlinkQueueReader lists and counts the in-flight downlink queue of a tenant.
type DownlinkQueueReader interface {
	ListTenantQueue(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter, limit, offset int) ([]*storage.DownlinkMessage, error)
	CountTenantQueue(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter) (int64, error)
}

// ListenerState reports whether the SCACI listener accepts connections.
type ListenerState interface {
	Listening() bool
}

// Deps wires the monitoring service. The status reports the service offline
// while the Listener is not listening, as with scaci_enabled off, or is nil.
type Deps struct {
	Sessions     SCACISessionReader
	Operations   SCACIOperationReader
	Queue        DownlinkQueueReader
	Listener     ListenerState
	ServiceStart time.Time
	SCEui        uint64
	Clock        clock.Clock
	Log          logger.Logger
}

// Service implements grpcservices.ScaciMonitoringService.
type Service struct {
	sessions     SCACISessionReader
	operations   SCACIOperationReader
	queue        DownlinkQueueReader
	listener     ListenerState
	serviceStart time.Time
	scEui        uint64
	clock        clock.Clock
	logger       logger.Logger
}

// percentScale converts a ratio to a percentage.
const percentScale = 100

// reconnectCountProbe fetches the smallest page: only the total matters.
const reconnectCountProbe = 1

var validSessionStatuses = map[string]struct{}{
	models.SCACISessionStatusActive:       {},
	models.SCACISessionStatusResumed:      {},
	models.SCACISessionStatusDisconnected: {},
	models.SCACISessionStatusTerminated:   {},
}

// New creates a new SCACI monitoring service.
func New(d Deps) *Service {
	return &Service{
		sessions:     d.Sessions,
		operations:   d.Operations,
		queue:        d.Queue,
		listener:     d.Listener,
		serviceStart: d.ServiceStart,
		scEui:        d.SCEui,
		clock:        d.Clock,
		logger:       d.Log,
	}
}

// ListSessions returns SCACI sessions for the given tenant.
func (s *Service) ListSessions(ctx context.Context, tenantID int64, filter grpcservices.ScaciSessionFilter, limit, offset int) ([]*grpcservices.ScaciSession, int64, error) {
	if filter.Status != nil {
		if _, ok := validSessionStatuses[*filter.Status]; !ok {
			return nil, 0, fmt.Errorf("%w: %s", ErrInvalidSessionStatus, *filter.Status)
		}
	}
	sessions, total, err := s.sessions.ListSessions(ctx, &models.SCACISessionFilter{
		TenantID:  &tenantID,
		Status:    filter.Status,
		CanResume: filter.CanResume,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, LogListSessionsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errCtxListSessions, err)
	}
	ids := make([]int64, len(sessions))
	for i, session := range sessions {
		ids[i] = session.ID
	}
	counts, err := s.operations.CountOperationsBySession(ctx, tenantID, ids)
	if err != nil {
		s.logger.ErrorContext(ctx, LogCountOperationsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errCtxCountOperations, err)
	}
	result := make([]*grpcservices.ScaciSession, len(sessions))
	for i, session := range sessions {
		result[i] = toScaciSession(session, counts[session.ID])
	}
	return result, total, nil
}

// GetSession returns a specific SCACI session.
func (s *Service) GetSession(ctx context.Context, tenantID int64, sessionID string) (*grpcservices.ScaciSession, error) {
	id, err := strconv.ParseInt(sessionID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidSessionID, err)
	}
	session, err := s.sessions.GetSessionByID(ctx, tenantID, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		s.logger.ErrorContext(ctx, LogGetSessionFailed, logger.FieldTenantID, tenantID, logger.FieldSessionID, sessionID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxGetSession, err)
	}
	counts, err := s.operations.CountOperationsBySession(ctx, tenantID, []int64{session.ID})
	if err != nil {
		s.logger.ErrorContext(ctx, LogCountOperationsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxCountOperations, err)
	}
	return toScaciSession(session, counts[session.ID]), nil
}

// GetStatistics returns SCACI operation statistics for the window.
func (s *Service) GetStatistics(ctx context.Context, tenantID int64, window grpcservices.ScaciWindow) (*grpcservices.ScaciStatistics, error) {
	from, to, err := s.resolveWindow(window)
	if err != nil {
		return nil, err
	}
	sessionStats, err := s.sessions.GetSessionStatistics(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogGetSessionStatisticsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxGetSessionStatistics, err)
	}
	opSummary, err := s.operations.GetTenantOperationSummaryBetween(ctx, tenantID, from, to)
	if err != nil {
		s.logger.ErrorContext(ctx, LogGetOperationSummaryFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxGetOperationSummary, err)
	}
	successfulOps := opSummary.StateCounts[string(models.OperationStateCompleted)] +
		opSummary.StateCounts[string(models.OperationStateCompletedWithWarnings)]
	failedOps := opSummary.StateCounts[string(models.OperationStateFailed)]
	var successRate float64
	if opSummary.TotalOperations > 0 {
		successRate = float64(successfulOps) / float64(opSummary.TotalOperations) * percentScale
	}
	uptime := s.serviceStart
	return &grpcservices.ScaciStatistics{
		TotalSessions:        sessionStats.TotalSessions,
		ActiveSessions:       sessionStats.ActiveSessions,
		TotalOperations:      opSummary.TotalOperations,
		SuccessfulOperations: successfulOps,
		FailedOperations:     failedOps,
		SuccessRate:          successRate,
		UptimeSince:          &uptime,
	}, nil
}

// ListErrors returns the failed-operation buckets of the window, newest first.
func (s *Service) ListErrors(ctx context.Context, tenantID int64, window grpcservices.ScaciWindow, limit, offset int) ([]*grpcservices.ScaciError, int64, error) {
	from, to, err := s.resolveWindow(window)
	if err != nil {
		return nil, 0, err
	}
	groups, total, err := s.operations.ListFailedOperationGroups(ctx, tenantID, from, to, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogListErrorsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errCtxListErrors, err)
	}
	result := make([]*grpcservices.ScaciError, len(groups))
	for i, g := range groups {
		result[i] = toScaciError(g)
	}
	return result, total, nil
}

// ListQueues returns the in-flight downlinks the organization queued.
func (s *Service) ListQueues(ctx context.Context, tenantID int64, orgID uuid.UUID, epEUI *[8]byte, limit, offset int) ([]*grpcservices.ScaciQueue, int64, error) {
	filter := storage.DownlinkQueueFilter{EpEUI: epEUI, OrganizationID: &orgID}
	total, err := s.queue.CountTenantQueue(ctx, tenantID, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, LogCountQueueFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errCtxCountQueue, err)
	}
	entries, err := s.queue.ListTenantQueue(ctx, tenantID, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogListQueueFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errCtxListQueue, err)
	}
	result := make([]*grpcservices.ScaciQueue, len(entries))
	for i, entry := range entries {
		result[i] = toScaciQueue(entry)
	}
	return result, total, nil
}

// GetStatus returns overall SCACI status; the pending operations are the
// in-flight downlinks the organization queued, and the ping and reconnect
// counters cover the window.
func (s *Service) GetStatus(ctx context.Context, tenantID int64, orgID uuid.UUID, window grpcservices.ScaciWindow) (*grpcservices.ScaciStatus, error) {
	from, to, err := s.resolveWindow(window)
	if err != nil {
		return nil, err
	}
	sessionStats, err := s.sessions.GetSessionStatistics(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogGetStatusStatisticsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxGetSessionStatistics, err)
	}
	queueDepth, err := s.queue.CountTenantQueue(ctx, tenantID, storage.DownlinkQueueFilter{OrganizationID: &orgID})
	if err != nil {
		s.logger.ErrorContext(ctx, LogCountQueueFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxCountQueue, err)
	}
	pings, err := s.operations.GetPingSummary(ctx, tenantID, scaci.CmdPing, from, to, dbconfig.PingInterval)
	if err != nil {
		s.logger.ErrorContext(ctx, LogGetPingSummaryFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxPingSummary, err)
	}
	_, reconnects, err := s.sessions.ListSessions(ctx, &models.SCACISessionFilter{
		TenantID:      &tenantID,
		ConnectedFrom: &from,
		ConnectedTo:   &to,
		Limit:         reconnectCountProbe,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, LogCountReconnectsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxCountReconnects, err)
	}
	lastConnect, err := s.operations.GetLatestOperation(ctx, tenantID, scaci.CmdConnect)
	if err != nil {
		s.logger.ErrorContext(ctx, LogGetLatestConnectFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errCtxLatestConnect, err)
	}
	uptime := s.serviceStart
	status := &grpcservices.ScaciStatus{
		ServiceOnline:     s.listener != nil && s.listener.Listening(),
		ActiveSessions:    clampInt32(sessionStats.ActiveSessions),
		PendingOperations: clampInt32(queueDepth),
		UptimeSince:       &uptime,
		ProtocolVersion:   scaci.ProtocolVersionString,
		SCEui:             mioty.FormatEUI64(s.scEui),
		LastPingAt:        pings.LastPingAt,
		LastPingRTT:       pings.LastPingRTT,
		MissedPings:       pings.MissedPings,
		ReconnectAttempts: reconnects,
	}
	if lastConnect != nil {
		status.LastConnectResult = lastConnect.State
	}
	return status, nil
}

// resolveWindow fills the missing bounds from the default lookback and
// rejects an inverted range.
func (s *Service) resolveWindow(window grpcservices.ScaciWindow) (time.Time, time.Time, error) {
	to := s.clock.Now()
	if window.To != nil {
		to = *window.To
	}
	from := to.Add(-config.SCACIDashboardDefaultWindow)
	if window.From != nil {
		from = *window.From
	}
	if window.From != nil && window.To == nil {
		to = from.Add(config.SCACIDashboardDefaultWindow)
		if now := s.clock.Now(); to.After(now) {
			to = now
		}
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, ErrInvalidTimeRange
	}
	return from, to, nil
}

func toScaciSession(session *models.SCACISession, operations int64) *grpcservices.ScaciSession {
	return &grpcservices.ScaciSession{
		ID:              strconv.FormatInt(session.ID, 10),
		AcEUI:           mioty.FormatEUIBytes(session.AcEUI[:]),
		Status:          session.Status,
		CanResume:       session.CanResume,
		ProtocolVersion: session.NegotiatedVersion,
		ConnectedAt:     session.ConnectedAt,
		LastActivityAt:  session.LastHeartbeat,
		DisconnectedAt:  session.DisconnectedAt,
		OperationsCount: operations,
		SnAcUUID:        uuid.UUID(session.SnAcUUID).String(),
		SnScUUID:        uuid.UUID(session.SnScUUID).String(),
		LastOpIDAc:      session.LastOpIDAc,
		LastOpIDSc:      session.LastOpIDSc,
	}
}

func toScaciError(g *models.SCACIOperationErrorGroup) *grpcservices.ScaciError {
	e := &grpcservices.ScaciError{
		OperationType: g.Command,
		SessionID:     strconv.FormatInt(g.SessionID, 10),
		OccurredAt:    g.LastSeen,
		FirstSeen:     g.FirstSeen,
		LastSeen:      g.LastSeen,
		Count:         g.Count,
	}
	if g.ErrorCode != nil {
		e.ErrorCode = strconv.Itoa(*g.ErrorCode)
	}
	if g.ErrorToken != nil {
		e.ErrorToken = *g.ErrorToken
	}
	if g.ErrorMessage != nil {
		e.ErrorMessage = *g.ErrorMessage
	}
	e.ID = g.Command + ":" + e.ErrorCode + ":" + e.ErrorToken
	return e
}

func toScaciQueue(entry *storage.DownlinkMessage) *grpcservices.ScaciQueue {
	return &grpcservices.ScaciQueue{
		ID:            strconv.FormatInt(entry.ID, 10),
		EpEUI:         entry.EPEUI,
		OperationType: scaci.CmdDLDataQueue,
		Status:        string(entry.Status),
		Payload:       entry.Payload,
		QueuedAt:      entry.CreatedAt,
		ProcessedAt:   entry.SentAt,
		QueID:         entry.QueID,
		Priority:      entry.Priority,
	}
}

func clampInt32(v int64) int32 {
	switch {
	case v > math.MaxInt32:
		return math.MaxInt32
	case v < 0:
		return 0
	default:
		return int32(v)
	}
}

// Ensure Service implements grpcservices.ScaciMonitoringService
var _ grpcservices.ScaciMonitoringService = (*Service)(nil)
