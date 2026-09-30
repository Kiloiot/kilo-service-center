package grpcservices

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// DownlinkListingService lists a tenant's downlink queue and results; a
// listing of one endpoint starts at its registration, so a deleted
// registration's downlinks stay with it.
type DownlinkListingService struct {
	results DownlinkResultsStore
	queue   DownlinkQueueLister
	window  RegistrationWindow
}

// NewDownlinkListingService creates the downlink listings of the gRPC layer.
func NewDownlinkListingService(results DownlinkResultsStore, queue DownlinkQueueLister, window RegistrationWindow) *DownlinkListingService {
	return &DownlinkListingService{results: results, queue: queue, window: window}
}

// ListDownlinkQueue pages the in-flight downlinks the filter selects and counts them.
func (s *DownlinkListingService) ListDownlinkQueue(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter, limit, offset int) ([]*storage.DownlinkMessage, int64, error) {
	if filter.EpEUI != nil {
		since, registered, err := s.window.Start(ctx, tenantID, filter.EpEUI[:], nil)
		if err != nil || !registered {
			return nil, 0, err
		}
		filter.QueuedFrom = since
	}
	total, err := s.queue.CountTenantQueue(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}
	entries, err := s.queue.ListTenantQueue(ctx, tenantID, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

// GetDownlinkResults pages the finished downlinks the filter selects.
func (s *DownlinkListingService) GetDownlinkResults(ctx context.Context, tenantID int64, orgID *uuid.UUID, filter storage.DownlinkResultFilter, limit, offset int) ([]*storage.DownlinkMessage, int, error) {
	if len(filter.EpEUI) > 0 {
		since, registered, err := s.window.Start(ctx, tenantID, filter.EpEUI, nil)
		if err != nil || !registered {
			return nil, 0, err
		}
		filter.QueuedFrom = since
	}
	return s.results.GetDownlinkResults(ctx, tenantID, orgID, filter, limit, offset)
}
