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
