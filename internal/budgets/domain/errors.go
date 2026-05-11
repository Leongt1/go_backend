package domain

import platformErrors "backend-go/internal/platform/errors"

var (
	ErrInvalidInput      = platformErrors.NewDomainError(platformErrors.CodeInvalidInput, "Invalid input")
	ErrBudgetNotFound    = platformErrors.NewDomainError(platformErrors.CodeBudgetNotFound, "Budget not found")
	ErrDuplicateName     = platformErrors.NewDomainError(platformErrors.CodeDuplicateBudgetName, "Budget name already exists")
	ErrCannotModifyOther = platformErrors.NewDomainError(platformErrors.CodeForbidden, "Cannot modify another user's Budget")
)
