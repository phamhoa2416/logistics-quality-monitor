package repository

import (
	"context"
	"errors"
	"fmt"
	"logistics-quality-monitor/internal/auth/models"
	"logistics-quality-monitor/internal/database"
	appErrors "logistics-quality-monitor/pkg/errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *database.Database
}

func NewRepository(db *database.Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	if err := r.db.DB.WithContext(ctx).Create(user).Error; err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "duplicate key") && strings.Contains(errStr, "email") {
			return appErrors.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appErrors.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.DB.WithContext(ctx).First(&user, "id = ?", userID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appErrors.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	result := r.db.DB.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"full_name":    user.FullName,
			"phone_number": user.PhoneNumber,
			"address":      user.Address,
			"updated_at":   user.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return appErrors.ErrUserNotFound
	}

	return nil
}

func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	result := r.db.DB.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password_hashed": passwordHash,
			"updated_at":      time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update password: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return appErrors.ErrUserNotFound
	}
	return nil
}

func (r *Repository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.Used = false

	if err := r.db.DB.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}
	return nil
}

func (r *Repository) GetPasswordResetToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	var resetToken models.PasswordResetToken
	err := r.db.DB.WithContext(ctx).
		Where("token = ? AND used = false AND expires_at > NOW()", token).
		First(&resetToken).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appErrors.ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reset token: %w", err)
	}

	return &resetToken, nil
}

func (r *Repository) MarkTokenAsUsed(ctx context.Context, tokenID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.PasswordResetToken{}).
		Where("id = ?", tokenID).
		Update("used", true)

	return result.Error
}
