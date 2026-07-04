package service

import (
	"backend-go/internal/budgets/domain"
	categoryDomain "backend-go/internal/categories/domain"
	platformErrors "backend-go/internal/platform/errors"
	transactionDomain "backend-go/internal/transactions/domain"
	"context"
	"math"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	budgetRepo      domain.BudgetRepository
	categoryRepo    categoryDomain.CategoryRepository
	transactionRepo transactionDomain.TransactionRepository
}

func NewService(budgetRepo domain.BudgetRepository, categoryRepo categoryDomain.CategoryRepository, transactionRepo transactionDomain.TransactionRepository) *Service {
	return &Service{
		budgetRepo:      budgetRepo,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
	}
}

// ListByUser retrieves all budgets for a user
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Budget, error) {
	return s.budgetRepo.ListByUser(ctx, userID)
}

// GetByID retrieves a specific budget and populates its categories
func (s *Service) GetByID(ctx context.Context, budgetID uuid.UUID) (*domain.Budget, error) {
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return nil, err
	}

	// Populate categories for this budget
	if budget.Type == domain.BudgetTypeCategory {
		categoryIDs, err := s.budgetRepo.GetCategoriesForBudget(ctx, budgetID)
		if err != nil {
			return nil, err
		}
		budget.CategoryIDs = categoryIDs
	}

	return budget, nil
}

// Create creates a new budget
func (s *Service) Create(
	ctx context.Context,
	userID uuid.UUID,
	name string,
	budgetType domain.BudgetType,
	budgetKind domain.BudgetKind,
	amount int64,
	periodUnit domain.PeriodUnit,
	periodValue int,
	startDate time.Time,
) (*domain.Budget, error) {
	// Validate budget type and kind
	if err := validateBudgetTypeAndKind(budgetType, budgetKind); err != nil {
		return nil, err
	}

	// Create budget
	budget, err := domain.NewBudget(userID, name, budgetType, budgetKind, amount, periodUnit, periodValue, startDate)
	if err != nil {
		return nil, err
	}

	// Persist to database
	if err := s.budgetRepo.Create(ctx, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

// Update updates an existing budget
func (s *Service) Update(
	ctx context.Context,
	budgetID, userID uuid.UUID,
	name *string,
	amount *int64,
	periodUnit *domain.PeriodUnit, periodValue *int,
	startDate *time.Time,
	bType *domain.BudgetType,
) error {
	// Get the budget to verify ownership
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return err
	}
	// Verify user owns this budget
	if budget.UserID != userID {
		return domain.ErrCannotModifyOther
	}

	// Clean up categories if type changes from 'category' to 'overall'
	if bType != nil && *bType == domain.BudgetTypeOverall && budget.Type == domain.BudgetTypeCategory {
		if err := s.budgetRepo.ClearCategoriesFromBudget(ctx, budgetID); err != nil {
			return err
		}
	}

	// Apply updates
	if err := budget.Update(name, amount, periodUnit, periodValue, startDate, bType); err != nil {
		return err
	}

	// Persist to database
	return s.budgetRepo.Update(ctx, budget)
}

// Delete deletes a budget
func (s *Service) Delete(ctx context.Context, budgetID uuid.UUID, userID uuid.UUID) error {
	// Get the budget to verify ownership
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return err
	}

	// Verify user owns this budget
	if budget.UserID != userID {
		return domain.ErrCannotModifyOther
	}

	return s.budgetRepo.Delete(ctx, budgetID)
}

// AddCategoryToBudget adds a category to a budget
func (s *Service) AddCategoryToBudget(ctx context.Context, budgetID, categoryID, userID uuid.UUID) error {
	// Verify budget exists and user owns it
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return err
	}

	if budget.UserID != userID {
		return domain.ErrCannotModifyOther
	}

	// Only category-type budgets can have categories
	if budget.Type != domain.BudgetTypeCategory {
		return platformErrors.NewDomainError(
			platformErrors.CodeInvalidInput,
			"Only category-type budgets can have categories",
		)
	}

	// Verify category exists and user owns it
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return err
	}

	if category.UserID != userID {
		return categoryDomain.ErrCategoryNotFound
	}

	// Check if category is hidden
	if category.Hidden {
		return platformErrors.NewDomainError(platformErrors.CodeCategoryHidden, "Cannot add a hidden category to a budget")
	}

	return s.budgetRepo.AddCategoryToBudget(ctx, budgetID, categoryID)
}

