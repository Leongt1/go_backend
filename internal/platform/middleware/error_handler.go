package middleware

import (
	platformErrors "backend-go/internal/platform/errors"
	"errors"
	"log"
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

		// Log the real error for debugging
		log.Printf("[ERROR] %s %s → %v", c.Request.Method, c.Request.URL.Path, err)

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
			if appErr.Err != nil {
				log.Printf("[APP ERROR] %s %s → [%s] %s: %v", c.Request.Method, c.Request.URL.Path, appErr.Code, appErr.Message, appErr.Err)
			} else {
				log.Printf("[APP ERROR] %s %s → [%s] %s", c.Request.Method, c.Request.URL.Path, appErr.Code, appErr.Message)
			}
			switch appErr.Code {
			case platformErrors.CodeDatabaseError, platformErrors.CodeInternalServer:
				c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
			default:
				c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
			}
			return
		}

		// Fallback for unexpected errors
		log.Printf("[UNEXPECTED ERROR] %s %s → %v", c.Request.Method, c.Request.URL.Path, err)
		c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
	}
}

func errorResponse(msg string) gin.H {
	return gin.H{"error": msg}
}
