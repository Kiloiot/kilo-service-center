package resilience

import (
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RegistrationRateLimiter provides per-IP rate limiting for sensitive gRPC methods.
type RegistrationRateLimiter struct {
	limiters  sync.Map // map[string]*rate.Limiter
	rate      rate.Limit
	burst     int
	methods   map[string]bool
	stopCh    chan struct{}
	closeOnce sync.Once
}

// NewRegistrationRateLimiter creates a rate limiter from gateway config.
func NewRegistrationRateLimiter(cfg config.GatewayRateLimitConfig) *RegistrationRateLimiter {
	rl := &RegistrationRateLimiter{
		rate:  rate.Limit(float64(cfg.RequestsPerMin) / 60.0),
		burst: cfg.Burst,
		methods: map[string]bool{
			"/kilocenter.api.v1.IdentityService/RegisterAccount":   true,
			"/kilocenter.api.v1.KiloCenterService/RegisterAccount": true,
		},
		stopCh: make(chan struct{}),
	}

	// Background goroutine cleans up stale per-IP limiters
	go rl.cleanup(cfg.CleanupInterval)

	return rl
}

// StreamInterceptor returns a gRPC stream interceptor that rate-limits configured methods.
func (rl *RegistrationRateLimiter) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !rl.methods[info.FullMethod] {
			return handler(srv, ss)
		}

		ip := extractIP(ss)
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			return status.Error(codes.ResourceExhausted,
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRateLimitExceeded))
		}

		return handler(srv, ss)
	}
}

// Close stops the background cleanup goroutine. Safe to call multiple times.
func (rl *RegistrationRateLimiter) Close() {
	rl.closeOnce.Do(func() {
		close(rl.stopCh)
	})
}

func (rl *RegistrationRateLimiter) getLimiter(ip string) *rate.Limiter {
	if v, ok := rl.limiters.Load(ip); ok {
		return v.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(rl.rate, rl.burst)
	actual, _ := rl.limiters.LoadOrStore(ip, limiter)
	return actual.(*rate.Limiter)
}

func (rl *RegistrationRateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.limiters.Range(func(key, value interface{}) bool {
				limiter := value.(*rate.Limiter)
				// Remove limiters that have recovered to full tokens (idle clients)
				if limiter.Tokens() >= float64(rl.burst) {
					rl.limiters.Delete(key)
				}
				return true
			})
		}
	}
}

func extractIP(ss grpc.ServerStream) string {
	p, ok := peer.FromContext(ss.Context())
	if !ok || p.Addr == nil {
		return "unknown"
	}
	// peer.Addr.String() returns "ip:port"; extract just the ip
	addr := p.Addr.String()
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
