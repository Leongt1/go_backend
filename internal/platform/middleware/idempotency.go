package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// idemEntry stores the outcome of the first request for a key. done is closed
// once the response is captured, so concurrent duplicates can wait for it.
type idemEntry struct {
	status      int
	contentType string
	body        []byte
	done        chan struct{}
	createdAt   time.Time
}

// bodyCapture tees the response body so it can be replayed for duplicates.
type bodyCapture struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *bodyCapture) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyCapture) WriteString(s string) (int, error) {
	w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Idempotency replays the stored response for unsafe requests (POST/PUT/
// PATCH/DELETE) that repeat the same Idempotency-Key within the TTL - double
// clicks and network retries get the first request's result instead of
// running twice. Requests without the header pass through untouched.
// Responses are cached in memory (single-instance deploy).
func Idempotency(ttl time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	entries := make(map[string]*idemEntry)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for key, e := range entries {
				select {
				case <-e.done:
					if time.Since(e.createdAt) > ttl {
						delete(entries, key)
					}
				default: // still in flight, keep it
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		// scope the key so different clients/routes cannot collide
		fullKey := c.Request.Method + "|" + c.Request.URL.Path + "|" + c.ClientIP() + "|" + key

		mu.Lock()
		if e, ok := entries[fullKey]; ok {
			mu.Unlock()
			<-e.done // if the original is still running, serve its eventual result
			c.Data(e.status, e.contentType, e.body)
			c.Abort()
			return
		}
		e := &idemEntry{done: make(chan struct{}), createdAt: time.Now()}
		entries[fullKey] = e
		mu.Unlock()

		w := &bodyCapture{ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		e.status = w.Status()
		e.contentType = w.Header().Get("Content-Type")
		e.body = w.buf.Bytes()

		// only pin successful outcomes. Errors must stay retryable: e.g. a 401
		// from an expired access token is retried by the frontend with the same
		// key after a silent refresh, and must reach the handler again.
		if e.status < http.StatusOK || e.status >= http.StatusMultipleChoices {
			mu.Lock()
			delete(entries, fullKey)
			mu.Unlock()
		}
		close(e.done)
	}
}
