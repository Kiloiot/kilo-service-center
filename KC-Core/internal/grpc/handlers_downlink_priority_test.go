package grpc

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// A downlink priority is a finite single-precision number (SCACI §3.10.1):
// every gRPC ingress refuses NaN and infinities before anything is queued or
// patched.
func TestDownlinkIngress_RefusesNonFinitePriority(t *testing.T) {
	for name, prio := range map[string]float32{
		"NaN":       float32(math.NaN()),
		"+Infinity": float32(math.Inf(1)),
		"-Infinity": float32(math.Inf(-1)),
	} {
		t.Run(name, func(t *testing.T) {
			svc := newRetainedService(t, &retainedFakes{})
			queuer := &fakeSCACIQueuer{}
			pending := &pendingMessageSvc{}
			svc.useDownlinks(downlinkFakes{queuer: queuer, messages: pending})

			_, sendErr := svc.SendDownlink(actorCtx(), &pb.SendDownlinkRequest{EpEui: auditTestEpEUI, Payloads: [][]byte{{0x01}}, Priority: prio})
			_, updateErr := svc.UpdatePendingDownlink(actorCtx(), &pb.UpdatePendingDownlinkRequest{
				EpEui: pendingTestEpEUI, QueId: pendingTestQueID, Payloads: [][]byte{{0xAA}}, Priority: prio,
			})

			for _, err := range []error{sendErr, updateErr} {
				assertCode(t, err, grpcerrors.ErrTokenDownlinkPriorityInvalid)
				assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDownlinkPriorityInvalid), status.Convert(err).Message())
			}
			assert.Zero(t, queuer.calls, "nothing is queued")
			assert.Zero(t, pending.calls, "nothing is patched")
		})
	}
}
