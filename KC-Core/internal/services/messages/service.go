// Package messages provides message listing service implementation for gRPC layer.
package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/streampoll"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// StoredUplinks reads uplinks in the order the database stored them.
type StoredUplinks interface {
	// ListStored lists one page of the tenant's uplinks, of the station when
	// bsEui is given, stored at or after since (all when nil), newest stored first.
	ListStored(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, since *time.Time, limit, offset int) ([]*mioty.ULDataMessage, error)
}

// MessageStore provides message persistence operations.
type MessageStore interface {
	StoredUplinks
	List(ctx context.Context, tenantID int64, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	GetByID(ctx context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error)
	GetBaseStationStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error)
	Search(ctx context.Context, tenantID int64, bsEui []byte, query string, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	Export(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, format string) ([]byte, error)
}

// RegistrationWindow tells where a view of an endpoint's history starts: the
// requested start, or its registration when that is later; registered is
// false for an EUI the tenant has not registered.
type RegistrationWindow interface {
	Start(ctx context.Context, tenantID int64, epEui []byte, requested *time.Time) (start *time.Time, registered bool, err error)
}

// DefaultStreamBatchSize is the default batch size for streaming operations.
const DefaultStreamBatchSize = 100

// Service implements grpcservices.MessageListingService.
type Service struct {
	messageStore       MessageStore
	window             RegistrationWindow
	streamPollInterval time.Duration
	streamOverlap      time.Duration
	streamWake         streampoll.Waker
	streamBatchSize    int
	logger             logger.Logger
}

// New creates a new message listing service; streamWake announces stored
// uplinks to the uplink streams, and each stream read reaches streamOverlap
// back in storage order.
func New(messageStore MessageStore, window RegistrationWindow, streamPollInterval, streamOverlap time.Duration, streamWake streampoll.Waker, streamBatchSize int, log logger.Logger) *Service {
	if streamBatchSize <= 0 {
		streamBatchSize = DefaultStreamBatchSize
	}
	return &Service{
		messageStore:       messageStore,
		window:             window,
		streamPollInterval: streamPollInterval,
		streamOverlap:      streamOverlap,
		streamWake:         streamWake,
		streamBatchSize:    streamBatchSize,
		logger:             log,
	}
}

// GetMessage returns a single message by ID for the given tenant.
func (s *Service) GetMessage(ctx context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error) {
	message, err := s.messageStore.GetByID(ctx, tenantID, messageID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogMessageGetFailed, logger.FieldTenantID, tenantID, logger.FieldMessageID, messageID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrGetMessage, err)
	}
	return message, nil
}

// ListMessages returns messages for the given tenant with optional filters.
func (s *Service) ListMessages(ctx context.Context, tenantID int64, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		return []*mioty.ULDataMessage{}, 0, nil
	}

	filter, registered, err := s.endpointWindow(ctx, tenantID, filters)
	if err != nil || !registered {
		return []*mioty.ULDataMessage{}, 0, err
	}

	messages, total, err := s.messageStore.List(ctx, tenantID, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogMessageListFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%w: %w", ErrListMessages, err)
	}

	return messages, total, nil
}

// endpointWindow starts a listing of one endpoint's uplinks at its
// registration; registered is false for an EUI the tenant has not registered.
func (s *Service) endpointWindow(ctx context.Context, tenantID int64, filter *grpcservices.MessageFilters) (*grpcservices.MessageFilters, bool, error) {
	if filter == nil || len(filter.EpEui) == 0 {
		return filter, true, nil
	}
	start, registered, err := s.window.Start(ctx, tenantID, filter.EpEui, filter.StartTime)
	if err != nil || !registered {
		return nil, false, err
	}
	scoped := *filter
	scoped.StartTime = start
	return &scoped, true, nil
}

// StreamMessages streams the tenant's uplinks stored after the stream opened.
func (s *Service) StreamMessages(ctx context.Context, tenantID int64, filters *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	return s.streamUplinks(ctx, tenantID, nil, filters), nil
}

