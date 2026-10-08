package scaci

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	// testBroadcastBound is how long a test waits before declaring a broadcast blocked.
	testBroadcastBound = 2 * time.Second
	// testSocketWriteTimeout is the frame write bound the stalled-peer test configures.
	testSocketWriteTimeout = 50 * time.Millisecond
)

// The frame write bound comes from the configuration alone (validated there):
// the server has no default of its own to fall back to, so a server without
// a bound is not built.
func TestNewServer_RefusesAConfigurationWithoutAWriteBound(t *testing.T) {
	_, err := newServerThroughPersistence(stubOrgDirectory{}, stubSnapshotSource{}, stubEndpointPropagator{},
		NewErrorRecorder(nil, nil, testLogger()), &sessionEventLog{})

	require.ErrorIs(t, err, nettransport.ErrWriteTimeoutRequired)
}

type frameReadResult struct {
	frame *mioty.Frame
	err   error
}

// An Application Center that stops reading must neither stall the broadcast
// nor starve the healthy session, and its connection is closed.
func TestBroadcastULData_StalledSessionDoesNotBlockHealthySession(t *testing.T) {
	t.Parallel()
	stalledConn, stalledPeer := net.Pipe()
	healthyConn, healthyPeer := net.Pipe()
	t.Cleanup(func() {
		for _, c := range []net.Conn{stalledConn, stalledPeer, healthyConn, healthyPeer} {
			_ = c.Close()
		}
	})

	received := make(chan frameReadResult, 1)
	go func() {
		frame, err := testFrameCodec.Read(healthyPeer)
		received <- frameReadResult{frame: frame, err: err}
	}()

	server := newBroadcastULDataServer(map[net.Conn]*Session{
		stalledConn: activeBroadcastSession(broadcastULDataTestTenant),
		healthyConn: activeBroadcastSession(broadcastULDataTestTenant),
	})
	server.codec = mustFrameCodec(testSocketWriteTimeout)

	done := make(chan error, 1)
	go func() {
		done <- server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, broadcastULDataFixture())
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, errBroadcastSessionFailed, "the stalled session fails the broadcast so the outbox retries")
	case <-time.After(testBroadcastBound):
		t.Fatal("BroadcastULData blocked on an Application Center that stopped reading")
	}

	select {
	case got := <-received:
		require.NoError(t, got.err)
		assert.NotEmpty(t, got.frame.Payload, "the healthy session receives the uplink")
	case <-time.After(testBroadcastBound):
		t.Fatal("the healthy session never received the uplink")
	}

	peerRead := make(chan error, 1)
	go func() {
		_, err := stalledPeer.Read(make([]byte, mioty.FrameHeaderSize))
		peerRead <- err
	}()
	select {
	case err := <-peerRead:
		assert.ErrorIs(t, err, io.EOF, "the stalled connection is closed after its write timed out")
	case <-time.After(testBroadcastBound):
		t.Fatal("the stalled connection was left open after its write timed out")
	}
}
