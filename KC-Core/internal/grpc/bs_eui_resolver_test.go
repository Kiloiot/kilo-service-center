package grpc

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// highBitEUI exceeds 2^53 and cannot round-trip through a JavaScript Number.
const highBitEUI uint64 = 0xCAFECAFECAFECAFE

func TestResolveBaseStationEUI(t *testing.T) {
	tests := []struct {
		name        string
		bsEuiHex    string
		legacyBsEui uint64
		want        uint64
		wantErrMsg  string
	}{
		{
			name:        "legacy numeric only",
			bsEuiHex:    "",
			legacyBsEui: 0x70B3D59CD00009E6,
			want:        0x70B3D59CD00009E6,
		},
		{
			name:     "hex only",
			bsEuiHex: "CAFECAFECAFECAFE",
			want:     highBitEUI,
		},
		{
			name:     "hex only dashed",
			bsEuiHex: "CA-FE-CA-FE-CA-FE-CA-FE",
			want:     highBitEUI,
		},
		{
			name:        "both present and equal",
			bsEuiHex:    "CAFECAFECAFECAFE",
			legacyBsEui: highBitEUI,
			want:        highBitEUI,
		},
		{
			name:        "both present and mismatched",
			bsEuiHex:    "CAFECAFECAFECAFE",
			legacyBsEui: 0x70B3D59CD00009E6,
			wantErrMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationEUIMismatch),
		},
		{
			name:       "missing both",
			bsEuiHex:   "",
			wantErrMsg: grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired),
		},
		{
			name:       "malformed hex",
			bsEuiHex:   "ZZZZCAFECAFECAFE",
			wantErrMsg: grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
		},
		{
			name:       "hex too short",
			bsEuiHex:   "CAFE",
			wantErrMsg: grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
		},
		{
			name:        "malformed hex with legacy fallback still rejected",
			bsEuiHex:    "not-a-eui",
			legacyBsEui: 0x70B3D59CD00009E6,
			wantErrMsg:  grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveBaseStationEUI(tt.bsEuiHex, tt.legacyBsEui)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error must be a gRPC status")
				assert.Equal(t, codes.InvalidArgument, st.Code())
				assert.Equal(t, tt.wantErrMsg, st.Message())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// euiSessionDir returns a non-nil session object for a single expected EUI.
type euiSessionDir struct {
	expectedEUI uint64
}

func (d *euiSessionDir) GetConnectedSessions() []map[string]interface{} { return nil }
func (d *euiSessionDir) GetSessionByEUI(eui uint64) interface{} {
	if eui == d.expectedEUI {
		return struct{}{}
	}
	return nil
}
func (d *euiSessionDir) SelectBidirectionalSession(_ int64, _ *uint64) (string, uint64, error) {
	return "", 0, nil
}
func (d *euiSessionDir) FindSessionForEndpointAttachment(_ uint64) (string, error) {
	return "", nil
}

// capturingStatusReq records that a status request was sent.
type capturingStatusReq struct {
	called bool
	opID   int64
}

func (r *capturingStatusReq) SendStatusRequest(_ interface{}) (int64, error) {
	r.called = true
	return r.opID, nil
}

// capturingPingCmd records the EUI passed to InitiatePing.
type capturingPingCmd struct {
	gotEUI uint64
	opID   int64
}

func (c *capturingPingCmd) InitiatePing(_ context.Context, bsEui uint64, _ int64) (int64, error) {
	c.gotEUI = bsEui
	return c.opID, nil
}

func highBitEUIBytes() []byte {
	return []byte{0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE}
}

func TestRequestBaseStationStatus_AcceptsBsEuiHex(t *testing.T) {
	var capturedEUIBytes []byte
	bsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, eui []byte, _ int64) (*models.BaseStation, error) {
			capturedEUIBytes = eui
			return &models.BaseStation{Name: "high-bit station"}, nil
		},
	}
	statusReq := &capturingStatusReq{opID: 42}
	svc := &CoreService{
		basestationSvc: bsSvc,
		sessionDir:     &euiSessionDir{expectedEUI: highBitEUI},
		statusReq:      statusReq,
		log:            &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	resp, err := svc.RequestBaseStationStatus(ctx, &pb.BaseStationStatusRequest{
		BsEuiHex: "CAFECAFECAFECAFE",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, int64(42), resp.OpId)
	assert.True(t, statusReq.called, "status request must be sent to the resolved session")
	assert.True(t, bytes.Equal(highBitEUIBytes(), capturedEUIBytes),
		"ownership check must receive big-endian bytes of the hex EUI")
}

func TestRequestBaseStationStatus_RejectsMissingEUI(t *testing.T) {
	svc := &CoreService{
		basestationSvc: &mockBasestationSvc{},
		sessionDir:     &euiSessionDir{},
		statusReq:      &capturingStatusReq{},
		log:            &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	_, err := svc.RequestBaseStationStatus(ctx, &pb.BaseStationStatusRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInitiatePing_AcceptsBsEuiHex(t *testing.T) {
	var capturedEUIBytes []byte
	bsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, eui []byte, _ int64) (*models.BaseStation, error) {
			capturedEUIBytes = eui
			return &models.BaseStation{Name: "high-bit station"}, nil
		},
	}
	pingCmd := &capturingPingCmd{opID: 7}
	svc := &CoreService{
		basestationSvc: bsSvc,
		pingCmd:        pingCmd,
		log:            &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	resp, err := svc.InitiatePing(ctx, &pb.InitiatePingRequest{
		BsEuiHex: "CA-FE-CA-FE-CA-FE-CA-FE",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, int64(7), resp.OpId)
	assert.Equal(t, highBitEUI, pingCmd.gotEUI,
		"ping must target the full-range EUI parsed from hex")
	assert.True(t, bytes.Equal(highBitEUIBytes(), capturedEUIBytes),
		"ownership check must receive big-endian bytes of the hex EUI")
}

func TestInitiatePing_RejectsMismatchedEUIs(t *testing.T) {
	svc := &CoreService{
		basestationSvc: &mockBasestationSvc{},
		pingCmd:        &capturingPingCmd{},
		log:            &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	_, err := svc.InitiatePing(ctx, &pb.InitiatePingRequest{
		BsEui:    0x70B3D59CD00009E6, //nolint:staticcheck // deprecated field set deliberately to prove the mismatch guard rejects it
		BsEuiHex: "CAFECAFECAFECAFE",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t,
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationEUIMismatch),
		status.Convert(err).Message())
}
