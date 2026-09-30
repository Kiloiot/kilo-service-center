package bssciservices

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// EndpointLocator reads where a tenant's endpoint was last heard and attached;
// storage.ErrNotFound when the tenant has no such endpoint.
type EndpointLocator interface {
	GetEndpointLocation(ctx context.Context, tenantID int64, epEUI uint64) (storage.EndpointLocation, error)
}

// ServingStationPolicy decides which base station serves a tenant's endpoint,
// for the downlinks queued for it and its DL RX status queries. Only a
// station that hears the endpoint can use its downlink window (radio §3.6.1),
// so the policy picks, in order:
//  1. the station that received the endpoint's latest uplink with the best SNR;
//  2. before it has been heard, the station it was attached or propagated through;
//  3. otherwise none. A downlink then waits for the endpoint's next dlOpen
//     uplink and is never handed to an arbitrary station.
type ServingStationPolicy struct {
	locations EndpointLocator
}

// NewServingStationPolicy builds the policy over the endpoint locations.
func NewServingStationPolicy(locations EndpointLocator) (*ServingStationPolicy, error) {
	if locations == nil {
		return nil, ErrNilEndpointLocator
	}
	return &ServingStationPolicy{locations: locations}, nil
}

// ServingStation returns the station serving the tenant's endpoint; known is
// false while no station heard or attached it. storage.ErrNotFound when the
// tenant has no such endpoint.
func (p *ServingStationPolicy) ServingStation(ctx context.Context, tenantID int64, epEUI uint64) (bsEUI uint64, known bool, err error) {
	location, err := p.locations.GetEndpointLocation(ctx, tenantID, epEUI)
	if err != nil {
		return 0, false, err
	}
	if best, heard := bestReception(location.LatestReceptions); heard {
		return best.BsEui, true, nil
	}
	if location.AttachedThrough != 0 {
		return location.AttachedThrough, true, nil
	}
	return 0, false, nil
}

// bestReception is the reception with the highest SNR; the first wins a tie.
func bestReception(receptions []mioty.BaseStationReception) (mioty.BaseStationReception, bool) {
	if len(receptions) == 0 {
		return mioty.BaseStationReception{}, false
	}
	best := receptions[0]
	for _, reception := range receptions[1:] {
		if reception.Snr > best.Snr {
			best = reception
		}
	}
	return best, true
}
