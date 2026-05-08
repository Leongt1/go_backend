package categories

import (
	"backend-go/internal/categories/handler"
	"backend-go/internal/platform/middleware"
	"backend-go/internal/platform/security"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, h *handler.CategoryHandler, jwtManager *security.JWTManager) {
	categories := r.Group("/categories")
	categories.Use(middleware.AuthMiddleware(jwtManager))
	{
		// list all categories for the authenticated user
		categories.GET("/", h.List)

		// create a new custom category
		categories.POST("/", h.Create)

		// user_categories row operations (custom + already-overridden defaults)
		categories.PATCH("/:id/rename", h.Rename)
		categories.PATCH("/:id/hide", h.Hide)
		categories.PATCH("/:id/unhide", h.Unhide)
		categories.DELETE("/:id", h.Delete)
	}
}
