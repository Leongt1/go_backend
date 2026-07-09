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
		// Vercel preview deployments (per-branch/per-commit URLs like
		// https://fin-ai-git-<branch>-leongt1s-projects.vercel.app).
		// Checked only when the origin is not in AllowOrigins.
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "https://fin-ai-") &&
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
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
