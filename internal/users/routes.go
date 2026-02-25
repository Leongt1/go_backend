package users

import (
	"backend-go/internal/users/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, userHandler *handler.UserHandler) {
	userGroup := r.Group("/users")
	{
		userGroup.GET("/", userHandler.ListUsers)
		userGroup.GET("/:id", userHandler.GetByID)
	}
}
