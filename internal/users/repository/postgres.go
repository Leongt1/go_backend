package repository

import (
	platformErrors "backend-go/internal/platform/errors"
	"backend-go/internal/users/domain"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, role, gender, date_of_birth, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Gender,
		user.DateOfBirth,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // Key value violates unique constraint
				return domain.ErrEmailAlreadyExists
			}
		}
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to create user",
			err,
		)
	}
	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, gender, date_of_birth, created_at, updated_at, created_by, updated_by
		FROM users
		WHERE email = $1
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Gender,
		&user.DateOfBirth,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.CreatedBy,
		&user.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to get user by email",
			err,
		)
	}
	return &user, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, gender, date_of_birth, created_at, updated_at, created_by, updated_by
		FROM users
		WHERE id = $1
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Gender,
		&user.DateOfBirth,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.CreatedBy,
		&user.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to get user by id",
			err,
		)
	}
	return &user, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, user *domain.User) error {
	query := `
		UPDATE users
		SET name = $1, role = $2, gender = $3, date_of_birth = $4, updated_at = $5, updated_by = $6
		WHERE id = $7
	`
	_, err := r.db.Exec(ctx, query,
		user.Name,
		user.Role,
		user.Gender,
		user.DateOfBirth,
		user.UpdatedAt,
		user.UpdatedBy,
		id,
	)
	if err != nil {
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to update user",
			err,
		)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to delete user",
			err,
		)
	}
	return nil
}

func (r *Repository) List(ctx context.Context) ([]domain.User, error) {
	query := `
		SELECT id, name, email, role, gender, date_of_birth, created_at, updated_at, created_by, updated_by
		FROM users
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, platformErrors.NewAppError(
			platformErrors.CodeDatabaseError,
			"failed to list users",
			err,
		)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Role,
			&user.Gender,
			&user.DateOfBirth,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.CreatedBy,
			&user.UpdatedBy,
		)
		if err != nil {
			return nil, platformErrors.NewAppError(
				platformErrors.CodeDatabaseError,
				"failed to scan user",
				err,
			)
		}
		users = append(users, user)
	}
	return users, nil
}
