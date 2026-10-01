package interceptors

import (
	"context"
	"crypto/subtle"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// PeerAuthenticator admits every call when no secret is set, which configuration allows only on a loopback address.
type PeerAuthenticator struct {
	secret []byte
}

// NewPeerAuthenticator authenticates internal peers against secret.
func NewPeerAuthenticator(secret string) PeerAuthenticator {
	return PeerAuthenticator{secret: []byte(secret)}
}

// Authenticate fails closed on a missing, repeated or different secret.
func (p PeerAuthenticator) Authenticate(ctx context.Context) error {
	if len(p.secret) == 0 {
		return nil
	}
	md, _ := metadata.FromIncomingContext(ctx)
	presented := md.Get(grpcconst.MetadataKeyInternalPeerSecret)
	if len(presented) != 1 || subtle.ConstantTimeCompare([]byte(presented[0]), p.secret) != 1 {
		return status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenGatewayInternalPeerRejected),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenGatewayInternalPeerRejected))
	}
	return nil
}
