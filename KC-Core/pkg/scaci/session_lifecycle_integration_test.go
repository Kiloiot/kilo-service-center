package scaci_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	scaciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// TestMain tears the shared database container down after the package's tests.
func TestMain(m *testing.M) { os.Exit(testsupport.Main(m)) }

const (
	harnessTenant               int64 = 1 // seeded default tenant
	harnessServerName                 = "scaci.kilocenter.test"
	harnessClientCN                   = "70B3D59CD0000A01"
	harnessSCEui                      = uint64(0x70B3D59CD0000001)
	harnessAcEui                      = uint64(0x70B3D59CD0000A01)
	harnessVersion                    = "1.0.0"
	harnessTLSMinVersion              = "1.2"
	harnessTLSEnabled                 = true
	harnessStrictOrgResolution        = false
	harnessCertLifetime               = time.Hour
	harnessFileMode                   = 0o600
	harnessIOTimeout                  = 5 * time.Second
	harnessWaitTimeout                = 5 * time.Second
	harnessWaitStep                   = 20 * time.Millisecond
	harnessHoldPeriod                 = time.Second
	harnessFirstAcOpID                = int64(1)
	harnessEstablishmentTimeout       = 300 * time.Millisecond
	harnessLoopback                   = "127.0.0.1:0"
	harnessSerialCA                   = 1
	harnessSerialServer               = 2
	harnessSerialClient               = 3
	harnessUUIDFirst            byte  = 0x11
	harnessUUIDSecond           byte  = 0x51
)

var frameCodec = harnessFrameCodec()

// harnessApplicationCenter is the application center of the harness client
// certificate, which falls back to the default tenant without organization.
var harnessApplicationCenter = scaci.ApplicationCenter{TenantID: harnessTenant, AcEui: harnessAcEui}

func harnessFrameCodec() nettransport.FrameCodec {
	codec, err := nettransport.NewFrameCodec(mioty.SCACIFrameIdentifier, dbconfig.MaxMessageSize, harnessIOTimeout)
	if err != nil {
		panic(err)
	}
	return codec
}

// fallbackResolver resolves no certificate, so the handshake falls back to
// the community default tenant.
type fallbackResolver struct{ org.Resolver }

var errNoCertificateMapping = errors.New("no certificate mapping")

func (fallbackResolver) ResolveCert(context.Context, *x509.Certificate) (uuid.UUID, int64, error) {
	return uuid.Nil, 0, errNoCertificateMapping
}

func (fallbackResolver) GetDefaultOrgForTenant(context.Context, int64) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// sessionCreations runs a session creation in a storage transaction, as the
// composition root's bridge does.
type sessionCreations struct {
	adapter *adapters.SCACISessionTransactionAdapter
}

func (c sessionCreations) Run(ctx context.Context, fn func(scaciservices.SessionCreationTx) error) error {
	return c.adapter.Run(ctx, func(tx adapters.SCACISessionTx) error { return fn(tx) })
}

type noSnapshots struct{}

func (noSnapshots) ConnectedSessionsSnapshot() []propagation.BaseStationSession { return nil }

type noPropagation struct{}

func (noPropagation) TriggerEndpointPropagate(context.Context, int64, []propagation.BaseStationSession) error {
	return nil
}

// harnessPKI is a throwaway CA with one server and one client leaf;
// issueClient issues further client leaves.
type harnessPKI struct {
	certFile, keyFile, caFile string
	caPool                    *x509.CertPool
	client                    tls.Certificate
	issueClient               func(cn string) tls.Certificate
}

