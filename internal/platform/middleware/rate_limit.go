package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// clientLimiter pairs a token bucket with its last activity so idle clients
// can be evicted.
type clientLimiter struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// ipKey buckets requests by client IP - the default for public/global limiting.
func ipKey(c *gin.Context) string { return c.ClientIP() }

// UserOrIPKey buckets requests by the authenticated user id (set on the context
// by AuthMiddleware), falling back to client IP when absent. Use this to key a
// limiter per account rather than per IP - mount it only AFTER AuthMiddleware.
func UserOrIPKey(c *gin.Context) string {
	if v, ok := c.Get(ContextUserID); ok {
		if s, ok := v.(string); ok && s != "" {
			return "user:" + s
		}
	}
	return "ip:" + c.ClientIP()
}

// RateLimit returns a per-client token-bucket limiter keyed by client IP.
// Each call creates an independent limiter set, so a stricter instance can be
// mounted on sensitive groups (e.g. /auth) on top of a global one.
func RateLimit(rps rate.Limit, burst int) gin.HandlerFunc {
	return RateLimitBy(ipKey, rps, burst)
}

// RateLimitBy is RateLimit with a configurable bucket key, so a limiter can be
// scoped per user (UserOrIPKey) instead of per IP - useful for authenticated,
// cost-sensitive endpoints (e.g. AI) where abuse maps to an account. Each call
// creates an independent limiter set.
func RateLimitBy(keyFn func(*gin.Context) string, rps rate.Limit, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*clientLimiter)

	// evict clients idle for 10+ minutes so the map cannot grow unbounded
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for key, cl := range clients {
				if time.Since(cl.lastSeen) > 10*time.Minute {
					delete(clients, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		key := keyFn(c)

		mu.Lock()
		cl, ok := clients[key]
		if !ok {
			cl = &clientLimiter{lim: rate.NewLimiter(rps, burst)}
			clients[key] = cl
		}
		cl.lastSeen = time.Now()
		mu.Unlock()

		// Reserve a token instead of Allow() so we can report an accurate
		// Retry-After. Delay() is 0 when a token is free now (the reservation
		// consumes it, like Allow); when it's not, we reject and Cancel() so the
		// rejected request doesn't hold the next token.
		res := cl.lim.Reserve()
		if delay := res.Delay(); delay > 0 || !res.OK() {
			res.Cancel()
			retryAfter := 1
			if res.OK() { // OK() is false only if burst is 0 (Delay is +Inf)
				retryAfter = max(1, int(math.Ceil(delay.Seconds())))
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, slow down",
			})
			return
		}

		c.Next()
	}
}
