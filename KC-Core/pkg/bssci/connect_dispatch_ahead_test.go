package bssci_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// The connecting station, an endpoint it serves and one another station serves.
const (
	aheadStation         = uint64(0x70B3D59CD00009E6)
	aheadOtherStation    = uint64(0x70B3D59CD00009E7)
	aheadServedEndpoint  = uint64(0x70B3D56770111505)
	aheadForeignEndpoint = uint64(0x70B3D56770111506)
	aheadServedQueue     = uint64(7_300_000_000_000_301)
	aheadForeignQueue    = uint64(7_300_000_000_000_302)
	aheadWait            = 2 * time.Second
	aheadPoll            = 10 * time.Millisecond
)

// servingByEndpoint names the station serving each endpoint.
type servingByEndpoint map[uint64]uint64

func (s servingByEndpoint) ServingStation(_ context.Context, _ int64, epEUI uint64) (uint64, bool, error) {
	station, known := s[epEUI]
	return station, known, nil
}

// connectingStationConn captures the frames a connecting station is sent.
type connectingStationConn struct {
	*testutil.TestConn
}

func (connectingStationConn) Close() error                     { return nil }
func (connectingStationConn) SetDeadline(time.Time) error      { return nil }
func (connectingStationConn) SetReadDeadline(time.Time) error  { return nil }
func (connectingStationConn) SetWriteDeadline(time.Time) error { return nil }
func (connectingStationConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: mockConnLocalPort}
}
func (connectingStationConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: mockConnRemotePort}
}

// TestConnectedStationIsSentThePendingDownlinksItServes: a downlink queued
// while the station serving its endpoint was offline waits pending. Right
// after the station completes its connect it is sent the downlink as a
// dlDataQue, ahead of the endpoint's next uplink (BSSCI §5.12: a queue
// operation may schedule downlink data a priori); a downlink of an endpoint
// another station serves stays pending.
func TestConnectedStationIsSentThePendingDownlinksItServes(t *testing.T) {
	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(log, nil)
	server := bssci.NewTestServer(log, nil, nil, ulTestTenantID, sessionSvc, downlinkSvc, statusSvc,
		&mockConnectionService{tenantID: ulTestTenantID}, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.SetConfig(&bssci.Config{
		ServiceCenterEUI: bssci.TestScEui01, Vendor: "Test Vendor", Model: "Test Model", Name: "Test SC", SoftwareVersion: "1.0.0",
		StatusRequestInitialDelay: time.Hour, StatusRequestInterval: time.Hour,
	})
	queue := newStationQueue()
	queue.addPending(aheadServedQueue, aheadServedEndpoint)
	queue.addPending(aheadForeignQueue, aheadForeignEndpoint)
	dispatcher, err := bssciservices.NewDownlinkDispatcher(log, queue, queue,
		&ulWindowClaims{claimed: map[ulWindowKey]bool{}}, server.SendDLDataQueue, clock.SystemClock{})
	require.NoError(t, err)
	reclaimer, err := bssciservices.NewDownlinkReclaimer(queue, &requeueLog{}, log)
	require.NoError(t, err)
	server.SetDownlinkDispatcher(dispatcher)
	server.SetDownlinkReclaimer(reclaimer)
	server.SetPendingDownlinks(queue)
	server.SetServingStations(servingByEndpoint{aheadServedEndpoint: aheadStation, aheadForeignEndpoint: aheadOtherStation})

	conn := connectingStationConn{&testutil.TestConn{Encoding: mioty.EncodingMessagePack}}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID: "ahead-of-window-station", BaseStationEUI: aheadStation, SessionUUID: make([]byte, 16), ResolvedTenantID: ulTestTenantID,
		},
		Conn: conn,
	}
	connect := map[string]interface{}{"version": mioty.MIOTYProtocolVersion, "bsEui": int64(aheadStation), "bidi": true} //nolint:gosec // test EUI fits int64
	require.NoError(t, server.CallHandleConnect(session, &bssci.Message{OpId: 0, Command: mioty.CmdConnect, Data: connect}, connect))
	require.NoError(t, server.CallHandleConnectComplete(session, &bssci.Message{OpId: 0, Command: mioty.CmdConnectComplete}, map[string]interface{}{}))

	require.Eventually(t, func() bool { return len(sentQueueIDs(t, conn.TestConn)) > 0 }, aheadWait, aheadPoll,
		"the pending downlink goes out right after the connect completes, before any uplink")
	assert.Equal(t, []uint64{aheadServedQueue}, sentQueueIDs(t, conn.TestConn))
	assert.Equal(t, mioty.DLQueueStatusQueued, queue.row(aheadServedQueue).status)
	assert.Equal(t, mioty.DLQueueStatusPending, queue.row(aheadForeignQueue).status, "another station serves that endpoint")
}
