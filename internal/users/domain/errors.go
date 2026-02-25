package domain

import platformErrors "backend-go/internal/platform/errors"

// domain specific errors
var (
	ErrInvalidInput       = platformErrors.NewDomainError(platformErrors.CodeInvalidInput, "Invalid input")
	ErrInvalidRole        = platformErrors.NewDomainError(platformErrors.CodeInvalidRole, "Invalid role")
	ErrInvalidCreds       = platformErrors.NewDomainError(platformErrors.CodeInvalidCredentials, "Invalid credentials")
	ErrInvalidGender      = platformErrors.NewDomainError(platformErrors.CodeInvalidGender, "Invalid gender")
	ErrUserNotFound       = platformErrors.NewDomainError(platformErrors.CodeUserNotFound, "User not found")
	ErrEmailAlreadyExists = platformErrors.NewDomainError(platformErrors.CodeEmailAlreadyExists, "email already exists")
)
