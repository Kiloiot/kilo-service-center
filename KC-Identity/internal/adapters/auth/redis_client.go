// Package auth provides authentication and user management services.
package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/go-redis/redis/v8"
)

// RedisClient provides state storage for OIDC/OAuth2 flows.
type RedisClient struct {
	client *redis.Client
	logger logger.Logger
}

const defaultRedisPort = "6379"

// unknownCommandErrorText identifies pre-6.2 Redis servers lacking GETDEL.
const unknownCommandErrorText = "unknown command"

// NewRedisClient creates a new Redis client from canonical KC-Core config.
// Returns error if connection fails within timeout. ctx bounds the startup
// connectivity check together with the connect timeout.
// defaultRedisPort is the standard Redis port used when none is configured.
func NewRedisClient(ctx context.Context, cfg *config.RedisConfig, log logger.Logger) (*RedisClient, error) {
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
	pingCtx, cancel := context.WithTimeout(ctx, config.AuthRedisConnectTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, errors.New(logRedisConnectionFailed + ": " + err.Error())
	}

	log.Info(logRedisConnected, logger.FieldHost, cfg.Host, logger.FieldPort, cfg.Port, logger.FieldDb, cfg.Database)

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
		r.logger.ErrorContext(ctx, logRedisSetFailed, logger.FieldKey, key, logger.FieldError, err)
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
		r.logger.ErrorContext(ctx, logRedisGetFailed, logger.FieldKey, key, logger.FieldError, err)
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
	if strings.Contains(err.Error(), unknownCommandErrorText) {
		result, getErr := r.client.Get(ctx, key).Bytes()
		if getErr != nil {
			if errors.Is(getErr, redis.Nil) {
				return nil, ErrKeyNotFound
			}
			r.logger.ErrorContext(ctx, logRedisGetDelFailed, logger.FieldKey, key, logger.FieldError, getErr)
			return nil, getErr
		}
		// A token left behind could be replayed, so a failed delete fails the read.
		if err := r.Del(ctx, key); err != nil {
			return nil, err
		}
		return result, nil
	}

	// Handle redis.Nil and other errors
	if errors.Is(err, redis.Nil) {
		return nil, ErrKeyNotFound
	}
	r.logger.ErrorContext(ctx, logRedisGetDelFailed, logger.FieldKey, key, logger.FieldError, err)
	return nil, err
}

// Del removes a key.
func (r *RedisClient) Del(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.ErrorContext(ctx, logRedisDelFailed, logger.FieldKey, key, logger.FieldError, err)
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

// formatPort converts int port to string, defaulting to the standard Redis
// port for non-positive values.
func formatPort(port int) string {
	if port <= 0 {
		return defaultRedisPort
	}
	return strconv.Itoa(port)
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
	// logRedisConnected indicates successful Redis connection.
	logRedisConnected = "redis client connected"
	// logRedisConnectionFailed indicates Redis connection failure.
	logRedisConnectionFailed = "redis connection failed"
	// logRedisSetFailed indicates Redis SET operation failure.
	logRedisSetFailed = "redis SET failed"
	// logRedisGetFailed indicates Redis GET operation failure.
	logRedisGetFailed = "redis GET failed"
	// logRedisGetDelFailed indicates Redis GETDEL operation failure.
	logRedisGetDelFailed = "redis GETDEL failed"
	// logRedisDelFailed indicates Redis DEL operation failure.
	logRedisDelFailed = "redis DEL failed"
)
