//go:build integration

package probe

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"testing"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	epClassBidirectional  = "A"
	epClassUnidirectional = "Z"
	epStatusAttachedAPI   = "attached"
	endpointShAddrMask    = 0xFFFF
	pollInterval          = 200 * time.Millisecond
	messagePageSize       = 100
	updateMaskStatus      = "status"
)

// testEndpoint is an endpoint registered for exactly one test.
type testEndpoint struct {
	eui    uint64
	key    []byte // pre-shared network key
	shAddr uint16
	tenant int64
}

type endpointOption func(*pb.EndPoint)

func unidirectional(ep *pb.EndPoint) { ep.EpClass = epClassUnidirectional }

// preAttached registers the endpoint with Pre-Attachment, which attaches it at
// creation, as a pre-personalized device is registered.
func preAttached(ep *pb.EndPoint) { ep.PreAttach = true }

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

// newEndpoint registers a fresh bidirectional endpoint through CreateEndPoint.
func newEndpoint(t *testing.T, tenant int64, opts ...endpointOption) testEndpoint {
	t.Helper()
	ep := testEndpoint{eui: nextEndpointEUI(), key: randomBytes(t, sessionKeyLen), tenant: tenant}
	ep.shAddr = uint16(ep.eui & endpointShAddrMask) //nolint:gosec // masked to 16 bits
	req := &pb.EndPoint{
		EpEui: euiHex(ep.eui), Name: fmt.Sprintf("conformance-%s", t.Name()), NwkSnKey: ep.key,
		EpClass: epClassBidirectional, ShAddr: uint32(ep.shAddr),
	}
	for _, opt := range opts {
		opt(req)
	}
	_, err := coreClient(t).CreateEndPoint(apiCtx(t, tenant), &pb.CreateEndPointRequest{Endpoint: req})
	require.NoError(t, err, "CreateEndPoint %s", req.EpEui)
	return ep
}

// markAttached records the endpoint as attached, the state a pre-attached
// endpoint has at the SC before any over-the-air traffic.
func markAttached(t *testing.T, ep testEndpoint) {
	t.Helper()
	_, err := coreClient(t).UpdateEndPoint(apiCtx(t, ep.tenant), &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: euiHex(ep.eui), Status: epStatusAttachedAPI},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{updateMaskStatus}},
	})
	require.NoError(t, err, "mark %016X attached", ep.eui)
}

func getEndpoint(t *testing.T, ep testEndpoint) *pb.EndPoint {
	t.Helper()
	got, err := coreClient(t).GetEndPoint(apiCtx(t, ep.tenant), &pb.GetEndPointRequest{EpEui: euiHex(ep.eui)})
	require.NoError(t, err, "GetEndPoint %016X", ep.eui)
	return got
}

// revealEndpoint reads the endpoint with the given keys in clear; the SC audits each reveal.
func revealEndpoint(t *testing.T, ep testEndpoint, keys ...pb.EndpointKey) *pb.EndPoint {
	t.Helper()
	req := &pb.GetEndPointRequest{EpEui: euiHex(ep.eui), RevealKeys: keys}
	got, err := coreClient(t).GetEndPoint(apiCtx(t, ep.tenant), req)
	require.NoError(t, err, "GetEndPoint %016X revealing %v", ep.eui, keys)
	return got
}

// storedMessages returns the uplink rows the SC stored for the endpoint, as
// seen by the given tenant.
func storedMessages(t *testing.T, tenant int64, eui uint64) []*pb.Message {
	t.Helper()
	rsp, err := coreClient(t).ListEndpointMessages(apiCtx(t, tenant),
		&pb.ListEndpointMessagesRequest{EpEui: euiHex(eui), PageSize: messagePageSize})
	require.NoError(t, err, "ListEndpointMessages %016X", eui)
	return rsp.GetMessages()
}

func messageWithCounter(msgs []*pb.Message, cnt uint32) *pb.Message {
	for _, m := range msgs {
		if m.GetPacketCounter() == cnt {
			return m
		}
	}
	return nil
}

// eventually polls cond until it holds or the frame wait elapses.
func eventually(cond func() bool) bool {
	deadline := time.Now().Add(frameWait)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(pollInterval)
	}
}

// dlRequest is a counter-independent downlink carrying one payload.
func dlRequest(payload []byte) *pb.SendDownlinkRequest {
	return &pb.SendDownlinkRequest{Payloads: [][]byte{payload}}
}

// queueDownlink queues a downlink through SendDownlink and returns its queId.
func queueDownlink(t *testing.T, ep testEndpoint, req *pb.SendDownlinkRequest) int64 {
	t.Helper()
	req.EpEui = euiHex(ep.eui)
	rsp, err := coreClient(t).SendDownlink(apiCtx(t, ep.tenant), req)
	require.NoError(t, err, "SendDownlink %016X", ep.eui)
	queID, err := strconv.ParseInt(rsp.GetId(), 10, 64)
	require.NoError(t, err, "SendDownlink queue id %q", rsp.GetId())
	return queID
}

// queueEntry returns the in-flight queue row of the downlink, nil once it
// left the queue.
func queueEntry(t *testing.T, ep testEndpoint, queID int64) *pb.DownlinkMessage {
	t.Helper()
	rsp, err := coreClient(t).ListDownlinkQueue(apiCtx(t, ep.tenant),
		&pb.ListDownlinkQueueRequest{EpEui: euiHex(ep.eui), QueId: &queID})
	require.NoError(t, err, "ListDownlinkQueue %d", queID)
	for _, m := range rsp.GetMessages() {
		if m.GetQueId() == queID {
			return m
		}
	}
	return nil
}

// downlinkResult returns the terminal result row of the downlink, nil while
// it has none.
func downlinkResult(t *testing.T, ep testEndpoint, queID int64) *pb.DownlinkMessage {
	t.Helper()
	rsp, err := coreClient(t).GetDownlinkResults(apiCtx(t, ep.tenant),
		&pb.GetDownlinkResultsRequest{EpEui: euiHex(ep.eui), QueId: &queID})
	require.NoError(t, err, "GetDownlinkResults %d", queID)
	for _, m := range rsp.GetResults() {
		if m.GetQueId() == queID {
			return m
		}
	}
	return nil
}

// awaitDownlinkResult polls for the downlink's terminal result.
func awaitDownlinkResult(t *testing.T, ep testEndpoint, queID int64) *pb.DownlinkMessage {
	t.Helper()
	var res *pb.DownlinkMessage
	eventually(func() bool {
		res = downlinkResult(t, ep, queID)
		return res != nil
	})
	return res
}
