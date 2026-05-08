package service

import (
	"backend-go/internal/categories/domain"
	transactionDomain "backend-go/internal/transactions/domain"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db	*pgxpool.Pool
	categoryRepo     domain.CategoryRepository
	userCategoryRepo domain.UserCategoryRepository
	transactionRepo	transactionDomain.TransactionRepository
}

func NewService(db *pgxpool.Pool, categoryRepo domain.CategoryRepository, userCategoryRepo domain.UserCategoryRepository, transactionRepo transactionDomain.TransactionRepository) *Service {
	return &Service{
		db: db,
		categoryRepo:     categoryRepo,
		userCategoryRepo: userCategoryRepo,
		transactionRepo:  transactionRepo,
	}
}

func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.ResolvedCategory, error) {
	// fetch system categories
	systemCategories, err := s.categoryRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	// fetch user categories
	userCategories, err := s.userCategoryRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// look-up map
	overrideMap := make(map[uuid.UUID]domain.UserCategory)
	for _, uc := range userCategories {
		if uc.CategoryID != nil {
			overrideMap[*uc.CategoryID] = uc
		}
	}

	var resolved []domain.ResolvedCategory
	// merge system default with user categories
	for _, sys := range systemCategories {
		if uc, exists := overrideMap[sys.ID]; exists {
			// has override - apply it
			resolved = append(resolved, uc.Resolve(&sys))
		} else {
			// no override - resolve directly
			resolved = append(resolved, domain.ResolvedCategory{
				ID:     sys.ID,
				Name:   sys.Name,
				Icon:   sys.Icon,
				Hidden: false,
				Custom: false,
			})
		}
	}

	// append the custom categories (category_id is nil)
	for _, uc := range userCategories {
		if uc.CategoryID == nil {
			resolved = append(resolved, uc.Resolve(nil))
		}
	}

	return resolved, nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, name, icon string) error {
	// check for duplicate
	exists, err := s.userCategoryRepo.ExistsByName(ctx, userID, name, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrDuplicateName
	}

	// create new user category
	uc, err := domain.NewUserCategory(userID, name, icon)
	if err != nil {
		return err
	}

	// create and persist to db
	return s.userCategoryRepo.Create(ctx, uc)
}

func (s *Service) RenameCategory(ctx context.Context, userID uuid.UUID, id uuid.UUID, name string, icon *string) error {
	// try user category first
	uc, err := s.userCategoryRepo.GetUserCategoryByID(ctx, id)
	if err == nil {
		// check if the user reqesting is the same user that owns it
		if uc.UserID != userID {
			return domain.ErrCannotModifyOther // return err if not the same user that's requesting
		}

		// check duplicate name before renaming
		exists, err := s.userCategoryRepo.ExistsByName(ctx, userID, name, &uc.ID)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrDuplicateName
		}

		// rename the struct when same user is requesting and doesn't have duplicate name
		if err := uc.Rename(name, icon); err != nil {
			return err
		}

		// persist to db
		return s.userCategoryRepo.Update(ctx, uc)
	}

	// try system category next
	_, err = s.categoryRepo.GetCategoryByID(ctx, id)
	if err != nil {
		return domain.ErrCategoryNotFound
	}

	// check duplicate name before renaming
	exists, err := s.userCategoryRepo.ExistsByName(ctx, userID, name, nil)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrDuplicateName
	}

	// check if override row already exists
	existing, err := s.userCategoryRepo.GetByUserAndCategory(ctx, userID, id)
	if err != nil {
		return err
	}

	if existing != nil {
		exists, err = s.userCategoryRepo.ExistsByName(ctx, userID, name, &existing.ID)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrDuplicateName
		}
		// override exists — just rename it
		if err := existing.Rename(name, icon); err != nil {
			return err
		}
		return s.userCategoryRepo.Update(ctx, existing)
	}

	// no override yet — create one with custom name set
	uc = domain.NewSystemOverride(userID, id)
	if err := uc.Rename(name, icon); err != nil {
		return err
	}

	// create and persist to db
	return s.userCategoryRepo.Create(ctx, uc)
}

