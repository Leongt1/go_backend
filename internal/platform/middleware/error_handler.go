package middleware

import (
	platformErrors "backend-go/internal/platform/errors"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
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
			logDomainError(c, domainErr)
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
			case platformErrors.CodeCategoryNotFound:
				c.JSON(http.StatusNotFound, errorResponse(domainErr.Message))
			case platformErrors.CodeCategoryHidden:
				c.JSON(http.StatusGone, errorResponse(domainErr.Message))
			case platformErrors.CodeDuplicateCategoryName:
				c.JSON(http.StatusConflict, errorResponse(domainErr.Message))
			case platformErrors.CodeTransactionNotFound:
				c.JSON(http.StatusNotFound, errorResponse(domainErr.Message))
			case platformErrors.CodeFailedToResetPassword:
				c.JSON(http.StatusInternalServerError, errorResponse(domainErr.Message))
			case platformErrors.CodeBudgetNotFound:
				c.JSON(http.StatusNotFound, errorResponse(domainErr.Message))
			case platformErrors.CodeDuplicateBudgetName:
				c.JSON(http.StatusConflict, errorResponse(domainErr.Message))
			case platformErrors.CodeAINoCredits:
				c.JSON(http.StatusPaymentRequired, errorResponse(domainErr.Message))
			case platformErrors.CodeAIUnavailable:
				c.JSON(http.StatusServiceUnavailable, errorResponse(domainErr.Message))
			default:
				c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
			}
			return
		}

		// platform errors (with stack trace)
		var appErr *platformErrors.AppError
		if errors.As(err, &appErr) {
			logAppError(c, appErr)
			switch appErr.Code {
			case platformErrors.CodeDatabaseError, platformErrors.CodeInternalServer:
				c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
			default:
				c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
			}
			return
		}

		// Fallback for unexpected errors (capture stack trace here)
		logUnexpectedError(c, err)
		c.JSON(http.StatusInternalServerError, errorResponse("something went wrong"))
	}
}

// logDomainError logs domain/validation errors with a short summary (no stack trace).
func logDomainError(c *gin.Context, err *platformErrors.DomainError) {
	log.Printf("\n%s══════════════════════════════════════════════%s\n"+
		"%s DOMAIN ERROR %s\n"+
		"%s══════════════════════════════════════════════%s\n"+
		"  %sRoute:%s    %s %s\n"+
		"  %sCode:%s     %s\n"+
		"  %sMessage:%s  %s\n"+
		"%s══════════════════════════════════════════════%s\n",
		colorYellow, colorReset,
		colorBold+colorYellow, colorReset,
		colorYellow, colorReset,
		colorCyan, colorReset, c.Request.Method, c.Request.URL.Path,
		colorCyan, colorReset, err.Code,
		colorCyan, colorReset, err.Message,
		colorYellow, colorReset,
	)
}

// logAppError logs application/infrastructure errors with the full stack trace.
func logAppError(c *gin.Context, err *platformErrors.AppError) {
	causeLine := ""
	if err.Err != nil {
		causeLine = fmt.Sprintf("  %sCause:%s    %v\n", colorCyan, colorReset, err.Err)
	}

	stackBlock := ""
	if err.StackTrace != "" {
		stackBlock = fmt.Sprintf("\n  %sStack Trace:%s\n%s\n",
			colorCyan, colorReset,
			indentStackTrace(err.StackTrace),
		)
	}

	log.Printf("\n%s══════════════════════════════════════════════%s\n"+
		"%s APP ERROR %s\n"+
		"%s══════════════════════════════════════════════%s\n"+
		"  %sRoute:%s    %s %s\n"+
		"  %sCode:%s     %s\n"+
		"  %sMessage:%s  %s\n"+
		"%s%s"+
		"%s══════════════════════════════════════════════%s\n",
		colorRed, colorReset,
		colorBold+colorRed, colorReset,
		colorRed, colorReset,
		colorCyan, colorReset, c.Request.Method, c.Request.URL.Path,
		colorCyan, colorReset, err.Code,
		colorCyan, colorReset, err.Message,
		causeLine,
		stackBlock,
		colorRed, colorReset,
	)
}

// logUnexpectedError logs unexpected errors with a stack trace captured at the middleware level.
func logUnexpectedError(c *gin.Context, err error) {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)

	log.Printf("\n%s══════════════════════════════════════════════%s\n"+
		"%s UNEXPECTED ERROR %s\n"+
		"%s══════════════════════════════════════════════%s\n"+
		"  %sRoute:%s    %s %s\n"+
		"  %sError:%s    %v\n"+
		"\n  %sStack Trace:%s\n%s\n"+
		"%s══════════════════════════════════════════════%s\n",
		colorRed, colorReset,
		colorBold+colorRed, colorReset,
		colorRed, colorReset,
		colorCyan, colorReset, c.Request.Method, c.Request.URL.Path,
		colorCyan, colorReset, err,
		colorCyan, colorReset, indentStackTrace(string(buf[:n])),
		colorRed, colorReset,
	)
}

// indentStackTrace adds gray coloring and indentation to each line of a stack trace.
func indentStackTrace(trace string) string {
	lines := strings.Split(strings.TrimSpace(trace), "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(fmt.Sprintf("  %s%s%s\n", colorGray, line, colorReset))
	}
	return b.String()
}

func errorResponse(msg string) gin.H {
	return gin.H{"error": msg}
}
