package repository

import (
	platformErrors "backend-go/internal/platform/errors"
	"backend-go/internal/transactions/domain"
	"context"
	"errors"
	"fmt"
	"strings"

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

func (r *Repository) Create(ctx context.Context, tx *domain.Transaction) error {
	query := `
		INSERT INTO transactions 
			(id, user_id, category_id, amount, description, type, date, created_at, updated_at, created_by)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		tx.ID,
		tx.UserID,
		tx.CategoryID,
		tx.Amount,
		tx.Description, // nil → NULL
		tx.Type,
		tx.Date,
		tx.CreatedAt,
		tx.UpdatedAt,
		tx.CreatedBy,
	)
	if err != nil {
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to create transaction",
			err,
		)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	query := `
		SELECT id, user_id, category_id, amount, description, type, date, created_at, updated_at, created_by, updated_by
		FROM transactions
		WHERE id = $1
	`

	var tx domain.Transaction
	err := r.db.QueryRow(ctx, query, id).Scan(
		&tx.ID,
		&tx.UserID,
		&tx.CategoryID,
		&tx.Amount,
		&tx.Description,
		&tx.Type,
		&tx.Date,
		&tx.CreatedAt,
		&tx.UpdatedAt,
		&tx.CreatedBy,
		&tx.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to get transaction",
			err,
		)
	}
	return &tx, nil
}

func (r *Repository) Update(ctx context.Context, tx *domain.Transaction) error {
	query := `
		UPDATE transactions
		SET category_id  = $1,
		    amount       = $2,
		    description  = $3,
		    type         = $4,
		    date         = $5,
		    updated_at   = $6,
		    updated_by   = $7
		WHERE id = $8
	`

	_, err := r.db.Exec(ctx, query,
		tx.CategoryID,
		tx.Amount,
		tx.Description,
		tx.Type,
		tx.Date,
		tx.UpdatedAt,
		tx.UpdatedBy,
		tx.ID,
	)
	if err != nil {
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to update transaction",
			err,
		)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM transactions WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to delete transaction",
			err,
		)
	}
	return nil
}

// buildFilterClause turns the dynamic filter into extra WHERE conditions and
// their args. userID is always $1; shared by List and Count so both stay in sync.
func buildFilterClause(userID uuid.UUID, filter domain.TransactionFilter) (string, []any) {
	// args holds the query parameters, userID is always first
	args := []any{userID}

	// conditions holds the extra WHERE clauses we build dynamically
	conditions := []string{}

	if filter.CategoryID != nil {
		args = append(args, *filter.CategoryID)
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", len(args)))
	}

	if filter.Type != nil {
		args = append(args, *filter.Type)
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
	}

	if filter.DateFrom != nil {
		args = append(args, *filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("date >= $%d", len(args)))
	}

	if filter.DateTo != nil {
		args = append(args, *filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("date < $%d", len(args)))
	}

	clause := ""
	if len(conditions) > 0 {
		clause = " AND " + strings.Join(conditions, " AND ")
	}
	return clause, args
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID, filter domain.TransactionFilter) ([]domain.Transaction, error) {
	base := `
		SELECT id, user_id, category_id, amount, description, type, date, created_at, updated_at, created_by, updated_by
		FROM transactions
		WHERE user_id = $1
	`

	clause, args := buildFilterClause(userID, filter)
	query := base + clause

	// created_at breaks ties so pages are stable when many rows share a date
	query += " ORDER BY date DESC, created_at DESC"

	if filter.Limit != nil {
		args = append(args, *filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to list transactions",
			err,
		)
	}
	defer rows.Close()

	transactions := []domain.Transaction{}
	for rows.Next() {
		var tx domain.Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.CategoryID,
			&tx.Amount,
			&tx.Description,
			&tx.Type,
			&tx.Date,
			&tx.CreatedAt,
			&tx.UpdatedAt,
			&tx.CreatedBy,
			&tx.UpdatedBy,
		); err != nil {
			return nil, platformErrors.NewAppError(
				platformErrors.CodeDatabaseError,
				"failed to scan transaction",
				err,
			)
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to iterate transactions",
			err,
		)
	}

	return transactions, nil
}

func (r *Repository) Count(ctx context.Context, userID uuid.UUID, filter domain.TransactionFilter) (int64, error) {
	clause, args := buildFilterClause(userID, filter)
	query := `SELECT COUNT(*) FROM transactions WHERE user_id = $1` + clause

	var total int64
	if err := r.db.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to count transactions",
			err,
		)
	}
	return total, nil
}

func (r *Repository) ReassignCategoryTx(ctx context.Context, tx pgx.Tx, userID, fromCategoryID, toCategoryID uuid.UUID) error {
	query := `
		UPDATE transactions
		SET category_id = $1
		WHERE user_id = $2 AND category_id = $3
	`
	_, err := tx.Exec(ctx, query, toCategoryID, userID, fromCategoryID)
	if err != nil {
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to reassign category",
			err,
		)
	}

	return nil
}
