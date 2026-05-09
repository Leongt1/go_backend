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

func (r *Repository) Create(ctx context.Context, uc *domain.Category) error {
	query := `
		INSERT INTO user_categories 
			(id, user_id, name, icon, hidden, created_at, updated_at)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		uc.ID,
		uc.UserID,
		uc.Name,
		uc.Icon,
		uc.Hidden,
		uc.CreatedAt,
		uc.UpdatedAt,
	)
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to create user category", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	query := `
		SELECT id, user_id, name, icon, hidden, created_at, updated_at
		FROM user_categories
		WHERE id = $1
	`

	var c domain.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, 
		&c.UserID,
		&c.Name, 
		&c.Icon, 
		&c.Hidden,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get category", err)
	}
	return &c, nil
}

func (r *Repository) GetByName(ctx context.Context, userId uuid.UUID, name string) (*domain.Category, error) {
	query := `
		SELECT id, user_id, name, icon, hidden, created_at, updated_at
		FROM user_categories
		WHERE user_id = $1 AND name = $2
	`

	var cat domain.Category
	err := r.pool.QueryRow(ctx, query, userId, name).Scan(
		&cat.ID,
		&cat.UserID,
		&cat.Name,
		&cat.Icon,
		&cat.Hidden,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound // no category with the name exists
		}
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to get category by name", err)
	}

	return &cat, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Category, error) {
	query := `
		SELECT id, user_id, name, icon, hidden, created_at, updated_at
		FROM user_categories
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to list user categories", err)
	}
	defer rows.Close()

	var results []domain.Category
	for rows.Next() {
		var uc domain.Category
		if err := rows.Scan(
			&uc.ID,
			&uc.UserID,
			&uc.Name,
			&uc.Icon,
			&uc.Hidden,
			&uc.CreatedAt,
			&uc.UpdatedAt,
		); err != nil {
			return nil, platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to scan user categories", err)
		}
		results = append(results, uc)
	}

	return results, nil
}

func (r *Repository) Update(ctx context.Context, uc *domain.Category) error {
	query := `
		UPDATE user_categories
		SET name = $1,
			icon = $2,
			hidden = $3,
			updated_at = $4
		WHERE id = $5
	`

	_, err := r.pool.Exec(ctx, query,
		uc.Name,
		uc.Icon,
		uc.Hidden,
		uc.UpdatedAt,
		uc.ID,
	)
	if err != nil {
		return platformErrors.NewAppError(platformErrors.CodeDatabaseError, "Failed to update user category", err)
	}
	return nil
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

func (r *Repository) ExistsByName(ctx context.Context, userID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM user_categories
			WHERE user_id = $1 AND name = $2 AND hidden = false
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
