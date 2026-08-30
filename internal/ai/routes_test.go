package ai

import (
	aiHandler "backend-go/internal/ai/handler"
	aiService "backend-go/internal/ai/service"
	"backend-go/internal/platform/security"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// End-to-end check of the real route wiring: the per-user limiter on /ai/chat
// (burst 5) must let the first 5 requests through to the handler and 429 the
// rest. Uses an unconfigured OpenAI client so Chat returns early (no DB, no
// OpenAI spend) - we only care that the limiter trips after the burst.
func TestChatRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwt := security.NewJWTManager("test-secret", "finai-test")
	token, err := jwt.GenerateToken(uuid.New(), "User", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	// nil deps are safe: an unconfigured client makes Chat return before using them
	client := aiService.NewOpenAIClient("", "", "")
	svc := aiService.NewService(nil, nil, nil, nil, nil, client)
	h := aiHandler.NewAIHandler(svc)

	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), h, jwt)

	call := func() int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat",
			strings.NewReader(`{"message":"hi"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	const burst = 5
	// first `burst` requests reach the handler (same non-429 status), then 429
	var firstStatus int
	for i := 0; i < burst; i++ {
		code := call()
		if code == http.StatusTooManyRequests {
			t.Fatalf("request %d was rate-limited before the burst was used", i+1)
		}
		if i == 0 {
			firstStatus = code
		} else if code != firstStatus {
			t.Fatalf("request %d status %d differs from first %d", i+1, code, firstStatus)
		}
	}
	if code := call(); code != http.StatusTooManyRequests {
		t.Fatalf("request %d: got %d, want 429 (limiter should trip after burst)", burst+1, code)
	}
}
