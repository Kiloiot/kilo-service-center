package nettransport

import (
	"crypto/tls"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
)

func deniedHandshake(t *testing.T, pki *testPKI, opts TLSOptions) error {
	t.Helper()
	return deniedHandshakeAt(t, pki, opts, tls.VersionTLS12)
}

func deniedHandshakeAt(t *testing.T, pki *testPKI, opts TLSOptions, version uint16) error {
	t.Helper()
	serverCfg, err := BuildServerTLSConfig(pki.files, opts)
	require.NoError(t, err)
	serverSide, clientSide := net.Pipe()
	t.Cleanup(func() { _ = serverSide.Close(); _ = clientSide.Close() })

	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		client := tls.Client(clientSide, &tls.Config{
			RootCAs:      pki.caPool,
			Certificates: []tls.Certificate{pki.clientCert},
			ServerName:   testServerName,
			MinVersion:   version,
			MaxVersion:   version,
		})
		_ = client.Handshake()
		_ = client.Close()
	}()
	err = tls.Server(serverSide, serverCfg).Handshake()
	_ = serverSide.Close()
	<-clientDone
	return err
}

func TestBuildServerTLSConfig_DeniedClientCertificate(t *testing.T) {
	pki := newTestPKI(t)
	clientFingerprint := crypto.CertFingerprintSHA256(pki.clientCert.Certificate[0])

	t.Run("a denied certificate is refused although the CA signed it", func(t *testing.T) {
		err := deniedHandshake(t, pki, TLSOptions{MinVersion: "1.2", DeniedCertFingerprints: []string{clientFingerprint}})

		require.ErrorIs(t, err, ErrDeniedClientCertificate)
	})
	t.Run("a denied certificate is refused over TLS 1.3", func(t *testing.T) {
		err := deniedHandshakeAt(t, pki, TLSOptions{MinVersion: "1.2", DeniedCertFingerprints: []string{clientFingerprint}}, tls.VersionTLS13)

		require.ErrorIs(t, err, ErrDeniedClientCertificate)
	})
	t.Run("the openssl form with colons and upper case is accepted", func(t *testing.T) {
		var b strings.Builder
		for i := 0; i < len(clientFingerprint); i += 2 {
			if i > 0 {
				b.WriteByte(':')
			}
			b.WriteString(strings.ToUpper(clientFingerprint[i : i+2]))
		}

		err := deniedHandshake(t, pki, TLSOptions{MinVersion: "1.2", DeniedCertFingerprints: []string{b.String()}})

		require.ErrorIs(t, err, ErrDeniedClientCertificate)
	})
	t.Run("another certificate still connects", func(t *testing.T) {
		err := deniedHandshake(t, pki, TLSOptions{MinVersion: "1.2", DeniedCertFingerprints: []string{strings.Repeat("0", 64)}})

		require.NoError(t, err)
	})
	t.Run("no list keeps the plain verification", func(t *testing.T) {
		cfg, err := BuildServerTLSConfig(pki.files, TLSOptions{MinVersion: "1.2"})

		require.NoError(t, err)
		assert.Nil(t, cfg.VerifyConnection)
	})
	t.Run("a malformed fingerprint stops the build", func(t *testing.T) {
		for _, bad := range []string{"", "abc", strings.Repeat("g", 64), strings.Repeat("0", 63)} {
			cfg, err := BuildServerTLSConfig(pki.files, TLSOptions{MinVersion: "1.2", DeniedCertFingerprints: []string{bad}})

			assert.Nil(t, cfg, bad)
			require.ErrorIs(t, err, ErrInvalidCertFingerprint, bad)
		}
	})
}
