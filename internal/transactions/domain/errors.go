package domain

import platformErrors "backend-go/internal/platform/errors"

var (
	ErrInvalidInput        = platformErrors.NewDomainError(platformErrors.CodeInvalidInput, "Invalid input")
	ErrTransactionNotFound = platformErrors.NewDomainError(platformErrors.CodeTransactionNotFound, "Transaction not found")
	ErrInvalidAmount       = platformErrors.NewDomainError(platformErrors.CodeInvalidInput, "Amount must be greater than zero")
	ErrFutureDate          = platformErrors.NewDomainError(platformErrors.CodeInvalidInput, "Transaction date cannot be in the future")
	ErrCannotModifyOther   = platformErrors.NewDomainError(platformErrors.CodeForbidden, "Cannot modify another user's transaction")
)