func newHarnessPKI(t *testing.T) *harnessPKI {
	t.Helper()
	dir := t.TempDir()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(harnessSerialCA),
		Subject:               pkix.Name{CommonName: harnessServerName},
		NotBefore:             time.Now().Add(-harnessCertLifetime),
		NotAfter:              time.Now().Add(harnessCertLifetime),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	issue := func(serial int64, cn string, usage x509.ExtKeyUsage) ([]byte, []byte) {
		key, keyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, keyErr)
		template := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: cn},
			DNSNames:     []string{harnessServerName},
			NotBefore:    time.Now().Add(-harnessCertLifetime),
			NotAfter:     time.Now().Add(harnessCertLifetime),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{usage},
		}
		der, certErr := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
		require.NoError(t, certErr)
		keyDER, marshalErr := x509.MarshalECPrivateKey(key)
		require.NoError(t, marshalErr)
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
			pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	}

	serverPEM, serverKeyPEM := issue(harnessSerialServer, harnessServerName, x509.ExtKeyUsageServerAuth)
	serial := int64(harnessSerialClient)
	issueClient := func(cn string) tls.Certificate {
		clientPEM, clientKeyPEM := issue(serial, cn, x509.ExtKeyUsageClientAuth)
		serial++
		client, pairErr := tls.X509KeyPair(clientPEM, clientKeyPEM)
		require.NoError(t, pairErr)
		return client
	}

	pki := &harnessPKI{
		certFile:    filepath.Join(dir, "server.crt"),
		keyFile:     filepath.Join(dir, "server.key"),
		caFile:      filepath.Join(dir, "ca.crt"),
		caPool:      x509.NewCertPool(),
		client:      issueClient(harnessClientCN),
		issueClient: issueClient,
	}
	pki.caPool.AddCert(caCert)
	require.NoError(t, os.WriteFile(pki.caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), harnessFileMode))
	require.NoError(t, os.WriteFile(pki.certFile, serverPEM, harnessFileMode))
	require.NoError(t, os.WriteFile(pki.keyFile, serverKeyPEM, harnessFileMode))
	return pki
}

// scaciHarness runs a real SCACI server over mutual TLS against a migrated
// per-test database; restart replaces the server on the same database.
type scaciHarness struct {
	server     *scaci.Server
	addr       string
	cfg        scaci.Config
	pki        *harnessPKI
	db         *postgres.DB
	sessions   *postgres.SCACISessionRepository
	operations *postgres.SCACIOperationRepository
	events     *postgres.SCACIEventStore
	downlinks  *scaci.MockDLService
	endpoints  *scaci.MockEndpointService
	// directory resolves client certificates to their organization.
	directory org.Resolver
	// stop stops the running server once, whether the test or its cleanup gets there first.
	stop func() error
}

func freeLoopbackAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", harnessLoopback)
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

func newSCACIHarness(t *testing.T, cfg scaci.Config) *scaciHarness {
	t.Helper()
	return newSCACIHarnessWith(t, cfg, fallbackResolver{})
}

// newSCACIHarnessWith runs the harness with the organization directory the
// handshake resolves client certificates with.
func newSCACIHarnessWith(t *testing.T, cfg scaci.Config, directory org.Resolver) *scaciHarness {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := teststore.Setup(t)
	t.Cleanup(cleanup)
	h := &scaciHarness{cfg: cfg, pki: newHarnessPKI(t), db: db, directory: directory, endpoints: &scaci.MockEndpointService{}}
	h.start(t)
	return h
}

