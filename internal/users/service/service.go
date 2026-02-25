package service

import (
	platformErrors "backend-go/internal/platform/errors"
	"backend-go/internal/users/domain"
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo domain.UserRepository // interface
}

func NewService(repo domain.UserRepository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	Name        string
	Email       string
	Password    string
	Role        domain.RoleType
	Gender      domain.GenderType
	DateOfBirth *time.Time
}

func (s *Service) CreateUser(ctx context.Context, req *CreateInput) (*domain.User, error) {
	// Validation
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	email := strings.ToLower(strings.TrimSpace(req.Email)) // Normalize email
	if email == "" {
		return nil, domain.ErrInvalidInput
	}
	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	password := req.Password
	if password == "" {
		return nil, domain.ErrInvalidInput
	}

	role := domain.RoleType(strings.ToTitle(strings.ToLower(strings.TrimSpace(string(req.Role)))))
	if role == "" {
		role = domain.RoleUser
	} else if role != domain.RoleAdmin && role != domain.RoleUser {
		return nil, domain.ErrInvalidRole
	}

	gender := domain.GenderType(strings.ToTitle(strings.ToLower(strings.TrimSpace(string(req.Gender)))))
	if gender == "" {
		return nil, domain.ErrInvalidInput
	} else if gender != domain.GenderMale && gender != domain.GenderFemale {
		return nil, domain.ErrInvalidGender
	}

	dateOfBirth := req.DateOfBirth
	if dateOfBirth != nil {
		if dateOfBirth.After(time.Now().UTC()) {
			return nil, domain.ErrInvalidInput
		}
	}

	// Generating password hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, platformErrors.ErrHashPassword
	}

	// Creating user
	user, err := domain.NewUser(name, email, role, gender)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = string(hash)
	user.DateOfBirth = dateOfBirth

	// Saving user to db
	err = s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) AuthenticateUser(ctx context.Context, emailRaw string, passwordRaw string) (*domain.User, error) {
	email := strings.ToLower(strings.TrimSpace(emailRaw)) // Normalize email
	password := passwordRaw

	if email == "" || password == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Comparing password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCreds
	}

	return user, nil
}

type UpdateInput struct {
	Name        *string
	Role        *domain.RoleType
	Gender      *domain.GenderType
	DateOfBirth *time.Time
	UpdatedBy   *uuid.UUID
}

func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, req *UpdateInput) (*domain.User, error) {

	// check if user exists
	existingUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Updating user using domain method
	err = existingUser.Update(req.Name, req.Role, req.Gender, req.DateOfBirth, req.UpdatedBy)
	if err != nil {
		return nil, err
	}

	// Saving user to db
	err = s.repo.Update(ctx, id, existingUser)
	if err != nil {
		return nil, err
	}

	return existingUser, nil // Getting updated user details
}

func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.ErrInvalidInput
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) GetByEmail(ctx context.Context, emailRaw string) (*domain.User, error) {
	email := strings.ToLower(strings.TrimSpace(emailRaw))
	if email == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]domain.User, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}
