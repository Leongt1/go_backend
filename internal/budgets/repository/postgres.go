package repository

import (
	"backend-go/internal/budgets/domain"
	platformErrors "backend-go/internal/platform/errors"
	transactionDomain "backend-go/internal/transactions/domain"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool            *pgxpool.Pool
	transactionRepo transactionDomain.TransactionRepository
}

func NewRepository(pool *pgxpool.Pool, transactionRepo transactionDomain.TransactionRepository) *Repository {
	return &Repository{
		pool:            pool,
		transactionRepo: transactionRepo,
	}
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Budget, error) {
	query := `
		SELECT id, user_id, name, type, kind, amount, period_unit, period_value, start_date, created_at, updated_at
		FROM budgets
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to list budgets by user", err)
	}
	defer rows.Close()

	var budgets []domain.Budget
	for rows.Next() {
		var budget domain.Budget
		if err := rows.Scan(
			&budget.ID,
			&budget.UserID,
			&budget.Name,
			&budget.Type,
			&budget.Kind,
			&budget.Amount,
			&budget.PeriodUnit,
			&budget.PeriodValue,
			&budget.StartDate,
			&budget.CreatedAt,
			&budget.UpdatedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrBudgetNotFound // No budgets found for the user
			}
			return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to list budgets by user", err)
		}
		budgets = append(budgets, budget)
	}
	if err := rows.Err(); err != nil {
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to list budgets by user", err)
	}

	return budgets, nil
}

func (r *Repository) GetByID(ctx context.Context, budgetID uuid.UUID) (*domain.Budget, error) {
	query := `
		SELECT id, user_id, name, type, kind, amount, period_unit, period_value, start_date, created_at, updated_at	
		FROM budgets
		WHERE id = $1
	`
	var budget domain.Budget
	if err := r.pool.QueryRow(ctx, query, budgetID).Scan(
		&budget.ID,
		&budget.UserID,
		&budget.Name,
		&budget.Type,
		&budget.Kind,
		&budget.Amount,
		&budget.PeriodUnit,
		&budget.PeriodValue,
		&budget.StartDate,
		&budget.CreatedAt,
		&budget.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrBudgetNotFound // Budget not found
		}
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get budget by ID", err)
	}

	return &budget, nil
}

func (r *Repository) Create(ctx context.Context, budget *domain.Budget) error {
	query := `
		INSERT INTO budgets 
			(id, user_id, name, type, kind, amount, period_unit, period_value, start_date, created_at, updated_at)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		budget.ID,
		budget.UserID,
		budget.Name,
		budget.Type,
		budget.Kind,
		budget.Amount,
		budget.PeriodUnit,
		budget.PeriodValue,
		budget.StartDate,
		budget.CreatedAt,
		budget.UpdatedAt,
	)
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to create budget", err)
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, budget *domain.Budget) error {
	query := `
		UPDATE budgets
		SET name = $1,
			type = $2,
			kind = $3,
			amount = $4,
			period_unit = $5,
			period_value = $6,
			start_date = $7,
			updated_at = $8
		WHERE id = $9
	`
	_, err := r.pool.Exec(ctx, query,
		budget.Name,
		budget.Type,
		budget.Kind,
		budget.Amount,
		budget.PeriodUnit,
		budget.PeriodValue,
		budget.StartDate,
		budget.UpdatedAt,
		budget.ID,
	)
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to update budget", err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, budgetID uuid.UUID) error {
	query := `
		DELETE FROM budgets
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, budgetID)
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to delete budget", err)
	}

	return nil
}

// AddCategoryToBudget adds a category to a category-type budget
func (r *Repository) AddCategoryToBudget(ctx context.Context, budgetID, categoryID uuid.UUID) error {
	query := `
		INSERT INTO budget_categories (id, budget_id, category_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (budget_id, category_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, uuid.New(), budgetID, categoryID, time.Now().UTC())
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to add category to budget", err)
	}
	return nil
}

// RemoveCategoryFromBudget removes a category from a budget
func (r *Repository) RemoveCategoryFromBudget(ctx context.Context, budgetID, categoryID uuid.UUID) error {
	query := `
		DELETE FROM budget_categories
		WHERE budget_id = $1 AND category_id = $2
	`
	_, err := r.pool.Exec(ctx, query, budgetID, categoryID)
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to remove category from budget", err)
	}
	return nil
}

// GetCategoriesForBudget retrieves all categories associated with a budget
func (r *Repository) GetCategoriesForBudget(ctx context.Context, budgetID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT category_id
		FROM budget_categories
		WHERE budget_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, budgetID)
	if err != nil {
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get categories for budget", err)
	}
	defer rows.Close()

	var categoryIDs []uuid.UUID
	for rows.Next() {
		var categoryID uuid.UUID
		if err := rows.Scan(&categoryID); err != nil {
			return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to scan category ID", err)
		}
		categoryIDs = append(categoryIDs, categoryID)
	}

	if err := rows.Err(); err != nil {
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get categories for budget", err)
	}

	return categoryIDs, nil
}

// GetSpentAmount calculates the spent amount for a budget based on its type and kind
// Rules:
// - Expense + Category: sum of expense transactions in those categories
// - Expense + Overall: sum of ALL expense transactions
// - Savings + Category: (income - expenses) for those categories
// - Savings + Overall: (total income - total expenses) across all categories
func (r *Repository) GetSpentAmount(
	ctx context.Context, budget *domain.Budget, start, end time.Time,
) (int64, error) {

	switch budget.Kind {
	case domain.BudgetKindExpense:
		return r.getExpenseSum(ctx, budget.UserID, budget.CategoryIDs, start, end)
	case domain.BudgetKindSavings:
		return r.getSavingsSum(ctx, budget.UserID, budget.CategoryIDs, start, end)
	default:
		return 0, platformErrors.NewAppError(platformErrors.CodeInvalidInput, "Invalid budget kind", nil)
	}
}

func (r *Repository) getExpenseSum(
	ctx context.Context,
	userID uuid.UUID,
	categoryIDs []uuid.UUID,
	start, end time.Time,
) (int64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = $1
		  AND type = 'Expense'
		  AND date >= $2
		  AND date < $3
	`

	args := []any{userID, start, end}

	if len(categoryIDs) > 0 {
		query += " AND category_id = ANY($4)"
		args = append(args, categoryIDs)
	}

	var sum int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&sum)
	if err != nil {
		return 0, err
	}

	return sum, nil
}

func (r *Repository) getSavingsSum(
	ctx context.Context,
	userID uuid.UUID,
	categoryIDs []uuid.UUID,
	start, end time.Time,
) (int64, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'Income' THEN amount END), 0) -
			COALESCE(SUM(CASE WHEN type = 'Expense' THEN amount END), 0)
		FROM transactions
		WHERE user_id = $1
		  AND date >= $2
		  AND date < $3
	`

	args := []any{userID, start, end}

	if len(categoryIDs) > 0 {
		query += " AND (type = 'Income' OR category_id = ANY($4))"
		args = append(args, categoryIDs)
	}

	var result int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

// Helper function to create pointers
func ptrOf[T any](v T) *T {
	return &v
}
