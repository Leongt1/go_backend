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
		INSERT INTO refresh_tokens (id, user_id, token_hash, family_id, expires_at, revoked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.FamilyID,
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

func (r *Repository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, family_id, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var token domain.RefreshToken

	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.FamilyID,
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

func (r *Repository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = true
		WHERE family_id = $1
	`

	_, err := r.db.Exec(ctx, query, familyID)
	if err != nil {
		return backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to revoke refresh token family",
			err,
		)
	}

	return nil
}

func (r *Repository) DeleteExpiredByUser(ctx context.Context, userID uuid.UUID) error {
	// revoked-but-unexpired rows are kept so replaying them still trips
	// family revocation; expired rows are dead weight either way
	query := `
		DELETE FROM refresh_tokens
		WHERE user_id = $1 AND expires_at <= NOW()
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to prune expired refresh tokens",
			err,
		)
	}

	return nil
}

func (r *Repository) RevokeActiveBeyondCap(ctx context.Context, userID uuid.UUID, keep int) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = true
		WHERE id IN (
			SELECT id FROM refresh_tokens
			WHERE user_id = $1 AND revoked = false AND expires_at > NOW()
			ORDER BY created_at DESC
			OFFSET $2
		)
	`

	_, err := r.db.Exec(ctx, query, userID, keep)
	if err != nil {
		return backendErrors.NewAppError(
			backendErrors.CodeDatabaseError,
			"failed to enforce refresh token cap",
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
			return nil, domain.ErrInvalidPasswordResetToken
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
