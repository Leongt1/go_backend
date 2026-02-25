package domain

import (
	platformErrors "backend-go/internal/platform/errors"
)

var (
	ErrInvalidInput        = platformErrors.NewDomainError(platformErrors.CodeInvalidInput, "Invalid input")
	ErrInvalidCredentials  = platformErrors.NewDomainError(platformErrors.CodeInvalidCredentials, "Invalid credentials")
	ErrInvalidRefreshToken = platformErrors.NewDomainError(platformErrors.CodeInvalidCredentials, "Invalid refresh token")
)
