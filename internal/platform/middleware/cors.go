package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",           // Vite dev server
			"https://fin-ai-wheat.vercel.app", // Production frontend
		},
		// Vercel preview deployments. Vercel mints two URL shapes per
		// deployment (branch alias fin-ai-git-<branch>-... and per-deploy
		// hash fin-<hash>-...); the stable part is the team suffix, so
		// match on that. Checked only when the origin is not in AllowOrigins.
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "https://") &&
				strings.HasSuffix(origin, "-leongt1s-projects.vercel.app")
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"Idempotency-Key",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
