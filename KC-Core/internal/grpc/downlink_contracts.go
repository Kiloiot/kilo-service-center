package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinks"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// DownlinkCommands queues, edits and revokes an organization's downlinks.
type DownlinkCommands interface {
	Queue(ctx context.Context, owner downlinks.Owner, content downlinks.Content) (*scaci.DLDataQueueResult, error)
	Update(ctx context.Context, target downlinks.Target, content downlinks.Content) (*storage.DownlinkMessage, error)
	Revoke(ctx context.Context, target downlinks.Target) (downlinks.RevokeResult, error)
}

// DownlinkQueueLister pages a tenant's in-flight downlinks.
type DownlinkQueueLister interface {
	ListDownlinkQueue(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter, limit, offset int) ([]*storage.DownlinkMessage, int64, error)
}

// DownlinkResultsReader pages a tenant's finished downlinks.
type DownlinkResultsReader interface {
	GetDownlinkResults(ctx context.Context, tenantID int64, orgID *uuid.UUID, filter storage.DownlinkResultFilter, limit, offset int) ([]*storage.DownlinkMessage, int, error)
}

// DLRXStatusReader reads the DL RX status reports of an endpoint (BSSCI §3.15).
type DLRXStatusReader interface {
	GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte,
		limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error)
	GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte,
		startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error)
}

// ServingStationLocator decides the base station serving a tenant's
// endpoint; known is false while no station heard or attached it.
type ServingStationLocator interface {
	ServingStation(ctx context.Context, tenantID int64, epEUI uint64) (bsEUI uint64, known bool, err error)
}

// EndpointSessionFinder finds the session of the base station an endpoint
// is reached through.
type EndpointSessionFinder interface {
	FindSessionForEndpointAttachment(bsEui uint64) (sessionID string, err error)
}

// BidirectionalSessionSelector picks the connected bidirectional base
// station session a transmission goes through: the requested station's, or
// any of the tenant's when none is requested.
type BidirectionalSessionSelector interface {
	SelectBidirectionalSession(tenantID int64, targetBsEui *uint64) (sessionID string, actualBsEui uint64, err error)
}

// BaseStationLookup finds a tenant's base station; storage.ErrNotFound when
// the tenant has none with the EUI.
type BaseStationLookup interface {
	GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error)
}
