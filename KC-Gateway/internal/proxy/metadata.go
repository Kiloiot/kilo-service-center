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
	grpcconst.MetadataKeyInternalPeerSecret,
	grpcconst.MetadataKeyAuthorization,
	grpcconst.MetadataKeyOrganizationID,
	grpcconst.MetadataKeyUserID,
}

// SanitizeAndInject strips spoofable headers from inbound metadata and injects
// trusted identity values from interceptor-populated context.
// Returns the outgoing metadata for the upstream KC-Core call.
func SanitizeAndInject(ctx context.Context) metadata.MD {
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

	return outMD
}
