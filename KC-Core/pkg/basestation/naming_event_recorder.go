package basestation

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// StationDirectory reads the registered station an event is about.
type StationDirectory interface {
	GetBaseStation(ctx context.Context, eui [8]byte) (*BaseStation, error)
}

// NamingEventRecorder records base station events under the station's
// registered name, so every station event reads with the name the station
// was registered under, not by its EUI alone.
type NamingEventRecorder struct {
	stations StationDirectory
	next     EventRecorder
	logger   logger.Logger
}

// NewNamingEventRecorder builds the recorder that names the station of every
// event before next records it.
func NewNamingEventRecorder(stations StationDirectory, next EventRecorder, log logger.Logger) *NamingEventRecorder {
	return &NamingEventRecorder{stations: stations, next: next, logger: log}
}

// RecordEvent names the station in the event data, then records the event.
func (r *NamingEventRecorder) RecordEvent(ctx context.Context, eui [8]byte, eventType string, occurredAt time.Time, data map[string]interface{}) error {
	return r.next.RecordEvent(ctx, eui, eventType, occurredAt, r.withStationName(ctx, eui, data))
}

// withStationName returns data with the station's registered name. Data that
// already names the station keeps its name, as for a station just removed;
// an unreadable station leaves the EUI alone.
func (r *NamingEventRecorder) withStationName(ctx context.Context, eui [8]byte, data map[string]interface{}) map[string]interface{} {
	if name, named := data[models.EventDetailKeyBaseStationName].(string); named && name != "" {
		return data
	}
	bs, err := r.stations.GetBaseStation(ctx, eui)
	if err != nil {
		r.logger.WarnContext(ctx, LogFailedToGetBaseStationDetails, logger.FieldError, err)
		return data
	}
	if bs == nil || bs.Name == "" {
		return data
	}
	named := make(map[string]interface{}, len(data)+1)
	for k, v := range data {
		named[k] = v
	}
	named[models.EventDetailKeyBaseStationName] = bs.Name
	return named
}
