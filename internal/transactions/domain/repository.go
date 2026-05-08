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
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	List(ctx context.Context, userID uuid.UUID, filter TransactionFilter) ([]Transaction, error)
	Update(ctx context.Context, tx *Transaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	ReassignCategoryTx(ctx context.Context, tx pgx.Tx, userID, fromCategoryID, toCategoryID uuid.UUID) error
}
