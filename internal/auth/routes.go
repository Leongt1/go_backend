package auth

import (
	"backend-go/internal/auth/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *handler.AuthHandler) {
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/login", handler.Login)
		authRoutes.POST("/refresh", handler.RefreshToken)
		authRoutes.POST("/logout", handler.Logout)
		authRoutes.POST("/signup", handler.Signup)
	}
}
