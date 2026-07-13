package service

import (
	"backend-go/internal/auth/domain"
	categoryDomain "backend-go/internal/categories/domain"
	"backend-go/internal/platform/security"
	userDomain "backend-go/internal/users/domain"
	userService "backend-go/internal/users/service"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---- fakes ----

type fakeUserRepo struct {
	byEmail map[string]*userDomain.User
	created []*userDomain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]*userDomain.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, user *userDomain.User) error {
	f.created = append(f.created, user)
	f.byEmail[user.Email] = user
	return nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*userDomain.User, error) {
	return nil, userDomain.ErrUserNotFound
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*userDomain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, userDomain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) Update(_ context.Context, _ uuid.UUID, _ *userDomain.User) error { return nil }
func (f *fakeUserRepo) Delete(_ context.Context, _ uuid.UUID) error                     { return nil }
func (f *fakeUserRepo) List(_ context.Context) ([]userDomain.User, error) {
	return []userDomain.User{}, nil
}
func (f *fakeUserRepo) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error { return nil }

type fakeCategoryRepo struct {
	created []*categoryDomain.Category
}

func (f *fakeCategoryRepo) Create(_ context.Context, c *categoryDomain.Category) error {
	f.created = append(f.created, c)
	return nil
}

func (f *fakeCategoryRepo) GetByID(_ context.Context, _ uuid.UUID) (*categoryDomain.Category, error) {
	return nil, categoryDomain.ErrCategoryNotFound
}
func (f *fakeCategoryRepo) GetByName(_ context.Context, _ uuid.UUID, _ string) (*categoryDomain.Category, error) {
	return nil, categoryDomain.ErrCategoryNotFound
}
func (f *fakeCategoryRepo) ListByUser(_ context.Context, _ uuid.UUID) ([]categoryDomain.Category, error) {
	return []categoryDomain.Category{}, nil
}
func (f *fakeCategoryRepo) Update(_ context.Context, _ *categoryDomain.Category) error { return nil }
func (f *fakeCategoryRepo) DeleteTx(_ context.Context, _ pgx.Tx, _, _ uuid.UUID) error { return nil }
func (f *fakeCategoryRepo) ExistsByName(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
	return false, nil
}

// ---- helpers ----

func newAuthService() (*Service, *fakeUserRepo, *fakeCategoryRepo) {
	userRepo := newFakeUserRepo()
	catRepo := &fakeCategoryRepo{}
	users := userService.NewService(userRepo)
	jwt := security.NewJWTManager("test-secret", "finai-test")
	svc := NewService(
		users, jwt, "http://localhost:5173",
		nil, nil, catRepo,
		nil, // email provider unused by signup
		15*time.Minute, 24*time.Hour, time.Hour,
	)
	return svc, userRepo, catRepo
}

func validSignup() *SignupInput {
	return &SignupInput{
		Name:     "New User",
		Email:    "new@example.com",
		Password: "longenough123",
		Gender:   "Female",
	}
}

// ---- tests ----

func TestSignupAlwaysCreatesRoleUser(t *testing.T) {
	svc, userRepo, catRepo := newAuthService()

	if err := svc.Signup(context.Background(), validSignup()); err != nil {
		t.Fatal(err)
	}

	if len(userRepo.created) != 1 {
		t.Fatalf("expected 1 user, got %d", len(userRepo.created))
	}
	// public signup can never grant elevated roles - there is no role input,
	// and the stored role must be User
	if userRepo.created[0].Role != userDomain.RoleUser {
		t.Errorf("signup created role %q, want %q", userRepo.created[0].Role, userDomain.RoleUser)
	}
	// default categories are seeded at signup
	if len(catRepo.created) == 0 {
		t.Errorf("signup must seed default categories")
	}
}

func TestSignupValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*SignupInput)
		wantErr error
	}{
		{
			name:    "short password rejected",
			mutate:  func(in *SignupInput) { in.Password = "short" },
			wantErr: domain.ErrWeakPassword,
		},
		{
			name:    "blank name rejected",
			mutate:  func(in *SignupInput) { in.Name = "  " },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "blank email rejected",
			mutate:  func(in *SignupInput) { in.Email = "" },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "invalid gender rejected",
			mutate:  func(in *SignupInput) { in.Gender = "Robot" },
			wantErr: userDomain.ErrInvalidGender,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, userRepo, _ := newAuthService()
			input := validSignup()
			tc.mutate(input)

			err := svc.Signup(context.Background(), input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if len(userRepo.created) != 0 {
				t.Errorf("no user should have been created")
			}
		})
	}
}

func TestSignupNormalizesEmail(t *testing.T) {
	svc, userRepo, _ := newAuthService()
	input := validSignup()
	input.Email = "  MiXeD@Example.COM "

	if err := svc.Signup(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if got := userRepo.created[0].Email; got != "mixed@example.com" {
		t.Errorf("email not normalized: %q", got)
	}
}
