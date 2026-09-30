package bssci

import (
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

var errPeerReset = errors.New("connection reset by peer")

// resetConn is a plain link whose peer resets it on the first read.
type resetConn struct{ net.Conn }

func (resetConn) Read([]byte) (int, error)        { return 0, errPeerReset }
func (resetConn) Close() error                    { return nil }
func (resetConn) SetReadDeadline(time.Time) error { return nil }

// resetPeerPort is the source port of the peer that resets the link.
const resetPeerPort = 40000

func (resetConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: resetPeerPort}
}

// A peer that is not a base station speaking TLS, or that drops the link, is
// a warning with its address; it says nothing about the service center.
func TestConnection_PeerFailuresAreWarningsWithTheRemoteAddress(t *testing.T) {
	t.Run("TLS handshake", func(t *testing.T) {
		log := newRecordingLogger()
		server := newSendFailureServer(t, log)
		serverSide, clientSide := net.Pipe()
		go func() {
			_, _ = clientSide.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
			_ = clientSide.Close()
		}()

		server.wg.Add(1)
		server.handleConnection(tls.Server(serverSide, &tls.Config{MinVersion: tls.VersionTLS12}))

		logged := entriesWithMessage(log, LogBSSCITLSHandshakeFailed)
		require.Len(t, logged, 1)
		assert.Equal(t, "WARN", logged[0].level)
		assert.Equal(t, serverSide.RemoteAddr().String(), logged[0].fields["remote"])
	})

	t.Run("read after the peer reset the link", func(t *testing.T) {
		log := newRecordingLogger()
		server := newSendFailureServer(t, log)
		server.ctx = testutil.TestContext()

		server.wg.Add(1)
		server.handleConnection(resetConn{})

		logged := entriesWithMessage(log, LogBSSCIFailedToReadFrame)
		require.Len(t, logged, 1)
		assert.Equal(t, "WARN", logged[0].level)
		assert.Equal(t, "203.0.113.7:40000", logged[0].fields["remote"])
		assert.ErrorIs(t, logged[0].fields["error"].(error), errPeerReset)
	})
}
