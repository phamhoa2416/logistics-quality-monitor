package errors

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidCredentials      = errors.New("invalid email or password")
	ErrInvalidToken            = errors.New("invalid or expired token")
	ErrUnauthorized            = errors.New("unauthorized access")
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserInactive      = errors.New("user account is inactive")
	ErrInvalidUserRole   = errors.New("invalid user role")

	ErrInvalidInput     = errors.New("invalid input data")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrWeakPassword     = errors.New("password does not meet requirements")
	ErrPasswordMismatch = errors.New("passwords do not match")

	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenInvalid   = errors.New("token is invalid")
	ErrResetTokenUsed = errors.New("reset token has already been used")
)

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}

	return e.Message
}

func NewAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
