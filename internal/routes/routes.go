package routes

import (
	"backend-go/internal/platform/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	gin.ForceConsoleColor()
	r := gin.Default()

	r.Use(middleware.CORS())
	// global per-IP limit: generous, just blocks abuse/floods
	r.Use(middleware.RateLimit(20, 40))
	// duplicate unsafe requests carrying the same Idempotency-Key are served
	// the first response instead of executing twice
	r.Use(middleware.Idempotency(10 * time.Minute))
	r.Use(middleware.ErrorHandler())

	return r
}
