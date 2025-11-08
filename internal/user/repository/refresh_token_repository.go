package repository

import (
	"context"
	"errors"
	"fmt"
	"logistics-quality-monitor/internal/database"
	"logistics-quality-monitor/internal/user/model"
	appErrors "logistics-quality-monitor/pkg/errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *database.Database
}

func NewRefreshTokenRepository(db *database.Database) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()
	token.Revoked = false

	if err := r.db.DB.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) GetRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	var refreshToken model.RefreshToken
	err := r.db.DB.WithContext(ctx).
		Where("token = ? AND revoked = false AND expires_at > NOW()", token).
		First(&refreshToken).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appErrors.ErrTokenInvalid
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return &refreshToken, nil
}

func (r *RefreshTokenRepository) RevokeToken(ctx context.Context, tokenID uuid.UUID) error {
	now := time.Now()
	result := r.db.DB.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ? AND revoked = false", tokenID).
		Updates(map[string]interface{}{
			"revoked":    true,
			"revoked_at": now,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to revoke token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.ErrTokenInvalid
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	result := r.db.DB.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked = false", userID).
		Updates(map[string]interface{}{
			"revoked":    true,
			"revoked_at": now,
			"updated_at": now,
		})

	return result.Error
}

func (r *RefreshTokenRepository) DeleteExpiredTokens(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	result := r.db.DB.WithContext(ctx).
		Where("expires_at < ? OR (revoked = true AND revoked_at < ?)", cutoffTime, cutoffTime).
		Delete(&model.RefreshToken{})

	return result.Error
}

func (r *RefreshTokenRepository) GetUserTokens(ctx context.Context, userID uuid.UUID) ([]model.RefreshToken, error) {
	var tokens []model.RefreshToken
	err := r.db.DB.WithContext(ctx).
		Where("user_id = ? AND revoked = false AND expires_at > NOW()", userID).
		Order("created_at DESC").
		Find(&tokens).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens: %w", err)
	}
	return tokens, nil
}
