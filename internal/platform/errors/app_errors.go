package errors

import "runtime"

type AppError struct {
	Code       string
	Message    string
	Err        error
	StackTrace string
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, err error) *AppError {
	// Capture stack trace (4096 bytes is usually enough for most call stacks)
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)

	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StackTrace: string(buf[:n]),
	}
}

var (
	ErrInternalServer = &AppError{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "something went wrong",
	}

	ErrDatabase = &AppError{
		Code:    "DATABASE_ERROR",
		Message: "database error",
	}

	ErrHashPassword = &AppError{
		Code:    "HASH_PASSWORD_ERROR",
		Message: "failed to hash password",
	}

	ErrComparePassword = &AppError{
		Code:    "COMPARE_PASSWORD_ERROR",
		Message: "failed to compare password",
	}
)
