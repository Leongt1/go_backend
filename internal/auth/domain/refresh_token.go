package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID     uuid.UUID
	UserID uuid.UUID
	// TokenHash is the SHA-256 hex digest of the token; the plaintext is never stored.
	TokenHash string
	// FamilyID groups a login session's rotation chain. A replayed revoked
	// token revokes the entire family (reuse detection).
	FamilyID  uuid.UUID
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

func NewRefreshToken(userID uuid.UUID, tokenHash string, familyID uuid.UUID, ttl time.Duration) *RefreshToken {
	now := time.Now().UTC()

	return &RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
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
