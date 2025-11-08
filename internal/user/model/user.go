package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	PasswordHashed string    `json:"-"`
	FullName       string    `json:"full_name"`
	PhoneNumber    *string   `json:"phone_number,omitempty"`
	Role           string    `json:"role"`
	Address        *string   `json:"address,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}
