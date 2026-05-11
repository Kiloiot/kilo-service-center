// Package adapters provides adapter implementations for KC-Core service interfaces.
package adapters

import (
	"context"
	"errors"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
)

// StateStoreAdapter provides OIDC/OAuth2 state storage with key prefixing and TTL.
type StateStoreAdapter struct {
	redis     *auth.RedisClient
	keyPrefix string // "auth:oidc:" or "auth:oauth2:"
}

// Ensure StateStoreAdapter implements StateStore interface.
var _ auth.StateStore = (*StateStoreAdapter)(nil)

// NewOIDCStateStoreAdapter creates a state store for OIDC auth flows.
// Uses auth:oidc: key prefix.
func NewOIDCStateStoreAdapter(redis *auth.RedisClient) *StateStoreAdapter {
	return &StateStoreAdapter{
		redis:     redis,
		keyPrefix: config.AuthRedisKeyPrefixOIDC,
	}
}

// NewOAuth2StateStoreAdapter creates a state store for OAuth2 auth flows.
// Uses auth:oauth2: key prefix.
func NewOAuth2StateStoreAdapter(redis *auth.RedisClient) *StateStoreAdapter {
	return &StateStoreAdapter{
		redis:     redis,
		keyPrefix: config.AuthRedisKeyPrefixOAuth2,
	}
}

// StoreState stores a state token with associated data and TTL.
// key: state token (random string), value: JSON-serialized State
func (a *StateStoreAdapter) StoreState(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := a.keyPrefix + key
	return a.redis.Set(ctx, fullKey, value, ttl)
}

// GetState retrieves and deletes state atomically (replay prevention).
// Returns auth.ErrStateNotFound if key doesn't exist or expired.
func (a *StateStoreAdapter) GetState(ctx context.Context, key string) ([]byte, error) {
	fullKey := a.keyPrefix + key
	value, err := a.redis.GetDel(ctx, fullKey)
	if err != nil {
		if errors.Is(err, auth.ErrKeyNotFound) {
			return nil, auth.ErrStateNotFound
		}
		return nil, err
	}
	return value, nil
}

// DeleteState explicitly removes a state token.
// Used for cleanup when auth flow is cancelled or fails.
func (a *StateStoreAdapter) DeleteState(ctx context.Context, key string) error {
	fullKey := a.keyPrefix + key
	return a.redis.Del(ctx, fullKey)
}
