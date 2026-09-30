package bssci

import (
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// bindingCertCN is the dashed EUI common name of the registered station
// TestBsEui01's certificate.
const bindingCertCN = "01-23-45-67-89-AB-CD-EF"

// communityConnect is a connect in the community edition, which has no
// organization enforcement, from a station presenting cert that names
// certEUI.
type communityConnect struct {
	server  *Server
	session *Session
	conn    *bsscitest.TestConn
}

func newCommunityConnect(t *testing.T, directory *fakeBSDirectory, cert *x509.Certificate, certEUI uint64) communityConnect {
	t.Helper()
	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	server := NewTestServer(log, storage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, interopConnectionService{}, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{
		MessageEncoding:  EncodingJSON,
		ServiceCenterEUI: TestBsEui02,
		Vendor:           "v", Model: "m", Name: "n", SoftwareVersion: "1.0.0",
	}
	bindStationCertificates(t, server, directory)
	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "community-binding",
			Encoding:         EncodingJSON,
			ResolvedTenantID: 1,
			ConnectState:     ConnectStateAwaitingConnect,
		},
		Conn:           conn,
		ClientCert:     cert,
		certSubjectEUI: &certEUI,
	}
	return communityConnect{server: server, session: session, conn: conn}
}

// claim sends the connect claiming to be the registered station TestBsEui01.
func (c communityConnect) claim(t *testing.T) {
	t.Helper()
	payload := connectPayload("1.0.0", uint64(TestBsEui01))
	require.NoError(t, c.server.CallHandleConnect(c.session, &Message{Command: mioty.CmdConnect, OpId: 0, Data: payload}, payload))
}

func (c communityConnect) assertRefused(t *testing.T, why string) {
	t.Helper()
	require.GreaterOrEqual(t, c.conn.MessageCount(), 1)
	frame := c.conn.GetMessage(c.conn.MessageCount() - 1)
	assert.Equal(t, mioty.CmdError, frame["command"], why)
	assert.Equal(t, ResolveErrorMessage(errBaseStationNotRegistered), frame["message"], "refused as an unregistered station would be")
	assert.Equal(t, ConnectStateAwaitingConnectErrorAck, c.session.ConnectState)
	assert.False(t, c.conn.SeenCommand(mioty.CmdConnectResponse), "no conRsp is offered")
}

// BSSCI §1 mutual TLS in the community edition: a station whose certificate
// names another station is refused, so it cannot take over the registered
// station's endpoints and downlinks.
func TestConnectCommunity_CertificateOfAnotherStationRefused(t *testing.T) {
	cert, _ := makeEnforcementCert(t, "11-11-11-11-11-11-11-11")
	c := newCommunityConnect(t, &fakeBSDirectory{}, cert, 0x1111111111111111)

	c.claim(t)

	c.assertRefused(t, "a certificate naming another station is refused")
}

// A certificate that names the station but is not the one pinned for it is
// refused in the community edition too.
func TestConnectCommunity_CertificateNotTheRegisteredOneRefused(t *testing.T) {
	cert, _ := makeEnforcementCert(t, bindingCertCN)
	pinned, _ := makeEnforcementCert(t, bindingCertCN)
	c := newCommunityConnect(t, &fakeBSDirectory{station: RegisteredBaseStation{ID: 7, TenantID: 1,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(pinned.Raw)}}, cert, uint64(TestBsEui01))

	c.claim(t)

	c.assertRefused(t, "a certificate other than the pinned one is refused")
}

// The registered station presenting its pinned certificate is admitted.
func TestConnectCommunity_RegisteredCertificateAdmitted(t *testing.T) {
	cert, _ := makeEnforcementCert(t, bindingCertCN)
	c := newCommunityConnect(t, &fakeBSDirectory{station: RegisteredBaseStation{ID: 7, TenantID: 1,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw)}}, cert, uint64(TestBsEui01))

	c.claim(t)

	assert.True(t, c.conn.SeenCommand(mioty.CmdConnectResponse), "the rightful station is offered conRsp")
}

// A station registered without a stored certificate is pinned to the first
// certificate that names it, so a later certificate that is not the same one
// is refused.
func TestConnectCommunity_FirstCertificateIsPinned(t *testing.T) {
	cert, _ := makeEnforcementCert(t, bindingCertCN)
	directory := &fakeBSDirectory{station: RegisteredBaseStation{ID: 7, TenantID: 1}, backfillResult: true}
	c := newCommunityConnect(t, directory, cert, uint64(TestBsEui01))

	c.claim(t)

	assert.True(t, c.conn.SeenCommand(mioty.CmdConnectResponse), "the first certificate naming the station is admitted")
	assert.Equal(t, 1, directory.backfillCalls, "its fingerprint is pinned")
}

// A resume is bound the same way: a certificate naming another station
// cannot resume the registered station's session.
func TestConnectCommunity_ResumeBoundToTheCertificate(t *testing.T) {
	cert, _ := makeEnforcementCert(t, "11-11-11-11-11-11-11-11")
	c := newCommunityConnect(t, &fakeBSDirectory{}, cert, 0x1111111111111111)
	resumable := make([]byte, 16)
	for i := range resumable {
		resumable[i] = byte(i + 1)
	}
	c.server.sessionSvc.(*mockSessionService).StoreSessionByUUID(&Session{ProtocolSessionState: ProtocolSessionState{
		ID: "registered-station-session", BaseStationEUI: TestBsEui01, SessionUUID: resumable, DbSessionID: 7,
	}})

	c.claim(t)

	c.assertRefused(t, "an impostor cannot resume the registered station's session")
	assert.False(t, c.session.IsResumed)
}

func bindStationCertificates(t *testing.T, server *Server, directory *fakeBSDirectory) {
	t.Helper()
	binder, err := NewStationCertificateBinder(directory, logger.NewNop())
	require.NoError(t, err)
	server.stationCertificates = binder
}
