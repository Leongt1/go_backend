package domain

import (
	platformErrors "backend-go/internal/platform/errors"
)

var (
	ErrInvalidInput = platformErrors.NewDomainError(
		platformErrors.CodeInvalidInput,
		"invalid input",
	)
	ErrNoCredits = platformErrors.NewDomainError(
		platformErrors.CodeAINoCredits,
		"no AI credits remaining",
	)
	ErrAINotConfigured = platformErrors.NewDomainError(
		platformErrors.CodeAIUnavailable,
		"AI assistant is not configured on this server",
	)
	ErrAIUnavailable = platformErrors.NewDomainError(
		platformErrors.CodeAIUnavailable,
		"AI assistant is temporarily unavailable, please try again",
	)
)
