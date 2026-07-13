package service

import (
	"backend-go/internal/users/domain"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepo struct {
	byEmail map[string]*domain.User
	byID    map[uuid.UUID]*domain.User
	created []*domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byEmail: map[string]*domain.User{},
		byID:    map[uuid.UUID]*domain.User{},
	}
}

func (f *fakeUserRepo) Create(_ context.Context, user *domain.User) error {
	f.created = append(f.created, user)
	f.byEmail[user.Email] = user
	f.byID[user.ID] = user
	return nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.User) error { return nil }
func (f *fakeUserRepo) Delete(_ context.Context, _ uuid.UUID) error                 { return nil }
func (f *fakeUserRepo) List(_ context.Context) ([]domain.User, error)               { return []domain.User{}, nil }
func (f *fakeUserRepo) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func validInput() *CreateInput {
	return &CreateInput{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
		Role:     domain.RoleUser,
		Gender:   domain.GenderMale,
	}
}

func TestCreateUser(t *testing.T) {
	future := time.Now().UTC().Add(48 * time.Hour)

	tests := []struct {
		name    string
		mutate  func(*CreateInput)
		seed    func(*fakeUserRepo)
		wantErr error
	}{
		{name: "valid user", mutate: func(in *CreateInput) {}},
		{
			name:    "blank name rejected",
			mutate:  func(in *CreateInput) { in.Name = "   " },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "blank password rejected",
			mutate:  func(in *CreateInput) { in.Password = "" },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:    "invalid role rejected",
			mutate:  func(in *CreateInput) { in.Role = "SuperAdmin" },
			wantErr: domain.ErrInvalidRole,
		},
		{
			name:    "invalid gender rejected",
			mutate:  func(in *CreateInput) { in.Gender = "Other" },
			wantErr: domain.ErrInvalidGender,
		},
		{
			name:    "future date of birth rejected",
			mutate:  func(in *CreateInput) { in.DateOfBirth = &future },
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:   "duplicate email rejected",
			mutate: func(in *CreateInput) {},
			seed: func(repo *fakeUserRepo) {
				repo.byEmail["test@example.com"] = &domain.User{Email: "test@example.com"}
			},
			wantErr: domain.ErrEmailAlreadyExists,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeUserRepo()
			if tc.seed != nil {
				tc.seed(repo)
			}
			svc := NewService(repo)
			input := validInput()
			tc.mutate(input)

			err := svc.CreateUser(context.Background(), input)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				if len(repo.created) != 1 {
					t.Fatalf("expected user to be persisted")
				}
				// password must be stored hashed, never plaintext
				stored := repo.created[0].PasswordHash
				if stored == input.Password {
					t.Errorf("password stored in plaintext")
				}
				if bcrypt.CompareHashAndPassword([]byte(stored), []byte("password123")) != nil {
					t.Errorf("stored hash does not verify against the password")
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestAuthenticateUser(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo)
	if err := svc.CreateUser(context.Background(), validInput()); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.AuthenticateUser(context.Background(), "TEST@Example.com ", "password123"); err != nil {
		t.Fatalf("valid credentials (unnormalized email) should authenticate, got %v", err)
	}
	if _, err := svc.AuthenticateUser(context.Background(), "test@example.com", "wrong-password"); !errors.Is(err, domain.ErrInvalidCreds) {
		t.Fatalf("wrong password: want ErrInvalidCreds, got %v", err)
	}
	if _, err := svc.AuthenticateUser(context.Background(), "nobody@example.com", "password123"); err == nil {
		t.Fatalf("unknown email must not authenticate")
	}
}
