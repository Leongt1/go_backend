package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// buildLimited returns an engine that sets the user id from an X-Test-User
// header (mimicking AuthMiddleware) and then applies a per-user limiter with the
// given burst. rate.Every(time.Hour) means no tokens refill during a test, so
// only the burst is available - making the boundary deterministic.
func buildLimited(burst int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if u := c.GetHeader("X-Test-User"); u != "" {
			c.Set(ContextUserID, u)
		}
		c.Next()
	})
	r.Use(RateLimitBy(UserOrIPKey, rate.Every(time.Hour), burst))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func do(r *gin.Engine, user string) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// A single key gets exactly `burst` requests, then 429.
func TestRateLimitBy_BlocksAfterBurst(t *testing.T) {
	r := buildLimited(2)
	want := []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests}
	for i, exp := range want {
		if got := do(r, "user-a"); got != exp {
			t.Errorf("request %d: got %d, want %d", i+1, got, exp)
		}
	}
}

// Different keys have independent buckets - one user exhausting its burst must
// not throttle another.
func TestRateLimitBy_IndependentPerKey(t *testing.T) {
	r := buildLimited(1)

	if got := do(r, "user-a"); got != http.StatusOK {
		t.Fatalf("user-a first request: got %d, want 200", got)
	}
	if got := do(r, "user-a"); got != http.StatusTooManyRequests {
		t.Fatalf("user-a second request: got %d, want 429", got)
	}
	// user-b is untouched by user-a hitting its limit
	if got := do(r, "user-b"); got != http.StatusOK {
		t.Errorf("user-b first request: got %d, want 200", got)
	}
}

// A blocked request's Retry-After header reflects the real wait until the next
// token, not a hardcoded value.
func TestRateLimitBy_RetryAfterReflectsWait(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// one token every 2s, burst 1: the second request must wait ~2s
	r.Use(RateLimitBy(func(*gin.Context) string { return "k" }, rate.Every(2*time.Second), 1))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	send := func() (int, string) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		return w.Code, w.Header().Get("Retry-After")
	}

	if code, _ := send(); code != http.StatusOK {
		t.Fatalf("first request: got %d, want 200", code)
	}
	code, retryAfter := send()
	if code != http.StatusTooManyRequests {
		t.Fatalf("second request: got %d, want 429", code)
	}
	n, err := strconv.Atoi(retryAfter)
	if err != nil {
		t.Fatalf("Retry-After %q is not an integer: %v", retryAfter, err)
	}
	if n < 1 || n > 2 {
		t.Errorf("Retry-After = %d, want ~2 (the refill interval)", n)
	}
}

// UserOrIPKey prefers the authenticated user id, falling back to client IP.
func TestUserOrIPKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	withUser, _ := gin.CreateTestContext(httptest.NewRecorder())
	withUser.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	withUser.Set(ContextUserID, "abc")
	if got := UserOrIPKey(withUser); got != "user:abc" {
		t.Errorf("with user id: got %q, want %q", got, "user:abc")
	}

	anon, _ := gin.CreateTestContext(httptest.NewRecorder())
	anon.Request = httptest.NewRequest(http.MethodGet, "/", nil) // RemoteAddr 192.0.2.1:1234
	if got := UserOrIPKey(anon); got != "ip:192.0.2.1" {
		t.Errorf("anonymous: got %q, want %q", got, "ip:192.0.2.1")
	}
}
