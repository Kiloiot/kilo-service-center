package grpcservices

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/google/uuid"
)

type messageService struct {
	store  *postgres.DB
	logger logger.Logger
}

// NewMessageService creates a new message service for gRPC layer
func NewMessageService(store *postgres.DB, log logger.Logger) MessageService {
	return &messageService{
		store:  store,
		logger: log,
	}
}

func (s *messageService) GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error) {
	return s.store.GetDownlinkByQueueID(ctx, queId, tenantID)
}

func (s *messageService) GetDownlinkQueue(ctx context.Context, epEui, tenantID string) ([]*storage.DownlinkMessage, error) {
	return s.store.GetDownlinkQueue(ctx, epEui, tenantID)
}

func (s *messageService) GetDownlinkResults(ctx context.Context, epEui, tenantID string, orgID *uuid.UUID, statusFilter string,
	timeFrom, timeTo *time.Time, limit, offset int) ([]*storage.DownlinkMessage, int, error) {
	return s.store.GetDownlinkResults(ctx, epEui, tenantID, orgID, statusFilter, timeFrom, timeTo, limit, offset)
}

func (s *messageService) GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte,
	limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error) {
	return s.store.GetDLRXStatusByEndpoint(ctx, tenantID, epEui, limit, offset, startTime, endTime)
}

func (s *messageService) GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte,
	startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error) {
	return s.store.GetAverageDLRXMetrics(ctx, tenantID, epEui, startTime, endTime)
}

func (s *messageService) GetEndpointBaseStation(ctx context.Context, epEui, tenantID string) (string, error) {
	return s.store.GetEndpointBaseStation(ctx, epEui, tenantID)
}
