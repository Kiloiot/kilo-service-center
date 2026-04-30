// Package auth provides authentication and user management services.
package auth

// Authentication protocol constants.

// ============================================================================
// PKCE Constants
// ============================================================================

const (
	// PKCEMethodS256 is the SHA-256 PKCE code challenge method (recommended).
	PKCEMethodS256 = "S256"

	// PKCEMethodPlain is the plain PKCE code challenge method (not recommended).
	PKCEMethodPlain = "plain"

	// PKCEVerifierLength is the length of the PKCE code verifier (43-128 chars per RFC 7636).
	PKCEVerifierLength = 64
)

// ============================================================================
// OAuth2/OIDC Protocol Parameter Constants
// ============================================================================

const (
	// ParamState is the OAuth2 state parameter name.
	ParamState = "state"

	// ParamCode is the OAuth2 authorization code parameter name.
	ParamCode = "code"

	// ParamError is the OAuth2 error parameter name.
	ParamError = "error"

	// ParamErrorDescription is the OAuth2 error description parameter name.
	ParamErrorDescription = "error_description"

	// ParamCodeChallenge is the PKCE code_challenge parameter name.
	ParamCodeChallenge = "code_challenge"

	// ParamCodeChallengeMethod is the PKCE code_challenge_method parameter name.
	ParamCodeChallengeMethod = "code_challenge_method"

	// ParamCodeVerifier is the PKCE code_verifier parameter name.
	ParamCodeVerifier = "code_verifier"

	// ParamNonce is the OIDC nonce parameter name.
	ParamNonce = "nonce"

	// ParamResponseType is the OAuth2 response_type parameter name.
	ParamResponseType = "response_type"

	// ParamClientID is the OAuth2 client_id parameter name.
	ParamClientID = "client_id"

	// ParamClientSecret is the OAuth2 client_secret parameter name.
	ParamClientSecret = "client_secret" //nolint:gosec // Parameter name constant

	// ParamRedirectURI is the OAuth2 redirect_uri parameter name.
	ParamRedirectURI = "redirect_uri"

	// ParamScope is the OAuth2 scope parameter name.
	ParamScope = "scope"

	// ParamGrantType is the OAuth2 grant_type parameter name.
	ParamGrantType = "grant_type"

	// ResponseTypeCode is the authorization code response type.
	ResponseTypeCode = "code"

	// GrantTypeAuthorizationCode is the authorization_code grant type.
	GrantTypeAuthorizationCode = "authorization_code"
)

// ============================================================================
// OIDC Well-Known Endpoints and Claims
// Note: ProviderOIDC, ProviderOAuth2 are defined in external_service.go
// ============================================================================

const (
	// OIDCWellKnownPath is the OIDC discovery endpoint path.
	OIDCWellKnownPath = "/.well-known/openid-configuration"

	// OIDCClaimSub is the standard subject claim.
	OIDCClaimSub = "sub"

	// OIDCClaimEmail is the email claim.
	OIDCClaimEmail = "email"

	// OIDCClaimEmailVerified is the email_verified claim.
	OIDCClaimEmailVerified = "email_verified"

	// OIDCClaimName is the name claim.
	OIDCClaimName = "name"

	// OIDCClaimNonce is the nonce claim in ID token.
	OIDCClaimNonce = "nonce"

	// OIDCClaimAud is the audience claim.
	OIDCClaimAud = "aud"

	// OIDCClaimIss is the issuer claim.
	OIDCClaimIss = "iss"

	// OIDCClaimExp is the expiration claim.
	OIDCClaimExp = "exp"

	// OIDCClaimIat is the issued-at claim.
	OIDCClaimIat = "iat"
)

// ============================================================================
// State Token Constants
// Note: StateTokenLength, NonceLength are defined in external_service.go
// ============================================================================
