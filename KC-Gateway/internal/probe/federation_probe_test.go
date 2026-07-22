//go:build integration

package probe

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// repoRoot returns the absolute path to the kilocenter-modules workspace root,
// by walking up from this test file's location.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// filename = .../kilocenter-modules/KC-Gateway/internal/probe/federation_probe_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

// Addr helpers — each returns env override or the default for the local harness.

func ceBSSCIAddr() string {
	if v := os.Getenv("BSSCI_CE_ADDR"); v != "" {
		return v
	}
	return "localhost:5002"
}

func eceCoreAddr() string {
	if v := os.Getenv("CORE_INTERNAL_ADDR"); v != "" {
		return v
	}
	return "localhost:50051"
}

func ceCoreAddr() string {
	if v := os.Getenv("CE_CORE_ADDR"); v != "" {
		return v
	}
	return "localhost:50053"
}

func eceDBDSN() string {
	if v := os.Getenv("ECE_DB_DSN"); v != "" {
		return v
	}
	return "postgres://kilocenter:changeme@localhost:5433/kilocenter_fed_ece?sslmode=disable"
}

func ceDBDSN() string {
	if v := os.Getenv("CE_DB_DSN"); v != "" {
		return v
	}
	return "postgres://kilocenter:changeme@localhost:5433/kilocenter_fed_ce?sslmode=disable"
}

// dialBSSCI opens a mutual-TLS connection to the given BSSCI address using
// the shared probe client certificate (BSSCI requires mutual TLS).
func dialBSSCI(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 5 * time.Second},
		"tcp", addr, bssciTLSConfig(t))
	require.NoError(t, err, "dial BSSCI at %s", addr)
	return conn
}

// euiBytes renders an EUI as the 8-byte big-endian BYTEA value the messages
// schema stores.
func euiBytes(eui uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, eui)
	return b
}

// seedEndpointOn ensures an endpoint with the given EUI exists on the KC-Core at coreAddr.
func seedEndpointOn(t *testing.T, coreAddr string, epEUI uint64) {
	t.Helper()
	conn, err := grpc.NewClient(coreAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "connect to KC-Core at %s", coreAddr)
	defer func() { _ = conn.Close() }()

	euiHex := fmt.Sprintf("%016x", epEUI)
	md := metadata.Pairs(
		grpcconst.MetadataKeyInternalTenantID, "1",
		grpcconst.MetadataKeyInternalOrgID, "11111111-2222-3333-4444-555555555555",
		grpcconst.MetadataKeyInternalUserID, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	var resp pb.EndPoint
	err = conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/CreateEndPoint",
		&pb.CreateEndPointRequest{
			Endpoint: &pb.EndPoint{
				EpEui:    euiHex,
				Name:     fmt.Sprintf("probe-fed-ep-%s", euiHex),
				NwkSnKey: probeNwkSnKey,
				EpClass:  "A",
				Status:   "active",
			},
		}, &resp)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() != codes.AlreadyExists {
			t.Fatalf("seed endpoint %s on %s: %v", euiHex, coreAddr, err)
		}
	}
}

// pollUntil polls check() at 500ms intervals until it returns true or timeout expires.
func pollUntil(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

// openDB opens a postgres connection and registers cleanup.
func openDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "open DB: %s", dsn)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedBaseStationOn ensures a base station with the given EUI exists on the
// KC-Core at coreAddr; the BSSCI connect handler only accepts registered
// stations.
func seedBaseStationOn(t *testing.T, coreAddr string, bsEUI uint64) {
	t.Helper()
	conn, err := grpc.NewClient(coreAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "connect to KC-Core at %s", coreAddr)
	defer func() { _ = conn.Close() }()

	euiHex := fmt.Sprintf("%016x", bsEUI)
	md := metadata.Pairs(
		grpcconst.MetadataKeyInternalTenantID, "1",
		grpcconst.MetadataKeyInternalOrgID, "11111111-2222-3333-4444-555555555555",
		grpcconst.MetadataKeyInternalUserID, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	var resp pb.BaseStation
	err = conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/CreateBaseStation",
		&pb.CreateBaseStationRequest{
			Basestation: &pb.BaseStation{
				BsEui: euiHex,
				Name:  fmt.Sprintf("probe-fed-bs-%s", euiHex),
			},
		}, &resp)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() != codes.AlreadyExists {
			t.Fatalf("seed base station %s on %s: %v", euiHex, coreAddr, err)
		}
	}
}

