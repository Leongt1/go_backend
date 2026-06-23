package domain

import (
	"time"

	"github.com/google/uuid"
)

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Duration
	UsedAt    *time.Time
	CreatedAt time.Time
}

func NewPasswordResetToken(
	userID uuid.UUID,
	tokenHash string,
	expiresAt time.Duration,
) *PasswordResetToken {
	return &PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
}