// start runs a new server on the harness database.
func (h *scaciHarness) start(t *testing.T) {
	t.Helper()
	repos := postgres.NewRepositories(h.db)
	log := logger.NewNop()
	clk := clock.SystemClock{}

	cfg := h.cfg
	cfg.ListenAddr = freeLoopbackAddr(t)
	cfg.TLS = config.TLSConfig{Enabled: harnessTLSEnabled, CertFile: h.pki.certFile, KeyFile: h.pki.keyFile, CAFile: h.pki.caFile, MinVersion: harnessTLSMinVersion}
	cfg.ServiceCenterEUI = harnessSCEui
	cfg.SoftwareVersion = harnessVersion
	cfg.SocketWriteTimeout = harnessIOTimeout
	cfg.PlatformTenantID = harnessTenant
	if cfg.ConnectionEstablishmentTimeout == 0 {
		cfg.ConnectionEstablishmentTimeout = harnessWaitTimeout
	}

	recorder := scaciservices.NewOperationRecorder(repos.SCACIOperations)
	rows, err := scaciservices.NewSessionRows(
		sessionCreations{adapter: adapters.NewSCACISessionTransactionAdapter(h.db)}, repos.SCACISessions, repos.SCACISessions, harnessSCEui)
	require.NoError(t, err)
	holder, err := scaciservices.NewResumeHolder(rows, repos.SCACIOperations,
		config.DefaultProtocolSCACIResumeMaxPendingOperations, log)
	require.NoError(t, err)
	require.NoError(t, holder.Load(testutil.TestContext(), repos.SCACISessions))
	registry, err := scaci.NewSessionRegistry(holder, rows, log)
	require.NoError(t, err)
	downlinks := &scaci.MockDLService{}
	handshake := scaciservices.NewHandshakeService(repos.SCACISessions, log, h.directory, harnessTenant, harnessStrictOrgResolution,
		scaciservices.NewCertificateVerifier(log), harnessSCEui, cfg.Vendor, cfg.Model, cfg.Name, cfg.SoftwareVersion,
		scaci.NewSessionFactory(clk))
	server, err := scaci.NewServer(&cfg, log, scaci.Dependencies{
		Registry:     registry,
		Operations:   repos.SCACIOperations,
		Handshake:    handshake,
		Endpoints:    h.endpoints,
		UL:           &scaci.MockULService{},
		DL:           downlinks,
		Status:       &scaci.MockStatusService{},
		Validator:    scaciservices.NewSessionValidator(),
		Recorder:     recorder,
		Persistence:  rows,
		OrgDirectory: h.directory,
		Snapshots:    noSnapshots{},
		Propagation:  noPropagation{},
		Errors:       scaci.NewErrorRecorder(repos.SCACIOperations, repos.SCACIEvents, log),
		Clock:        clk,

		SessionEvents: repos.SCACIEvents,
	})
	require.NoError(t, err)
	require.NoError(t, server.Start())
	var once sync.Once
	var stopErr error
	stop := func() error {
		once.Do(func() { stopErr = server.Stop() })
		return stopErr
	}
	h.server, h.addr, h.stop = server, cfg.ListenAddr, stop
	h.sessions, h.operations, h.downlinks = repos.SCACISessions, repos.SCACIOperations, downlinks
	h.events = repos.SCACIEvents
	t.Cleanup(func() { require.NoError(t, stop()) })
	require.Eventually(t, server.Listening, harnessWaitTimeout, harnessWaitStep)
}

// restart stops the server and starts a new one on the same database, as a
// service center restart does.
func (h *scaciHarness) restart(t *testing.T) {
	t.Helper()
	require.NoError(t, h.stop())
	h.start(t)
}

// acClient is a minimal application center speaking SCACI framing.
type acClient struct {
	t     *testing.T
	conn  *tls.Conn
	acEui uint64
}

func (h *scaciHarness) dial(t *testing.T) *acClient {
	t.Helper()
	return h.dialAs(t, h.pki.client, harnessAcEui)
}

// dialAs connects as the application center of the client certificate,
// which connects with acEui.
func (h *scaciHarness) dialAs(t *testing.T, cert tls.Certificate, acEui uint64) *acClient {
	t.Helper()
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: harnessIOTimeout}, "tcp", h.addr, &tls.Config{
		RootCAs:      h.pki.caPool,
		Certificates: []tls.Certificate{cert},
		ServerName:   harnessServerName,
		MinVersion:   tls.VersionTLS12,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return &acClient{t: t, conn: conn, acEui: acEui}
}

func (c *acClient) send(msg interface{}) {
	c.t.Helper()
	payload, err := msgpack.Marshal(msg)
	require.NoError(c.t, err)
	frame, err := frameCodec.Encode(payload)
	require.NoError(c.t, err)
	require.NoError(c.t, frameCodec.Send(c.conn, frame))
}

// receive reads one message; io.EOF reports the server closing the connection.
func (c *acClient) receive() (map[string]interface{}, error) {
	require.NoError(c.t, c.conn.SetReadDeadline(time.Now().Add(harnessIOTimeout)))
	frame, err := frameCodec.Read(c.conn)
	if err != nil {
		return nil, err
	}
	var msg map[string]interface{}
	require.NoError(c.t, msgpack.Unmarshal(frame.Payload, &msg))
	return msg, nil
}

func harnessUUID(seed byte) scaci.UUID16 {
	var id scaci.UUID16
	for i := range id {
		id[i] = seed + byte(i)
	}
	return id
}

