package users

import (
	"backend-go/internal/platform/middleware"
	"backend-go/internal/platform/security"
	"backend-go/internal/users/domain"
	"backend-go/internal/users/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, h *handler.UserHandler, jwtManager *security.JWTManager) {
	userGroup := r.Group("/users")
	userGroup.Use(middleware.AuthMiddleware(jwtManager))
	{
		userGroup.GET("/", middleware.RequireRole(domain.RoleAdmin), h.ListUsers)
		userGroup.GET("/:id", middleware.RequireRole(domain.RoleAdmin, domain.RoleUser), h.GetByID)
		userGroup.PATCH("/:id", middleware.RequireRole(domain.RoleAdmin, domain.RoleUser), h.Update)
		userGroup.DELETE("/:id", middleware.RequireRole(domain.RoleAdmin), h.Delete)
	}
}
