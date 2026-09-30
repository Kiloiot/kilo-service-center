package postgres

import (
	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// MIOTYDownlinkRepository is the downlink queue, one type per port it
// serves; each consumer depends on the narrow port it uses.
type MIOTYDownlinkRepository struct {
	*DownlinkQueueWriter
	*DownlinkQueueLookup
	*DownlinkStationOutcomes
	*DownlinkRevocations
	*DownlinkReservations
	*DownlinkExpirySweep
	*DownlinkResultsReader
	*EndpointLocations
}

// NewMIOTYDownlinkRepository creates the downlink queue repository.
// Every statement takes its time from the clock, never from the database.
func NewMIOTYDownlinkRepository(db sqlx.ExtContext, queueReader *DownlinkQueueReader, clk clock.Clock, log logger.Logger) *MIOTYDownlinkRepository {
	return &MIOTYDownlinkRepository{
		DownlinkQueueWriter:     &DownlinkQueueWriter{db: db, clock: clk},
		DownlinkQueueLookup:     &DownlinkQueueLookup{db: db, queueReader: queueReader, log: log},
		DownlinkStationOutcomes: &DownlinkStationOutcomes{db: db, clock: clk},
		DownlinkRevocations:     &DownlinkRevocations{db: db, clock: clk},
		DownlinkReservations:    &DownlinkReservations{db: db, clock: clk},
		DownlinkExpirySweep:     &DownlinkExpirySweep{db: db, clock: clk, log: log},
		DownlinkResultsReader:   &DownlinkResultsReader{db: db, log: log},
		EndpointLocations:       &EndpointLocations{db: db},
	}
}
