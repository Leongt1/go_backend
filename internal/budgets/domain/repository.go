package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BudgetRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Budget, error)
	GetByID(ctx context.Context, budgetID uuid.UUID) (*Budget, error)
	Create(ctx context.Context, budget *Budget) error
	Update(ctx context.Context, budget *Budget) error
	Delete(ctx context.Context, budgetID uuid.UUID) error

	// Category management
	AddCategoryToBudget(ctx context.Context, budgetID, categoryID uuid.UUID) error
	RemoveCategoryFromBudget(ctx context.Context, budgetID, categoryID uuid.UUID) error
	GetCategoriesForBudget(ctx context.Context, budgetID uuid.UUID) ([]uuid.UUID, error)

	// Spent amount calculation
	GetSpentAmount(ctx context.Context, budget *Budget, start, end time.Time) (int64, error)
}
