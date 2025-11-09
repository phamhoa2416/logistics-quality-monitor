package postgres

import (
	"context"
	"errors"
	"fmt"
	"logistics-quality-monitor/server/domain/user"
	"logistics-quality-monitor/server/infrastructure/database/postgres/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshTokenRepository implements domain.User.RefreshTokenRepository interface
type RefreshTokenRepository struct {
	db *DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *DB) user.RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *user.RefreshToken) error {
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()
	token.Revoked = false

	dbModel := toRefreshTokenModel(token)
	if err := r.db.DB.WithContext(ctx).Create(dbModel).Error; err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	token.ID = dbModel.ID
	token.CreatedAt = dbModel.CreatedAt
	token.UpdatedAt = dbModel.UpdatedAt

	return nil
}

func (r *RefreshTokenRepository) GetByToken(ctx context.Context, token string) (*user.RefreshToken, error) {
	var dbModel models.RefreshTokenModel
	err := r.db.DB.WithContext(ctx).
		Where("token = ? AND revoked = false AND expires_at > NOW()", token).
		First(&dbModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return toRefreshTokenEntity(&dbModel), nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenID uuid.UUID) error {
	now := time.Now()
	result := r.db.DB.WithContext(ctx).
		Model(&models.RefreshTokenModel{}).
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
		return user.ErrTokenInvalid
	}

	return nil
}

func (r *RefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	result := r.db.DB.WithContext(ctx).
		Model(&models.RefreshTokenModel{}).
		Where("user_id = ? AND revoked = false", userID).
		Updates(map[string]interface{}{
			"revoked":    true,
			"revoked_at": now,
			"updated_at": now,
		})

	return result.Error
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	result := r.db.DB.WithContext(ctx).
		Where("expires_at < ? OR (revoked = true AND revoked_at < ?)", cutoffTime, cutoffTime).
		Delete(&models.RefreshTokenModel{})

	return result.Error
}

func (r *RefreshTokenRepository) GetUserTokens(ctx context.Context, userID uuid.UUID) ([]*user.RefreshToken, error) {
	var dbModels []models.RefreshTokenModel
	err := r.db.DB.WithContext(ctx).
		Where("user_id = ? AND revoked = false AND expires_at > NOW()", userID).
		Order("created_at DESC").
		Find(&dbModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens: %w", err)
	}

	tokens := make([]*user.RefreshToken, len(dbModels))
	for i, dbModel := range dbModels {
		tokens[i] = toRefreshTokenEntity(&dbModel)
	}

	return tokens, nil
}

// Helper functions to convert between domain entities and database models

func toRefreshTokenModel(t *user.RefreshToken) *models.RefreshTokenModel {
	var revokedAt *time.Time
	if !t.RevokedAt.IsZero() {
		revokedAt = &t.RevokedAt
	}

	return &models.RefreshTokenModel{
		ID:        t.ID,
		UserID:    t.UserID,
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
		Revoked:   t.Revoked,
		RevokedAt: revokedAt,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func toRefreshTokenEntity(m *models.RefreshTokenModel) *user.RefreshToken {
	var revokedAt time.Time
	if m.RevokedAt != nil {
		revokedAt = *m.RevokedAt
	}

	return &user.RefreshToken{
		ID:        m.ID,
		UserID:    m.UserID,
		Token:     m.Token,
		ExpiresAt: m.ExpiresAt,
		Revoked:   m.Revoked,
		RevokedAt: revokedAt,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