// bssciConnect performs BSSCI version negotiation (con → conRsp → conCmp) using bsEUI.
// The station is registered on the CE core first, and every call starts a new
// session so operation counters never resume from a previous run.
// Returns the open connection for subsequent operations.
func bssciConnect(t *testing.T, addr string, bsEUI uint64) net.Conn {
	t.Helper()
	seedBaseStationOn(t, ceCoreAddr(), bsEUI)
	conn := dialBSSCI(t, addr)

	conReq := mioty.Connect{
		BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdConnect, OpId: 0},
		Version:     mioty.MIOTYProtocolVersion,
		BsEui:       bsEUI,
		Bidi:        true,
		SnBsUuid:    freshSessionUUID(),
	}
	writeFrame(t, conn, conReq)

	resp := awaitFrame(t, conn, mioty.CmdConnectResponse)
	require.Equal(t, mioty.CmdConnectResponse, resp["command"],
		"expected conRsp, got %v", resp["command"])

	conCmp := mioty.ConnectComplete{
		BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdConnectComplete, OpId: 0},
	}
	writeFrame(t, conn, conCmp)
	return conn
}

// bssciSendULData sends a ulData + waits for ulDataRsp + sends ulDataCmp.
func bssciSendULData(t *testing.T, conn net.Conn, epEUI uint64, opID int64, userData []byte) {
	t.Helper()
	ul := mioty.ULData{
		BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdULData, OpId: opID},
		EpEui:       epEUI,
		RxTime:      time.Now().UnixNano(),
		PacketCnt:   freshPacketCnt(),
		Snr:         12.0,
		Rssi:        -75.0,
		UserData:    userData,
	}
	writeFrame(t, conn, ul)

	resp := awaitFrame(t, conn, mioty.CmdULDataResponse)
	require.Equal(t, mioty.CmdULDataResponse, resp["command"],
		"expected ulDataRsp, got %v", resp["command"])

	ulCmp := mioty.ULDataComplete{
		BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdULDataComplete, OpId: opID},
	}
	writeFrame(t, conn, ulCmp)
}

// bssciSendULDataWithFrame sends a fully-specified ULData frame + waits for ulDataRsp + sends ulDataCmp.
func bssciSendULDataWithFrame(t *testing.T, conn net.Conn, ul mioty.ULData) {
	t.Helper()
	writeFrame(t, conn, ul)
	resp := awaitFrame(t, conn, mioty.CmdULDataResponse)
	require.Equal(t, mioty.CmdULDataResponse, resp["command"],
		"expected ulDataRsp, got %v", resp["command"])
	writeFrame(t, conn, mioty.ULDataComplete{
		BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdULDataComplete, OpId: ul.OpId},
	})
}

// ingressBinaryOnce builds the federation-ingress binary at most once per test process run.
var (
	ingressBinaryOnce sync.Once
	ingressBinaryPath string
	ingressBinaryErr  error
)

// buildIngressBinary compiles the federation-ingress binary once and returns its path.
func buildIngressBinary(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	binPath := filepath.Join(os.TempDir(), fmt.Sprintf("kilocenter-federation-ingress-test-%d", os.Getpid()))
	ingressBinaryOnce.Do(func() {
		cmd := exec.Command("go", "build", "-o", binPath, "./KC-Core/cmd/federation-ingress")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			ingressBinaryErr = fmt.Errorf("build federation-ingress: %w\n%s", err, out)
		}
		ingressBinaryPath = binPath
	})
	require.NoError(t, ingressBinaryErr, "build federation-ingress binary")
	return ingressBinaryPath
}

// startIngress starts the federation-ingress binary as a test-owned subprocess.
// KILOCENTER_STORAGE_* env vars are inherited from the test process.
// Returns an explicit stop function for early termination.
func startIngress(t *testing.T) (stop func()) {
	t.Helper()
	binary := buildIngressBinary(t)
	root := repoRoot(t)
	cfg := filepath.Join(root, "KC-Core", "config.federation-ingress.yaml")

	cmd := exec.Command(binary, "-config", cfg) //nolint:gosec
	cmd.Dir = root
	cmd.Env = os.Environ() // inherits KILOCENTER_STORAGE_* from test process
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	require.NoError(t, cmd.Start(), "start federation-ingress")

	stopped := false
	doStop := func() {
		if stopped {
			return
		}
		stopped = true
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			done := make(chan struct{})
			go func() { _ = cmd.Wait(); close(done) }()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}
	}
	t.Cleanup(doStop)

	// Allow time for the process to bind its gRPC port.
	time.Sleep(3 * time.Second)
	return doStop
}

