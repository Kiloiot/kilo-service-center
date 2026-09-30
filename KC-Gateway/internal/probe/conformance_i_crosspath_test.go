//go:build integration

package probe

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
)

// MQTT command and event topics of the SC (KC-MQTT constants; the MQTT
// ingress is SC-specific, not part of the MIOTY specifications).
const (
	mqttCommandTopicFmt = "%s/%s/device/%016x/command/down"
	mqttEventTopicFmt   = "%s/%s/device/%016x/event/#"
	mqttResultSuffix    = "/event/downlink_result"
	mqttKeyData         = "data"
	mqttKeyConfirmed    = "confirmed"
	mqttKeyQueID        = "queId"
	mqttQoS             = 1
	mqttDisconnectMs    = 250
)

type mqttMessage struct {
	topic   string
	payload map[string]interface{}
}

// mqttTap is an MQTT client subscribed to one device's event topics.
type mqttTap struct {
	client paho.Client
	mu     sync.Mutex
	msgs   []mqttMessage
}

func newMQTTTap(t *testing.T, eui uint64) *mqttTap {
	t.Helper()
	opts := paho.NewClientOptions().AddBroker("tcp://" + stackEnv(t, envMQTTAddr)).
		SetClientID(fmt.Sprintf("conformance-%d", time.Now().UnixNano())).SetConnectTimeout(dialTimeout)
	tap := &mqttTap{client: paho.NewClient(opts)}
	tok := tap.client.Connect()
	require.True(t, tok.WaitTimeout(dialTimeout), "MQTT connect timed out")
	require.NoError(t, tok.Error(), "MQTT connect")
	t.Cleanup(func() { tap.client.Disconnect(mqttDisconnectMs) })
	topic := fmt.Sprintf(mqttEventTopicFmt, stackEnv(t, envMQTTPrefix), tenantOrg(t, tenantPrimary), eui)
	tok = tap.client.Subscribe(topic, mqttQoS, func(_ paho.Client, m paho.Message) {
		var payload map[string]interface{}
		dec := json.NewDecoder(bytes.NewReader(m.Payload()))
		dec.UseNumber()
		_ = dec.Decode(&payload)
		tap.mu.Lock()
		tap.msgs = append(tap.msgs, mqttMessage{topic: m.Topic(), payload: payload})
		tap.mu.Unlock()
	})
	require.True(t, tok.WaitTimeout(dialTimeout), "MQTT subscribe timed out")
	require.NoError(t, tok.Error(), "MQTT subscribe")
	return tap
}

func (m *mqttTap) messages() []mqttMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mqttMessage(nil), m.msgs...)
}

// publishDownlink publishes an MQTT downlink command for the endpoint.
func (m *mqttTap) publishDownlink(t *testing.T, eui uint64, data []byte, confirmed bool) {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{mqttKeyData: base64.StdEncoding.EncodeToString(data), mqttKeyConfirmed: confirmed})
	require.NoError(t, err)
	topic := fmt.Sprintf(mqttCommandTopicFmt, stackEnv(t, envMQTTPrefix), tenantOrg(t, tenantPrimary), eui)
	tok := m.client.Publish(topic, mqttQoS, false, body)
	require.True(t, tok.WaitTimeout(dialTimeout), "MQTT publish timed out")
	require.NoError(t, tok.Error(), "MQTT publish")
}

// I2 BSSCI §3.12.1: an MQTT downlink becomes a spec-correct dlDataQue, with
// confirmed mapped to responseExp.
func TestConformanceI2_MQTTDownlink(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	tap := newMQTTTap(t, ep.eui)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	tap.publishDownlink(t, ep.eui, []byte{0x01}, true)
	que := bs.awaitCommand(t, from, cmdDLDataQue, forEndpoint(ep.eui))
	exp, _ := que.bool(keyResponseExp)
	require.True(t, exp, "dlDataQue.responseExp for confirmed=true: %v", que.fields)
	require.Equal(t, [][]byte{{0x01}}, userDataEntries(t, que))
}

// I3 originator feedback (no spec section): an MQTT downlink the SC refuses
// (201 bytes, over the radio limit) is reported to the publisher on MQTT.
func TestConformanceI3_MQTTRejectionReported(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary)
	tap := newMQTTTap(t, ep.eui)
	tap.publishDownlink(t, ep.eui, make([]byte, oversizedRadioLen), false)
	require.True(t, eventually(func() bool { return len(tap.messages()) > 0 }),
		"no MQTT event tells the publisher the downlink was refused")
}

