package ai

import (
	"backend-go/internal/ai/handler"
	"backend-go/internal/platform/middleware"
	"backend-go/internal/platform/security"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *handler.AIHandler, jwtManager *security.JWTManager) {
	aiRoutes := r.Group("/ai")
	aiRoutes.Use(middleware.AuthMiddleware(jwtManager))
	{
		aiRoutes.POST("/chat", handler.Chat)
		aiRoutes.POST("/transactions/extract", handler.ExtractTransactions)
		aiRoutes.GET("/credits", handler.Credits)
	}
}
