package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TransactionFilter struct {
	CategoryID *uuid.UUID
	Type       *TransactionType
	DateFrom   *time.Time
	DateTo     *time.Time
	// Limit/Offset paginate the result set; a nil Limit means "return everything"
	// (the pre-pagination behavior, kept for backward compatibility).
	Limit  *int
	Offset int
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	List(ctx context.Context, userID uuid.UUID, filter TransactionFilter) ([]Transaction, error)
	Count(ctx context.Context, userID uuid.UUID, filter TransactionFilter) (int64, error)
	Update(ctx context.Context, tx *Transaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	ReassignCategoryTx(ctx context.Context, tx pgx.Tx, userID, fromCategoryID, toCategoryID uuid.UUID) error
}