// I4 BSSCI §3.12 l.610, SCACI §3.10 l.499: every ingress accepts an empty
// userData as a pure acknowledgement (the SCACI forms are F7).
func TestConformanceI4_EmptyUserDataAllIngresses(t *testing.T) {
	t.Run("grpc", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary, preAttached)
		bs := connectServing(t, stationA, ep)
		from := bs.mark()
		que := bs.dlDataQueFor(t, from, ep, queueDownlink(t, ep, &pb.SendDownlinkRequest{}))
		require.Equal(t, [][]byte{{}}, userDataEntries(t, que))
	})
	t.Run("mqtt", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary, preAttached)
		tap := newMQTTTap(t, ep.eui)
		bs := connectServing(t, stationA, ep)
		from := bs.mark()
		tap.publishDownlink(t, ep.eui, nil, false)
		que := bs.awaitCommand(t, from, cmdDLDataQue, forEndpoint(ep.eui))
		require.Equal(t, [][]byte{{}}, userDataEntries(t, que))
	})
}

// I6 results back to MQTT (no spec section): the MQTT originator receives its
// downlink's result on the queuing organization's topic, under an id it was
// given when the downlink was queued.
func TestConformanceI6_MQTTResultCorrelatable(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	tap := newMQTTTap(t, ep.eui)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	tap.publishDownlink(t, ep.eui, []byte{0x06}, false)
	que := bs.awaitCommand(t, from, cmdDLDataQue, forEndpoint(ep.eui))
	queID, _ := que.int(keyQueID)
	requireRsp(t, bs.request(t, dlDataResSent(ep.eui, queID, uplinkCounter)), cmdDLDataRes)
	want := strconv.FormatInt(queID, 10)
	var msgs []mqttMessage
	resultAt := -1
	require.True(t, eventually(func() bool {
		msgs = tap.messages()
		for i, m := range msgs {
			if strings.HasSuffix(m.topic, mqttResultSuffix) && fmt.Sprint(m.payload[mqttKeyQueID]) == want {
				resultAt = i
				return true
			}
		}
		return false
	}), "no downlink_result on the queuing organization's topic")
	announced := false
	for _, m := range msgs[:resultAt] {
		announced = announced || fmt.Sprint(m.payload[mqttKeyQueID]) == want
	}
	require.True(t, announced, "the publisher was never told queue id %s, so it cannot correlate the result", want)
}

// I9 tenant isolation (CLAUDE.md): a foreign-tenant station cannot write the
// result of another tenant's downlink under a wrong endpoint.
func TestConformanceI9_ForeignResultRefused(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	bs := connectServing(t, stationA, ep)
	from := bs.mark()
	queID := queueDownlink(t, ep, dlRequest([]byte{0x09}))
	bs.dlDataQueFor(t, from, ep, queID)
	bs.settle(cmdDLDataQueCmp)
	roamer := connectStation(t, stationRoamer)
	requireProtocolError(t, roamer.request(t, dlDataResSent(nextEndpointEUI(), queID, uplinkCounter)))
	row := queueEntry(t, ep, queID)
	require.NotNil(t, row, "the downlink left the queue")
	require.Equal(t, queuedStatus, row.GetStatus())
	require.Empty(t, row.GetResult())
}

// I11 BSSCI §3.14, tenant isolation (CLAUDE.md): only the station holding a
// downlink reports its result. Another station naming the downlink's own
// endpoint and queue id, or the station it was taken from, is refused and
// leaves the downlink queued at its holder without telling any originator;
// the holder, of the owner's tenant or roaming from another, completes it.
func TestConformanceI11_ResultOnlyFromHoldingStation(t *testing.T) {
	t.Run("another station with the downlink's ids", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary, preAttached)
		tap := newMQTTTap(t, ep.eui)
		bsA := connectServing(t, stationA, ep)
		from := bsA.mark()
		queID := queueDownlink(t, ep, dlRequest([]byte{0x11}))
		bsA.dlDataQueFor(t, from, ep, queID)
		bsA.settle(cmdDLDataQueCmp)
		bsB := connectStation(t, stationB)
		requireProtocolError(t, bsB.request(t, dlDataResSent(ep.eui, queID, uplinkCounter)))
		requireHeldWithoutResult(t, ep, queID, stationA, tap)
		requireRsp(t, bsA.request(t, dlDataResSent(ep.eui, queID, uplinkCounter)), cmdDLDataRes)
		requireReportedOnce(t, tap, queID)
	})
	t.Run("the station the downlink was taken from", testLateResultAfterReassignment)
	t.Run("a roaming holder of another tenant", func(t *testing.T) {
		ep := newEndpoint(t, tenantPrimary)
		tap := newMQTTTap(t, ep.eui)
		roamer := connectStation(t, stationRoamer)
		requireRsp(t, roamer.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01})), cmdULData)
		from := roamer.mark()
		queID := queueDownlink(t, ep, dlRequest([]byte{0x13}))
		roamer.dlDataQueFor(t, from, ep, queID)
		roamer.settle(cmdDLDataQueCmp)
		requireRsp(t, roamer.request(t, dlDataResSent(ep.eui, queID, uplinkCounter)), cmdDLDataRes)
		requireReportedOnce(t, tap, queID)
	})
}

