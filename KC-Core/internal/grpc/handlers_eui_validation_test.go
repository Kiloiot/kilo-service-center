package grpc

import (
	"testing"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
)

// Endpoint EUIs are exactly eight bytes; anything else must be rejected
// before a queue id is drawn, a downlink is queued or an audit row is written.
var malformedEndpointEUIs = map[string]string{
	"empty":            "",
	"single digit":     "1",
	"seven bytes":      "70B3D59CD00000",
	"odd length":       "70B3D59CD000000",
	"non hexadecimal":  "70B3D59CD000000G",
	"nine bytes":       "70B3D59CD0000002AA",
	"whitespace inner": "70B3D59C D0000002",
}

func TestSendDownlink_RejectsMalformedEndpointEUI(t *testing.T) {
	for name, eui := range malformedEndpointEUIs {
		t.Run(name, func(t *testing.T) {
			queuer := &fakeSCACIQueuer{}
			sink := &captureAuditRecorder{}
			svc := testCoreService(coreFields{scaciQueuer: queuer, log: logger.NewNop(), audit: sink})

			_, err := svc.SendDownlink(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), &pb.SendDownlinkRequest{EpEui: eui, Payloads: [][]byte{{0x01}}})
			require.Error(t, err)
			assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEUIFormat), status.Code(err), "EUI %q", eui)
			assert.Zero(t, queuer.calls, "nothing may be queued for %q", eui)
			assert.Empty(t, sink.events, "no audit row may be written for %q", eui)
		})
	}
}

func TestSendULTransmit_RejectsMalformedEndpointEUI(t *testing.T) {
	for name, eui := range malformedEndpointEUIs {
		if eui == "" {
			continue // the empty EUI has its own required-field token
		}
		t.Run(name, func(t *testing.T) {
			called := false
			svc := testCoreService(coreFields{
				ulTransmit: &mockULTransmit{sendFunc: func(string, uint64, []byte, uint16, uint32, []byte, string, uint8) (int64, error) {
					called = true
					return 0, nil
				}},
				log: &mockLogger{},
			})
			_, err := svc.SendULTransmit(testutil.TestContextWithTenant(1), &pb.SendULTransmitRequest{
				EpEui: eui, BsEui: "70B3D59CD00009E6", NwkSnKey: testNetworkKey(), PacketCnt: 1, UserData: []byte{0x01},
			})
			require.Error(t, err)
			assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat), status.Code(err), "EUI %q", eui)
			assert.False(t, called, "transmit must not be attempted for %q", eui)
		})
	}
}
