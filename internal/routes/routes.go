package routes

import (
	"backend-go/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	gin.ForceConsoleColor()
	r := gin.Default()

	r.Use(middleware.CORS())
	r.Use(middleware.ErrorHandler())

	return r
}
