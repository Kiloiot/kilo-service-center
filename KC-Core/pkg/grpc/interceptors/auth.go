// Package interceptors provides exported gRPC interceptors shared between KC-Core and KC-Gateway.
package interceptors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/config"
	grpcconst "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-DB/storage/models"
	pkgcontext "github.com/kilocenter/pkg/context"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor provides JWT and API key bearer authentication for gRPC.
type AuthInterceptor struct {
	enabled          bool
	keySet           jwk.Set
	hmacKey          []byte
	issuer           string
	audience         string
	tenantClaim      string
	userClaim        string
	orgResolver      OrganizationResolver
	tenantResolver   TenantResolver
	apiKeyAuth       APIKeyAuthenticator
	eventWriter      grpcconst.EventWriter
	platformTenantID int64
}

// NewAuthInterceptor creates a new authentication interceptor.
func NewAuthInterceptor(cfg AuthConfig) (*AuthInterceptor, error) {
	ai := &AuthInterceptor{
		enabled:          cfg.Enabled,
		issuer:           cfg.Issuer,
		audience:         cfg.Audience,
		tenantClaim:      cfg.TenantClaim,
		userClaim:        cfg.UserClaim,
		eventWriter:      cfg.EventWriter,
		platformTenantID: cfg.PlatformTenantID,
	}

	if !cfg.Enabled {
		return ai, nil
	}

	if cfg.JWKSEndpoint != "" {
		keySet, err := jwk.Fetch(context.Background(), cfg.JWKSEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", cfg.JWKSEndpoint, err)
		}
		ai.keySet = keySet
	} else if cfg.HMACSecret != "" {
		ai.hmacKey = []byte(cfg.HMACSecret)
	}

	if ai.tenantClaim == "" {
		ai.tenantClaim = "tenant_id"
	}

	if ai.userClaim == "" {
		ai.userClaim = config.AuthDefaultUserIDClaim
	}

	return ai, nil
}

// WithOrganizationResolver sets the organization resolver for the interceptor.
func (ai *AuthInterceptor) WithOrganizationResolver(resolver OrganizationResolver) *AuthInterceptor {
	ai.orgResolver = resolver
	return ai
}

// WithTenantResolver sets the tenant resolver for org UUID → tenant ID mapping.
func (ai *AuthInterceptor) WithTenantResolver(resolver TenantResolver) *AuthInterceptor {
	ai.tenantResolver = resolver
	return ai
}

// WithAPIKeyAuthenticator sets the API key authenticator for opaque bearer token support.
func (ai *AuthInterceptor) WithAPIKeyAuthenticator(auth APIKeyAuthenticator) *AuthInterceptor {
	ai.apiKeyAuth = auth
	return ai
}

// UnaryInterceptor returns the unary server interceptor.
func (ai *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if grpcconst.IsPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		newCtx, err := ai.authenticate(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// StreamInterceptor returns the stream server interceptor.
func (ai *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if grpcconst.IsPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		newCtx, err := ai.authenticate(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		wrappedStream := &authenticatedServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// authenticate validates the bearer token (JWT or API key) and extracts tenant/user information.
func (ai *AuthInterceptor) authenticate(ctx context.Context, method string) (context.Context, error) {
	if !ai.enabled {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "missing gRPC metadata")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenMissingMetadata),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingMetadata))
	}

	authorization := md.Get(grpcconst.MetadataKeyAuthorization)
	if len(authorization) == 0 {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "missing authorization header")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenMissingAuthToken),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingAuthToken))
	}

	tokenString := strings.TrimPrefix(authorization[0], grpcconst.BearerPrefix)
	if tokenString == authorization[0] {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "invalid authorization format")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenInvalidAuthFormat),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInvalidAuthFormat))
	}

	if isJWTShaped(tokenString) {
		return ai.authenticateJWT(ctx, tokenString, method)
	}

	if ai.apiKeyAuth != nil {
		return ai.authenticateAPIKey(ctx, tokenString, method)
	}

	ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "unrecognized token format")
	return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenInvalidToken),
		grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInvalidToken))
}

// isJWTShaped returns true if the token has the header.payload.signature structure of a JWT.
func isJWTShaped(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && len(parts[0]) > 0 && len(parts[1]) > 0 && len(parts[2]) > 0
}

// authenticateAPIKey validates an opaque bearer token as an API key.
func (ai *AuthInterceptor) authenticateAPIKey(ctx context.Context, tokenString string, method string) (context.Context, error) {
	hash := sha256.Sum256([]byte(tokenString))
	keyHash := hex.EncodeToString(hash[:])

	key, err := ai.apiKeyAuth.LookupByHash(ctx, keyHash)
	if err != nil {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthAPIKeyRejected, models.EventTitleAuthAPIKeyRejected, "API key not found")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenInvalidToken),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInvalidToken))
	}

	if !key.IsActive {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthAPIKeyRejected, models.EventTitleAuthAPIKeyRejected, "API key inactive")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenApiKeyInactive),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenApiKeyInactive))
	}

	if key.IsExpired {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthAPIKeyRejected, models.EventTitleAuthAPIKeyRejected, "API key expired")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenApiKeyExpired),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenApiKeyExpired))
	}

	_ = ai.apiKeyAuth.UpdateLastUsed(ctx, key.ID)

	ctx = pkgcontext.WithTenantID(ctx, key.TenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, key.OrganizationID)

	if key.UserID != nil {
		ctx = pkgcontext.WithUserID(ctx, key.UserID.String())
	}

	return ctx, nil
}

