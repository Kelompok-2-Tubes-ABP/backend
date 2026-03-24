package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	limiters   map[string]*tokenBucket
	limit      int           // requests per window
	window     time.Duration // time window
	bucketSize int           // max tokens in bucket
}

type tokenBucket struct {
	tokens    int
	lastToken time.Time
}

// NewRateLimiter creates a new rate limiter
// limit: maximum requests per window
// window: time window duration
// bucketSize: maximum tokens (burst capacity)
func NewRateLimiter(limit int, window time.Duration, bucketSize int) *RateLimiter {
	return &RateLimiter{
		limiters:   make(map[string]*tokenBucket),
		limit:      limit,
		window:     window,
		bucketSize: bucketSize,
	}
}

// RateLimit returns a Gin middleware for rate limiting
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clean up old limiters periodically
		rl.cleanup()

		// Get client IP
		clientIP := c.ClientIP()

		rl.mu.Lock()
		bucket, exists := rl.limiters[clientIP]
		if !exists {
			bucket = &tokenBucket{
				tokens:    rl.bucketSize,
				lastToken: time.Now(),
			}
			rl.limiters[clientIP] = bucket
		}
		rl.mu.Unlock()

		// Refill tokens based on time elapsed
		now := time.Now()
		elapsed := now.Sub(bucket.lastToken)
		bucket.lastToken = now

		// Calculate tokens to add
		tokensToAdd := int(elapsed.Nanoseconds() / rl.window.Nanoseconds() * int64(rl.limit))
		if tokensToAdd > 0 {
			bucket.tokens += tokensToAdd
			if bucket.tokens > rl.bucketSize {
				bucket.tokens = rl.bucketSize
			}
		}

		// Check if request is allowed
		if bucket.tokens > 0 {
			bucket.tokens--
			c.Next()
		} else {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests",
				"retry_after": int(rl.window.Seconds()),
			})
			c.Abort()
		}
	}
}

// cleanup removes old limiters that haven't been used
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, bucket := range rl.limiters {
		if now.Sub(bucket.lastToken) > rl.window*2 {
			delete(rl.limiters, ip)
		}
	}
}

// Global rate limiter instance
var (
	globalRL     *RateLimiter
	globalRLOnce sync.Once
)

// GetGlobalRateLimiter returns the global rate limiter
func GetGlobalRateLimiter() *RateLimiter {
	globalRLOnce.Do(func() {
		// 5 requests per minute, burst up to 10
		globalRL = NewRateLimiter(5, time.Minute, 10)
	})
	return globalRL
}

// RateLimitMiddleware returns a Gin middleware for global rate limiting
func RateLimitMiddleware() gin.HandlerFunc {
	return GetGlobalRateLimiter().RateLimit()
}

// StrictRateLimit returns a stricter rate limiter (3 requests per minute)
func StrictRateLimit() gin.HandlerFunc {
	strictRL := NewRateLimiter(3, time.Minute, 5)
	return strictRL.RateLimit()
}
