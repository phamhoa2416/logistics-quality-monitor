package models

import (
	"time"

	"github.com/google/uuid"
)

// UserModel represents the database model for User
type UserModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username       string    `gorm:"type:varchar(100);not null;uniqueIndex"`
	Email          string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	PasswordHashed string    `gorm:"type:varchar(255);not null"`
	FullName       string    `gorm:"type:varchar(255);not null"`
	PhoneNumber    *string   `gorm:"type:varchar(20)"`
	Role           string    `gorm:"type:varchar(50);not null;default:'user'"`
	Address        *string   `gorm:"type:text"`
	IsActive       bool      `gorm:"default:true;not null"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (UserModel) TableName() string {
	return "users"
}

// PasswordResetTokenModel represents the database model for PasswordResetToken
type PasswordResetTokenModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Token     string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null;index"`
	Used      bool      `gorm:"default:false;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (PasswordResetTokenModel) TableName() string {
	return "password_reset_tokens"
}

// RefreshTokenModel represents the database model for RefreshToken
type RefreshTokenModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Token     string     `gorm:"type:varchar(500);not null;unique;index"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	Revoked   bool       `gorm:"default:false;index"`
	RevokedAt *time.Time `gorm:"type:timestamp"`
	CreatedAt time.Time  `gorm:"not null"`
	UpdatedAt time.Time  `gorm:"not null"`
}

func (RefreshTokenModel) TableName() string {
	return "refresh_tokens"
}
