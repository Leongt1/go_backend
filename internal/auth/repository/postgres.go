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

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, token *domain.RefreshToken) error {
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

func (r *Repository) GetByToken(ctx context.Context, tokenStr string) (*domain.RefreshToken, error) {
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

func (r *Repository) Revoke(ctx context.Context, id uuid.UUID) error {
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

func (r *Repository) CreatePasswordResetToken(
	ctx context.Context,
	token *domain.PasswordResetToken,
) error {
	query := `
		INSERT INTO password_reset_tokens (
			id,
			user_id,
			token_hash,
			expires_at,
			used_at,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.UsedAt,
		token.CreatedAt,
	)

	if err != nil {
		return backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to create password reset token",
			err,
		)
	}

	return nil
}

func (r *Repository) DeletePasswordResetTokensByUserID(
	ctx context.Context,
	userID uuid.UUID,
) error {

	query := `
		DELETE
		FROM password_reset_tokens
		WHERE user_id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)

	if err != nil {
		return backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to delete password reset tokens",
			err,
		)
	}

	return nil
}

func (r *Repository) GetPasswordResetTokenByHash(
	ctx context.Context,
	hash string,
) (*domain.PasswordResetToken, error) {

	query := `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			used_at,
			created_at
		FROM password_reset_tokens
		WHERE
			token_hash = $1
			AND used_at IS NULL
			AND expires_at > NOW()
	`

	var token domain.PasswordResetToken

	err := r.db.QueryRow(ctx, query, hash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidRefreshToken
		}

		return nil, backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to fetch password reset token",
			err,
		)
	}

	return &token, nil
}

func (r *Repository) MarkPasswordResetTokenUsed(
	ctx context.Context,
	id uuid.UUID,
) error {

	query := `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)

	if err != nil {
		return backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to mark password reset token as used",
			err,
		)
	}

	return nil
}