// ensureCEOnboarded calls GetCEStatus and completes onboarding if required.
// Does NOT wait for FederationConnected — call waitForFederationConnected separately
// after the ingress is started.
func ensureCEOnboarded(t *testing.T) {
	t.Helper()
	conn, err := grpc.NewClient(ceCoreAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	ctx := context.Background()
	var statusResp pb.GetCEStatusResponse
	require.NoError(t, conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/GetCEStatus",
		&pb.GetCEStatusRequest{}, &statusResp))
	if statusResp.OnboardingRequired {
		var r pb.CompleteCEOnboardingResponse
		require.NoError(t, conn.Invoke(ctx,
			"/kilocenter.api.v1.CoreService/CompleteCEOnboarding",
			&pb.CompleteCEOnboardingRequest{CompanyName: "Smoke Test CE"}, &r))
	}
}

// waitForFederationConnected polls GetCEStatus until FederationConnected == true.
// Must be called after the ingress is started.
func waitForFederationConnected(t *testing.T) {
	t.Helper()
	conn, err := grpc.NewClient(ceCoreAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	ctx := context.Background()
	pollUntil(t, 30*time.Second, func() bool {
		var r pb.GetCEStatusResponse
		callErr := conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/GetCEStatus",
			&pb.GetCEStatusRequest{}, &r)
		return callErr == nil && r.FederationConnected
	})
}

// uniquePayload returns seed bytes prefixed with the current Unix nanosecond
// timestamp so each test run produces a globally unique payload.
func uniquePayload(seed []byte) []byte {
	ts := make([]byte, 8)
	binary.LittleEndian.PutUint64(ts, uint64(time.Now().UnixNano())) //nolint:gosec
	return append(ts, seed...)
}

