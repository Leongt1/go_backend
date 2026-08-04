package ai

import (
	"backend-go/internal/ai/handler"
	"backend-go/internal/platform/middleware"
	"backend-go/internal/platform/security"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func RegisterRoutes(r *gin.RouterGroup, handler *handler.AIHandler, jwtManager *security.JWTManager) {
	aiRoutes := r.Group("/ai")
	aiRoutes.Use(middleware.AuthMiddleware(jwtManager))
	{
		// The AI endpoints call OpenAI and cost money. Credits cap total spend,
		// but not burst velocity (and admins bypass credits), so each is capped
		// per user on top of the global limit. Keyed per account, since cost maps
		// to an account. Mounted after AuthMiddleware so the user id is on context.

		// chat: burst 5 then ~1 per 5s (~12/min) - comfortable for interactive use
		aiRoutes.POST("/chat",
			middleware.RateLimitBy(middleware.UserOrIPKey, rate.Every(5*time.Second), 5),
			handler.Chat)

		// bulk import is the costliest call (parses up to 4000 chars); users
		// paste-review-save, so a tighter cap: burst 3 then ~1 per 15s (~4/min)
		aiRoutes.POST("/transactions/extract",
			middleware.RateLimitBy(middleware.UserOrIPKey, rate.Every(15*time.Second), 3),
			handler.ExtractTransactions)

		// credits is a cheap GET - global limit is enough
		aiRoutes.GET("/credits", handler.Credits)
	}
}