// ListBaseStationMessages returns messages for a specific base station.
func (s *Service) ListBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		return []*mioty.ULDataMessage{}, 0, nil
	}

	messages, total, err := s.messageStore.ListByBaseStation(ctx, tenantID, bsEui, filters, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogMessageListBaseStationFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%w: %w", ErrListBaseStationMessages, err)
	}

	return messages, total, nil
}

// GetBaseStationMessage returns a specific message; when bsEui is given, the
// station must have received it, as the primary receiver or as one of the
// SCACI §3.8.1 receptions, the same stations the listing shows it to.
func (s *Service) GetBaseStationMessage(ctx context.Context, tenantID int64, bsEui []byte, messageID string) (*mioty.ULDataMessage, error) {
	message, err := s.messageStore.GetByID(ctx, tenantID, messageID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogMessageGetFailed, logger.FieldTenantID, tenantID, logger.FieldMessageID, messageID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrGetMessage, err)
	}
	if len(bsEui) != dbconfig.EUISize {
		return message, nil
	}
	station := mioty.EUI64FromBytes(bsEui)
	if _, received := message.ReceptionAt(station); message.BsEui != station && !received {
		return nil, ErrMessageNotOwnedByBaseStation
	}
	return message, nil
}

// GetBaseStationMessageStats returns the statistics of the uplinks the base
// station received within the window; a nil bound leaves that side open.
func (s *Service) GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error) {
	stats, err := s.messageStore.GetBaseStationStats(ctx, tenantID, bsEui, startTime, endTime)
	if err != nil {
		s.logger.ErrorContext(ctx, LogMessageStatsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrGetMessageStats, err)
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
		s.logger.ErrorContext(ctx, LogMessageSearchFailed, logger.FieldTenantID, tenantID, logger.FieldQuery, query, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%w: %w", ErrSearchMessages, err)
	}

	return messages, total, nil
}

// ExportBaseStationMessages exports messages for a base station.
func (s *Service) ExportBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, format string) ([]byte, error) {
	// Direction guard: base station messages are uplink-only in storage
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		return []byte{}, nil
	}

	data, err := s.messageStore.Export(ctx, tenantID, bsEui, filters, format)
	if err != nil {
		s.logger.ErrorContext(ctx, LogMessageExportFailed, logger.FieldTenantID, tenantID, logger.FieldFormat, format, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrExportMessages, err)
	}

	return data, nil
}

// StreamBaseStationMessages streams the uplinks of one base station stored after the stream opened.
func (s *Service) StreamBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	return s.streamUplinks(ctx, tenantID, bsEui, filters), nil
}

// streamUplinks polls for the tenant's uplinks, of the station when bsEui is
// given, stored after the stream opened. Base station messages are
// uplink-only in storage, so a downlink filter streams nothing.
func (s *Service) streamUplinks(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters) <-chan *mioty.ULDataMessage {
	if filters != nil && filters.Direction == grpcservices.DirectionDownlink {
		ch := make(chan *mioty.ULDataMessage)
		close(ch)
		return ch
	}
	return streampoll.Stream(ctx, s.streamPollInterval, s.streamBatchSize, streampoll.Source[*mioty.ULDataMessage]{
		Fetch: func(ctx context.Context, since *time.Time, offset int) ([]*mioty.ULDataMessage, error) {
			return s.messageStore.ListStored(ctx, tenantID, bsEui, filters, since, s.streamBatchSize, offset)
		},
		PageSize: s.streamBatchSize,
		StoredAt: func(msg *mioty.ULDataMessage) time.Time { return msg.StoredAt },
		Key:      func(msg *mioty.ULDataMessage) string { return msg.ID },
		Overlap:  s.streamOverlap,
		OnError: func(err error) {
			s.logger.ErrorContext(ctx, LogMessageStreamPollError, logger.FieldTenantID, tenantID, logger.FieldError, err)
		},
		Wake: s.streamWake,
	})
}

// Ensure Service implements grpcservices.MessageListingService
var _ grpcservices.MessageListingService = (*Service)(nil)
