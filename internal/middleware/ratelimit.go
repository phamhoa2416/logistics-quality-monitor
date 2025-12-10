package middleware

import (
	"cargo-tracker/internal/logger"
	"cargo-tracker/pkg/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter holds rate limiters for different clients.
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    burst,
	}

	go rl.cleanup()

	return rl
}

// getLimiter returns the rate limiter for the given IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists = rl.limiters[ip]
	if exists {
		return limiter
	}

	limiter = rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters[ip] = limiter
	return limiter
}

// cleanup removes old limiters periodically to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, limiter := range rl.limiters {
			if limiter.AllowN(time.Now(), rl.burst) {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(rps float64, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rps, burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.getLimiter(ip)

		if !l.Allow() {
			requestID := GetRequestID(c)
			logger.Warn("Rate limit exceeded",
				zap.String("request_id", requestID),
				zap.String("ip", ip),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)

			utils.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}
