package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

func TestExchange_DeactivatedAccountIsRefusedLikeWrongCredentials(t *testing.T) {
	refused := func(context.Context, string, string) (*grpcservices.AuthLoginResult, error) {
		return nil, fmt.Errorf("find account: %w", auth.ErrUserNotActive)
	}
	svc := &IdentityService{
		audit:           discardAudit{},
		externalAuthSvc: &mockExternalAuthService{completeOIDCLoginFunc: refused, completeOAuth2LoginFunc: refused},
		log:             &mockLogger{},
	}
	exchanges := []struct {
		name     string
		exchange func(context.Context) (*pb.LoginResponse, error)
	}{
		{name: "ExchangeOIDC", exchange: func(ctx context.Context) (*pb.LoginResponse, error) {
			return svc.ExchangeOIDC(ctx, &pb.ExchangeOIDCRequest{Code: "code", State: "state"})
		}},
		{name: "ExchangeOAuth2", exchange: func(ctx context.Context) (*pb.LoginResponse, error) {
			return svc.ExchangeOAuth2(ctx, &pb.ExchangeOAuth2Request{Code: "code", State: "state"})
		}},
	}
	for _, tt := range exchanges {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.exchange(testutil.TestContext())

			assert.Nil(t, resp)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidCredentials), st.Code())
			assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidCredentials), st.Message())
		})
	}
}
