package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user entity in the domain
type User struct {
	ID             uuid.UUID
	Username       string
	Email          string
	PasswordHashed string
	FullName       string
	PhoneNumber    *string
	Role           string
	Address        *string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PasswordResetToken represents a password reset token entity
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

// RefreshToken represents a refresh token entity
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	Revoked   bool
	RevokedAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsExpired checks if the refresh token is expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsActive checks if the refresh token is active (not revoked and not expired)
func (rt *RefreshToken) IsActive() bool {
	return !rt.Revoked && !rt.IsExpired()
}
