package service

import (
	categoryDomain "backend-go/internal/categories/domain"
	"backend-go/internal/transactions/domain"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---- fakes ----

type fakeTxRepo struct {
	created    []*domain.Transaction
	byID       map[uuid.UUID]*domain.Transaction
	listResult []domain.Transaction
	countCalls int
}

func (f *fakeTxRepo) Create(_ context.Context, tx *domain.Transaction) error {
	f.created = append(f.created, tx)
	return nil
}

func (f *fakeTxRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Transaction, error) {
	tx, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrTransactionNotFound
	}
	return tx, nil
}

func (f *fakeTxRepo) List(_ context.Context, _ uuid.UUID, _ domain.TransactionFilter) ([]domain.Transaction, error) {
	if f.listResult == nil {
		return []domain.Transaction{}, nil
	}
	return f.listResult, nil
}

func (f *fakeTxRepo) Count(_ context.Context, _ uuid.UUID, _ domain.TransactionFilter) (int64, error) {
	f.countCalls++
	return int64(len(f.listResult)), nil
}

func (f *fakeTxRepo) Update(_ context.Context, _ *domain.Transaction) error { return nil }
func (f *fakeTxRepo) Delete(_ context.Context, _ uuid.UUID) error           { return nil }
func (f *fakeTxRepo) ReassignCategoryTx(_ context.Context, _ pgx.Tx, _, _, _ uuid.UUID) error {
	return nil
}

type fakeCategoryRepo struct {
	byID map[uuid.UUID]*categoryDomain.Category
}

func (f *fakeCategoryRepo) GetByID(_ context.Context, id uuid.UUID) (*categoryDomain.Category, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, categoryDomain.ErrCategoryNotFound
	}
	return c, nil
}

func (f *fakeCategoryRepo) Create(_ context.Context, _ *categoryDomain.Category) error { return nil }
func (f *fakeCategoryRepo) GetByName(_ context.Context, _ uuid.UUID, _ string) (*categoryDomain.Category, error) {
	return nil, categoryDomain.ErrCategoryNotFound
}
func (f *fakeCategoryRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]categoryDomain.Category, error) {
	return []categoryDomain.Category{}, nil
}
func (f *fakeCategoryRepo) Update(_ context.Context, _ *categoryDomain.Category) error { return nil }
func (f *fakeCategoryRepo) DeleteTx(_ context.Context, _ pgx.Tx, _, _ uuid.UUID) error { return nil }
func (f *fakeCategoryRepo) ExistsByName(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
	return false, nil
}

// ---- helpers ----

func newTestService() (*Service, *fakeTxRepo, *fakeCategoryRepo, uuid.UUID, uuid.UUID) {
	userID := uuid.New()
	categoryID := uuid.New()
	catRepo := &fakeCategoryRepo{byID: map[uuid.UUID]*categoryDomain.Category{
		categoryID: {ID: categoryID, UserID: userID, Name: "Food", Hidden: false},
	}}
	txRepo := &fakeTxRepo{byID: map[uuid.UUID]*domain.Transaction{}}
	return NewService(txRepo, catRepo), txRepo, catRepo, userID, categoryID
}

// ---- tests ----

func TestCreateTransaction(t *testing.T) {
	yesterday := time.Now().UTC().Add(-24 * time.Hour)
	tomorrow := time.Now().UTC().Add(24 * time.Hour)

	tests := []struct {
		name    string
		mutate  func(*CreateInput, *fakeCategoryRepo)
		wantErr error
		wantOK  bool
	}{
		{
			name:   "valid expense",
			mutate: func(in *CreateInput, _ *fakeCategoryRepo) {},
			wantOK: true,
		},
		{
			name:    "zero amount rejected",
			mutate:  func(in *CreateInput, _ *fakeCategoryRepo) { in.Amount = 0 },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "negative amount rejected",
			mutate:  func(in *CreateInput, _ *fakeCategoryRepo) { in.Amount = -100 },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "future date rejected",
			mutate:  func(in *CreateInput, _ *fakeCategoryRepo) { in.Date = tomorrow },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "unknown type rejected",
			mutate:  func(in *CreateInput, _ *fakeCategoryRepo) { in.Type = "Transfer" },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "other user's category rejected",
			mutate: func(in *CreateInput, cats *fakeCategoryRepo) {
				cats.byID[in.CategoryID].UserID = uuid.New()
			},
			wantErr: categoryDomain.ErrCannotModifyOther,
		},
		{
			name: "hidden category rejected",
			mutate: func(in *CreateInput, cats *fakeCategoryRepo) {
				cats.byID[in.CategoryID].Hidden = true
			},
			wantErr: categoryDomain.ErrCategoryHidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, txRepo, catRepo, userID, categoryID := newTestService()
			input := CreateInput{
				UserID:     userID,
				CategoryID: categoryID,
				Amount:     25000,
				Type:       "Expense",
				Date:       yesterday,
			}
			tc.mutate(&input, catRepo)

			err := svc.CreateTransaction(context.Background(), input)

			if tc.wantOK {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				if len(txRepo.created) != 1 {
					t.Fatalf("expected 1 created transaction, got %d", len(txRepo.created))
				}
				if txRepo.created[0].UserID != userID {
					t.Errorf("created transaction has wrong user")
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if len(txRepo.created) != 0 {
				t.Errorf("no transaction should have been created")
			}
		})
	}
}

func TestGetByIDOwnership(t *testing.T) {
	svc, txRepo, _, userID, categoryID := newTestService()
	other := uuid.New()
	txID := uuid.New()
	txRepo.byID[txID] = &domain.Transaction{ID: txID, UserID: other, CategoryID: categoryID}

	if _, err := svc.GetByID(context.Background(), userID, txID); !errors.Is(err, domain.ErrCannotModifyOther) {
		t.Fatalf("want ErrCannotModifyOther, got %v", err)
	}
	if _, err := svc.GetByID(context.Background(), other, txID); err != nil {
		t.Fatalf("owner should read own transaction, got %v", err)
	}
}

func TestListTransactionsPagination(t *testing.T) {
	svc, txRepo, _, userID, categoryID := newTestService()
	txRepo.listResult = []domain.Transaction{
		{ID: uuid.New(), UserID: userID, CategoryID: categoryID, Amount: 100, Type: domain.TransactionTypeExpense},
	}

	// unpaginated: no count query, Total sentinel -1
	out, err := svc.ListTransactions(context.Background(), &ListInput{UserID: userID})
	if err != nil {
		t.Fatal(err)
	}
	if out.Total != -1 {
		t.Errorf("unpaginated Total: want -1, got %d", out.Total)
	}
	if txRepo.countCalls != 0 {
		t.Errorf("unpaginated list must not count, got %d calls", txRepo.countCalls)
	}

	// paginated: count runs, Total populated
	limit := 10
	out, err = svc.ListTransactions(context.Background(), &ListInput{UserID: userID, Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if out.Total != 1 {
		t.Errorf("paginated Total: want 1, got %d", out.Total)
	}
	if txRepo.countCalls != 1 {
		t.Errorf("paginated list must count once, got %d calls", txRepo.countCalls)
	}
}
