// Package proxy provides gRPC proxy forwarding and metadata handling for KC-Gateway.
package proxy

import (
	"context"
	"strconv"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// spoofableHeaders lists headers that must be stripped from inbound client requests
// before forwarding to KC-Core. KC-Gateway re-injects trusted values from interceptor context.
var spoofableHeaders = []string{
	grpcconst.MetadataKeyInternalTenantID,
	grpcconst.MetadataKeyInternalOrgID,
	grpcconst.MetadataKeyInternalUserID,
	grpcconst.MetadataKeyInternalServiceAccountID,
	grpcconst.MetadataKeyInternalPeerSecret,
	grpcconst.MetadataKeyAuthorization,
	grpcconst.MetadataKeyOrganizationID,
	grpcconst.MetadataKeyUserID,
}

// SanitizeAndInject strips spoofable headers, injects trusted identity and presents the peer secret when set.
func SanitizeAndInject(ctx context.Context, peerSecret string) metadata.MD {
	// Start with incoming metadata (or empty)
	inMD, _ := metadata.FromIncomingContext(ctx)
	outMD := inMD.Copy()

	// Strip all spoofable headers
	for _, key := range spoofableHeaders {
		delete(outMD, key)
	}

	// Inject trusted identity from interceptor context
	tenantID, err := pkgcontext.GetTenantID(ctx)
	if err == nil && tenantID > 0 {
		outMD.Set(grpcconst.MetadataKeyInternalTenantID, strconv.FormatInt(tenantID, 10))
	}

	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err == nil && orgID != uuid.Nil {
		outMD.Set(grpcconst.MetadataKeyInternalOrgID, orgID.String())
	}

	userID, err := pkgcontext.GetUserID(ctx)
	if err == nil && userID != "" {
		outMD.Set(grpcconst.MetadataKeyInternalUserID, userID)
	}

	if keyID, err := pkgcontext.GetServiceAccountID(ctx); err == nil {
		outMD.Set(grpcconst.MetadataKeyInternalServiceAccountID, keyID.String())
	}

	if peerSecret != "" {
		outMD.Set(grpcconst.MetadataKeyInternalPeerSecret, peerSecret)
	}

	return outMD
}
