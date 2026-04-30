// Package auth provides authentication and user management services.
package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/logger"
)

// RedisClient provides state storage for OIDC/OAuth2 flows.
type RedisClient struct {
	client *redis.Client
	logger logger.Logger
}

// NewRedisClient creates a new Redis client from canonical KC-Core config.
// Returns error if connection fails within timeout.
func NewRedisClient(cfg *config.RedisConfig, log logger.Logger) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         formatRedisAddr(cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.Database,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  config.AuthRedisConnectTimeout,
		ReadTimeout:  config.AuthRedisConnectTimeout,
		WriteTimeout: config.AuthRedisConnectTimeout,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), config.AuthRedisConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.New(LogRedisConnectionFailed + ": " + err.Error())
	}

	log.Info(LogRedisConnected, "host", cfg.Host, "port", cfg.Port, "db", cfg.Database)

	return &RedisClient{
		client: client,
		logger: log,
	}, nil
}

// Client returns the underlying redis.Client for health check integration.
// Required by health.NewRedisChecker for liveness probes.
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// Set stores a value with TTL expiration.
func (r *RedisClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		r.logger.ErrorContext(ctx, LogRedisSetFailed, "key", key, "error", err)
		return err
	}
	return nil
}

// Get retrieves a value by key.
// Returns ErrKeyNotFound if key doesn't exist.
func (r *RedisClient) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrKeyNotFound
		}
		r.logger.ErrorContext(ctx, LogRedisGetFailed, "key", key, "error", err)
		return nil, err
	}
	return val, nil
}

// GetDel retrieves and deletes a value atomically (replay prevention).
// Returns ErrKeyNotFound if key doesn't exist.
// Includes fallback for Redis < 6.2 which doesn't support GETDEL command.
func (r *RedisClient) GetDel(ctx context.Context, key string) ([]byte, error) {
	// Try GETDEL first (Redis 6.2+)
	val, err := r.client.GetDel(ctx, key).Bytes()
	if err == nil {
		return val, nil
	}

	// Fallback: GET + DEL for Redis < 6.2 (check for "unknown command" error)
	if strings.Contains(err.Error(), "unknown command") {
		result, getErr := r.client.Get(ctx, key).Bytes()
		if getErr != nil {
			if errors.Is(getErr, redis.Nil) {
				return nil, ErrKeyNotFound
			}
			r.logger.ErrorContext(ctx, LogRedisGetDelFailed, "key", key, "error", getErr)
			return nil, getErr
		}
		// Best-effort delete (non-atomic with GET, but acceptable for state tokens)
		_ = r.client.Del(ctx, key)
		return result, nil
	}

	// Handle redis.Nil and other errors
	if errors.Is(err, redis.Nil) {
		return nil, ErrKeyNotFound
	}
	r.logger.ErrorContext(ctx, LogRedisGetDelFailed, "key", key, "error", err)
	return nil, err
}

// Del removes a key.
func (r *RedisClient) Del(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.ErrorContext(ctx, LogRedisDelFailed, "key", key, "error", err)
		return err
	}
	return nil
}

// Close closes the Redis connection.
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// formatRedisAddr formats host:port address string.
func formatRedisAddr(host string, port int) string {
	return host + ":" + formatPort(port)
}

// formatPort converts int port to string.
func formatPort(port int) string {
	// Use simple integer formatting to avoid fmt dependency
	if port <= 0 {
		return "6379" // Default Redis port
	}
	return intToString(port)
}

// intToString converts int to string without fmt package.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ============================================================================
// Redis Client Error Sentinels
// ============================================================================

// ErrKeyNotFound indicates the requested key does not exist in Redis.
var ErrKeyNotFound = errors.New("redis: key not found")

// ============================================================================
// Redis Client Log Messages
// ============================================================================

const (
	// LogRedisConnected indicates successful Redis connection.
	LogRedisConnected = "redis client connected"
	// LogRedisConnectionFailed indicates Redis connection failure.
	LogRedisConnectionFailed = "redis connection failed"
	// LogRedisSetFailed indicates Redis SET operation failure.
	LogRedisSetFailed = "redis SET failed"
	// LogRedisGetFailed indicates Redis GET operation failure.
	LogRedisGetFailed = "redis GET failed"
	// LogRedisGetDelFailed indicates Redis GETDEL operation failure.
	LogRedisGetDelFailed = "redis GETDEL failed"
	// LogRedisDelFailed indicates Redis DEL operation failure.
	LogRedisDelFailed = "redis DEL failed"
)
