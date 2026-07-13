package service

import (
	"backend-go/internal/budgets/domain"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeBudgetRepo struct {
	byID  map[uuid.UUID]*domain.Budget
	spent int64
}

func (f *fakeBudgetRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]domain.Budget, error) {
	return []domain.Budget{}, nil
}

func (f *fakeBudgetRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Budget, error) {
	b, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrBudgetNotFound
	}
	return b, nil
}

func (f *fakeBudgetRepo) Create(_ context.Context, _ *domain.Budget) error { return nil }
func (f *fakeBudgetRepo) Update(_ context.Context, _ *domain.Budget) error { return nil }
func (f *fakeBudgetRepo) Delete(_ context.Context, _ uuid.UUID) error      { return nil }
func (f *fakeBudgetRepo) AddCategoryToBudget(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (f *fakeBudgetRepo) RemoveCategoryFromBudget(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (f *fakeBudgetRepo) GetCategoriesForBudget(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}
func (f *fakeBudgetRepo) ClearCategoriesFromBudget(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (f *fakeBudgetRepo) GetSpentAmount(_ context.Context, _ *domain.Budget, _, _ time.Time) (int64, error) {
	return f.spent, nil
}

func seedBudget(repo *fakeBudgetRepo, kind domain.BudgetKind, amount int64) (*domain.Budget, uuid.UUID) {
	userID := uuid.New()
	b := &domain.Budget{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        "Test Budget",
		Type:        domain.BudgetTypeOverall,
		Kind:        kind,
		Amount:      amount,
		PeriodUnit:  domain.PeriodUnitMonth,
		PeriodValue: 1,
		StartDate:   time.Now().UTC().Add(-48 * time.Hour),
	}
	repo.byID[b.ID] = b
	return b, userID
}

func TestGetBudgetStatusMath(t *testing.T) {
	tests := []struct {
		name         string
		kind         domain.BudgetKind
		amount       int64 // paisa
		spent        int64 // paisa
		wantStatus   BudgetHealthStatus
		wantProgress float64
		wantRemain   int64
	}{
		{"expense under 75% is healthy", domain.BudgetKindExpense, 10000, 5000, BudgetStatusHealthy, 50, 5000},
		{"expense at 75% warns", domain.BudgetKindExpense, 10000, 7500, BudgetStatusWarning, 75, 2500},
		{"expense over 100% exceeded", domain.BudgetKindExpense, 10000, 11000, BudgetStatusExceeded, 110, -1000},
		{"savings under goal is healthy", domain.BudgetKindSavings, 10000, 5000, BudgetStatusHealthy, 50, 5000},
		{"savings at goal achieved", domain.BudgetKindSavings, 10000, 10000, BudgetStatusAchieved, 100, 0},
		{"savings past goal stays achieved", domain.BudgetKindSavings, 10000, 15000, BudgetStatusAchieved, 150, -5000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeBudgetRepo{byID: map[uuid.UUID]*domain.Budget{}, spent: tc.spent}
			budget, userID := seedBudget(repo, tc.kind, tc.amount)
			svc := NewService(repo, nil, nil)

			status, err := svc.GetBudgetStatus(context.Background(), budget.ID, userID)
			if err != nil {
				t.Fatal(err)
			}
			if status.Status != tc.wantStatus {
				t.Errorf("status: want %s, got %s", tc.wantStatus, status.Status)
			}
			if status.ProgressPercent != tc.wantProgress {
				t.Errorf("progress: want %v, got %v", tc.wantProgress, status.ProgressPercent)
			}
			if status.Remaining != tc.wantRemain {
				t.Errorf("remaining: want %d, got %d", tc.wantRemain, status.Remaining)
			}
		})
	}
}

func TestGetBudgetStatusOwnership(t *testing.T) {
	repo := &fakeBudgetRepo{byID: map[uuid.UUID]*domain.Budget{}}
	budget, _ := seedBudget(repo, domain.BudgetKindExpense, 10000)
	svc := NewService(repo, nil, nil)

	if _, err := svc.GetBudgetStatus(context.Background(), budget.ID, uuid.New()); !errors.Is(err, domain.ErrCannotModifyOther) {
		t.Fatalf("want ErrCannotModifyOther, got %v", err)
	}
}

func TestCreateBudgetValidation(t *testing.T) {
	repo := &fakeBudgetRepo{byID: map[uuid.UUID]*domain.Budget{}}
	svc := NewService(repo, nil, nil)
	userID := uuid.New()
	start := time.Now().UTC()

	if _, err := svc.Create(context.Background(), userID, "Groceries", domain.BudgetTypeOverall, domain.BudgetKindExpense, 50000, domain.PeriodUnitMonth, 1, start); err != nil {
		t.Fatalf("valid budget: %v", err)
	}
	if _, err := svc.Create(context.Background(), userID, "Bad", "weekly", domain.BudgetKindExpense, 50000, domain.PeriodUnitMonth, 1, start); err == nil {
		t.Fatal("invalid type must be rejected")
	}
	if _, err := svc.Create(context.Background(), userID, "Bad", domain.BudgetTypeOverall, domain.BudgetKindExpense, 0, domain.PeriodUnitMonth, 1, start); err == nil {
		t.Fatal("zero amount must be rejected")
	}
}