// hide custom + override categories
func (s *Service) HideCategory(ctx context.Context, userID, ucID uuid.UUID) error {
	// try user category first
	uc, err := s.userCategoryRepo.GetUserCategoryByID(ctx, ucID)
	if err == nil {
		// check if the user reqesting is the same user that owns it
		if uc.UserID != userID {
			return domain.ErrCannotModifyOther // return err if not the same user that's requesting
		}

		// hide the struct when same user is requesting
		uc.Hide()
		return s.userCategoryRepo.Update(ctx, uc)
	}

	// try system category next
	_, err = s.categoryRepo.GetCategoryByID(ctx, ucID)
	if err != nil {
		return domain.ErrCategoryNotFound
	}

	// check if override row already exists
	existing, err := s.userCategoryRepo.GetByUserAndCategory(ctx, userID, ucID)
	if err != nil {
		return err
	}

	if existing != nil {
		// override exists — just hide it
		existing.Hide()
		return s.userCategoryRepo.Update(ctx, existing)
	}

	// no override yet — create one with hidden = true
	uc = domain.NewSystemOverride(userID, ucID)
	uc.Hide()
	return s.userCategoryRepo.Create(ctx, uc)
}

// unhide custom + override categories
func (s *Service) Unhide(ctx context.Context, userID, ucID uuid.UUID) error {
	uc, err := s.userCategoryRepo.GetUserCategoryByID(ctx, ucID)
	if err != nil {
		return err
	}

	if uc.UserID != userID {
		return domain.ErrCannotModifyOther
	}

	uc.Unhide()
	return s.userCategoryRepo.Update(ctx, uc)
}

// HideSystemCategory handles the case where a user hides a system default.
// System defaults don't have a user_categories row yet — we create one lazily.
func (s *Service) HideSystemCategory(ctx context.Context, userID, categoryID uuid.UUID) error {
	// verify the system category actually exists
	_, err := s.categoryRepo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		return err
	}

	// check if override row already exists
	existing, err := s.userCategoryRepo.GetByUserAndCategory(ctx, userID, categoryID)
	if err != nil {
		return err
	}

	if existing != nil {
		// override exists — just hide it
		existing.Hide()
		return s.userCategoryRepo.Update(ctx, existing)
	}

	// no override yet — create one with hidden = true
	uc := domain.NewSystemOverride(userID, categoryID)
	uc.Hide()
	return s.userCategoryRepo.Create(ctx, uc)
}

func (s *Service) DeleteCategory(ctx context.Context, userId, categoryID uuid.UUID) error {
	// get category to delete, check if it's custom or override and verify ownership
	uc, err := s.userCategoryRepo.GetUserCategoryByID(ctx, categoryID)
	if err != nil {
		return err
	}
	if uc.UserID != userId {
		return domain.ErrCannotModifyOther
	}

	// not allowing deletion of uncategorised category
	if uc.CategoryID != nil {
		sys, err := s.categoryRepo.GetCategoryByID(ctx, *uc.CategoryID)
		if err == nil && sys.Name == "Uncategorised" {
			return domain.ErrCannotDeleteSystemCategory
		}
	}

	// get the "Uncategorised" category to reassign transactions before deleting the category
	uncategorised, err := s.categoryRepo.GetByName(ctx, "Uncategorised")
	if err != nil {
		return err
	}
	if uncategorised == nil {
		fmt.Println("uncategorised not found")
		return domain.ErrCategoryNotFound
	}

	// check if the user already has an override for "Uncategorised" category, if not create one
	uncategoriesdUC, err := s.userCategoryRepo.GetByUserAndCategory(ctx, userId, uncategorised.ID)
	if err != nil {
		return err
	}

	if uncategoriesdUC == nil {
		uncategoriesdUC = domain.NewSystemOverride(userId, uncategorised.ID)
		if err := s.userCategoryRepo.Create(ctx, uncategoriesdUC); err != nil {
			return err
		}
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// reassign transactions to "Uncategorised" category before deleting the category (only for custom categories)
	if uc.CategoryID == nil {
		if err := s.transactionRepo.ReassignCategoryTx(ctx, tx, userId, uc.ID, uncategoriesdUC.ID); err != nil {
			return err
		}
	}
	if err := s.userCategoryRepo.DeleteTx(ctx, tx, userId, categoryID); err != nil {
		return err
	}
	
	return tx.Commit(ctx)
}