package users

import (
	"backend-go/internal/platform/middleware"
	"backend-go/internal/platform/security"
	"backend-go/internal/users/domain"
	"backend-go/internal/users/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, userHandler *handler.UserHandler, jwtManager *security.JWTManager) {
	userGroup := r.Group("/users")
	userGroup.Use(middleware.AuthMiddleware(jwtManager))
	{
		userGroup.GET("/", userHandler.ListUsers)
		userGroup.GET("/:id", userHandler.GetByID)
		userGroup.PATCH("/:id", middleware.RequireRole(domain.RoleAdmin), userHandler.Update)
		userGroup.DELETE("/:id", middleware.RequireRole(domain.RoleAdmin), userHandler.Delete)
	}
}
