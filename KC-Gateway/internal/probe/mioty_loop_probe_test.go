//go:build integration

package probe

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aead/cmac"
	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/kilocenter/KC-Core/pkg/grpc"
	mioty "github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// bssciAddr returns the BSSCI server address from env or default.
func bssciAddr() string {
	if addr := os.Getenv("BSSCI_ADDR"); addr != "" {
		return addr
	}
	return "localhost:5000"
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
	frame := mioty.Frame{Identifier: mioty.MIOTYFrameIdentifier, Payload: payload}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write(frame.Serialize())
	require.NoError(t, err, "write frame")
}

func readFrame(t *testing.T, conn net.Conn) map[string]interface{} {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	header := make([]byte, 12)
	_, err := io.ReadFull(conn, header)
	require.NoError(t, err, "read frame header")
	require.True(t, bytes.Equal(header[:8], mioty.MIOTYFrameIdentifier[:]),
		"expected MIOTYB01, got %s", string(header[:8]))
	size := binary.LittleEndian.Uint32(header[8:])
	require.Less(t, size, uint32(1024*1024), "payload too large")
	buf := make([]byte, size)
	_, err = io.ReadFull(conn, buf)
	require.NoError(t, err, "read frame payload")
	var resp map[string]interface{}
	require.NoError(t, msgpack.Unmarshal(buf, &resp), "unmarshal response")
	return resp
}

// probeNwkSnKey is the 16-byte preshared key seeded with the test endpoint.
// Matches testPresharedKey() in attach_replay_protection_test.go.
var probeNwkSnKey = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

// computeAttachSignature computes CMAC signature per MIOTY radio spec §3.7.1.3.
// Pattern from KC-Core/pkg/bssci/attach_replay_protection_test.go:29.
func computeAttachSignature(epEUI uint64, attachCnt uint32, presharedKey []byte) [4]byte {
	iv := make([]byte, 15)
	binary.BigEndian.PutUint64(iv[0:8], epEUI)
	iv[8] = 0xFF
	iv[9] = 0x00
	maskedCnt := attachCnt & 0xFFFFFF
	iv[10] = byte(maskedCnt >> 16)
	iv[11] = byte(maskedCnt >> 8)
	iv[12] = byte(maskedCnt)
	iv[13] = 0xFF
	iv[14] = 0xFF

	block, _ := aes.NewCipher(presharedKey)
	mac, _ := cmac.New(block)
	mac.Write(iv) //nolint:errcheck
	result := mac.Sum(nil)
	var sig [4]byte
	copy(sig[:], result[:4])
	return sig
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
//   - MetadataKeyInternalUserID (optional but included for completeness, valid UUID)
func seedTestEndpoint(t *testing.T) {
	t.Helper()
	conn, err := grpc.NewClient(coreInternalAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "connect to KC-Core internal gRPC")
	defer func() { _ = conn.Close() }()

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
			&tls.Config{InsecureSkipVerify: true}, //nolint:gosec
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
				SnBsUuid:    [16]byte{1},
			}
			writeFrame(t, conn, conReq)
			resp := readFrame(t, conn)
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
			var attachCnt uint32 = 1
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
			resp := readFrame(t, conn)
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
				PacketCnt:   1,
				Snr:         12.0,
				Rssi:        -75.0,
				UserData:    []byte{0xDE, 0xAD},
			}
			writeFrame(t, conn, ul)
			resp := readFrame(t, conn)
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
