// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	messagesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// messageListingStore covers the uplink reads the listing adapter performs
// against the message repository. Satisfied structurally by the KC-DB MIOTY
// message repository.
type messageListingStore interface {
	ListULDataMessages(ctx context.Context, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error)
	ListStoredULData(ctx context.Context, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, error)
	GetULDataMessage(ctx context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error)
	GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error)
}

// MessageListingStoreAdapter adapts the KC-DB message repository to messagesservice.MessageStore.
// Implements Export with domain-only constants.
type MessageListingStoreAdapter struct {
	repo messageListingStore
}

// NewMessageListingStoreAdapter creates a new adapter for message listing.
func NewMessageListingStoreAdapter(repo messageListingStore) *MessageListingStoreAdapter {
	return &MessageListingStoreAdapter{repo: repo}
}

// List returns messages for the given tenant with filters.
func (a *MessageListingStoreAdapter) List(ctx context.Context, tenantID int64, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	dbFilter := convertMessageFilter(tenantID, filters, limit, offset)
	return a.repo.ListULDataMessages(ctx, dbFilter)
}

// ListByBaseStation returns messages for a specific base station.
func (a *MessageListingStoreAdapter) ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	dbFilter := convertMessageFilter(tenantID, filters, limit, offset)
	dbFilter.BsEui = mioty.OptionalEUI64FromBytes(bsEui)
	return a.repo.ListULDataMessages(ctx, dbFilter)
}

// ListStored lists the uplinks stored at or after since, of the station when
// bsEui is given, newest stored first.
func (a *MessageListingStoreAdapter) ListStored(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, since *time.Time, limit, offset int) ([]*mioty.ULDataMessage, error) {
	dbFilter := convertMessageFilter(tenantID, filters, limit, offset)
	dbFilter.BsEui = mioty.OptionalEUI64FromBytes(bsEui)
	dbFilter.StoredSince = since
	return a.repo.ListStoredULData(ctx, dbFilter)
}

// GetByID returns a specific message by ID.
func (a *MessageListingStoreAdapter) GetByID(ctx context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error) {
	return a.repo.GetULDataMessage(ctx, messageID, tenantID)
}

// GetBaseStationStats returns the statistics of the uplinks the base station
// received within the window; a nil bound leaves that side open.
func (a *MessageListingStoreAdapter) GetBaseStationStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error) {
	return a.repo.GetBaseStationMessageStats(ctx, tenantID, bsEui, startTime, endTime)
}

// Search searches messages by query string.
func (a *MessageListingStoreAdapter) Search(ctx context.Context, tenantID int64, bsEui []byte, query string, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	dbFilter := mioty.ULDataMessageFilter{
		TenantID:   tenantID,
		Limit:      limit,
		Offset:     offset,
		SearchTerm: &query,
	}
	dbFilter.BsEui = mioty.OptionalEUI64FromBytes(bsEui)
	return a.repo.ListULDataMessages(ctx, dbFilter)
}

// Export exports messages in the specified format.
// Returns domain error for unsupported format; handler maps to gRPC token.
func (a *MessageListingStoreAdapter) Export(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, format string) ([]byte, error) {
	// Fetch messages (limited for export using domain constant)
	dbFilter := convertMessageFilter(tenantID, filters, messagesservice.ExportMaxLimit, 0)
	dbFilter.BsEui = mioty.OptionalEUI64FromBytes(bsEui)

	msgs, _, err := a.repo.ListULDataMessages(ctx, dbFilter)
	if err != nil {
		return nil, err
	}

	switch format {
	case messagesservice.ExportFormatJSON:
		return exportJSON(msgs)
	case messagesservice.ExportFormatCSV:
		return exportCSV(msgs)
	default:
		// Return domain error (not literal string)
		return nil, messagesservice.ErrExportUnsupportedFormat
	}
}

// convertMessageFilter renders the service filter as the repository's.
// Direction stays unfiltered here: the service answers a downlink direction
// itself because stored messages are uplinks only.
func convertMessageFilter(tenantID int64, filter *grpcservices.MessageFilters, limit, offset int) mioty.ULDataMessageFilter {
	dbFilter := mioty.ULDataMessageFilter{TenantID: tenantID, Limit: limit, Offset: offset}
	if filter == nil {
		return dbFilter
	}
	dbFilter.EpEui = mioty.OptionalEUI64FromBytes(filter.EpEui)
	dbFilter.BsEui = mioty.OptionalEUI64FromBytes(filter.BsEui)
	dbFilter.StartTime = filter.StartTime
	dbFilter.EndTime = filter.EndTime
	if filter.Direction != "" {
		dbFilter.Direction = &filter.Direction
	}
	dbFilter.Duplicate = filter.Duplicate
	dbFilter.DlOpen = filter.DlOpen
	if filter.Profile != "" {
		dbFilter.Profile = &filter.Profile
	}
	if filter.Mode != "" {
		dbFilter.Mode = &filter.Mode
	}
	return dbFilter
}

// Ensure MessageListingStoreAdapter implements messagesservice.MessageStore
var _ messagesservice.MessageStore = (*MessageListingStoreAdapter)(nil)
