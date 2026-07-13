package domain

import (
	"context"

	"github.com/google/uuid"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	// RevokeFamily revokes every token in a rotation family (reuse detection).
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error
	// DeleteExpiredByUser prunes rows past their expiry (revoked rows are kept
	// until expiry so replays can still be detected).
	DeleteExpiredByUser(ctx context.Context, userID uuid.UUID) error
	// RevokeActiveBeyondCap keeps the newest `keep` active tokens for the user
	// and revokes the rest (session cap).
	RevokeActiveBeyondCap(ctx context.Context, userID uuid.UUID, keep int) error
}

type PasswordResetRepository interface {
	CreatePasswordResetToken(ctx context.Context, token *PasswordResetToken) error
	GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	DeletePasswordResetTokensByUserID(ctx context.Context, userID uuid.UUID) error
	MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error
}
