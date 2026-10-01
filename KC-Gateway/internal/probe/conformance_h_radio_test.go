//go:build integration

package probe

import (
	"testing"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// dlWindowDelay is the delay of the downlink window after an uplink with RX
// open (RADIO §3.6.1 l.341-348, about 6.88 s).
const dlWindowDelay = 6880 * time.Millisecond

// H1 RADIO §3.6.6.3 l.924: a downlink payload is at most 200 bytes; 201 is
// refused at the API, 200 is queued and sent whole.
func TestConformanceH1_DLPayloadLimitAPI(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	req := dlRequest(make([]byte, oversizedRadioLen))
	req.EpEui = euiHex(ep.eui)
	_, err := coreClient(t).SendDownlink(apiCtx(t, ep.tenant), req)
	require.Equal(t, codes.InvalidArgument, status.Code(err), "201-byte downlink: %v", err)

	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	payload := randomBytes(t, maxRadioPayload)
	que := bs.dlDataQueFor(t, from, ep, queueDownlink(t, ep, &pb.SendDownlinkRequest{Payloads: [][]byte{payload}}))
	require.Equal(t, [][]byte{payload}, userDataEntries(t, que))
}

// H5 RADIO §3.6.1 l.341-348: an AC that answers an uplink with a downlink
// within 2 s gets it to the station inside the downlink window.
func TestConformanceH5_ResponseInsideWindow(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	ac := newAC(t)
	bs := connectStation(t, stationA)
	fromAC, fromBS := ac.mark(), bs.mark()
	uplink := ulDataOpen(ep, uplinkCounter, []byte{0x01})
	uplink[keyResponseExp] = true
	sentAt := time.Now()
	requireRsp(t, bs.request(t, uplink), cmdULData)
	ac.awaitCommand(t, fromAC, cmdULData, forEndpoint(ep.eui))
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x01})), cmdDLDataQue)
	bs.awaitCommand(t, fromBS, cmdDLDataQue, forEndpoint(ep.eui))
	require.Less(t, time.Since(sentAt), dlWindowDelay, "downlink reached the station after the window")
}
