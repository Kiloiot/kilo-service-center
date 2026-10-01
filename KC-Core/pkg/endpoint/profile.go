package endpoint

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// StationProfile is the endpoint configuration base stations hold: the
// attach propagate parameters of BSSCI §3.8.1. Base stations keep the profile
// they were last sent until the endpoint is attached again.
type StationProfile struct {
	bidi          bool
	nwkSnKey      string
	shAddr        uint16
	hasShAddr     bool
	lastPacketCnt uint32
	dualChan      bool
	repetition    bool
	wideCarrOff   bool
	longBlkDist   bool
}

// StationProfileOf returns the station profile of ep.
func StationProfileOf(ep *models.EndPoint) StationProfile {
	profile := StationProfile{
		bidi:          ep.Bidi,
		nwkSnKey:      string(ep.NwkSnKey),
		lastPacketCnt: ep.LastPacketCnt,
		dualChan:      ep.DualChan,
		repetition:    ep.Repetition,
		wideCarrOff:   ep.WideCarrOff,
		longBlkDist:   ep.LongBlkDist,
	}
	if ep.ShAddr != nil {
		profile.shAddr, profile.hasShAddr = *ep.ShAddr, true
	}
	return profile
}

// RecordProfileChange stamps ep with now when its station profile differs
// from prior.
func RecordProfileChange(ep *models.EndPoint, prior StationProfile, now time.Time) {
	if StationProfileOf(ep) == prior {
		return
	}
	ep.ProfileChangedAt = &now
}

// ReattachPending reports whether base stations hold an earlier profile of an
// attached endpoint: its profile changed after the last attach propagate a
// base station completed.
func ReattachPending(ep *models.EndPoint) bool {
	if ep.EpStatus != EndpointStatusAttached || ep.ProfileChangedAt == nil {
		return false
	}
	return ep.PropagatedAt == nil || ep.ProfileChangedAt.After(*ep.PropagatedAt)
}