// TestFederationRelayUnknownULData verifies that a CE-side uplink for an endpoint
// known only to the ECE is relayed end-to-end and persisted in the ECE message store.
func TestFederationRelayUnknownULData(t *testing.T) {
	runStart := time.Now().UTC()

	// epEUI=2: unknown to CE, known to ECE only.
	// bsEUI=0x7000000000000011: CE-side BS EUI, distinct from the synthetic ingress EUI (0x7000000000000001).
	const epEUI uint64 = 0x0000000000000002
	const bsEUI uint64 = 0x7000000000000011
	seedEndpointOn(t, eceCoreAddr(), epEUI)

	ensureCEOnboarded(t)
	waitForFederationConnected(t)

	eceDB := openDB(t, eceDBDSN())
	ceDB := openDB(t, ceDBDSN())

	payload := uniquePayload([]byte{0xDE, 0xAD})

	conn := bssciConnect(t, ceBSSCIAddr(), bsEUI)
	defer func() { _ = conn.Close() }()

	bssciSendULDataWithFrame(t, conn, mioty.ULData{
		BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdULData, OpId: 1},
		EpEui:       epEUI,
		RxTime:      time.Now().UnixNano(),
		PacketCnt:   freshPacketCnt(),
		Snr:         12.0,
		Rssi:        -75.0,
		UserData:    payload,
		DlOpen:      true,
		ResponseExp: true,
		DlAck:       false,
	})

	// Wait until CE outbox row is acked; scan its relay_id for downstream assertions.
	var relayID string
	pollUntil(t, 30*time.Second, func() bool {
		var s string
		err := ceDB.QueryRow(
			`SELECT relay_id::text, status FROM federation_outbox
			 WHERE ep_eui = $1 AND created_at >= $2
			 ORDER BY created_at DESC LIMIT 1`,
			int64(epEUI), runStart, //nolint:gosec
		).Scan(&relayID, &s)
		return err == nil && s == "acked"
	})
	require.NotEmpty(t, relayID)

	// ECE receipt: accepted, message_id populated, ce_bs_eui matches CE-side BS EUI.
	var accepted bool
	var messageIDStr sql.NullString
	var ceBsEUI int64
	err := eceDB.QueryRow(
		`SELECT accepted, message_id::text, ce_bs_eui FROM federation_receipts WHERE relay_id = $1`,
		relayID,
	).Scan(&accepted, &messageIDStr, &ceBsEUI)
	require.NoError(t, err, "receipt must exist for relay_id %s", relayID)
	require.True(t, accepted)
	require.True(t, messageIDStr.Valid, "message_id must be populated on accepted receipt")
	require.Equal(t, int64(bsEUI), ceBsEUI) //nolint:gosec

	// ECE messages: verify the relayed uplink was stored with the synthetic BS EUI and correct payload.
	var msgBsEUI []byte
	var msgPayload []byte
	err = eceDB.QueryRow(
		`SELECT m.bs_eui, m.user_data FROM messages m WHERE m.id = $1::uuid`,
		messageIDStr.String,
	).Scan(&msgBsEUI, &msgPayload)
	require.NoError(t, err, "ECE message must exist for message_id %s", messageIDStr.String)
	require.Equal(t, euiBytes(0x7000000000000001), msgBsEUI, "ECE message must carry synthetic BS EUI")
	require.Equal(t, payload, msgPayload)

	// ECE messages: verify uplink flags were forwarded correctly.
	var dlOpen, responseExp, dlAck bool
	err = eceDB.QueryRow(
		`SELECT dl_open, response_exp, dl_ack FROM messages WHERE id = $1::uuid`,
		messageIDStr.String,
	).Scan(&dlOpen, &responseExp, &dlAck)
	require.NoError(t, err)
	require.True(t, dlOpen)
	require.True(t, responseExp)
	require.False(t, dlAck)

	// Idempotency: exactly one receipt and one message row for this relay.
	var receiptCount, msgCount int
	require.NoError(t, eceDB.QueryRow(
		`SELECT COUNT(*) FROM federation_receipts WHERE relay_id = $1`, relayID,
	).Scan(&receiptCount))
	require.Equal(t, 1, receiptCount)
	require.NoError(t, eceDB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE id = $1::uuid`, messageIDStr.String,
	).Scan(&msgCount))
	require.Equal(t, 1, msgCount)

	// CE messages: no rows for epEUI in this run (endpoint is unknown locally).
	var ceCount int
	err = ceDB.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE ep_eui = $1 AND created_at >= $2`,
		euiBytes(epEUI), runStart,
	).Scan(&ceCount)
	require.NoError(t, err)
	require.Equal(t, 0, ceCount, "CE should not store messages for endpoints it does not own")

	// ECE ce_registry: at least one packet relayed.
	var connStatus string
	var totalRelayed int64
	err = eceDB.QueryRow(
		`SELECT connection_status, total_packets_relayed FROM ce_registry ORDER BY created_at DESC LIMIT 1`,
	).Scan(&connStatus, &totalRelayed)
	require.NoError(t, err)
	require.Equal(t, "online", connStatus)
	require.GreaterOrEqual(t, totalRelayed, int64(1))
}

// TestFederationKnownEndpointStaysLocal verifies that a CE-side uplink for an endpoint
// known to both CE and ECE is stored locally and never relayed to the ECE.
func TestFederationKnownEndpointStaysLocal(t *testing.T) {
	runStart := time.Now().UTC()

	const epEUI uint64 = 0x0000000000000003
	const bsEUI uint64 = 0x7000000000000011

	// Seed on both sides so CE recognizes the endpoint as local.
	seedEndpointOn(t, eceCoreAddr(), epEUI)
	seedEndpointOn(t, ceCoreAddr(), epEUI)

	ensureCEOnboarded(t)
	waitForFederationConnected(t)

	ceDB := openDB(t, ceDBDSN())
	eceDB := openDB(t, eceDBDSN())

	payload := uniquePayload([]byte{0xBE, 0xEF})

	conn := bssciConnect(t, ceBSSCIAddr(), bsEUI)
	defer func() { _ = conn.Close() }()

	bssciSendULData(t, conn, epEUI, 2, payload)

	// CE messages: local row for epEUI must appear.
	pollUntil(t, 10*time.Second, func() bool {
		var count int
		err := ceDB.QueryRow(
			`SELECT COUNT(*) FROM messages WHERE ep_eui = $1 AND created_at >= $2`,
			euiBytes(epEUI), runStart,
		).Scan(&count)
		return err == nil && count > 0
	})

	// ECE federation_receipts: no relay for this endpoint in this run.
	var receiptCount int
	err := eceDB.QueryRow(
		`SELECT COUNT(*) FROM federation_receipts WHERE ep_eui = $1 AND received_at >= $2`,
		int64(epEUI), runStart, //nolint:gosec
	).Scan(&receiptCount)
	require.NoError(t, err)
	require.Equal(t, 0, receiptCount, "ECE should receive no relay for a locally-known endpoint")

	// CE federation_outbox: no entry for this endpoint in this run.
	var outboxCount int
	err = ceDB.QueryRow(
		`SELECT COUNT(*) FROM federation_outbox WHERE ep_eui = $1 AND created_at >= $2`,
		int64(epEUI), runStart, //nolint:gosec
	).Scan(&outboxCount)
	require.NoError(t, err)
	require.Equal(t, 0, outboxCount, "CE outbox should have no entry for a locally-known endpoint")
}

