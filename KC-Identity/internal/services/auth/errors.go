package auth

import "errors"

// Authentication service errors.
var (
	// ErrInvalidCredentials indicates email or password is incorrect
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserNotActive indicates user account is not in active state
	ErrUserNotActive = errors.New("user not active")

	// ErrNoPasswordSet indicates user has no password hash stored
	ErrNoPasswordSet = errors.New("no password set")

	// ErrInvalidPHCFormat indicates password hash is not valid PHC format
	ErrInvalidPHCFormat = errors.New("invalid PHC format")

	// ErrMembershipRequired indicates user has no active organization membership
	ErrMembershipRequired = errors.New("membership required")

	// ErrInvalidRefreshToken indicates refresh token is invalid or expired
	ErrInvalidRefreshToken = errors.New("invalid refresh token")

	// ErrRefreshTokenRevoked indicates refresh token has been explicitly revoked
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")

	// ErrRefreshTokenReused indicates refresh token was already used (reuse attack detected)
	ErrRefreshTokenReused = errors.New("refresh token reused")

	// ErrLocalLoginDisabled indicates local login is not enabled in config
	ErrLocalLoginDisabled = errors.New("local login disabled")

	// ErrRefreshTokenDisabled indicates refresh token rotation is not enabled in config
	ErrRefreshTokenDisabled = errors.New("refresh token disabled")

	// ErrUserPasswordWeak indicates password does not meet minimum requirements
	ErrUserPasswordWeak = errors.New("password too weak")

	// ErrUserNotFound indicates the requested user does not exist
	ErrUserNotFound = errors.New("user not found")

	// ErrTokenGenerationDisabled indicates token issuer is not configured.
	ErrTokenGenerationDisabled = errors.New("token generation disabled")
)

// External authentication service errors.
var (
	// ErrOIDCProviderDisabled indicates OIDC authentication is not enabled
	ErrOIDCProviderDisabled = errors.New("oidc provider disabled")

	// ErrOAuth2ProviderDisabled indicates OAuth2 authentication is not enabled
	ErrOAuth2ProviderDisabled = errors.New("oauth2 provider disabled")

	// ErrStateNotFound indicates auth state token was not found or expired
	ErrStateNotFound = errors.New("auth state not found")

	// ErrInvalidState indicates auth state token is malformed or doesn't match
	ErrInvalidState = errors.New("invalid auth state")

	// ErrNonceMismatch indicates OIDC nonce in ID token doesn't match stored nonce
	ErrNonceMismatch = errors.New("nonce mismatch")

	// ErrTokenExchangeFailed indicates OAuth2 token exchange failed
	ErrTokenExchangeFailed = errors.New("token exchange failed")

	// ErrUserInfoFailed indicates userinfo endpoint request failed
	ErrUserInfoFailed = errors.New("userinfo request failed")

	// ErrEmailNotVerified indicates user email is not verified (and assume_email_verified=false)
	ErrEmailNotVerified = errors.New("email not verified")

	// ErrRegistrationDisabled indicates new user registration via external auth is disabled
	ErrRegistrationDisabled = errors.New("registration disabled")

	// ErrIDTokenInvalid indicates OIDC ID token validation failed
	ErrIDTokenInvalid = errors.New("invalid ID token")

	// ErrRedisUnavailable indicates Redis is unreachable for state storage
	ErrRedisUnavailable = errors.New("redis unavailable")

	// ErrOrgResolutionFailed indicates external org claim could not be resolved to a valid organization
	ErrOrgResolutionFailed = errors.New("external org claim resolution failed")
)
