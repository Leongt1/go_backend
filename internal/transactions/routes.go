package transactions

import (
	"backend-go/internal/platform/middleware"
	"backend-go/internal/platform/security"
	"backend-go/internal/transactions/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, h *handler.TransactionHandler, jwtManager *security.JWTManager) {
	transactions := r.Group("/transactions")
	transactions.Use(middleware.AuthMiddleware(jwtManager))
	{
		transactions.GET("/", h.List)
		transactions.POST("/", h.Create)
		transactions.GET("/:id", h.GetByID)
		transactions.PATCH("/:id", h.Update)
		transactions.DELETE("/:id", h.Delete)
	}
}