// RemoveCategoryFromBudget removes a category from a budget
func (s *Service) RemoveCategoryFromBudget(ctx context.Context, budgetID, categoryID, userID uuid.UUID) error {
	// Verify budget exists and user owns it
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return err
	}

	if budget.UserID != userID {
		return domain.ErrCannotModifyOther
	}

	return s.budgetRepo.RemoveCategoryFromBudget(ctx, budgetID, categoryID)
}

// GetBudgetStatus returns budget information including spent and remaining amounts
func (s *Service) GetBudgetStatus(ctx context.Context, budgetID uuid.UUID, userID uuid.UUID) (*BudgetStatus, error) {
	// Get the budget
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return nil, err
	}

	// Verify user owns this budget
	if budget.UserID != userID {
		return nil, domain.ErrCannotModifyOther
	}

	// Populate categories if needed
	if budget.Type == domain.BudgetTypeCategory {
		categoryIDs, err := s.budgetRepo.GetCategoriesForBudget(ctx, budgetID)
		if err != nil {
			return nil, err
		}
		budget.CategoryIDs = categoryIDs
	}

	// Calculate spent amount
	start, end := domain.CurrentPeriod(budget.StartDate, budget.PeriodUnit, budget.PeriodValue)

	spent, err := s.budgetRepo.GetSpentAmount(ctx, budget, start, end)
	if err != nil {
		return nil, err
	}

	// Calculate remaining
	remaining := budget.Amount - spent

	// Calculate progress percentage
	var progressPercent float64
	if budget.Amount > 0 {
		progressPercent = (float64(spent) / float64(budget.Amount)) * 100
		progressPercent = math.Round(progressPercent*100) / 100 // round to 2 decimal places
	}

	// Determine status. For expense budgets crossing the amount is bad;
	// for savings budgets reaching the amount means the goal is achieved.
	status := BudgetStatusHealthy
	if budget.Kind == domain.BudgetKindSavings {
		if progressPercent >= 100 {
			status = BudgetStatusAchieved
		}
	} else {
		if progressPercent >= 100 {
			status = BudgetStatusExceeded
		} else if progressPercent >= 75 {
			status = BudgetStatusWarning
		}
	}

	return &BudgetStatus{
		BudgetID:        budget.ID,
		Name:            budget.Name,
		BudgetAmount:    budget.Amount,
		Spent:           spent,
		Remaining:       remaining,
		ProgressPercent: progressPercent,
		Status:          status,
		PeriodStart:     start,
		PeriodEnd:       end,
	}, nil
}

// Helper functions

func validateBudgetTypeAndKind(bType domain.BudgetType, kind domain.BudgetKind) error {
	validTypes := map[domain.BudgetType]bool{
		domain.BudgetTypeCategory: true,
		domain.BudgetTypeOverall:  true,
	}

	validKinds := map[domain.BudgetKind]bool{
		domain.BudgetKindExpense: true,
		domain.BudgetKindSavings: true,
	}

	if !validTypes[bType] {
		return domain.ErrInvalidInput
	}

	if !validKinds[kind] {
		return domain.ErrInvalidInput
	}

	return nil
}

// BudgetStatus represents the current status of a budget
type BudgetStatus struct {
	BudgetID        uuid.UUID
	Name            string
	BudgetAmount    int64
	Spent           int64
	Remaining       int64
	ProgressPercent float64
	Status          BudgetHealthStatus
	PeriodStart     time.Time
	PeriodEnd       time.Time
}

type BudgetHealthStatus string

const (
	BudgetStatusHealthy  BudgetHealthStatus = "healthy"
	BudgetStatusWarning  BudgetHealthStatus = "warning"
	BudgetStatusExceeded BudgetHealthStatus = "exceeded"
	BudgetStatusAchieved BudgetHealthStatus = "achieved"
)
