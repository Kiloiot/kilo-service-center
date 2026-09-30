package bssci

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	// testStalledSendBound is how long a test waits before declaring a send blocked.
	testStalledSendBound = 2 * time.Second
	// testSocketWriteTimeout is the frame write bound the stalled-peer tests configure.
	testSocketWriteTimeout = 50 * time.Millisecond
)

// A base station that stops reading must not hold the sender past the write
// timeout; the failure is ambiguous so the caller closes and relies on resume.
func TestSendMessage_StalledBaseStationReturnsWithinWriteTimeout(t *testing.T) {
	serverConn, stalledPeer := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = stalledPeer.Close()
	})

	frames, err := nettransport.NewFrameCodec(mioty.MIOTYFrameIdentifier, dbconfig.MaxMessageSize, testSocketWriteTimeout)
	require.NoError(t, err)
	server := &Server{
		clock:    clock.SystemClock{},
		config:   &Config{},
		logger:   logger.NewNop(),
		commands: mustTestCommandRegistry(),
		frames:   frames,
	}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{Encoding: EncodingMessagePack, HandshakeComplete: true},
		Conn:                 serverConn,
	}

	done := make(chan error, 1)
	go func() {
		done <- server.sendMessage(session, map[string]interface{}{"command": mioty.CmdPing, "opId": int64(-1)})
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrAmbiguousWrite)
		require.ErrorIs(t, err, os.ErrDeadlineExceeded)
	case <-time.After(testStalledSendBound):
		t.Fatal("sendMessage blocked on a base station that stopped reading")
	}
}

// A server takes the frame write bound from its configuration and substitutes
// none: a missing configuration or a bound that is not positive is refused.
func TestNewServerRefusesAMissingOrNonPositiveWriteTimeout(t *testing.T) {
	deps := Dependencies{Protocol: ProtocolServices{Status: struct{ StatusService }{}}}

	_, err := NewServer(nil, logger.NewNop(), deps)
	require.ErrorIs(t, err, errServerConfigRequired)
	for _, timeout := range []time.Duration{0, -testSocketWriteTimeout} {
		_, err := NewServer(&Config{SocketWriteTimeout: timeout}, logger.NewNop(), deps)
		require.ErrorIs(t, err, nettransport.ErrWriteTimeoutRequired, "write timeout %s", timeout)
	}
}
