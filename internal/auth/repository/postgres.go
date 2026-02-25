package repository

import (
	"backend-go/internal/auth/domain"
	backendErrors "backend-go/internal/platform/errors"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *RefreshRepository {
	return &RefreshRepository{db: db}
}

func (r *RefreshRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, revoked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.Token,
		token.ExpiresAt,
		token.Revoked,
		token.CreatedAt,
	)

	if err != nil {
		return backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to create refresh token",
			err,
		)
	}

	return nil
}

func (r *RefreshRepository) GetByToken(ctx context.Context, tokenStr string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token = $1
	`

	var token domain.RefreshToken

	err := r.db.QueryRow(ctx, query, tokenStr).Scan(
		&token.ID,
		&token.UserID,
		&token.Token,
		&token.ExpiresAt,
		&token.Revoked,
		&token.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidRefreshToken
		}
		return nil, backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to get refresh token",
			err,
		)
	}

	return &token, nil
}

func (r *RefreshRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = true
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to revoke refresh token",
			err,
		)
	}

	return nil
}
