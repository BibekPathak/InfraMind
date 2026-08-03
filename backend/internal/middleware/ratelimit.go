package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter limits requests per client (IP or token). Uses Redis when
// available for shared state across instances; otherwise in-memory.
type RateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	visitors map[string]*counter
	redis    *redis.Client
}

type counter struct {
	count       int
	windowStart time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		window:   window,
		limit:    limit,
		visitors: make(map[string]*counter),
	}
}

// SetRedis enables distributed rate limiting.
func (rl *RateLimiter) SetRedis(client *redis.Client) {
	rl.redis = client
}

func (rl *RateLimiter) allow(key string) bool {
	if rl.redis != nil {
		return rl.allowRedis(key)
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	c, ok := rl.visitors[key]
	if !ok || now.Sub(c.windowStart) > rl.window {
		rl.visitors[key] = &counter{count: 1, windowStart: now}
		return true
	}
	if c.count >= rl.limit {
		return false
	}
	c.count++
	return true
}

func (rl *RateLimiter) allowRedis(key string) bool {
	ctx := context.Background()
	window := int64(rl.window.Seconds())
	if window < 1 {
		window = 1
	}

	// INCR + EXPIRE: count requests in a fixed window.
	count, err := rl.redis.Incr(ctx, "ratelimit:"+key).Result()
	if err != nil {
		return true // fail open on Redis errors
	}
	if count == 1 {
		rl.redis.Expire(ctx, "ratelimit:"+key, rl.window)
	}
	return count <= int64(rl.limit)
}

// Middleware wraps a handler with per-IP + per-token rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.allow("ip:" + ip) {
			http.Error(w, `{"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
			return
		}

		if token := bearerToken(r); token != "" {
			if !rl.allow("token:" + token) {
				http.Error(w, `{"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
		return auth[len(prefix):]
	}
	return ""
}
