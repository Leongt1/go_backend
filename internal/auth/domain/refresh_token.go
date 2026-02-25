package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

func NewRefreshToken(userID uuid.UUID, token string, ttl time.Duration) *RefreshToken {
	now := time.Now().UTC()

	return &RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: now.Add(ttl),
		Revoked:   false,
		CreatedAt: now,
	}
}

func (r *RefreshToken) IsExpired() bool {
	return time.Now().UTC().After(r.ExpiresAt)
}

func (r *RefreshToken) Revoke() {
	r.Revoked = true
}
