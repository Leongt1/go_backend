package main

import (
	"backend-go/internal/auth"
	authHandler "backend-go/internal/auth/handler"
	authRepo "backend-go/internal/auth/repository"
	authService "backend-go/internal/auth/service"
	"backend-go/internal/platform/config"
	"backend-go/internal/platform/db"
	"backend-go/internal/platform/security"
	"backend-go/internal/routes"
	"backend-go/internal/users"
	userHandler "backend-go/internal/users/handler"
	userRepo "backend-go/internal/users/repository"
	userService "backend-go/internal/users/service"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	// Loading .env from current directory or parent directories
	godotenv.Load(".env")
	godotenv.Load("../../.env")

	// Load config
	cfg := config.Load()

	// Initialize db
	pool, err := db.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	fmt.Print("Connected to db successfully")

	// Initialize router
	router := routes.NewRouter()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Initialize dependencies
	userRepo := userRepo.NewRepository(pool)
	userService := userService.NewService(userRepo)
	userHandler := userHandler.NewUserHandler(userService)

	// Parse TTLs
	accessTTL, err := time.ParseDuration(cfg.JWT.AccessTTL)
	if err != nil {
		log.Fatal("Invalid ACCESS_TTL:", err)
	}
	refreshTTL, err := time.ParseDuration(cfg.JWT.RefreshTTL)
	if err != nil {
		log.Fatal("Invalid REFRESH_TTL:", err)
	}

	jwtManager := security.NewJWTManager(cfg.JWT.Secret, "finai-api")
	authRepo := authRepo.NewRepository(pool)
	authService := authService.NewService(userService, jwtManager, authRepo, accessTTL, refreshTTL)
	authHandler := authHandler.NewAuthHandler(authService, refreshTTL)

	api := router.Group("/api/v1")
	// Initialize routes
	users.RegisterRoutes(api, userHandler)
	auth.RegisterRoutes(api, authHandler)

	// Seed admin user
	if err := users.SeedUser(ctx, userRepo, cfg.Admin); err != nil {
		log.Fatal(err)
	}

	// Initialize server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-quit
	log.Println("Shutting down server ...")

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
