package main

import (
	"backend-go/internal/auth"
	authHandler "backend-go/internal/auth/handler"
	authRepo "backend-go/internal/auth/repository"
	authService "backend-go/internal/auth/service"
	"backend-go/internal/budgets"
	budgetHandler "backend-go/internal/budgets/handler"
	budgetRepo "backend-go/internal/budgets/repository"
	budgetService "backend-go/internal/budgets/service"
	"backend-go/internal/categories"
	categoryHandler "backend-go/internal/categories/handler"
	categoryRepo "backend-go/internal/categories/repository"
	categoryService "backend-go/internal/categories/service"
	"backend-go/internal/platform/config"
	"backend-go/internal/platform/db"
	"backend-go/internal/platform/email"
	"backend-go/internal/platform/security"
	"backend-go/internal/routes"
	"backend-go/internal/transactions"
	transactionHandler "backend-go/internal/transactions/handler"
	transactionRepo "backend-go/internal/transactions/repository"
	transactionService "backend-go/internal/transactions/service"
	"backend-go/internal/users"
	userHandler "backend-go/internal/users/handler"
	userRepo "backend-go/internal/users/repository"
	userService "backend-go/internal/users/service"
	"context"
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
	log.Println("Connected to db successfully")

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
	resetPasswordTTL, err := time.ParseDuration(cfg.ResetPassword.ResetPasswordTTL)
	if err != nil {
		log.Fatal("Invalid RESET_PASSWORD_TOKEN:", err)
	}

	jwtManager := security.NewJWTManager(cfg.JWT.Secret, "finai-api")

	// For testing — swap in NewResendProvider when ready
	emailProvider := email.NewSMTPProvider(
		cfg.Email.SMTP.Host,
		cfg.Email.SMTP.Port,
		cfg.Email.SMTP.Username,
		cfg.Email.SMTP.Password,
		cfg.Email.SMTP.From,
	)

	// or for Resend:
	// emailProvider := email.NewResendProvider(
	// 	cfg.Email.Resend.APIKey,
	// 	cfg.Email.Resend.From,
	// )

	categoryRepo := categoryRepo.NewRepository(pool)

	txRepo := transactionRepo.NewRepository(pool)
	txService := transactionService.NewService(txRepo, categoryRepo)
	txHandler := transactionHandler.NewTransactionHandler(txService)

	categoryService := categoryService.NewService(pool, categoryRepo, txRepo)
	categoryHandler := categoryHandler.NewCategoryHandler(categoryService)

	budgetRepo := budgetRepo.NewRepository(pool, txRepo)
	budgetService := budgetService.NewService(budgetRepo, categoryRepo, txRepo)
	budgetHandler := budgetHandler.NewBudgetHandler(budgetService)

	authRepo := authRepo.NewRepository(pool)
	authService := authService.NewService(
		userService, jwtManager,
		authRepo, authRepo, categoryRepo,
		emailProvider,
		accessTTL, refreshTTL, resetPasswordTTL,
	)
	authHandler := authHandler.NewAuthHandler(authService, refreshTTL)

	api := router.Group("/api/v1")
	// Initialize routes
	auth.RegisterRoutes(api, authHandler)
	users.RegisterRoutes(api, userHandler, jwtManager)
	categories.RegisterRoutes(api, categoryHandler, jwtManager)
	transactions.RegisterRoutes(api, txHandler, jwtManager)
	budgets.RegisterRoutes(api, budgetHandler, jwtManager)

	// Seed admin user
	if err := users.SeedUser(ctx, userRepo, cfg.Admin); err != nil {
		log.Fatal(err)
	}

	// seed categories for admin user
	adminUser, err := userService.GetByEmail(ctx, cfg.Admin.Email)
	if err != nil {
		log.Fatal("Failed to get admin user:", err)
	}

	existing, err := categoryRepo.ListByUser(ctx, adminUser.ID)
	if err != nil {
		log.Fatal("Failed to check admin categories:", err)
	}
	if len(existing) == 0 {
		if err := categories.SeedCategoriesForUser(ctx, adminUser.ID, categoryRepo); err != nil {
			log.Fatal("Failed to seed admin categories:", err)
		}
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
