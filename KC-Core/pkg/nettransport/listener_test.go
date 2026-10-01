package nettransport

import (
	"context"
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testDialTimeout      = 5 * time.Second
	testPollInterval     = 20 * time.Millisecond
	testWaitTimeout      = 5 * time.Second
	testWaitStep         = 10 * time.Millisecond
	testLoopbackAddr     = "127.0.0.1:0"
	testListenerName     = "TEST"
	testHandlerHelloByte = 0x42
)

func newTestListener(pki *testPKI, minVersion string, poll time.Duration, onListening func(), handle ConnHandler) *Listener {
	if handle == nil {
		handle = handshakeAndClose
	}
	return NewListener(ListenerConfig{
		Name:             testListenerName,
		Addr:             testLoopbackAddr,
		Files:            pki.files,
		TLS:              TLSOptions{MinVersion: minVersion},
		CertPollInterval: poll,
		OnListening:      onListening,
	}, logger.NewNop(), handle)
}

// handshakeAndClose completes the server side of the TLS handshake, which
// tls.Listen defers to the first read, and then closes the connection.
func handshakeAndClose(conn net.Conn) {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		_ = tlsConn.Handshake()
	}
	_ = conn.Close()
}

// handshake dials the listener with a client capped at maxVersion and returns
// the negotiated protocol version, or the handshake error.
func handshake(pki *testPKI, addr string, maxVersion uint16) (uint16, error) {
	dialer := &net.Dialer{Timeout: testDialTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		RootCAs:      pki.caPool,
		Certificates: []tls.Certificate{pki.clientCert},
		ServerName:   testServerName,
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   maxVersion,
	})
	if err != nil {
		return 0, err
	}
	defer func() { _ = conn.Close() }()
	return conn.ConnectionState().Version, nil
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(testWaitTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(testWaitStep)
	}
	t.Fatal(msg)
}

func TestListenerFloorFollowsConfig(t *testing.T) {
	pki := newTestPKI(t)

	t.Run("floor 1.2 admits a TLS 1.2 client and still prefers 1.3", func(t *testing.T) {
		l := newTestListener(pki, "1.2", testPollInterval, nil, nil)
		require.NoError(t, l.Start(testutil.TestContext()))
		t.Cleanup(l.Stop)
		addr := l.Addr().String()

		version, err := handshake(pki, addr, tls.VersionTLS12)
		require.NoError(t, err)
		assert.Equal(t, uint16(tls.VersionTLS12), version)

		version, err = handshake(pki, addr, tls.VersionTLS13)
		require.NoError(t, err)
		assert.Equal(t, uint16(tls.VersionTLS13), version)
	})

	t.Run("floor 1.3 rejects a TLS 1.2 client", func(t *testing.T) {
		l := newTestListener(pki, "1.3", testPollInterval, nil, nil)
		require.NoError(t, l.Start(testutil.TestContext()))
		t.Cleanup(l.Stop)
		addr := l.Addr().String()

		_, err := handshake(pki, addr, tls.VersionTLS12)
		require.Error(t, err, "a client capped at TLS 1.2 must not pass a 1.3 floor")

		version, err := handshake(pki, addr, tls.VersionTLS13)
		require.NoError(t, err)
		assert.Equal(t, uint16(tls.VersionTLS13), version)
	})

	t.Run("blank floor follows the config parser default", func(t *testing.T) {
		l := newTestListener(pki, "", testPollInterval, nil, nil)
		require.NoError(t, l.Start(testutil.TestContext()))
		t.Cleanup(l.Stop)

		version, err := handshake(pki, l.Addr().String(), tls.VersionTLS12)
		require.NoError(t, err)
		assert.Equal(t, uint16(tls.VersionTLS12), version)
	})

	t.Run("unsupported floor binds nothing", func(t *testing.T) {
		l := newTestListener(pki, "1.1", testPollInterval, nil, nil)
		err := l.Start(testutil.TestContext())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TLS configuration")
		assert.Nil(t, l.Addr(), "no listener may be bound with an unsupported floor")
		l.Stop()
	})
}

func TestListenerHandsConnectionsToTheHandler(t *testing.T) {
	pki := newTestPKI(t)
	var listening atomic.Int32
	received := make(chan byte, 1)
	handle := func(conn net.Conn) {
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 1)
		if _, err := conn.Read(buf); err == nil {
			received <- buf[0]
		}
	}
	l := newTestListener(pki, "1.2", testPollInterval, func() { listening.Add(1) }, handle)
	require.NoError(t, l.Start(testutil.TestContext()))
	t.Cleanup(l.Stop)
	assert.Equal(t, int32(1), listening.Load(), "OnListening runs once the socket is bound")

	conn, err := tls.Dial("tcp", l.Addr().String(), &tls.Config{
		RootCAs:      pki.caPool,
		Certificates: []tls.Certificate{pki.clientCert},
		ServerName:   testServerName,
		MinVersion:   tls.VersionTLS12,
	})
	require.NoError(t, err)
	_, err = conn.Write([]byte{testHandlerHelloByte})
	require.NoError(t, err)
	select {
	case b := <-received:
		assert.Equal(t, byte(testHandlerHelloByte), b)
	case <-time.After(testWaitTimeout):
		t.Fatal("handler did not receive the connection")
	}
	_ = conn.Close()
}

func TestListenerRejectsClientWithoutCertificate(t *testing.T) {
	pki := newTestPKI(t)
	l := newTestListener(pki, "1.2", testPollInterval, nil, nil)
	require.NoError(t, l.Start(testutil.TestContext()))
	t.Cleanup(l.Stop)

	conn, err := tls.Dial("tcp", l.Addr().String(), &tls.Config{
		RootCAs:    pki.caPool,
		ServerName: testServerName,
		MinVersion: tls.VersionTLS12,
	})
	if err == nil {
		// TLS 1.3 reports the missing client certificate on the first read.
		_, err = conn.Read(make([]byte, 1))
		_ = conn.Close()
	}
	require.Error(t, err, "mutual TLS must refuse a client without a certificate")
}

