package errors

const (
	// Common
	CodeInvalidInput = "INVALID_INPUT"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"

	// Auth
	CodeInvalidCredentials = "INVALID_CREDENTIALS"

	// User
	CodeUserNotFound       = "USER_NOT_FOUND"
	CodeInvalidRole        = "INVALID_ROLE"
	CodeInvalidGender      = "INVALID_GENDER"
	CodeEmailAlreadyExists = "EMAIL_ALREADY_EXISTS"

	// Transactions
	CodeTransactionNotFound = "TRANSACTION_NOT_FOUND"

	// Infrastructure
	CodeInternalServer = "INTERNAL_SERVER_ERROR"
	CodeDatabaseError  = "DATABASE_ERROR"
)
