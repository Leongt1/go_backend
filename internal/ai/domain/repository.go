package domain

import (
	"context"

	"github.com/google/uuid"
)

// CreditRepository manages the per-user AI credit balance (users.ai_credits).
type CreditRepository interface {
	GetCredits(ctx context.Context, userID uuid.UUID) (int, error)
	// ConsumeCredit atomically spends one credit and returns the new balance;
	// returns ErrNoCredits when the balance is already zero.
	ConsumeCredit(ctx context.Context, userID uuid.UUID) (int, error)
	// RefundCredit returns a credit when the provider call failed after the
	// credit was consumed.
	RefundCredit(ctx context.Context, userID uuid.UUID) error
}
