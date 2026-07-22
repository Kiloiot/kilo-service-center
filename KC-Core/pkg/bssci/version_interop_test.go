package bssci

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// interopSCEui exercises exact outbound encoding: the high bit is set and the
// value exceeds float64 exact-integer range, so any lossy projection corrupts it.
const interopSCEui = uint64(0xFFFFFFFFFFFFFFFF)

// interopConnectionService mirrors the production registry semantics for a
// base station registered under tenant 1 (any EUI accepted).
type interopConnectionService struct{}

func (interopConnectionService) GetBaseStation(_ context.Context, eui [8]byte, _ *basestation.ConnectionManager) (*basestation.BaseStation, error) {
	return &basestation.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: "Interop BS"}, nil
}

func (interopConnectionService) GetBaseStationGlobal(_ context.Context, eui [8]byte, _ *basestation.ConnectionManager) (*basestation.BaseStation, error) {
	return &basestation.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: "Interop BS"}, nil
}

func (interopConnectionService) RegisterConnection(_ context.Context, _ *Session, _ *basestation.BaseStation, _ *basestation.ConnectionManager) error {
	return nil
}

// interopHarness drives a real Server through net.Pipe with framed wire
// traffic in the configured encoding.
type interopHarness struct {
	t        *testing.T
	conn     net.Conn
	encoding string
	server   *Server
	done     chan struct{}
}

func startInteropServer(t *testing.T, encoding string) *interopHarness {
	t.Helper()

	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(log, nil)

	server := NewTestServer(log, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, interopConnectionService{}, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{
		ServiceCenterEUI:    interopSCEui,
		Vendor:              "test-vendor",
		Model:               "test-model",
		Name:                "test-sc",
		SoftwareVersion:     "1.0.0",
		MessageEncoding:     encoding,
		OperationAckTimeout: 2 * time.Second,
	}
	server.RegisterHandlers()
	ctx, cancel := context.WithCancel(context.Background())
	server.ctx = ctx
	server.cancel = cancel
	t.Cleanup(cancel)

	client, srvSide := net.Pipe()
	done := make(chan struct{})
	server.wg.Add(1)
	go func() {
		defer close(done)
		server.handleConnection(srvSide)
	}()
	t.Cleanup(func() {
		_ = client.Close()
		<-done
	})

	return &interopHarness{t: t, conn: client, encoding: encoding, server: server, done: done}
}

func (h *interopHarness) writeFrame(payload map[string]interface{}) {
	h.t.Helper()
	raw, err := encodeMessage(payload, h.encoding)
	require.NoError(h.t, err)

	header := make([]byte, HeaderSize)
	copy(header[:8], mioty.MIOTYFrameIdentifier[:])
	binary.LittleEndian.PutUint32(header[8:], uint32(len(raw)))

	require.NoError(h.t, h.conn.SetWriteDeadline(time.Now().Add(2*time.Second)))
	_, err = h.conn.Write(append(header, raw...))
	require.NoError(h.t, err)
}

// readFrame decodes the next frame; JSON numbers surface as json.Number so
// exact values can be asserted.
func (h *interopHarness) readFrame() map[string]interface{} {
	h.t.Helper()
	raw := h.readFrameRaw()
	decoded, err := decodeMessage(raw, h.encoding)
	require.NoError(h.t, err)
	return decoded
}

func (h *interopHarness) readFrameRaw() []byte {
	h.t.Helper()
	require.NoError(h.t, h.conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	header := make([]byte, HeaderSize)
	_, err := io.ReadFull(h.conn, header)
	require.NoError(h.t, err)
	require.True(h.t, bytes.Equal(header[:8], mioty.MIOTYFrameIdentifier[:]), "frame identifier")

	payload := make([]byte, binary.LittleEndian.Uint32(header[8:]))
	_, err = io.ReadFull(h.conn, payload)
	require.NoError(h.t, err)
	return payload
}

// expectClosed asserts the server side closes the connection.
func (h *interopHarness) expectClosed() {
	h.t.Helper()
	// A closed pipe can already fail the deadline call - that is the
	// expected outcome
	if err := h.conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		require.ErrorIs(h.t, err, io.ErrClosedPipe)
		return
	}
	buf := make([]byte, 1)
	_, err := h.conn.Read(buf)
	require.Error(h.t, err, "connection must be closed by the service center")
	require.True(h.t, errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe),
		"expected closed connection, got %v", err)
}

