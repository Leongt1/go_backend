package domain

import platformErrors "backend-go/internal/platform/errors"

var (
	ErrInvalidInput      = platformErrors.NewDomainError(platformErrors.CodeInvalidInput, "Invalid input")
	ErrCategoryNotFound  = platformErrors.NewDomainError(platformErrors.CodeCategoryNotFound, "Category not found")
	ErrCategoryHidden    = platformErrors.NewDomainError(platformErrors.CodeCategoryHidden, "Category is hidden")
	ErrDuplicateName     = platformErrors.NewDomainError(platformErrors.CodeDuplicateCategoryName, "Category name already exists")
	ErrCannotModifyOther = platformErrors.NewDomainError(platformErrors.CodeForbidden, "Cannot modify another user's category")
	ErrCannotModifyUncategorised = platformErrors.NewDomainError(platformErrors.CodeForbidden, "Uncategorised category cannot be modified deleted")
	ErrCannotDeleteSystemCategory = platformErrors.NewDomainError(platformErrors.CodeForbidden, "Cannot delete a system category")
)
