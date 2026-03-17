package domain

import (
	"context"

	"github.com/google/uuid"
)

// CategoryRepository is the interface for category repository (system defaults)
// read-only, only written at seed time
type CategoryRepository interface {
	List(ctx context.Context) ([]Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*Category, error)
}

type CategorySeeder interface {
	List(ctx context.Context) ([]Category, error)
	CreateCategory(ctx context.Context, c *Category) error
}

// UserCategoryRepository handles per-user overrides and custom categories.
type UserCategoryRepository interface {
	Create(ctx context.Context, uc *UserCategory) error
	GetUserCategoryByID(ctx context.Context, id uuid.UUID) (*UserCategory, error)
	GetByUserAndCategory(ctx context.Context, userID, categoryID uuid.UUID) (*UserCategory, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]UserCategory, error)
	Update(ctx context.Context, uc *UserCategory) error
	ExistsByName(ctx context.Context, userID uuid.UUID, name string) (bool, error)
}
