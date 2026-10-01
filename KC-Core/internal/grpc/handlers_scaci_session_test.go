package grpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	scacimonitoring "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci_monitoring"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

const testLogLevelError = "error"

// scaciSessionLookup fails every session lookup with err.
type scaciSessionLookup struct {
	retainedFakes
	err error
}

func (f *scaciSessionLookup) GetSession(context.Context, int64, string) (*grpcservices.ScaciSession, error) {
	return nil, f.err
}

func TestGetScaciSession_MapsEachLookupFailure(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		token  string
		logged bool
	}{
		{name: "unknown session", err: scacimonitoring.ErrSessionNotFound, token: grpcerrors.ErrTokenScaciSessionNotFound},
		{name: "malformed id", err: fmt.Errorf("%w: %w", scacimonitoring.ErrInvalidSessionID, strconv.ErrSyntax), token: grpcerrors.ErrTokenInvalidIDFormat},
		{name: "store failure", err: errors.New("session store unavailable"), token: grpcerrors.ErrTokenInternalError, logged: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			log := newCapturingLogger()
			svc := testCoreService(coreFields{scaciMonitorSvc: &scaciSessionLookup{err: tc.err}, log: log})

			_, err := svc.GetScaciSession(ownerCtx(), &pb.GetScaciSessionRequest{Id: retainedSessionID})

			assert.Equal(t, grpcerrors.GetGRPCCode(tc.token), status.Code(err))
			assert.Equal(t, grpcerrors.ResolveErrorMessage(tc.token), status.Convert(err).Message())
			assert.Equal(t, tc.logged, log.hasEntry(testLogLevelError, LogGetSCACISessionFailed), "only an unexpected failure is logged as an error")
		})
	}
}
