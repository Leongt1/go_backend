package domain

import (
	platformErrors "backend-go/internal/platform/errors"
)

var (
	ErrInvalidInput              = platformErrors.NewDomainError(platformErrors.CodeInvalidInput, "Invalid input")
	ErrInvalidCredentials        = platformErrors.NewDomainError(platformErrors.CodeInvalidCredentials, "Invalid credentials")
	ErrInvalidRefreshToken       = platformErrors.NewDomainError(platformErrors.CodeInvalidCredentials, "Invalid refresh token")
	ErrInvalidPasswordResetToken = platformErrors.NewDomainError(platformErrors.CodeInvalidCredentials, "Invalid password reset token")
	ErrPasswordResetFailed       = platformErrors.NewDomainError(platformErrors.CodeFailedToResetPassword, "Failed to reset password")
)
