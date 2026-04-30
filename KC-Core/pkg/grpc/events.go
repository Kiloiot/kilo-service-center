package grpc

import (
	"context"

	"github.com/kilocenter/KC-DB/storage/models"
)

// EventWriter writes system events to the event store.
type EventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}
