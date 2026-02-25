package middleware

import (
	platformErrors "backend-go/internal/platform/errors"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		// domain errors
		var dbErr *platformErrors.DomainError
		if errors.As(err, &dbErr) {
			switch dbErr.Code {
			case "INVALID_INPUT":
				c.JSON(http.StatusBadRequest, gin.H{
					"error": dbErr.Message,
				})
			case "INVALID_CREDENTIALS":
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": dbErr.Message,
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": dbErr.Message,
				})
			}
			return
		}

		// platform errors
		var pErr *platformErrors.AppError
		if errors.As(err, &pErr) {
			switch pErr.Code {
			case "INTERNAL_SERVER_ERROR":
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": pErr.Message,
				})
			case "DATABASE_ERROR":
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": pErr.Message,
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": pErr.Message,
				})
			}
			return
		}
	}
}
