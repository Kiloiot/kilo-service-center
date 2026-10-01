//go:build integration

package probe

import (
	"context"
	"crypto/aes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/aead/cmac"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// bssciAddr returns the BSSCI server address from env or default.
func bssciAddr() string {
	if addr := os.Getenv("BSSCI_ADDR"); addr != "" {
		return addr
	}
	return "localhost:5000"
}

// bssciTLSConfig builds the probe's BSSCI TLS client configuration. The BSSCI
// listener requires mutual TLS, so the probe authenticates like a real base
// station: BSSCI_CLIENT_CERT and BSSCI_CLIENT_KEY name a CA-signed client
// certificate pair.
func bssciTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	cfg := &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	certFile := os.Getenv("BSSCI_CLIENT_CERT")
	keyFile := os.Getenv("BSSCI_CLIENT_KEY")
	if certFile == "" || keyFile == "" {
		t.Fatalf("BSSCI_CLIENT_CERT and BSSCI_CLIENT_KEY must point at a CA-signed client certificate pair (the BSSCI listener requires mutual TLS)")
	}
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err, "load BSSCI client certificate pair")
	cfg.Certificates = []tls.Certificate{pair}
	return cfg
}

// probeResult captures structured probe output for automation.
type probeResult struct {
	Probe      string `json:"probe"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

func reportResult(t *testing.T, result probeResult) {
	t.Helper()
	data, _ := json.Marshal(result)
	t.Logf("PROBE_RESULT: %s", string(data))
}

func writeFrame(t *testing.T, conn net.Conn, msg interface{}) {
	t.Helper()
	payload, err := msgpack.Marshal(msg)
	require.NoError(t, err, "msgpack marshal")
	require.NoError(t, writeFramePayload(conn, mioty.MIOTYFrameIdentifier, payload), "write frame")
}

func readFrame(t *testing.T, conn net.Conn) map[string]interface{} {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf, err := readFramePayload(conn, mioty.MIOTYFrameIdentifier)
	require.NoError(t, err, "read frame")
	var resp map[string]interface{}
	require.NoError(t, msgpack.Unmarshal(buf, &resp), "unmarshal response")
	return resp
}

// probeNwkSnKey is the 16-byte preshared key seeded with the test endpoint.
// Matches testPresharedKey() in attach_replay_protection_test.go.
var probeNwkSnKey = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

// computeAttachSignature computes the attach signature per MIOTY radio spec
// §3.7.1.3 over the 16-byte Fig. 3-15 IV
// [EUI64 | 0xFF | 0x00 | attachCnt (4 bytes) | 0xFF 0xFF].
func computeAttachSignature(epEUI uint64, attachCnt uint32, presharedKey []byte) [4]byte {
	iv := make([]byte, 16)
	binary.BigEndian.PutUint64(iv[0:8], epEUI)
	iv[8] = 0xFF
	iv[9] = 0x00
	binary.BigEndian.PutUint32(iv[10:14], attachCnt)
	iv[14] = 0xFF
	iv[15] = 0xFF

	block, _ := aes.NewCipher(presharedKey)
	mac, _ := cmac.New(block)
	mac.Write(iv) //nolint:errcheck
	result := mac.Sum(nil)
	var sig [4]byte
	copy(sig[:], result[:4])
	return sig
}

// probeUserID is the principal the probe acts as: the seeded default
// administrator, whose roles admit endpoint registration.
func probeUserID() string {
	if id := os.Getenv("PROBE_USER_ID"); id != "" {
		return id
	}
	return "00000000-0000-0000-0000-000000000001"
}

func coreInternalAddr() string {
	if addr := os.Getenv("CORE_INTERNAL_ADDR"); addr != "" {
		return addr
	}
	return "localhost:50051"
}

// seedTestEndpoint ensures the probe's test endpoint exists in KC-Core.
// Uses internal trust mode with all three required headers:
//   - MetadataKeyInternalTenantID (required, positive int64)
//   - MetadataKeyInternalOrgID (required for non-org-exempt methods like CreateEndPoint, valid UUID)
//   - MetadataKeyInternalUserID (the acting user, whose roles must include endpoint management)
func seedTestEndpoint(t *testing.T) {
	t.Helper()
	conn, err := grpc.NewClient(coreInternalAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "connect to KC-Core internal gRPC")
	defer func() { _ = conn.Close() }()

	md := metadata.Pairs(
		grpcconst.MetadataKeyInternalTenantID, "1",
		grpcconst.MetadataKeyInternalOrgID, "11111111-2222-3333-4444-555555555555",
		grpcconst.MetadataKeyInternalUserID, probeUserID(),
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	var resp pb.EndPoint
	err = conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/CreateEndPoint",
		&pb.CreateEndPointRequest{
			Endpoint: &pb.EndPoint{
				EpEui:    "0000000000000002",
				Name:     "probe-test-endpoint",
				NwkSnKey: probeNwkSnKey,
				EpClass:  "A",
				Status:   "active",
			},
		}, &resp)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() != codes.AlreadyExists {
			t.Fatalf("failed to seed test endpoint: %v", err)
		}
	}

	// The BSSCI connect handler only accepts registered base stations and pins a
	// station to the first certificate that names it. The probe certificate is
	// issued per run, so the station is registered anew for each run.
	err = conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/DeleteBaseStation",
		&pb.DeleteBaseStationRequest{BsEui: "0000000000000001"}, &emptypb.Empty{})
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() != codes.NotFound {
			t.Fatalf("failed to remove the previous run's test base station: %v", err)
		}
	}
	var bsResp pb.BaseStation
	err = conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/CreateBaseStation",
		&pb.CreateBaseStationRequest{
			Basestation: &pb.BaseStation{
				BsEui: "0000000000000001",
				Name:  "probe-test-basestation",
			},
		}, &bsResp)
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() != codes.AlreadyExists {
			t.Fatalf("failed to seed test base station: %v", err)
		}
	}
}

// awaitFrame reads frames until the wanted command arrives, servicing the
// SC-initiated operations a live Service Center interleaves with responses
// (e.g. attach-propagate reconciliation after connect): attPrp/detPrp get
// their response so the SC can complete its handshake, and completes are
// consumed silently.
func awaitFrame(t *testing.T, conn net.Conn, want string) map[string]interface{} {
	t.Helper()
	for range [16]int{} {
		resp := readFrame(t, conn)
		cmd, _ := resp["command"].(string)
		if cmd == want {
			return resp
		}
		switch cmd {
		case mioty.CmdAttachPropagate:
			writeFrame(t, conn, mioty.AttachPropagateResponse{
				BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdAttachPropagateResponse, OpId: frameOpID(resp)},
			})
		case mioty.CmdDetachPropagate:
			writeFrame(t, conn, mioty.DetachPropagateResponse{
				BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdDetachPropagateResponse, OpId: frameOpID(resp)},
			})
		case mioty.CmdStatus:
			writeFrame(t, conn, mioty.StatusResponse{
				BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdStatusResponse, OpId: frameOpID(resp)},
				Code:        0,
				Message:     "ok",
				Time:        time.Now().UnixNano(),
				DutyCycle:   0,
			})
		case mioty.CmdAttachPropagateComplete, mioty.CmdDetachPropagateComplete,
			mioty.CmdStatusComplete, mioty.CmdPing:
			// Consumed; nothing to answer at this layer.
		default:
			require.Equal(t, want, cmd, "expected %s, got %v", want, cmd)
		}
	}
	t.Fatalf("no %s frame within 16 reads", want)
	return nil
}

// freshSessionUUID builds a unique BS session UUID so every probe run starts
// a new session instead of resuming the previous run's operation counters.
func freshSessionUUID() [16]byte {
	var id [16]byte
	binary.BigEndian.PutUint64(id[0:8], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint64(id[8:16], uint64(os.Getpid()))
	return id
}

// freshPacketCnt returns a per-run monotonic MIOTY packet counter so repeated
// probe runs never collide in the deduplicator.
func freshPacketCnt() uint32 {
	return uint32(time.Now().Unix() & 0x7FFFFFFF) //nolint:gosec
}

// frameOpID extracts the operation ID from a decoded frame.
func frameOpID(resp map[string]interface{}) int64 {
	switch v := resp["opId"].(type) {
	case int64:
		return v
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case uint64:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// TestMIOTYLoopProbe verifies the BS → SC → app path via BSSCI protocol.
// Connects to the BSSCI port, performs version negotiation, attach handshake
// with CMAC-signed signature, and UL data handshake.
func TestMIOTYLoopProbe(t *testing.T) {
	addr := bssciAddr()
	start := time.Now()

	// Seed endpoint before BSSCI protocol tests
	seedTestEndpoint(t)

	t.Run("Step1_BSSCIConnect", func(t *testing.T) {
		stepStart := time.Now()
		conn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: 5 * time.Second},
			"tcp", addr,
			bssciTLSConfig(t),
		)
		if err != nil {
			reportResult(t, probeResult{Probe: "bssci_connect", Status: "fail",
				Message:    fmt.Sprintf("TLS dial to %s failed: %v", addr, err),
				DurationMS: time.Since(stepStart).Milliseconds()})
			t.Fatalf("BSSCI server not reachable at %s: %v", addr, err)
		}
		defer func() { _ = conn.Close() }()
		reportResult(t, probeResult{Probe: "bssci_connect", Status: "pass",
			DurationMS: time.Since(stepStart).Milliseconds()})

		// Step 2: Version negotiation
		t.Run("Step2_VersionNegotiation", func(t *testing.T) {
			conReq := mioty.Connect{
				BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdConnect, OpId: 0},
				Version:     mioty.MIOTYProtocolVersion,
				BsEui:       1,
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
			reportResult(t, probeResult{Probe: "version_negotiation", Status: "pass"})
		})

		// Step 3: Attach handshake (strict attRsp required)
		// Compute CMAC signature matching seeded NwkSnKey per MIOTY radio spec §3.7.1.3.
		// EpEui=2 matches "0000000000000002" seeded in seedTestEndpoint.
		t.Run("Step3_AttachHandshake", func(t *testing.T) {
			// Monotonic across probe runs: replay protection requires the
			// attach counter to advance beyond the endpoint's stored value.
			attachCnt := uint32(time.Now().Unix() & 0xFFFFFF)
			sign := computeAttachSignature(2, attachCnt, probeNwkSnKey)
			att := mioty.Attach{
				BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdAttach, OpId: 1},
				EpEui:       2,
				RxTime:      time.Now().UnixNano(),
				AttachCnt:   attachCnt,
				Snr:         10.0,
				Rssi:        -80.0,
				Nonce:       [4]byte{0, 0, 0, 0},
				Sign:        sign,
			}
			writeFrame(t, conn, att)
			resp := awaitFrame(t, conn, mioty.CmdAttachResponse)
			require.Equal(t, mioty.CmdAttachResponse, resp["command"],
				"expected attRsp, got %v", resp["command"])

			attCmp := mioty.AttachComplete{
				BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdAttachComplete, OpId: 1},
			}
			writeFrame(t, conn, attCmp)
			reportResult(t, probeResult{Probe: "attach_handshake", Status: "pass"})
		})

		// Step 4: UL Data handshake (strict ulDataRsp required)
		t.Run("Step4_ULDataHandshake", func(t *testing.T) {
			ul := mioty.ULData{
				BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdULData, OpId: 2},
				EpEui:       2,
				RxTime:      time.Now().UnixNano(),
				PacketCnt:   freshPacketCnt(),
				Snr:         12.0,
				Rssi:        -75.0,
				UserData:    []byte{0xDE, 0xAD},
			}
			writeFrame(t, conn, ul)
			resp := awaitFrame(t, conn, mioty.CmdULDataResponse)
			require.Equal(t, mioty.CmdULDataResponse, resp["command"],
				"expected ulDataRsp, got %v", resp["command"])

			ulCmp := mioty.ULDataComplete{
				BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdULDataComplete, OpId: 2},
			}
			writeFrame(t, conn, ulCmp)
			reportResult(t, probeResult{Probe: "uldata_handshake", Status: "pass"})
		})
	})

	reportResult(t, probeResult{Probe: "mioty_loop_total", Status: "pass",
		DurationMS: time.Since(start).Milliseconds()})
}

// TestBSSCIPortReachable is a fast connectivity check for BSSCI.
func TestBSSCIPortReachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr := bssciAddr()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		t.Fatalf("BSSCI port %s not reachable: %v", addr, err)
		return
	}
	_ = conn.Close()
	t.Logf("BSSCI port %s is reachable", addr)
}