func connectPayload(version string, bsEui interface{}) map[string]interface{} {
	// snBsUuid is mandatory in con (§5.3.1); a JSON encoder emits it as a
	// numeric array, msgpack as a binary - the numeric array decodes
	// correctly under both
	snBsUUID := make([]interface{}, 16)
	for i := range snBsUUID {
		snBsUUID[i] = int64(i + 1)
	}
	payload := map[string]interface{}{
		"command":  mioty.CmdConnect,
		"opId":     int64(0),
		"bsEui":    bsEui,
		"bidi":     true,
		"snBsUuid": snBsUUID,
	}
	if version != "" {
		payload["version"] = version
	}
	return payload
}

func frameCommand(frame map[string]interface{}) string {
	cmd, _ := frame["command"].(string)
	return cmd
}

func frameUint64(t *testing.T, frame map[string]interface{}, key string) uint64 {
	t.Helper()
	v, err := coerceUint64(frame[key])
	require.NoError(t, err, "field %s must coerce exactly (got %T)", key, frame[key])
	return v
}

// TestBSSCIVersionInterop_NegotiationMatrix drives the full connect version
// arbitration over real framed traffic (rev1 §4.2, §5.3.2): compatible
// requests receive the service center's selected version in conRsp and the
// session activates on conCmp.
func TestBSSCIVersionInterop_NegotiationMatrix(t *testing.T) {
	compatible := []string{"1.0.0", "1.0.99", "1.1.0", "1.5.7"}
	for _, encoding := range []string{EncodingJSON, EncodingMessagePack} {
		for _, requested := range compatible {
			t.Run(fmt.Sprintf("%s_%s", encoding, requested), func(t *testing.T) {
				h := startInteropServer(t, encoding)
				h.writeFrame(connectPayload(requested, uint64(0x70B3D59CD00009E6)))

				conRsp := h.readFrame()
				require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp))
				assert.Equal(t, mioty.MIOTYProtocolVersion, conRsp["version"],
					"conRsp must carry the service center's selected version")

				// Base station agrees by completing the operation
				h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})

				// An active session answers ping with pingRsp (BSSCI §5.4)
				h.writeFrame(map[string]interface{}{"command": mioty.CmdPing, "opId": int64(1)})
				pingRsp := h.readFrame()
				assert.Equal(t, mioty.CmdPingResponse, frameCommand(pingRsp))
			})
		}
	}
}

// TestBSSCIVersionInterop_IncompatibleMajor verifies con with a different
// major version follows error, errorAck, close (§4.1, §5.17).
func TestBSSCIVersionInterop_IncompatibleMajor(t *testing.T) {
	for _, encoding := range []string{EncodingJSON, EncodingMessagePack} {
		for _, requested := range []string{"0.9.0", "2.0.0"} {
			t.Run(fmt.Sprintf("%s_%s", encoding, requested), func(t *testing.T) {
				h := startInteropServer(t, encoding)
				h.writeFrame(connectPayload(requested, uint64(0x70B3D59CD00009E6)))

				errFrame := h.readFrame()
				require.Equal(t, mioty.CmdError, frameCommand(errFrame))
				code, err := coerceInt64(errFrame["code"])
				require.NoError(t, err)
				assert.Equal(t, int64(POSIX_EPROTO), code)

				// The error acknowledgement completes the failed operation
				h.writeFrame(map[string]interface{}{"command": mioty.CmdErrorAck, "opId": int64(0)})
				h.expectClosed()
			})
		}
	}
}

// TestBSSCIVersionInterop_BaseStationRejectsOffer verifies the base station
// rejecting the offered version: con, conRsp, error, errorAck, close (§5.17).
func TestBSSCIVersionInterop_BaseStationRejectsOffer(t *testing.T) {
	for _, encoding := range []string{EncodingJSON, EncodingMessagePack} {
		t.Run(encoding, func(t *testing.T) {
			h := startInteropServer(t, encoding)
			h.writeFrame(connectPayload("1.1.0", uint64(0x70B3D59CD00009E6)))

			conRsp := h.readFrame()
			require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp))

			// Base station cannot accept the offered version
			h.writeFrame(map[string]interface{}{
				"command": mioty.CmdError,
				"opId":    int64(0),
				"code":    int64(POSIX_EPROTO),
				"message": "unsupported version",
			})

			ack := h.readFrame()
			assert.Equal(t, mioty.CmdErrorAck, frameCommand(ack))
			h.expectClosed()
		})
	}
}