// connect runs the connect operation of a new session (SCACI §3.3) and
// returns the conRsp.
func (c *acClient) connect(snAcUUID scaci.UUID16) map[string]interface{} {
	c.t.Helper()
	return c.runConnect(scaci.Connect{SnAcUUID: snAcUUID})
}

// resume runs the connect operation asking to resume the session of snAcUUID
// with the application center's view of the operation IDs (SCACI §3.3.1).
func (c *acClient) resume(snAcUUID scaci.UUID16, snAcOpID, snScOpID int64) map[string]interface{} {
	c.t.Helper()
	return c.runConnect(scaci.Connect{SnAcUUID: snAcUUID, SnAcOpId: &snAcOpID, SnScOpId: &snScOpID})
}

func (c *acClient) runConnect(con scaci.Connect) map[string]interface{} {
	c.t.Helper()
	rsp := c.startConnect(con)
	c.completeConnect()
	return rsp
}

// startConnect sends con and returns the conRsp, leaving the connect
// operation open until completeConnect (SCACI §3.3).
func (c *acClient) startConnect(con scaci.Connect) map[string]interface{} {
	c.t.Helper()
	con.BaseMessage = scaci.BaseMessage{Command: scaci.CmdConnect, OpId: scaci.OpIDConnect}
	con.Version = harnessVersion
	con.AcEui = c.acEui
	c.send(con)
	rsp, err := c.receive()
	require.NoError(c.t, err)
	require.Equal(c.t, scaci.CmdConnectResponse, rsp["command"], "connect must be answered with conRsp, got %v", rsp)
	return rsp
}

func (c *acClient) completeConnect() {
	c.t.Helper()
	c.send(scaci.ConnectComplete{BaseMessage: scaci.BaseMessage{Command: scaci.CmdConnectComplete, OpId: scaci.OpIDConnect}})
}

// ping starts an application center keepalive (SCACI §3.4) and returns the
// service center's answer.
func (c *acClient) ping(opID int64) map[string]interface{} {
	c.t.Helper()
	c.send(scaci.Ping{BaseMessage: scaci.BaseMessage{Command: scaci.CmdPing, OpId: opID}})
	rsp, err := c.receive()
	require.NoError(c.t, err)
	return rsp
}

func (h *scaciHarness) session(t *testing.T, snAcUUID scaci.UUID16) *models.SCACISession {
	t.Helper()
	session, err := h.sessions.GetSessionByAcUUID(testutil.TestContext(), harnessApplicationCenter.Key(), snAcUUID)
	require.NoError(t, err)
	return session
}

// waitForSession polls on the test goroutine, so no lookup outlives the test
// and its database.
func (h *scaciHarness) waitForSession(t *testing.T, snAcUUID scaci.UUID16, reached func(*models.SCACISession) bool, what string) *models.SCACISession {
	t.Helper()
	deadline := time.Now().Add(harnessWaitTimeout)
	for {
		session := h.session(t, snAcUUID)
		if reached(session) {
			return session
		}
		if time.Now().After(deadline) {
			t.Fatalf("session never reached %s; status %s", what, session.Status)
		}
		time.Sleep(harnessWaitStep)
	}
}

// waitForEvents polls the system event log until the session's events are
// exactly want, in the order they were filed.
func (h *scaciHarness) waitForEvents(t *testing.T, sessionID int64, want []string) {
	t.Helper()
	deadline := time.Now().Add(harnessWaitTimeout)
	for {
		events, err := h.events.ListSCACIEvents(testutil.TestContext(), harnessTenant, &sessionID, "", len(want)+1, 0)
		require.NoError(t, err)
		got := make([]string, len(events))
		for i, e := range events {
			got[len(events)-1-i] = e.EventType
		}
		if slices.Equal(got, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("session events %v, want %v", got, want)
		}
		time.Sleep(harnessWaitStep)
	}
}