// authenticateJWT validates a JWT-shaped bearer token.
func (ai *AuthInterceptor) authenticateJWT(ctx context.Context, tokenString string, method string) (context.Context, error) {
	var token jwt.Token
	var err error
	if ai.hmacKey != nil {
		token, err = jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.HS256, ai.hmacKey))
	} else if ai.keySet != nil {
		token, err = jwt.Parse([]byte(tokenString), jwt.WithKeySet(ai.keySet))
	} else {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "no signing key configured")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenInvalidToken),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInvalidToken))
	}
	if err != nil {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "JWT validation failed")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenInvalidToken),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInvalidToken))
	}

	// Validate issuer
	if ai.issuer != "" {
		if iss, ok := token.Get("iss"); !ok || iss != ai.issuer {
			ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "invalid token issuer")
			return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenInvalidTokenIssuer),
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInvalidTokenIssuer))
		}
	}

	// Validate audience
	if ai.audience != "" {
		aud, ok := token.Get("aud")
		if !ok {
			ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "missing audience claim")
			return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenMissingAudience),
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingAudience))
		}

		audValid := false
		switch v := aud.(type) {
		case string:
			audValid = v == ai.audience
		case []string:
			for _, a := range v {
				if a == ai.audience {
					audValid = true
					break
				}
			}
		case []interface{}:
			for _, a := range v {
				if a == ai.audience {
					audValid = true
					break
				}
			}
		}

		if !audValid {
			ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "invalid audience")
			return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenInvalidAudience),
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInvalidAudience))
		}
	}

	// Extract tenant claim
	claimValue, ok := token.Get(ai.tenantClaim)
	if !ok {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "missing tenant claim")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenMissingTenantClaim),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingTenantClaim))
	}

	var tenantID int64
	var orgID uuid.UUID

	if claimStr, ok := claimValue.(string); ok {
		if parsedUUID, err := uuid.Parse(claimStr); err == nil {
			if ai.tenantResolver == nil {
				ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "tenant resolver not configured")
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenTenantResolverRequired),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenTenantResolverRequired))
			}
			orgID = parsedUUID
			resolvedTenant, err := ai.tenantResolver.LookupTenant(ctx, orgID)
			if err != nil {
				ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "org resolution failed")
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgResolutionFailed),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgResolutionFailed))
			}
			tenantID = resolvedTenant
		} else {
			tenantID = extractTenantFromClaims(token, ai.tenantClaim)
		}
	} else {
		tenantID = extractTenantFromClaims(token, ai.tenantClaim)
	}

	if tenantID == 0 {
		ai.emitSecurityEvent(ctx, method, models.EventTypeAuthInvalidToken, models.EventTitleAuthInvalidToken, "tenant claim resolved to zero")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenMissingTenantClaim),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingTenantClaim))
	}

	ctx = pkgcontext.WithTenantID(ctx, tenantID)

	if orgID != uuid.Nil {
		ctx = pkgcontext.WithOrganizationID(ctx, orgID)
	} else if ai.orgResolver != nil {
		resolvedOrg, err := ai.orgResolver.ResolveOrganization(ctx, tenantID)
		if err == nil && resolvedOrg != uuid.Nil {
			ctx = pkgcontext.WithOrganizationID(ctx, resolvedOrg)
		}
	}

	// Extract user ID from JWT token
	if ai.userClaim != "" {
		if userIDVal, ok := token.Get(ai.userClaim); ok {
			if userID, ok := userIDVal.(string); ok && userID != "" {
				ctx = pkgcontext.WithUserID(ctx, userID)
			}
		}
	}

	return ctx, nil
}

// extractTenantFromClaims extracts tenant ID from JWT claims.
func extractTenantFromClaims(token jwt.Token, claimPath string) int64 {
	if tenantVal, ok := token.Get(claimPath); ok {
		switch v := tenantVal.(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case string:
			var tenantID int64
			_, _ = fmt.Sscanf(v, "%d", &tenantID)
			return tenantID
		}
	}

	parts := strings.Split(claimPath, ":")
	if len(parts) > 1 {
		if val, ok := token.Get(claimPath); ok {
			switch v := val.(type) {
			case string:
				var tenantID int64
				_, _ = fmt.Sscanf(v, "%d", &tenantID)
				return tenantID
			case float64:
				return int64(v)
			}
		}
	}

	alternatives := []string{"tenant_id", "org_id", "organization_id", "project_id"}
	for _, alt := range alternatives {
		if val, ok := token.Get(alt); ok {
			switch v := val.(type) {
			case float64:
				return int64(v)
			case int64:
				return v
			case string:
				var tenantID int64
				_, _ = fmt.Sscanf(v, "%d", &tenantID)
				return tenantID
			}
		}
	}

	return 0
}

// authenticatedServerStream wraps a ServerStream with authenticated context.
type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}

// emitSecurityEvent persists a security event when an event writer is configured.
func (ai *AuthInterceptor) emitSecurityEvent(ctx context.Context, method, eventType, title, reason string) {
	if ai.eventWriter == nil {
		return
	}
	tenantID := ai.platformTenantID
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil {
		tenantID = tid
	}
	details, _ := json.Marshal(map[string]interface{}{
		"method": method,
		"reason": reason,
	})
	_ = ai.eventWriter.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   eventType,
		Category:    models.EventCategorySecurity,
		Severity:    models.EventSeverityWarning,
		Title:       title,
		Description: reason,
		SourceType:  models.SourceTypeAPI,
		SourceName:  method,
		Details:     details,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
}