func TestListenerDefersUntilCertificatesAppear(t *testing.T) {
	pki := newTestPKI(t)
	staged := TLSFiles{
		Cert: pki.files.Cert + ".staged",
		Key:  pki.files.Key + ".staged",
		CA:   pki.files.CA + ".staged",
	}
	for _, pair := range [][2]string{{pki.files.Cert, staged.Cert}, {pki.files.Key, staged.Key}, {pki.files.CA, staged.CA}} {
		require.NoError(t, os.Rename(pair[0], pair[1]))
	}

	var listening atomic.Int32
	l := newTestListener(pki, "1.2", testPollInterval, func() { listening.Add(1) }, nil)
	require.NoError(t, l.Start(testutil.TestContext()), "missing certificates defer the listener instead of failing Start")
	t.Cleanup(l.Stop)
	assert.Nil(t, l.Addr())
	time.Sleep(testPollInterval * 3)
	assert.Nil(t, l.Addr(), "nothing binds while the files are absent")
	assert.Equal(t, int32(0), listening.Load())

	for _, pair := range [][2]string{{staged.Cert, pki.files.Cert}, {staged.Key, pki.files.Key}, {staged.CA, pki.files.CA}} {
		require.NoError(t, os.Rename(pair[0], pair[1]))
	}
	// The socket is published before OnListening runs, so wait for the callback.
	waitFor(t, func() bool { return listening.Load() == 1 }, "listener did not bind after the certificates appeared")
	require.NotNil(t, l.Addr())
	_, err := handshake(pki, l.Addr().String(), tls.VersionTLS13)
	require.NoError(t, err)
}

func TestListenerDeferredPollStopsWithStop(t *testing.T) {
	pki := newTestPKI(t)
	require.NoError(t, os.Remove(pki.files.CA))
	l := newTestListener(pki, "1.2", testPollInterval, nil, nil)
	require.NoError(t, l.Start(testutil.TestContext()))
	done := make(chan struct{})
	go func() {
		l.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(testWaitTimeout):
		t.Fatal("Stop did not end the certificate poll")
	}
}

func TestListenerLifecycle(t *testing.T) {
	pki := newTestPKI(t)

	t.Run("Start requires a positive poll interval", func(t *testing.T) {
		l := newTestListener(pki, "1.2", 0, nil, nil)
		require.ErrorIs(t, l.Start(testutil.TestContext()), ErrInvalidCertPollInterval)
	})
	t.Run("Stop is idempotent and safe before Start", func(t *testing.T) {
		l := newTestListener(pki, "1.2", testPollInterval, nil, nil)
		l.Stop()
		l.Stop()
		require.ErrorIs(t, l.Start(testutil.TestContext()), ErrListenerStopped)
	})
	t.Run("a second Start is refused", func(t *testing.T) {
		l := newTestListener(pki, "1.2", testPollInterval, nil, nil)
		require.NoError(t, l.Start(testutil.TestContext()))
		t.Cleanup(l.Stop)
		require.ErrorIs(t, l.Start(testutil.TestContext()), ErrListenerAlreadyStarted)
	})
	t.Run("Stop closes the socket and drains handlers", func(t *testing.T) {
		release := make(chan struct{})
		entered := make(chan struct{}, 1)
		handle := func(conn net.Conn) {
			if tlsConn, ok := conn.(*tls.Conn); ok {
				_ = tlsConn.Handshake()
			}
			entered <- struct{}{}
			<-release
			_ = conn.Close()
		}
		l := newTestListener(pki, "1.2", testPollInterval, nil, handle)
		require.NoError(t, l.Start(testutil.TestContext()))
		addr := l.Addr().String()
		conn, err := tls.Dial("tcp", addr, &tls.Config{
			RootCAs:      pki.caPool,
			Certificates: []tls.Certificate{pki.clientCert},
			ServerName:   testServerName,
			MinVersion:   tls.VersionTLS12,
		})
		require.NoError(t, err)
		require.NoError(t, conn.Handshake())
		<-entered

		stopped := make(chan struct{})
		go func() {
			l.Stop()
			close(stopped)
		}()
		select {
		case <-stopped:
			t.Fatal("Stop returned while a handler was still running")
		case <-time.After(testPollInterval * 3):
		}
		close(release)
		select {
		case <-stopped:
		case <-time.After(testWaitTimeout):
			t.Fatal("Stop did not return after the handler finished")
		}
		_ = conn.Close()
		_, err = net.DialTimeout("tcp", addr, testDialTimeout)
		require.Error(t, err, "the socket must be closed after Stop")
	})
	t.Run("a cancelled context ends the accept loop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(testutil.TestContext())
		l := newTestListener(pki, "1.2", testPollInterval, nil, nil)
		require.NoError(t, l.Start(ctx))
		cancel()
		l.Stop()
	})
}

func TestListenerSocketFile(t *testing.T) {
	pki := newTestPKI(t)
	// Binding a path that is not host:port fails in tls.Listen and must
	// surface from Start rather than from a background goroutine.
	l := NewListener(ListenerConfig{
		Name:             testListenerName,
		Addr:             filepath.Join(t.TempDir(), "not-a-port"),
		Files:            pki.files,
		TLS:              TLSOptions{MinVersion: "1.2"},
		CertPollInterval: testPollInterval,
	}, logger.NewNop(), handshakeAndClose)
	err := l.Start(testutil.TestContext())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create TLS listener")
	l.Stop()
}
