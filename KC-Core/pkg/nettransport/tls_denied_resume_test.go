package nettransport

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
)

// handshakeOverLoopback returns the server's handshake error and whether the session was resumed.
func handshakeOverLoopback(t *testing.T, serverCfg *tls.Config, pki *testPKI, cache tls.ClientSessionCache, version uint16) (error, bool) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		raw, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			return
		}
		_ = raw.SetDeadline(time.Now().Add(5 * time.Second))
		client := tls.Client(raw, &tls.Config{
			RootCAs:            pki.caPool,
			Certificates:       []tls.Certificate{pki.clientCert},
			ServerName:         testServerName,
			MinVersion:         version,
			MaxVersion:         version,
			ClientSessionCache: cache,
		})
		_ = client.Handshake()
		_, _ = client.Read(make([]byte, 1))
		_ = client.Close()
	}()

	conn, err := ln.Accept()
	require.NoError(t, err)
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	server := tls.Server(conn, serverCfg)
	err = server.Handshake()
	resumed := false
	if err == nil {
		resumed = server.ConnectionState().DidResume
		_, _ = server.Write([]byte("x"))
	}
	<-clientDone
	_ = server.Close()
	return err, resumed
}

func TestBuildServerTLSConfig_DeniedCertificateCannotResumeAnEarlierSession(t *testing.T) {
	for name, version := range map[string]uint16{"TLS 1.2": tls.VersionTLS12, "TLS 1.3": tls.VersionTLS13} {
		t.Run(name, func(t *testing.T) {
			pki := newTestPKI(t)
			fingerprint := crypto.CertFingerprintSHA256(pki.clientCert.Certificate[0])
			denying, err := BuildServerTLSConfig(pki.files, TLSOptions{MinVersion: "1.2", DeniedCertFingerprints: []string{fingerprint}})
			require.NoError(t, err)
			// Shared ticket keys make a ticket issued before the denial valid on the denying config.
			denying.SetSessionTicketKeys([][32]byte{{1}})
			before := denying.Clone()
			before.VerifyConnection = nil
			before.VerifyPeerCertificate = nil
			cache := tls.NewLRUClientSessionCache(4)

			err, _ = handshakeOverLoopback(t, before, pki, cache, version)
			require.NoError(t, err)

			err, resumed := handshakeOverLoopback(t, before, pki, cache, version)
			require.NoError(t, err)
			require.True(t, resumed, "the client must hold a resumable session for the check to mean anything")

			err, _ = handshakeOverLoopback(t, denying, pki, cache, version)
			require.ErrorIs(t, err, ErrDeniedClientCertificate)
		})
	}
}
