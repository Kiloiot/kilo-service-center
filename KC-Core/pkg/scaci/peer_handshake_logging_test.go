package scaci

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
)

// A peer that does not speak TLS to the SCACI port is a warning with its
// address; it says nothing about the service center.
func TestHandshakeTLS_PeerFailureIsAWarningWithTheRemoteAddress(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	server := &Server{logger: log, config: &Config{}}
	serverSide, clientSide := net.Pipe()
	go func() {
		_, _ = clientSide.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
		_ = clientSide.Close()
	}()

	_, ok := server.handshakeTLS(tls.Server(serverSide, &tls.Config{MinVersion: tls.VersionTLS12}), time.Now().Add(time.Second))

	assert.False(t, ok)
	logged := log.FilterMessage(LogSCACITLSHandshakeFailed)
	require.Len(t, logged, 1)
	assert.Equal(t, "WARN", logged[0].Level)
	assert.Equal(t, serverSide.RemoteAddr().String(), logged[0].FieldMap()["remote"])
}
