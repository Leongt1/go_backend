package budgets

import (
	"backend-go/internal/budgets/handler"
	"backend-go/internal/platform/middleware"
	"backend-go/internal/platform/security"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, h *handler.BudgetHandler, jwtManager *security.JWTManager) {
	budgets := r.Group("/budgets")
	budgets.Use(middleware.AuthMiddleware(jwtManager))
	{
		// List all budgets for the authenticated user
		budgets.GET("/", h.List)

		// Get specific budget with full details
		budgets.GET("/:id", h.GetByID)

		// Get budget status (spent, remaining, progress)
		budgets.GET("/:id/status", h.GetStatus)

		// Create a new budget
		budgets.POST("/", h.Create)

		// Update budget details
		budgets.PUT("/:id", h.Update)

		// Delete a budget
		budgets.DELETE("/:id", h.Delete)

		// Manage budget categories
		budgets.POST("/:id/categories", h.AddCategory)
		budgets.DELETE("/:id/categories/:categoryId", h.RemoveCategory)
	}
}
