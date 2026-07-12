package repository

import (
	"backend-go/internal/ai/domain"
	platformErrors "backend-go/internal/platform/errors"
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

func (r *Repository) GetCredits(ctx context.Context, userID uuid.UUID) (int, error) {
	var credits int
	err := r.db.QueryRow(ctx,
		`SELECT ai_credits FROM users WHERE id = $1`, userID,
	).Scan(&credits)
	if err != nil {
		return 0, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to get AI credits",
			err,
		)
	}
	return credits, nil
}

func (r *Repository) ConsumeCredit(ctx context.Context, userID uuid.UUID) (int, error) {
	// the WHERE guard makes the spend atomic: two concurrent prompts can never
	// take the balance below zero
	var remaining int
	err := r.db.QueryRow(ctx,
		`UPDATE users SET ai_credits = ai_credits - 1
		 WHERE id = $1 AND ai_credits > 0
		 RETURNING ai_credits`, userID,
	).Scan(&remaining)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrNoCredits
		}
		return 0, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to consume AI credit",
			err,
		)
	}
	return remaining, nil
}

func (r *Repository) RefundCredit(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET ai_credits = ai_credits + 1 WHERE id = $1`, userID,
	)
	if err != nil {
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to refund AI credit",
			err,
		)
	}
	return nil
}