// holdsFor fails as soon as the session stops satisfying holds within period.
func (h *scaciHarness) holdsFor(t *testing.T, snAcUUID scaci.UUID16, holds func(*models.SCACISession) bool, period time.Duration, what string) {
	t.Helper()
	for deadline := time.Now().Add(period); time.Now().Before(deadline); time.Sleep(harnessWaitStep) {
		if session := h.session(t, snAcUUID); !holds(session) {
			t.Fatalf("%s; status %s", what, session.Status)
		}
	}
}

func (h *scaciHarness) waitForStatus(t *testing.T, snAcUUID scaci.UUID16, status string) *models.SCACISession {
	t.Helper()
	return h.waitForSession(t, snAcUUID, func(s *models.SCACISession) bool { return s.Status == status }, status)
}

// A lost connection leaves the session disconnected and resumable (SCACI §1).
func TestSCACIServer_LostConnectionLeavesSessionResumable(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	client := h.dial(t)
	client.connect(harnessUUID(harnessUUIDFirst))
	h.waitForStatus(t, harnessUUID(harnessUUIDFirst), models.SCACISessionStatusActive)

	require.NoError(t, client.conn.Close())

	session := h.waitForStatus(t, harnessUUID(harnessUUIDFirst), models.SCACISessionStatusDisconnected)
	assert.True(t, session.CanResume)
	assert.NotNil(t, session.DisconnectedAt)
}

// Opening a session, losing its connection and resuming it reach the system
// event log under the session's tenant (SCACI §1, §3.3).
func TestSCACIServer_FilesTheSessionLifecycle(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	first := h.dial(t)
	first.connect(harnessUUID(harnessUUIDFirst))
	session := h.waitForStatus(t, harnessUUID(harnessUUIDFirst), models.SCACISessionStatusActive)
	h.waitForEvents(t, session.ID, []string{models.EventTypeSCACISessionOpened})

	require.NoError(t, first.conn.Close())
	h.waitForStatus(t, harnessUUID(harnessUUIDFirst), models.SCACISessionStatusDisconnected)
	h.waitForEvents(t, session.ID, []string{models.EventTypeSCACISessionOpened, models.EventTypeSCACISessionClosed})

	second := h.dial(t)
	assert.Equal(t, true, second.resume(harnessUUID(harnessUUIDFirst), 0, 0)["snResume"])
	h.waitForEvents(t, session.ID, []string{
		models.EventTypeSCACISessionOpened, models.EventTypeSCACISessionClosed, models.EventTypeSCACISessionResumed,
	})
}

// A returning application center that starts a new session instead of
// resuming the old one is served, and the old session is discarded (SCACI §1).
func TestSCACIServer_ReturningApplicationCenterOpensNewSession(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	first := h.dial(t)
	first.connect(harnessUUID(harnessUUIDFirst))
	require.NoError(t, first.conn.Close())

	second := h.dial(t)
	rsp := second.connect(harnessUUID(harnessUUIDSecond))

	assert.Equal(t, false, rsp["snResume"])
	h.waitForStatus(t, harnessUUID(harnessUUIDSecond), models.SCACISessionStatusActive)
	retired := h.waitForStatus(t, harnessUUID(harnessUUIDFirst), models.SCACISessionStatusTerminated)
	assert.False(t, retired.CanResume)
}

// A new session of an application center whose previous connection is still
// open takes that connection over: the old connection is closed and the old
// session discarded, and its events read opened, then closed.
func TestSCACIServer_NewSessionSupersedesOpenConnection(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	first := h.dial(t)
	first.connect(harnessUUID(harnessUUIDFirst))
	require.Equal(t, scaci.CmdPingResponse, first.ping(harnessFirstAcOpID)["command"], "the service center completed the connect")

	second := h.dial(t)
	second.connect(harnessUUID(harnessUUIDSecond))

	_, err := first.receive()
	assert.ErrorIs(t, err, io.EOF, "the superseded connection is closed by the service center")
	superseded := h.waitForStatus(t, harnessUUID(harnessUUIDFirst), models.SCACISessionStatusTerminated)
	assert.Equal(t, models.SCACISessionStatusActive, h.session(t, harnessUUID(harnessUUIDSecond)).Status,
		"closing the superseded connection must not touch the new session")
	h.waitForEvents(t, superseded.ID, []string{models.EventTypeSCACISessionOpened, models.EventTypeSCACISessionClosed})
}

