package grpc

import (
	"fmt"
	"testing"

	"github.com/kilocenter/KC-Core/pkg/scaci"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestMapSCACIErrorToGRPC verifies SCACI error tokens map to expected gRPC status codes.
func TestMapSCACIErrorToGRPC(t *testing.T) {
	tokens := []struct {
		name     string
		token    string
		expected codes.Code
	}{
		// InvalidArgument
		{"CntDependMismatch", scaci.ErrCntDependMismatch, codes.InvalidArgument},
		{"CntDependPacketCntOmit", scaci.ErrCntDependPacketCntOmit, codes.InvalidArgument},
		{"NonCntDependMultiPayload", scaci.ErrNonCntDependMultiPayload, codes.InvalidArgument},
		{"QueueIDOutOfRange", scaci.ErrQueueIDOutOfRange, codes.InvalidArgument},
		{"FailedPersistDownlink", scaci.ErrFailedPersistDownlink, codes.InvalidArgument},
		// AlreadyExists
		{"QueIDExists", scaci.ErrQueIDExists, codes.AlreadyExists},
		// Unavailable
		{"SchedulerUnavailable", scaci.ErrSchedulerUnavailable, codes.Unavailable},
		{"BaseStationUnavailable", scaci.ErrBaseStationUnavailable, codes.Unavailable},
		// NotFound
		{"EndpointNotFound", scaci.ErrEndpointNotFound, codes.NotFound},
		{"DownlinkNotFound", scaci.ErrDownlinkNotFound, codes.NotFound},
		// Unknown → Internal
		{"UnknownToken", "scaci.error.unknown_token", codes.Internal},
	}

	for _, tc := range tokens {
		t.Run(tc.name, func(t *testing.T) {
			err := fmt.Errorf("SCACI dlDataQue failed: %s", tc.token)
			if got := status.Code(mapSCACIErrorToGRPC(err)); got != tc.expected {
				t.Fatalf("mapSCACIErrorToGRPC(%s) = %v, want %v", tc.token, got, tc.expected)
			}
		})
	}
}
