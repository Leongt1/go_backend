package users

import (
	"backend-go/internal/platform/config"
	"backend-go/internal/users/domain"
	"context"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func SeedUser(ctx context.Context, repo domain.UserRepository, adminCfg config.AdminConfig) error {

	adminName := adminCfg.Name
	adminEmail := adminCfg.Email
	adminPassword := adminCfg.Password

	if adminEmail == "" || adminPassword == "" {
		log.Println("Admin credentials not found, skipping seed")
		return nil
	}

	// Check if user already exists
	existingUser, _ := repo.GetByEmail(ctx, adminEmail)
	if existingUser != nil {
		log.Println("Admin user already exists, skipping seed")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin, err := domain.NewUser(adminName, adminEmail, domain.RoleAdmin, domain.GenderMale)
	if err != nil {
		return err
	}

	admin.PasswordHash = string(hash)
	admin.CreatedAt = time.Now().UTC()
	admin.UpdatedAt = time.Now().UTC()

	if err := repo.Create(ctx, admin); err != nil {
		return err
	}

	log.Println("Admin user seeded successfully")
	return nil
}
