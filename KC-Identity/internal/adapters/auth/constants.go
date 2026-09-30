package auth

import "time"

// HTTP header names and values used by the identity transport adapters.
const (
	// HeaderContentType is the standard content-type header name.
	HeaderContentType = "Content-Type"
	// HeaderAuthorization is the standard authorization header name.
	HeaderAuthorization = "Authorization"
	// MediaTypeFormURLEncoded is the form-encoded request media type.
	MediaTypeFormURLEncoded = "application/x-www-form-urlencoded"
	// MediaTypeJSON is the JSON media type.
	MediaTypeJSON = "application/json"
	// BearerPrefix prefixes bearer-token authorization values.
	BearerPrefix = "Bearer "
)

// HTTPClientTimeout bounds every outbound identity-provider HTTP call.
const HTTPClientTimeout = 30 * time.Second

// ID-token validation failure reasons logged by the OIDC client.
const (
	oidcReasonInvalidFormat      = "invalid format"
	oidcReasonBase64DecodeFailed = "base64 decode failed"
	oidcReasonUnmarshalRawFailed = "json unmarshal raw failed"
	oidcReasonUnmarshalFailed    = "json unmarshal failed"
	oidcReasonIssuerMismatch     = "issuer mismatch"
	oidcReasonAudienceMismatch   = "audience mismatch"
	oidcReasonTokenExpired       = "token expired"
)
