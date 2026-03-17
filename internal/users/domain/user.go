package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type RoleType string
type GenderType string

const (
	RoleUser     RoleType   = "User"
	RoleAdmin    RoleType   = "Admin"
	GenderMale   GenderType = "Male"
	GenderFemale GenderType = "Female"
)

func ParseRole(s string) (RoleType, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrInvalidRole
	}
	normalized := RoleType(strings.ToUpper(s[:1]) + strings.ToLower(s[1:]))
	switch normalized {
	case RoleUser, RoleAdmin:
		return normalized, nil
	}
	return "", ErrInvalidRole
}

func ParseGender(s string) (GenderType, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrInvalidGender
	}
	normalized := GenderType(strings.ToUpper(s[:1]) + strings.ToLower(s[1:]))
	switch normalized {
	case GenderMale, GenderFemale:
		return normalized, nil
	}
	return "", ErrInvalidGender
}

type User struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         RoleType   `json:"role"`
	Gender       GenderType `json:"gender"`
	DateOfBirth  *time.Time `json:"date_of_birth"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	CreatedBy *uuid.UUID `json:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by"`
}

func NewUser(name, email string, role RoleType, gender GenderType) (*User, error) {
	if role != RoleUser && role != RoleAdmin {
		return nil, ErrInvalidRole
	}

	if gender != GenderMale && gender != GenderFemale {
		return nil, ErrInvalidGender
	}

	return &User{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		Role:      role,
		Gender:    gender,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (u *User) Update(name *string, role *RoleType, gender *GenderType, dateOfBirth *time.Time, updatedBy *uuid.UUID) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return ErrInvalidInput
		}
		u.Name = trimmed
	}
	if role != nil {
		if *role != RoleUser && *role != RoleAdmin {
			return ErrInvalidRole
		}
		u.Role = *role
	}

	if gender != nil {
		if *gender != GenderMale && *gender != GenderFemale {
			return ErrInvalidGender
		}
		u.Gender = *gender
	}

	if dateOfBirth != nil {
		if dateOfBirth.After(time.Now().UTC()) {
			return ErrInvalidInput
		}
		u.DateOfBirth = dateOfBirth
	}

	u.UpdatedAt = time.Now().UTC()
	u.UpdatedBy = updatedBy

	return nil
}