// TestFederationReplayAfterIngressOutage verifies that uplinks buffered in the CE outbox
// while the ingress is down are delivered when the ingress restarts.
// This test owns the ingress subprocess; skip if FEDERATION_HARNESS_EXTERNAL=1.
func TestFederationReplayAfterIngressOutage(t *testing.T) {
	if os.Getenv("FEDERATION_HARNESS_EXTERNAL") == "1" {
		t.Skip("ingress managed externally; skipping outage test to avoid process conflict")
	}

	runStart := time.Now().UTC()

	const epEUI uint64 = 0x0000000000000002
	const bsEUI uint64 = 0x7000000000000011

	seedEndpointOn(t, eceCoreAddr(), epEUI)
	ensureCEOnboarded(t)

	ceDB := openDB(t, ceDBDSN())
	eceDB := openDB(t, eceDBDSN())

	stop := startIngress(t)
	waitForFederationConnected(t)

	conn := bssciConnect(t, ceBSSCIAddr(), bsEUI)
	defer func() { _ = conn.Close() }()

	payload1 := uniquePayload([]byte{0xDE, 0xAD})
	payload2 := uniquePayload([]byte{0xFE, 0xED})

	// First uplink — ingress is up, should be relayed and acked.
	bssciSendULData(t, conn, epEUI, 3, payload1)
	pollUntil(t, 15*time.Second, func() bool {
		var s string
		err := ceDB.QueryRow(
			`SELECT status FROM federation_outbox
			 WHERE ep_eui = $1 AND created_at >= $2
			 ORDER BY created_at ASC LIMIT 1`,
			int64(epEUI), runStart, //nolint:gosec
		).Scan(&s)
		return err == nil && s == "acked"
	})

	// Kill ingress to simulate outage.
	stop()
	time.Sleep(500 * time.Millisecond)

	// Second uplink — ingress is down, should buffer in CE outbox as pending/sent.
	bssciSendULData(t, conn, epEUI, 4, payload2)

	time.Sleep(3 * time.Second)
	var pendingStatus string
	var relayID string
	err := ceDB.QueryRow(
		`SELECT relay_id::text, status FROM federation_outbox
		 WHERE ep_eui = $1 AND created_at >= $2
		 ORDER BY created_at DESC LIMIT 1`,
		int64(epEUI), runStart, //nolint:gosec
	).Scan(&relayID, &pendingStatus)
	require.NoError(t, err, "outbox row for second uplink must exist while ingress is down")
	require.NotEqual(t, "acked", pendingStatus, "second uplink must not be acked while ingress is down")

	// Restart ingress — outbox replay should deliver the buffered uplink.
	startIngress(t)

	// Idempotency: poll until receipt appears for the relay_id of the buffered uplink.
	pollUntil(t, 15*time.Second, func() bool {
		var count int
		err := eceDB.QueryRow(
			`SELECT COUNT(*) FROM federation_receipts WHERE relay_id = $1`, relayID,
		).Scan(&count)
		return err == nil && count == 1
	})

	// CE outbox: second uplink should now be acked.
	pollUntil(t, 15*time.Second, func() bool {
		var s string
		err := ceDB.QueryRow(
			`SELECT status FROM federation_outbox WHERE relay_id = $1`, relayID,
		).Scan(&s)
		return err == nil && s == "acked"
	})

	// Verify the relayed payload round-trips correctly.
	var storedPayload []byte
	err = eceDB.QueryRow(
		`SELECT m.user_data FROM messages m
		 JOIN federation_receipts r ON r.message_id = m.id
		 WHERE r.relay_id = $1`,
		relayID,
	).Scan(&storedPayload)
	require.NoError(t, err, "ECE message must be linked via receipt")
	require.Equal(t, payload2, storedPayload)
}

