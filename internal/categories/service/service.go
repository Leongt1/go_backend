package service

import (
	"backend-go/internal/categories/domain"
	transactionDomain "backend-go/internal/transactions/domain"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db              *pgxpool.Pool
	categoryRepo    domain.CategoryRepository
	transactionRepo transactionDomain.TransactionRepository
}

func NewService(db *pgxpool.Pool, categoryRepo domain.CategoryRepository, transactionRepo transactionDomain.TransactionRepository) *Service {
	return &Service{
		db:              db,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
	}
}

func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Category, error) {
	return s.categoryRepo.ListByUser(ctx, userID)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, name, icon string) error {
	// check for duplicate
	exists, err := s.categoryRepo.ExistsByName(ctx, userID, name, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrDuplicateName
	}

	// create new user category
	c, err := domain.NewCategory(userID, name, icon)
	if err != nil {
		return err
	}

	// create and persist to db
	return s.categoryRepo.Create(ctx, c)
}

func (s *Service) RenameCategory(ctx context.Context, userID uuid.UUID, id uuid.UUID, name string, icon *string) error {
	// try user category first
	c, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.UserID != userID {
		return domain.ErrCannotModifyOther
	}

	if c.Name == "Uncategorised" {
		return domain.ErrCannotModifyUncategorised
	}

	exists, err := s.categoryRepo.ExistsByName(ctx, userID, name, &id)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrDuplicateName
	}

	if err := c.Rename(name, icon); err != nil {
		return err
	}

	// persist to db
	return s.categoryRepo.Update(ctx, c)
	
}

// hide custom + override categories
func (s *Service) HideCategory(ctx context.Context, userID, ucID uuid.UUID) error {
	// try user category first
	c, err := s.categoryRepo.GetByID(ctx, ucID)
	if err != nil {
		return err
	}
	if c.UserID != userID {
		return domain.ErrCannotModifyOther // return err if not the same user that's requesting
	}

	if c.Name == "Uncategorised" {
		return domain.ErrCannotModifyUncategorised
	}

	c.Hide()
	return s.categoryRepo.Update(ctx, c)
}

// unhide custom + override categories
func (s *Service) Unhide(ctx context.Context, userID, ucID uuid.UUID) error {
	c, err := s.categoryRepo.GetByID(ctx, ucID)
	if err != nil {
		return err
	}
	if c.UserID != userID {
		return domain.ErrCannotModifyOther
	}

	c.Unhide()
	return s.categoryRepo.Update(ctx, c)
}

func (s *Service) DeleteCategory(ctx context.Context, userId, categoryID uuid.UUID) error {
	// get category to delete, check if it's custom or override and verify ownership
	c, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return err
	}
	if c.UserID != userId {
		return domain.ErrCannotModifyOther
	}

	if c.Name == "Uncategorised" {
		return domain.ErrCannotModifyUncategorised
	}

	// get the "Uncategorised" category to reassign transactions before deleting the category
	uncategorised, err := s.categoryRepo.GetByName(ctx, userId, "Uncategorised")
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// reassign transactions to "Uncategorised" category before deleting the category (only for custom categories)
	if err := s.transactionRepo.ReassignCategoryTx(ctx, tx, userId, c.ID, uncategorised.ID); err != nil {
		return err
	}
	
	if err := s.categoryRepo.DeleteTx(ctx, tx, userId, categoryID); err != nil {
		return err
	}
	
	return tx.Commit(ctx)
}