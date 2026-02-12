package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters per IP address
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

// NewRateLimiter creates a new rate limiter with the specified rate and burst.
// rate: requests per second (e.g., rate.Every(time.Minute/5) for 5 per minute)
// burst: maximum burst size
// ttl: time to keep limiter entries for inactive IPs (currently unused, kept for future cleanup implementation)
func NewRateLimiter(rateLimit rate.Limit, burst int, ttl time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rateLimit,
		burst:    burst,
		ttl:      ttl,
	}
	// Note: Cleanup can be added later with last-access-time tracking
	// For auth endpoints, number of unique IPs is typically small, so keeping all limiters is acceptable
	return rl
}

// getLimiter returns the rate limiter for the given IP, creating one if needed
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = limiter
	}
	return limiter
}


// RateLimitMiddleware creates a middleware that rate limits requests by IP address
func RateLimitMiddleware(requestsPerMinute int, burst int) gin.HandlerFunc {
	// Convert requests per minute to rate.Limit
	rateLimit := rate.Every(time.Minute / time.Duration(requestsPerMinute))
	limiter := NewRateLimiter(rateLimit, burst, 15*time.Minute)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.RemoteIP()
		}

		rl := limiter.getLimiter(ip)
		if !rl.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthRateLimitMiddleware is for authentication endpoints (login, register, logout, me, stream).
// Allows normal usage (login, a few /me or /stream, logout) without hitting the limit.
func AuthRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(30, 15)
}