// The resumed session belongs to the new connection: tearing down the old
// connection afterwards must not mark it disconnected.
func TestSCACIServer_ResumeKeepsSessionActive(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	first := h.dial(t)
	first.connect(harnessUUID(harnessUUIDFirst))
	h.waitForStatus(t, harnessUUID(harnessUUIDFirst), models.SCACISessionStatusActive)

	second := h.dial(t)
	rsp := second.resume(harnessUUID(harnessUUIDFirst), 0, 0)
	assert.Equal(t, true, rsp["snResume"])

	_, err := first.receive()
	assert.ErrorIs(t, err, io.EOF, "the connection that held the resumed session is closed")

	h.holdsFor(t, harnessUUID(harnessUUIDFirst), func(s *models.SCACISession) bool {
		return s.Status == models.SCACISessionStatusActive
	}, harnessHoldPeriod, "the resumed session stays active")
}

// An application center that lost the response of its last operation resumes
// requiring only what it saw completed, then reissues that operation with its
// original ID (SCACI §1, §3.2, §3.3.1); the service center serves it and
// keeps rejecting IDs that go backwards.
func TestSCACIServer_ResumedSessionServesReissuedOperation(t *testing.T) {
	const lostOpID, nextOpID = int64(1), int64(2)
	h := newSCACIHarness(t, scaci.Config{})
	first := h.dial(t)
	first.connect(harnessUUID(harnessUUIDFirst))
	assert.Equal(t, scaci.CmdPingResponse, first.ping(lostOpID)["command"])
	require.NoError(t, first.conn.Close())
	h.waitForSession(t, harnessUUID(harnessUUIDFirst), func(s *models.SCACISession) bool {
		return s.Status == models.SCACISessionStatusDisconnected && s.LastOpIDAc == lostOpID
	}, "disconnected after the application center operation")

	second := h.dial(t)
	rsp := second.resume(harnessUUID(harnessUUIDFirst), lostOpID-1, 0)
	require.Equal(t, true, rsp["snResume"], "the service center knows more than the application center requires")

	assert.Equal(t, scaci.CmdPingResponse, second.ping(lostOpID)["command"], "the reissued operation is served")
	assert.Equal(t, scaci.CmdPingResponse, second.ping(nextOpID)["command"], "IDs continue after the reissue")
	assert.Equal(t, scaci.CmdError, second.ping(lostOpID)["command"], "an ID below the sequence is rejected")
}

// Stop does not wait for application centers to hang up: it closes their
// connections, and the sessions stay resumable (SCACI §1).
func TestSCACIServer_StopClosesConnectionsAndLeavesSessionsResumable(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	client := h.dial(t)
	client.connect(harnessUUID(harnessUUIDFirst))
	require.Equal(t, scaci.CmdPingResponse, client.ping(harnessFirstAcOpID)["command"], "the connection is established and idle")

	stopped := make(chan error, 1)
	go func() { stopped <- h.stop() }()
	select {
	case err := <-stopped:
		require.NoError(t, err)
	case <-time.After(harnessWaitTimeout):
		t.Fatal("Stop is blocked by a connected application center")
	}

	_, err := client.receive()
	assert.ErrorIs(t, err, io.EOF, "the service center closes the connection on shutdown")
	session := h.session(t, harnessUUID(harnessUUIDFirst))
	assert.Equal(t, models.SCACISessionStatusDisconnected, session.Status, "shutdown records the lost connection before it returns")
	assert.True(t, session.CanResume)
}

// A peer that never finishes the TLS handshake cannot hold a connection
// handler past the establishment timeout.
func TestSCACIServer_SilentPeerBeforeTLSIsDisconnected(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{ConnectionEstablishmentTimeout: harnessEstablishmentTimeout})
	conn, err := net.Dial("tcp", h.addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(harnessIOTimeout)))
	_, err = conn.Read(make([]byte, 1))
	assert.ErrorIs(t, err, io.EOF, "the service center drops a peer that never completes the TLS handshake")
}

