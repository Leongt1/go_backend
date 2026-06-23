package domain

import (
	"context"

	"github.com/google/uuid"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByToken(ctx context.Context, token string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type PasswordResetRepository interface {
	CreatePasswordResetToken(ctx context.Context, token *PasswordResetToken) error
	GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	DeletePasswordResetTokensByUserID(ctx context.Context, userID uuid.UUID) error
	MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error
}
