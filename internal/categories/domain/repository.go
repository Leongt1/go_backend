package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CategoryRepository is the interface for category repository (system defaults)
// read-only, only written at seed time
type CategoryRepository interface {
	Create(ctx context.Context, uc *Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*Category, error)
	GetByName(ctx context.Context, userId uuid.UUID, name string) (*Category, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Category, error)
	Update(ctx context.Context, c *Category) error
	DeleteTx(ctx context.Context, tx pgx.Tx, userId, categoryID uuid.UUID) error
	ExistsByName(ctx context.Context, userID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)
}
