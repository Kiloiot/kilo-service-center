// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"time"

	messagesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
	miotyformat "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// MessageListingStoreAdapter adapts interfaces.MIOTYMessageRepository to messagesservice.MessageStore.
// Implements Export with domain-only constants.
type MessageListingStoreAdapter struct {
	repo interfaces.MIOTYMessageRepository
}

// NewMessageListingStoreAdapter creates a new adapter for message listing.
func NewMessageListingStoreAdapter(repo interfaces.MIOTYMessageRepository) *MessageListingStoreAdapter {
	return &MessageListingStoreAdapter{repo: repo}
}

// List returns messages for the given tenant with filters.
func (a *MessageListingStoreAdapter) List(ctx context.Context, tenantID int64, filters *messagesservice.MessageFilter, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	dbFilter := convertMessageFilter(tenantID, filters, limit, offset)
	return a.repo.ListULDataMessages(ctx, dbFilter)
}

// ListByBaseStation returns messages for a specific base station.
func (a *MessageListingStoreAdapter) ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *messagesservice.MessageFilter, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	dbFilter := convertMessageFilter(tenantID, filters, limit, offset)
	if len(bsEui) >= 8 {
		euiVal := bytesToUint64(bsEui)
		dbFilter.BsEui = &euiVal
	}
	return a.repo.ListULDataMessages(ctx, dbFilter)
}

// GetByID returns a specific message by ID.
func (a *MessageListingStoreAdapter) GetByID(ctx context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error) {
	return a.repo.GetULDataMessage(ctx, messageID, tenantID)
}

// GetBaseStationStats returns message statistics for a base station.
func (a *MessageListingStoreAdapter) GetBaseStationStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime time.Time) (*mioty.BaseStationMessageStats, error) {
	return a.repo.GetBaseStationMessageStats(ctx, tenantID, bsEui, &startTime, &endTime)
}

// Search searches messages by query string.
func (a *MessageListingStoreAdapter) Search(ctx context.Context, tenantID int64, bsEui []byte, query string, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
	dbFilter := mioty.ULDataMessageFilter{
		TenantID:   tenantID,
		Limit:      limit,
		Offset:     offset,
		SearchTerm: &query,
	}
	if len(bsEui) >= 8 {
		euiVal := bytesToUint64(bsEui)
		dbFilter.BsEui = &euiVal
	}
	return a.repo.ListULDataMessages(ctx, dbFilter)
}

// Export exports messages in the specified format.
// Returns domain error for unsupported format; handler maps to gRPC token.
func (a *MessageListingStoreAdapter) Export(ctx context.Context, tenantID int64, bsEui []byte, filters *messagesservice.MessageFilter, format string) ([]byte, error) {
	// Fetch messages (limited for export using domain constant)
	dbFilter := convertMessageFilter(tenantID, filters, messagesservice.ExportMaxLimit, 0)
	if len(bsEui) >= 8 {
		euiVal := bytesToUint64(bsEui)
		dbFilter.BsEui = &euiVal
	}

	msgs, _, err := a.repo.ListULDataMessages(ctx, dbFilter)
	if err != nil {
		return nil, err
	}

	switch format {
	case messagesservice.ExportFormatJSON:
		return json.Marshal(msgs)
	case messagesservice.ExportFormatCSV:
		return a.exportToCSV(msgs)
	default:
		// Return domain error (not literal string)
		return nil, messagesservice.ErrExportUnsupportedFormat
	}
}

// exportToCSV converts messages to CSV bytes using domain-layer headers.
func (a *MessageListingStoreAdapter) exportToCSV(msgs []*mioty.ULDataMessage) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Use domain-layer headers (NO pkg/grpc dependency)
	if err := w.Write(messagesservice.ExportCSVHeaders); err != nil {
		return nil, err
	}

	for _, m := range msgs {
		record := []string{
			m.ID,
			miotyformat.FormatEUI64(m.EpEui),
			miotyformat.FormatEUI64(m.BsEui),
			miotyformat.FormatTimestampRFC3339(m.RxTime),
			miotyformat.FormatInt(int64(m.PacketCnt)), // PacketCnt is uint32, cast to int64
			miotyformat.FormatFloat2Dec(m.SNR),
			miotyformat.FormatFloat2Dec(m.RSSI),
			miotyformat.FormatUserDataHex(m.UserData),
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// convertMessageFilter converts messagesservice.MessageFilter to mioty.ULDataMessageFilter.
func convertMessageFilter(tenantID int64, filter *messagesservice.MessageFilter, limit, offset int) mioty.ULDataMessageFilter {
	dbFilter := mioty.ULDataMessageFilter{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	}

	if filter != nil {
		if len(filter.EpEui) >= 8 {
			epEuiVal := bytesToUint64(filter.EpEui)
			dbFilter.EpEui = &epEuiVal
		}
		if len(filter.BsEui) >= 8 {
			bsEuiVal := bytesToUint64(filter.BsEui)
			dbFilter.BsEui = &bsEuiVal
		}
		dbFilter.StartTime = filter.StartTime
		dbFilter.EndTime = filter.EndTime
		// Direction: pass through for potential future DB-level filtering
		// Note: Direction guard is enforced at service layer (uplink-only in storage)
		if filter.Direction != "" {
			dbFilter.Direction = &filter.Direction
		}
		// Note: MinRSSI/MaxRSSI not supported in DB filter - filtering done at application layer if needed
	}

	return dbFilter
}

// bytesToUint64 converts byte slice to uint64 (big-endian).
func bytesToUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}

// Ensure MessageListingStoreAdapter implements messagesservice.MessageStore
var _ messagesservice.MessageStore = (*MessageListingStoreAdapter)(nil)
