// Package messages provides message listing service implementation for gRPC layer.
package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/mioty"
)

// MessageStore provides message persistence operations.
type MessageStore interface {
	List(ctx context.Context, tenantID int64, filters *MessageFilter, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *MessageFilter, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	GetByID(ctx context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error)
	GetBaseStationStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime time.Time) (*mioty.BaseStationMessageStats, error)
	Search(ctx context.Context, tenantID int64, bsEui []byte, query string, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	Export(ctx context.Context, tenantID int64, bsEui []byte, filters *MessageFilter, format string) ([]byte, error)
}

// MessageFilter contains filtering criteria for messages.
type MessageFilter struct {
	EpEui     []byte
	BsEui     []byte
	StartTime *time.Time
	EndTime   *time.Time
	MinRSSI   *float64
	MaxRSSI   *float64
	Direction string // Filter by direction: "uplink" | "downlink" | "" (all)
}

// DefaultStreamBatchSize is the default batch size for streaming operations.
const DefaultStreamBatchSize = 100

// Service implements grpcservices.MessageListingService.
type Service struct {
	messageStore       MessageStore
	streamPollInterval time.Duration
	streamBatchSize    int
	logger             logger.Logger
}

// New creates a new message listing service.
func New(messageStore MessageStore, streamPollInterval time.Duration, streamBatchSize int, log logger.Logger) *Service {
	if streamBatchSize <= 0 {
		streamBatchSize = DefaultStreamBatchSize
	}
	return &Service{
		messageStore:       messageStore,
		streamPollInterval: streamPollInterval,
		streamBatchSize:    streamBatchSize,
		logger:             log,
	}
}

// GetMessage returns a single message by ID for the given tenant.
func (s *Service) GetMessage(ctx context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error) {
	message, err := s.messageStore.GetByID(ctx, tenantID, messageID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get message", "tenantID", tenantID, "messageID", messageID, "error", err)
		return nil, fmt.Errorf("get message: %w", err)
	}
	return message, nil
}

// ListMessages returns messages for the given tenant with optional filters.
func (s *Service) ListMessages(ctx context.Context, tenantID int64, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		return []*mioty.ULDataMessage{}, 0, nil
	}

	filter := convertFilters(filters)

	messages, total, err := s.messageStore.List(ctx, tenantID, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list messages", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list messages: %w", err)
	}

	return messages, total, nil
}

// StreamMessages streams messages for the given tenant.
func (s *Service) StreamMessages(ctx context.Context, tenantID int64, filters *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		ch := make(chan *mioty.ULDataMessage)
		close(ch) // Return closed channel for empty stream
		return ch, nil
	}

	ch := make(chan *mioty.ULDataMessage, s.streamBatchSize)
	filter := convertFilters(filters)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(s.streamPollInterval)
		defer ticker.Stop()

		var lastTimestamp *time.Time

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Set start time filter to only get new messages
				if lastTimestamp != nil {
					filter.StartTime = lastTimestamp
				}

				messages, _, err := s.messageStore.List(ctx, tenantID, filter, s.streamBatchSize, 0)
				if err != nil {
					s.logger.ErrorContext(ctx, "stream poll error", "tenantID", tenantID, "error", err)
					continue
				}

				for _, msg := range messages {
					select {
					case ch <- msg:
						// Update last timestamp
						msgTime := time.Unix(0, msg.RxTime)
						lastTimestamp = &msgTime
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// ListBaseStationMessages returns messages for a specific base station.
func (s *Service) ListBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		return []*mioty.ULDataMessage{}, 0, nil
	}

	filter := convertFilters(filters)

	messages, total, err := s.messageStore.ListByBaseStation(ctx, tenantID, bsEui, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list base station messages", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list base station messages: %w", err)
	}

	return messages, total, nil
}

// GetBaseStationMessage returns a specific message, optionally validating it belongs to the given base station.
func (s *Service) GetBaseStationMessage(ctx context.Context, tenantID int64, bsEui []byte, messageID string) (*mioty.ULDataMessage, error) {
	message, err := s.messageStore.GetByID(ctx, tenantID, messageID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get message", "tenantID", tenantID, "messageID", messageID, "error", err)
		return nil, fmt.Errorf("get message: %w", err)
	}

	// Validate message belongs to the requested base station when bsEui is provided
	if len(bsEui) == 8 {
		var requestedEui uint64
		for i := 0; i < 8; i++ {
			requestedEui = (requestedEui << 8) | uint64(bsEui[i])
		}
		if message.BsEui != requestedEui {
			return nil, fmt.Errorf("message does not belong to requested base station")
		}
	}

	return message, nil
}

// GetBaseStationMessageStats returns message statistics for a base station.
func (s *Service) GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error) {
	// Default to last 24 hours if not specified
	end := time.Now()
	start := end.Add(-24 * time.Hour)
	if startTime != nil {
		start = *startTime
	}
	if endTime != nil {
		end = *endTime
	}

	stats, err := s.messageStore.GetBaseStationStats(ctx, tenantID, bsEui, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get message stats", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get message stats: %w", err)
	}

	return stats, nil
}

// SearchBaseStationMessages searches messages for a base station.
func (s *Service) SearchBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, query string, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		return []*mioty.ULDataMessage{}, 0, nil
	}

	messages, total, err := s.messageStore.Search(ctx, tenantID, bsEui, query, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to search messages", "tenantID", tenantID, "query", query, "error", err)
		return nil, 0, fmt.Errorf("search messages: %w", err)
	}

	return messages, total, nil
}

// ExportBaseStationMessages exports messages for a base station.
func (s *Service) ExportBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, format string) ([]byte, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		return []byte{}, nil
	}

	filter := convertFilters(filters)

	data, err := s.messageStore.Export(ctx, tenantID, bsEui, filter, format)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to export messages", "tenantID", tenantID, "format", format, "error", err)
		return nil, fmt.Errorf("export messages: %w", err)
	}

	return data, nil
}

// StreamBaseStationMessages streams messages for a specific base station.
func (s *Service) StreamBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		ch := make(chan *mioty.ULDataMessage)
		close(ch) // Return closed channel for empty stream
		return ch, nil
	}

	ch := make(chan *mioty.ULDataMessage, s.streamBatchSize)
	filter := convertFilters(filters)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(s.streamPollInterval)
		defer ticker.Stop()

		var lastTimestamp *time.Time

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Set start time filter to only get new messages
				if lastTimestamp != nil {
					filter.StartTime = lastTimestamp
				}

				messages, _, err := s.messageStore.ListByBaseStation(ctx, tenantID, bsEui, filter, s.streamBatchSize, 0)
				if err != nil {
					s.logger.ErrorContext(ctx, "stream poll error", "tenantID", tenantID, "error", err)
					continue
				}

				for _, msg := range messages {
					select {
					case ch <- msg:
						// Update last timestamp
						msgTime := time.Unix(0, msg.RxTime)
						lastTimestamp = &msgTime
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// convertFilters converts grpcservices.MessageFilters to internal MessageFilter.
func convertFilters(filters *grpcservices.MessageFilters) *MessageFilter {
	if filters == nil {
		return nil
	}

	return &MessageFilter{
		EpEui:     filters.EpEui,
		BsEui:     filters.BsEui,
		StartTime: filters.StartTime,
		EndTime:   filters.EndTime,
		MinRSSI:   filters.MinRSSI,
		MaxRSSI:   filters.MaxRSSI,
		Direction: filters.Direction,
	}
}

// Ensure Service implements grpcservices.MessageListingService
var _ grpcservices.MessageListingService = (*Service)(nil)
