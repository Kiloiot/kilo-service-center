package endpoint

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var profileEditTime = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func attachedEndpoint() *models.EndPoint {
	shAddr := uint16(0x1505)
	return &models.EndPoint{
		EpStatus: EndpointStatusAttached,
		NwkSnKey: []byte{0x01, 0x02},
		ShAddr:   &shAddr,
		Bidi:     true,
	}
}

func TestRecordProfileChangeStampsOnlyAStationProfileChange(t *testing.T) {
	cases := []struct {
		name    string
		edit    func(*models.EndPoint)
		stamped bool
	}{
		{"short address", func(ep *models.EndPoint) { v := uint16(0x1506); ep.ShAddr = &v }, true},
		{"network session key", func(ep *models.EndPoint) { ep.NwkSnKey = []byte{0x03} }, true},
		{"bidirectional", func(ep *models.EndPoint) { ep.Bidi = false }, true},
		{"last packet counter", func(ep *models.EndPoint) { ep.LastPacketCnt = 9 }, true},
		{"dual channel", func(ep *models.EndPoint) { ep.DualChan = true }, true},
		{"repetition", func(ep *models.EndPoint) { ep.Repetition = true }, true},
		{"wide carrier offset", func(ep *models.EndPoint) { ep.WideCarrOff = true }, true},
		{"long block distance", func(ep *models.EndPoint) { ep.LongBlkDist = true }, true},
		{"name", func(ep *models.EndPoint) { ep.Name = "renamed" }, false},
		{"application key", func(ep *models.EndPoint) { ep.AppKey = []byte{0x09} }, false},
		{"carrier offset", func(ep *models.EndPoint) { ep.CarrierOffset = 120 }, false},
		{"pre-attach", func(ep *models.EndPoint) { ep.PreAttach = true }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ep := attachedEndpoint()
			prior := StationProfileOf(ep)
			tc.edit(ep)

			RecordProfileChange(ep, prior, profileEditTime)

			if tc.stamped {
				assert.Equal(t, &profileEditTime, ep.ProfileChangedAt)
			} else {
				assert.Nil(t, ep.ProfileChangedAt)
			}
		})
	}
}

func TestReattachPendingUntilTheNextAttachPropagate(t *testing.T) {
	before := profileEditTime.Add(-time.Minute)
	after := profileEditTime.Add(time.Minute)
	cases := []struct {
		name         string
		status       string
		changedAt    *time.Time
		propagatedAt *time.Time
		pending      bool
	}{
		{"never edited", EndpointStatusAttached, nil, &before, false},
		{"edited after the last propagate", EndpointStatusAttached, &profileEditTime, &before, true},
		{"edited with no propagate yet", EndpointStatusAttached, &profileEditTime, nil, true},
		{"propagated after the edit", EndpointStatusAttached, &profileEditTime, &after, false},
		{"detached endpoints take the profile on their next attach", EndpointStatusDetached, &profileEditTime, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ep := attachedEndpoint()
			ep.EpStatus = tc.status
			ep.ProfileChangedAt = tc.changedAt
			ep.PropagatedAt = tc.propagatedAt

			assert.Equal(t, tc.pending, ReattachPending(ep))
		})
	}
}