// A peer that completes TLS but never sends con is dropped the same way: the
// whole connect operation must finish in time (SCACI §3.3).
func TestSCACIServer_SilentPeerAfterTLSIsDisconnected(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{ConnectionEstablishmentTimeout: harnessEstablishmentTimeout})
	client := h.dial(t)

	_, err := client.receive()
	assert.ErrorIs(t, err, io.EOF, "the service center drops a peer that never starts the connect operation")
}

// An established session reads without a deadline: liveness is the ping
// operation's job (SCACI §3.4), not the establishment timeout's.
func TestSCACIServer_EstablishedSessionOutlivesTheEstablishmentTimeout(t *testing.T) {
	const opID = int64(1)
	h := newSCACIHarness(t, scaci.Config{ConnectionEstablishmentTimeout: harnessEstablishmentTimeout})
	client := h.dial(t)
	client.connect(harnessUUID(harnessUUIDFirst))
	require.Equal(t, scaci.CmdPingResponse, client.ping(opID)["command"])

	time.Sleep(2 * harnessEstablishmentTimeout)

	assert.Equal(t, scaci.CmdPingResponse, client.ping(opID + 1)["command"])
}

// The connect session UUIDs are Numeric[16] (SCACI §3.3.1, §3.3.2): the
// service center accepts snAcUuid as an array and answers snScUuid as one.
func TestSCACIServer_ConnectCarriesSessionUUIDsAsNumericArrays(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	client := h.dial(t)
	snAcUUID := harnessUUID(harnessUUIDFirst)
	values := make([]int, len(snAcUUID))
	for i, b := range snAcUUID {
		values[i] = int(b)
	}

	client.send(map[string]interface{}{
		"command":  scaci.CmdConnect,
		"opId":     scaci.OpIDConnect,
		"version":  harnessVersion,
		"acEui":    harnessAcEui,
		"snAcUuid": values,
	})
	rsp, err := client.receive()
	require.NoError(t, err)

	require.Equal(t, scaci.CmdConnectResponse, rsp["command"], "a Numeric[16] snAcUuid is accepted, got %v", rsp)
	snScUUID, isArray := rsp["snScUuid"].([]interface{})
	require.True(t, isArray, "snScUuid is a Numeric[16] array, got %T", rsp["snScUuid"])
	assert.Len(t, snScUUID, len(snAcUUID))
	client.send(scaci.ConnectComplete{BaseMessage: scaci.BaseMessage{Command: scaci.CmdConnectComplete, OpId: scaci.OpIDConnect}})
	h.waitForStatus(t, snAcUUID, models.SCACISessionStatusActive)
}

// sendJSON frames msg as a JSON object, the other SCACI encoding (§1, §3).
func (c *acClient) sendJSON(msg map[string]interface{}) {
	c.t.Helper()
	payload, err := json.Marshal(msg)
	require.NoError(c.t, err)
	frame, err := frameCodec.Encode(payload)
	require.NoError(c.t, err)
	require.NoError(c.t, frameCodec.Send(c.conn, frame))
}

// An Application Center that speaks JSON is served: connect and ping run
// end to end with JSON frames (SCACI §1, §3).
func TestSCACIServer_ServesJSONFrames(t *testing.T) {
	h := newSCACIHarness(t, scaci.Config{})
	client := h.dial(t)
	snAcUUID := harnessUUID(harnessUUIDFirst)

	client.sendJSON(map[string]interface{}{
		"command":  scaci.CmdConnect,
		"opId":     scaci.OpIDConnect,
		"version":  harnessVersion,
		"acEui":    harnessAcEui,
		"snAcUuid": snAcUUID,
	})
	rsp, err := client.receive()
	require.NoError(t, err)
	require.Equal(t, scaci.CmdConnectResponse, rsp["command"], "a JSON con is answered with conRsp, got %v", rsp)
	client.sendJSON(map[string]interface{}{"command": scaci.CmdConnectComplete, "opId": scaci.OpIDConnect})
	h.waitForStatus(t, snAcUUID, models.SCACISessionStatusActive)

	client.sendJSON(map[string]interface{}{"command": scaci.CmdPing, "opId": harnessFirstAcOpID})
	pong, err := client.receive()
	require.NoError(t, err)
	assert.Equal(t, scaci.CmdPingResponse, pong["command"], "a JSON ping is answered, got %v", pong)
}
