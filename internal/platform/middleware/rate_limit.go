package middleware

import (
	"net/http"
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

// RateLimit returns a per-client token-bucket limiter keyed by client IP.
// Each call creates an independent limiter set, so a stricter instance can be
// mounted on sensitive groups (e.g. /auth) on top of a global one.
func RateLimit(rps rate.Limit, burst int) gin.HandlerFunc {
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
		key := c.ClientIP()

		mu.Lock()
		cl, ok := clients[key]
		if !ok {
			cl = &clientLimiter{lim: rate.NewLimiter(rps, burst)}
			clients[key] = cl
		}
		cl.lastSeen = time.Now()
		mu.Unlock()

		if !cl.lim.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, slow down",
			})
			return
		}

		c.Next()
	}
}
