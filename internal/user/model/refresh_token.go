package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key:default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Token     string    `gorm:"type:varchar(500);not null;unique;index"`
	ExpiresAt time.Time `gorm:"not null; index"`
	Revoked   bool      `gorm:"default:false;index"`
	RevokedAt time.Time `gorm:"type:timestamp"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

func (rt *RefreshToken) IsActive() bool {
	return !rt.Revoked && !rt.IsExpired()
}
