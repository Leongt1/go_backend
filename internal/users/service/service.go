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

func (s *Service) CreateUser(ctx context.Context, req *CreateInput) error {
	// Validation
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.ErrInvalidInput
	}

	email := strings.ToLower(strings.TrimSpace(req.Email)) // Normalize email
	if email == "" {
		return domain.ErrInvalidInput
	}

	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return domain.ErrEmailAlreadyExists
	}

	password := req.Password
	if password == "" {
		return domain.ErrInvalidInput
	}

	// trust the caller to pass valid role/gender — already validated via ParseRole/ParseGender
	if req.Role != domain.RoleUser && req.Role != domain.RoleAdmin {
		return domain.ErrInvalidRole
	}

	if req.Gender != domain.GenderMale && req.Gender != domain.GenderFemale {
		return domain.ErrInvalidGender
	}

	dateOfBirth := req.DateOfBirth
	if dateOfBirth != nil {
		if dateOfBirth.After(time.Now().UTC()) {
			return domain.ErrInvalidInput
		}
	}

	// Generating password hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return platformErrors.ErrHashPassword
	}

	// Creating user
	user, err := domain.NewUser(name, email, req.Role, req.Gender)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	user.DateOfBirth = dateOfBirth

	// Saving user to db
	err = s.repo.Create(ctx, user)
	if err != nil {
		return err
	}
	return nil
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
	Role        *string
	Gender      *string
	DateOfBirth *time.Time
	UpdatedBy   *uuid.UUID
}

func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, req *UpdateInput) (*domain.User, error) {
	// check if user exists
	existingUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var parsedRole *domain.RoleType
	if req.Role != nil {
		role, err := domain.ParseRole(*req.Role)
		if err != nil {
			return nil, err
		}
		parsedRole = &role
	}
	var parsedGender *domain.GenderType
	if req.Gender != nil {
		gender, err := domain.ParseGender(*req.Gender)
		if err != nil {
			return nil, err
		}
		parsedGender = &gender
	}

	// Updating user using domain method
	err = existingUser.Update(req.Name, parsedRole, parsedGender, req.DateOfBirth, req.UpdatedBy)
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
