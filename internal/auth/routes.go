package auth

import (
	"backend-go/internal/auth/handler"
	"backend-go/internal/platform/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func RegisterRoutes(r *gin.RouterGroup, handler *handler.AuthHandler) {
	authRoutes := r.Group("/auth")
	// stricter per-IP limit for credential endpoints: burst of 10, then one
	// request per 3s (~20/min) - slows brute force without hurting real users
	authRoutes.Use(middleware.RateLimit(rate.Every(3*time.Second), 10))
	{
		authRoutes.POST("/login", handler.Login)
		authRoutes.POST("/refresh", handler.RefreshToken)
		authRoutes.POST("/logout", handler.Logout)
		authRoutes.POST("/signup", handler.Signup)
		authRoutes.POST("/forgot-password", handler.ForgotPassword)
		authRoutes.POST("/reset-password", handler.ResetPassword)
	}
}
