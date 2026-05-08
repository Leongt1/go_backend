package repository

import (
	"backend-go/internal/categories/domain"
	platformErrors "backend-go/internal/platform/errors"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateCategory(ctx context.Context, c *domain.Category) error {
	query := `
		INSERT INTO categories (id, name, icon, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, query, c.ID, c.Name, c.Icon, c.CreatedAt)
	if err != nil {
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to create category",
			err,
		)
	}
	return nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Category, error) {
	query := `
	SELECT id, name, icon, created_at
	FROM categories
	ORDER BY name ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to list categories", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.CreatedAt); err != nil {
			return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to scan categories", err)
		}
		categories = append(categories, c)
	}

	return categories, nil
}

func (r *Repository) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	query := `
		SELECT id, name, icon, created_at
		FROM categories
		WHERE id = $1
	`

	var c domain.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.Icon, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get category", err)
	}
	return &c, nil
}

func (r *Repository) Create(ctx context.Context, uc *domain.UserCategory) error {
	query := `
		INSERT INTO user_categories 
			(id, user_id, category_id, custom_name, icon, hidden, deleted_at, created_at, updated_at)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		uc.ID,
		uc.UserID,
		uc.CategoryID,
		uc.CustomName,
		uc.Icon,
		uc.Hidden,
		uc.DeletedAt,
		uc.CreatedAt,
		uc.UpdatedAt,
	)
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to create user category", err)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, uc *domain.UserCategory) error {
	query := `
		UPDATE user_categories
		SET custom_name = $1,
			icon = $2,
			hidden = $3,
			deleted_at = $4,
			updated_at = $5
		WHERE id = $6
	`

	_, err := r.pool.Exec(ctx, query,
		uc.CustomName,
		uc.Icon,
		uc.Hidden,
		uc.DeletedAt,
		uc.UpdatedAt,
		uc.ID,
	)
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to update user category", err)
	}
	return nil
}

func (r *Repository) GetUserCategoryByID(ctx context.Context, id uuid.UUID) (*domain.UserCategory, error) {
	query := `
		SELECT id, user_id, category_id, custom_name, icon, hidden, deleted_at, created_at, updated_at
		FROM user_categories
		WHERE id = $1
	`

	var uc domain.UserCategory
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&uc.ID,
		&uc.UserID,
		&uc.CategoryID,
		&uc.CustomName,
		&uc.Icon,
		&uc.Hidden,
		&uc.DeletedAt,
		&uc.CreatedAt,
		&uc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get user category", err)
	}

	return &uc, nil
}

func (r *Repository) GetByUserAndCategory(ctx context.Context, userID, categoryID uuid.UUID) (*domain.UserCategory, error) {
	query := `
		SELECT id, user_id, category_id, custom_name, icon, hidden, deleted_at, created_at, updated_at
		FROM user_categories
		WHERE user_id = $1 AND category_id = $2
	`

	var uc domain.UserCategory
	err := r.pool.QueryRow(ctx, query, userID, categoryID).Scan(
		&uc.ID,
		&uc.UserID,
		&uc.CategoryID,
		&uc.CustomName,
		&uc.Icon,
		&uc.Hidden,
		&uc.DeletedAt,
		&uc.CreatedAt,
		&uc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no overrides exists yet
		}
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get user category override", err)
	}

	return &uc, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.UserCategory, error) {
	query := `
		SELECT id, user_id, category_id, custom_name, icon, hidden, deleted_at, created_at, updated_at
		FROM user_categories
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to list user categories", err)
	}
	defer rows.Close()

	var results []domain.UserCategory
	for rows.Next() {
		var uc domain.UserCategory
		if err := rows.Scan(
			&uc.ID,
			&uc.UserID,
			&uc.CategoryID,
			&uc.CustomName,
			&uc.Icon,
			&uc.Hidden,
			&uc.DeletedAt,
			&uc.CreatedAt,
			&uc.UpdatedAt,
		); err != nil {
			return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to scan user categories", err)
		}
		results = append(results, uc)
	}

	return results, nil
}

func (r *Repository) ExistsByName(ctx context.Context, userID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM user_categories
			WHERE user_id = $1 AND custom_name = $2 AND hidden = false
			AND ($3::uuid IS NULL OR id != $3)
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, name, excludeID).Scan(&exists)
	if err != nil {
		return false, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to check category name", err)
	}

	return exists, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	query := `
		SELECT id, name, icon, created_at
		FROM categories
		WHERE name = $1
	`

	var cat domain.Category
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&cat.ID,
		&cat.Name,
		&cat.Icon,
		&cat.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound // no category with the name exists
		}
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get category by name", err)
	}

	return &cat, nil
}

func (r *Repository) DeleteTx(ctx context.Context, tx pgx.Tx, userId, categoryID uuid.UUID) error {
	query := `
		DELETE from user_categories
		WHERE user_id = $1
		AND id = $2
	`
	if _, err := tx.Exec(ctx, query, userId, categoryID); err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to delete user category", err)
	}

	return nil
}