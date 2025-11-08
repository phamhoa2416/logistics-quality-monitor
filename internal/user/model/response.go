package model

import (
	"time"

	"github.com/google/uuid"
)

type UserResponse struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	FullName       string    `json:"full_name"`
	PhoneNumber    *string   `json:"phone_number"`
	Role           string    `json:"role"`
	DefaultAddress *string   `json:"default_address"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

type AuthResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresAt    int64         `json:"expires_at"`
}

func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		FullName:       u.FullName,
		PhoneNumber:    u.PhoneNumber,
		Role:           u.Role,
		DefaultAddress: u.Address,
		IsActive:       u.IsActive,
		CreatedAt:      u.CreatedAt,
	}
}
