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
		var domainErr *platformErrors.DomainError
		if errors.As(err, &domainErr) {
			switch domainErr.Code {
			case platformErrors.CodeInvalidInput:
				c.JSON(http.StatusBadRequest, errorResponse(domainErr.Message))
			case platformErrors.CodeInvalidCredentials:
				c.JSON(http.StatusUnauthorized, errorResponse(domainErr.Message))
			case platformErrors.CodeUserNotFound:
				c.JSON(http.StatusNotFound, errorResponse(domainErr.Message))
			case platformErrors.CodeEmailAlreadyExists:
				c.JSON(http.StatusConflict, errorResponse(domainErr.Message))
			case platformErrors.CodeInvalidRole, platformErrors.CodeInvalidGender:
				c.JSON(http.StatusBadRequest, errorResponse(domainErr.Message))
			case platformErrors.CodeForbidden:
				c.JSON(http.StatusForbidden, errorResponse(domainErr.Message))
			default:
				c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
			}
			return
		}

		// platform errors
		var appErr *platformErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code {
			case platformErrors.CodeDatabaseError, platformErrors.CodeInternalServer:
				c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
			default:
				c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
			}
			return
		}

		// Fallback for unexpected errors
		c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
	}
}

func errorResponse(msg string) gin.H {
	return gin.H{"error": msg}
}
