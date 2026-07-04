package service

import (
	categoryDomain "backend-go/internal/categories/domain"
	"backend-go/internal/transactions/domain"
	"context"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo         domain.TransactionRepository
	categoryRepo categoryDomain.CategoryRepository
}

func NewService(
	repo domain.TransactionRepository,
	categoryRepo categoryDomain.CategoryRepository,
) *Service {
	return &Service{
		repo:         repo,
		categoryRepo: categoryRepo,
	}
}

type CreateInput struct {
	UserID      uuid.UUID
	CategoryID  uuid.UUID
	Amount      int64
	Description *string
	Type        string
	Date        time.Time
}

func (s *Service) CreateTransaction(
	ctx context.Context,
	input CreateInput,
) error {
	// resolves category and returns the user_categories row ID to store
	if err := s.validateCategory(ctx, input.UserID, input.CategoryID); err != nil {
		return err
	}

	txType, err := domain.ParseTransactionType(input.Type)
	if err != nil {
		return err
	}

	tx, err := domain.NewTransaction(
		input.UserID,
		input.CategoryID,
		input.Description,
		input.Amount,
		txType,
		input.Date,
	)
	if err != nil {
		return err
	}

	// Create transaction
	return s.repo.Create(ctx, tx)
}

func (s *Service) GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.Transaction, error) {
	tx, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// make sure this transaction belongs to the requesting user
	if tx.UserID != userID {
		return nil, domain.ErrCannotModifyOther
	}

	return tx, nil
}

type ListInput struct {
	UserID     uuid.UUID
	CategoryID *uuid.UUID
	Type       *string
	DateFrom   *time.Time
	DateTo     *time.Time
}

func (s *Service) ListTransactions(ctx context.Context, req *ListInput) ([]domain.Transaction, error) {
	filter := domain.TransactionFilter{
		CategoryID: req.CategoryID,
		DateFrom:   req.DateFrom,
		DateTo:     req.DateTo,
	}

	// parse type filter if provided
	if req.Type != nil {
		txType, err := domain.ParseTransactionType(*req.Type)
		if err != nil {
			return nil, err
		}
		filter.Type = &txType
	}

	return s.repo.List(ctx, req.UserID, filter)
}

type UpdateInput struct {
	CategoryID  *uuid.UUID
	Amount      *int64
	Description *string
	Type        *string
	Date        *time.Time
	UpdatedBy   *uuid.UUID
}

func (s *Service) UpdateTransaction(ctx context.Context, userID, id uuid.UUID, req *UpdateInput) (*domain.Transaction, error) {
	// fetch and verify ownership
	tx, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if tx.UserID != userID {
		return nil, domain.ErrCannotModifyOther
	}

	// validate new category if provided
	if req.CategoryID != nil {
		if err := s.validateCategory(ctx, userID, *req.CategoryID); err != nil {
			return nil, err
		}
	}

	// parse type if provided
	var txType *domain.TransactionType
	if req.Type != nil {
		parsed, err := domain.ParseTransactionType(*req.Type)
		if err != nil {
			return nil, err
		}
		txType = &parsed
	}

	if err := tx.Update(
		req.CategoryID,
		req.Amount,
		req.Description,
		txType,
		req.Date,
		req.UpdatedBy,
	); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *Service) DeleteTransaction(ctx context.Context, userID, id uuid.UUID) error {
	tx, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if tx.UserID != userID {
		return domain.ErrCannotModifyOther
	}

	return s.repo.Delete(ctx, id)
}

// validateCategory replaces the old resolveAndValidateCategory —
// no more lazy creation, no more two-table lookup.
// Every category is already a user_categories row seeded at signup.
func (s *Service) validateCategory(ctx context.Context, userID, categoryID uuid.UUID) error {
	c, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return err
	}

	if c.UserID != userID {
		return categoryDomain.ErrCannotModifyOther
	}

	if c.Hidden {
		return categoryDomain.ErrCategoryHidden
	}

	return nil
}