// TestFederationRevocationStopsRelay verifies that revoking a CE instance via DB
// causes the ECE to terminate the active stream and the CE to stop relaying.
// Skips if FEDERATION_HARNESS_EXTERNAL=1 to avoid state pollution during shared-harness runs.
func TestFederationRevocationStopsRelay(t *testing.T) {
	if os.Getenv("FEDERATION_HARNESS_EXTERNAL") == "1" {
		t.Skip("skipping revocation test during shared harness run to avoid CE state pollution")
	}

	runStart := time.Now().UTC()

	const epEUI uint64 = 0x0000000000000002
	const bsEUI uint64 = 0x7000000000000011

	seedEndpointOn(t, eceCoreAddr(), epEUI)
	ensureCEOnboarded(t)
	waitForFederationConnected(t)

	eceDB := openDB(t, eceDBDSN())
	ceDB := openDB(t, ceDBDSN())

	// Send one uplink before revocation to confirm the relay path is live.
	preRevokePayload := uniquePayload([]byte{0xAA, 0xBB})
	bssciConn := bssciConnect(t, ceBSSCIAddr(), bsEUI)
	defer func() { _ = bssciConn.Close() }()
	bssciSendULData(t, bssciConn, epEUI, 10, preRevokePayload)
	pollUntil(t, 15*time.Second, func() bool {
		var s string
		err := ceDB.QueryRow(
			`SELECT status FROM federation_outbox
			 WHERE ep_eui = $1 AND created_at >= $2
			 ORDER BY created_at ASC LIMIT 1`,
			int64(epEUI), runStart, //nolint:gosec
		).Scan(&s)
		return err == nil && s == "acked"
	})

	// Read the active CE ID from the ECE registry.
	var ceIDStr string
	err := eceDB.QueryRow(
		`SELECT ce_id FROM ce_registry WHERE status = 'active' ORDER BY first_seen_at DESC LIMIT 1`,
	).Scan(&ceIDStr)
	require.NoError(t, err, "no active CE found in ECE registry; run federation-up first")

	// Revoke directly via DB — simulates an operator action.
	_, err = eceDB.Exec(
		`UPDATE ce_registry SET status = 'revoked', revocation_reason = $1, revoked_at = NOW() WHERE ce_id = $2`,
		"smoke-test", ceIDStr,
	)
	require.NoError(t, err, "revoke CE in ECE registry")

	// IngressService polls at the configured interval; allow up to 20s for detection and stream close.
	pollUntil(t, 20*time.Second, func() bool {
		conn, dialErr := grpc.NewClient(ceCoreAddr(),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if dialErr != nil {
			return false
		}
		defer func() { _ = conn.Close() }()

		md := metadata.Pairs(
			grpcconst.MetadataKeyInternalTenantID, "1",
			grpcconst.MetadataKeyInternalOrgID, "11111111-2222-3333-4444-555555555555",
			grpcconst.MetadataKeyInternalUserID, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		)
		ctx := metadata.NewOutgoingContext(context.Background(), md)

		var resp pb.GetCEStatusResponse
		callErr := conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/GetCEStatus",
			&pb.GetCEStatusRequest{}, &resp)
		return callErr == nil && !resp.FederationConnected
	})

	// Send an uplink after revocation; it must stay pending and never reach ECE.
	postRevokePayload := uniquePayload([]byte{0xCA, 0xFE})

	bssciSendULData(t, bssciConn, epEUI, 11, postRevokePayload)
	time.Sleep(5 * time.Second)

	var outboxRelayID string
	var outboxStatus string
	err = ceDB.QueryRow(
		`SELECT relay_id::text, status FROM federation_outbox
		 WHERE ep_eui = $1 AND created_at >= $2
		 ORDER BY created_at DESC LIMIT 1`,
		int64(epEUI), runStart, //nolint:gosec
	).Scan(&outboxRelayID, &outboxStatus)
	require.NoError(t, err, "outbox row must exist for post-revocation uplink")
	require.NotEqual(t, "acked", outboxStatus, "post-revocation uplink must not be acked")

	// No ECE receipt should exist for this uplink.
	var receiptCount int
	require.NoError(t, eceDB.QueryRow(
		`SELECT COUNT(*) FROM federation_receipts WHERE relay_id = $1`, outboxRelayID,
	).Scan(&receiptCount))
	require.Equal(t, 0, receiptCount, "no ECE processing must happen for post-revocation uplink")
}
