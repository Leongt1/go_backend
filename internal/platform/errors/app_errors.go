package errors

type AppError struct {
	Code    string
	Message string
	Err     error
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
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
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
