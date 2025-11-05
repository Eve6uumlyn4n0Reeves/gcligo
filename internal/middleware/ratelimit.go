package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type limiterEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// ttlLimiterCache is a simple TTL map for per-key limiters with opportunistic sweeping.
type ttlLimiterCache struct {
	mu        sync.RWMutex
	items     map[string]*limiterEntry
	ttl       time.Duration
	lastSweep time.Time
}

func newTTLLimiterCache(ttl time.Duration) *ttlLimiterCache {
	return &ttlLimiterCache{items: make(map[string]*limiterEntry), ttl: ttl}
}

func (c *ttlLimiterCache) get(key string, makeFn func() *rate.Limiter) *rate.Limiter {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[key]; ok {
		e.lastSeen = now
		return e.lim
	}
	lim := makeFn()
	c.items[key] = &limiterEntry{lim: lim, lastSeen: now}
	// update gauge on insert
	SetRateLimitKeyGauge(len(c.items))
	// opportunistic sweep every ~2 minutes
	if c.lastSweep.IsZero() || now.Sub(c.lastSweep) > 2*time.Minute {
		c.sweepLocked(now)
		c.lastSweep = now
	}
	return lim
}

func (c *ttlLimiterCache) sweepLocked(now time.Time) {
	if c.ttl <= 0 {
		c.ttl = 15 * time.Minute
	}
	for k, e := range c.items {
		if now.Sub(e.lastSeen) > c.ttl {
			delete(c.items, k)
		}
	}
	// update metrics
	SetRateLimitKeyGauge(len(c.items))
	RecordRateLimitSweep()
}

// RateLimiter creates a rate limiting middleware
func RateLimiter(rps int, burst int) gin.HandlerFunc {
	limiters := &sync.Map{}

	return func(c *gin.Context) {
		key := c.ClientIP()

		limiterI, _ := limiters.LoadOrStore(key, rate.NewLimiter(rate.Limit(rps), burst))
		limiter := limiterI.(*rate.Limiter)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimiterAutoKey applies rate limit using API key if present (Authorization/x-api-key/x-goog-api-key),
// otherwise falls back to client IP. Additionally enforces a lightweight global limiter.
func RateLimiterAutoKey(rps int, burst int) gin.HandlerFunc {
	if rps <= 0 {
		rps = 10
	}
	if burst <= 0 {
		burst = 20
	}
	cache := newTTLLimiterCache(15 * time.Minute)
	global := rate.NewLimiter(rate.Limit(rps*5), burst*5) // simple global guard (5x per-key defaults)
	return func(c *gin.Context) {
		// Global limiter first
		if !global.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"message": "Global rate limit exceeded", "type": "rate_limit_error"}})
			c.Abort()
			return
		}
		key := extractAPIKey(c)
		if key == "" {
			key = c.ClientIP()
		}
		li := cache.get(key, func() *rate.Limiter { return rate.NewLimiter(rate.Limit(rps), burst) })
		if !li.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"message": "Rate limit exceeded", "type": "rate_limit_error"}})
			c.Abort()
			return
		}
		c.Next()
	}
}

func extractAPIKey(c *gin.Context) string {
	if v, ok := c.Get("api_key"); ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if v := strings.TrimSpace(c.GetHeader("x-api-key")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("x-goog-api-key")); v != "" {
		return v
	}
	if v, err := c.Cookie("mgmt_session"); err == nil && strings.TrimSpace(v) != "" {
		return v
	}
	return ""
}
