package service

import (
	"backend-go/internal/categories/domain"
	"context"

	"github.com/google/uuid"
)

type Service struct {
	categoryRepo     domain.CategoryRepository
	userCategoryRepo domain.UserCategoryRepository
}

func NewService(categoryRepo domain.CategoryRepository, userCategoryRepo domain.UserCategoryRepository) *Service {
	return &Service{
		categoryRepo:     categoryRepo,
		userCategoryRepo: userCategoryRepo,
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