// testLateResultAfterReassignment: while station A is away the endpoint is
// heard at B, A's fresh session returns the Application Center's downlink to
// the queue, B takes it at the endpoint's next window, and A's late result
// reaches neither the Application Center nor MQTT; B's result reaches both,
// naming B.
func testLateResultAfterReassignment(t *testing.T) {
	ep := newEndpoint(t, tenantPrimary, preAttached)
	tap := newMQTTTap(t, ep.eui)
	ac := newAC(t)
	bsA := connectServing(t, stationA, ep)
	from := bsA.mark()
	requireRsp(t, ac.request(t, acDLDataQue(ep, acQueID(), []byte{0x12})), cmdDLDataQue)
	queID, _ := bsA.awaitCommand(t, from, cmdDLDataQue, forEndpoint(ep.eui)).int(keyQueID)
	bsA.awaitCommand(t, from, cmdDLDataQueCmp, nil)
	dropAbortively(bsA.conn)
	bsB := connectStation(t, stationB)
	requireRsp(t, bsB.request(t, ulDataMsg(ep, uplinkCounter, []byte{0x01})), cmdULData)
	staleA := connectStation(t, stationA)
	staleA.assertQuiet(t, 0, cmdDLDataQue, forQueue(ep.eui, queID))
	from = bsB.mark()
	requireRsp(t, bsB.request(t, ulDataOpen(ep, uplinkCounter+1, []byte{0x02})), cmdULData)
	bsB.dlDataQueFor(t, from, ep, queID)
	bsB.awaitCommand(t, from, cmdDLDataQueCmp, nil)

	fromAC := ac.mark()
	requireProtocolError(t, staleA.request(t, dlDataResSent(ep.eui, queID, uplinkCounter)))
	ac.assertQuiet(t, fromAC, cmdDLDataRes, forEndpoint(ep.eui))
	requireHeldWithoutResult(t, ep, queID, stationB, tap)
	requireRsp(t, bsB.request(t, dlDataResSent(ep.eui, queID, uplinkCounter)), cmdDLDataRes)
	res := ac.awaitCommand(t, fromAC, cmdDLDataRes, forEndpoint(ep.eui))
	bsEui, _ := res.uint(keyBsEui)
	require.Equal(t, stationB.eui, bsEui, "the result names the holder as transmitter: %v", res.fields)
	requireReportedOnce(t, tap, queID)
}

// requireHeldWithoutResult asserts the downlink is still queued at the
// station with no result recorded, and that no MQTT result was published.
func requireHeldWithoutResult(t *testing.T, ep testEndpoint, queID int64, holder station, tap *mqttTap) {
	t.Helper()
	row := queueEntry(t, ep, queID)
	require.NotNil(t, row, "the downlink left the queue")
	require.Equal(t, queuedStatus, row.GetStatus())
	require.Equal(t, euiHex(holder.eui), row.GetBsEui(), "the downlink changed holder")
	require.Empty(t, row.GetResult())
	time.Sleep(quietWindow)
	require.Zero(t, mqttResults(tap, queID), "a refused result reached MQTT")
}

// requireReportedOnce asserts the downlink's result reached MQTT exactly once.
func requireReportedOnce(t *testing.T, tap *mqttTap, queID int64) {
	t.Helper()
	require.True(t, eventually(func() bool { return mqttResults(tap, queID) > 0 }), "no downlink_result for queue id %d", queID)
	time.Sleep(quietWindow)
	require.Equal(t, 1, mqttResults(tap, queID), "downlink_result count for queue id %d", queID)
}

// mqttResults counts the downlink_result events published for the queue id.
func mqttResults(tap *mqttTap, queID int64) int {
	want := strconv.FormatInt(queID, 10)
	count := 0
	for _, m := range tap.messages() {
		if strings.HasSuffix(m.topic, mqttResultSuffix) && fmt.Sprint(m.payload[mqttKeyQueID]) == want {
			count++
		}
	}
	return count
}
