package user

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserInactive      = errors.New("user account is inactive")
	ErrInvalidUserRole   = errors.New("invalid user role")

	ErrTokenInvalid   = errors.New("token is invalid")
	ErrTokenExpired   = errors.New("token has expired")
	ErrResetTokenUsed = errors.New("reset token has already been used")
)