// TestBSSCIVersionInterop_MissingAndMalformedVersion verifies the mandatory
// version field over the wire (rev1 §5.3.1).
func TestBSSCIVersionInterop_MissingAndMalformedVersion(t *testing.T) {
	cases := map[string]string{"missing": "", "malformed": "1.0", "signed": "-1.0.0"}
	for _, encoding := range []string{EncodingJSON, EncodingMessagePack} {
		for name, version := range cases {
			t.Run(fmt.Sprintf("%s_%s", encoding, name), func(t *testing.T) {
				h := startInteropServer(t, encoding)
				h.writeFrame(connectPayload(version, uint64(0x70B3D59CD00009E6)))

				errFrame := h.readFrame()
				require.Equal(t, mioty.CmdError, frameCommand(errFrame))
			})
		}
	}
}

// TestBSSCIVersionInterop_UnknownFieldsIgnored verifies §2.4 forward
// compatibility: BSSCI 1.1 fields such as bsClass and subchan are dropped
// with a warning and the connect succeeds.
func TestBSSCIVersionInterop_UnknownFieldsIgnored(t *testing.T) {
	for _, encoding := range []string{EncodingJSON, EncodingMessagePack} {
		t.Run(encoding, func(t *testing.T) {
			h := startInteropServer(t, encoding)
			payload := connectPayload("1.1.0", uint64(0x70B3D59CD00009E6))
			payload["bsClass"] = int64(1)
			payload["subchan"] = int64(3)
			h.writeFrame(payload)

			conRsp := h.readFrame()
			require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp))
			assert.Equal(t, mioty.MIOTYProtocolVersion, conRsp["version"])
		})
	}
}

// TestBSSCIVersionInterop_EUI64Matrix verifies the full unsigned EUI-64 range
// end to end over both encodings: inbound bsEui values above INT64_MAX are
// accepted and the outbound scEui survives exactly.
func TestBSSCIVersionInterop_EUI64Matrix(t *testing.T) {
	euis := []uint64{
		0x0000000000000001,
		0x7FFFFFFFFFFFFFFF,
		0x8000000000000000,
		0xCAFECAFECAFECAFE,
		0xFFFFFFFFFFFFFFFF,
	}
	for _, encoding := range []string{EncodingJSON, EncodingMessagePack} {
		for _, eui := range euis {
			t.Run(fmt.Sprintf("%s_%016X", encoding, eui), func(t *testing.T) {
				h := startInteropServer(t, encoding)

				var bsEui interface{} = eui
				if encoding == EncodingJSON {
					// JSON wire input for the write helper: json.Marshal of
					// uint64 emits the exact decimal digits
					bsEui = json.Number(fmt.Sprintf("%d", eui))
				}
				h.writeFrame(connectPayload("1.0.0", bsEui))

				conRsp := h.readFrame()
				require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp),
					"bsEui %016X must be accepted", eui)

				// Outbound scEui must be bit-exact in the emitted frame
				assert.Equal(t, interopSCEui, frameUint64(t, conRsp, "scEui"),
					"outbound scEui must survive encoding exactly")

				h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
				h.writeFrame(map[string]interface{}{"command": mioty.CmdPing, "opId": int64(1)})
				pingRsp := h.readFrame()
				assert.Equal(t, mioty.CmdPingResponse, frameCommand(pingRsp))
			})
		}
	}
}

// TestBSSCIVersionInterop_OliverScenario replays the exact field report:
// an AVA base station (BSSCI 1.1, EUI CA-FE-CA-FE-CA-FE-CA-FE) connects over
// MessagePack, negotiates down to 1.0.0, and activates.
func TestBSSCIVersionInterop_OliverScenario(t *testing.T) {
	h := startInteropServer(t, EncodingMessagePack)

	payload := connectPayload("1.1.0", uint64(0xCAFECAFECAFECAFE))
	payload["bsClass"] = int64(1)
	payload["subchan"] = int64(3)
	payload["vendor"] = "DIEHL Metering"
	payload["model"] = "AVA"
	h.writeFrame(payload)

	conRsp := h.readFrame()
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp),
		"the 1.1 base station must receive conRsp, not error 71")
	assert.Equal(t, mioty.MIOTYProtocolVersion, conRsp["version"])

	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
	h.writeFrame(map[string]interface{}{"command": mioty.CmdPing, "opId": int64(1)})
	pingRsp := h.readFrame()
	assert.Equal(t, mioty.CmdPingResponse, frameCommand(pingRsp))
}
