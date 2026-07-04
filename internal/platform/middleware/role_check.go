package middleware

import (
	"backend-go/internal/users/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRole(roles ...domain.RoleType) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		for _, r := range roles {
			if domain.RoleType(role.(string)) == r {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}
