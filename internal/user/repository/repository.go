package repository

import (
	"context"
	"errors"
	"fmt"
	"logistics-quality-monitor/internal/database"
	"logistics-quality-monitor/internal/user/model"
	appErrors "logistics-quality-monitor/pkg/errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *database.Database
}

func NewRepository(db *database.Database) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
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

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appErrors.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.DB.WithContext(ctx).First(&user, "id = ?", userID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, appErrors.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()

	result := r.db.DB.WithContext(ctx).Model(&model.User{}).
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

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	result := r.db.DB.WithContext(ctx).Model(&model.User{}).
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

func (r *UserRepository) CreatePasswordResetToken(ctx context.Context, token *model.PasswordResetToken) error {
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.Used = false

	if err := r.db.DB.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}
	return nil
}

func (r *UserRepository) GetPasswordResetToken(ctx context.Context, token string) (*model.PasswordResetToken, error) {
	var resetToken model.PasswordResetToken
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

func (r *UserRepository) MarkTokenAsUsed(ctx context.Context, tokenID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&model.PasswordResetToken{}).
		Where("id = ?", tokenID).
		Update("used", true)

	return result.Error
}
