package postgres

import (
	"context"
	"errors"
	"fmt"
	"logistics-quality-monitor/server/domain/user"
	"logistics-quality-monitor/server/infrastructure/database/postgres/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository implements domain.User.Repository interface
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *DB) user.Repository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	u.IsActive = true

	dbModel := toUserModel(u)
	if err := r.db.DB.WithContext(ctx).Create(dbModel).Error; err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "duplicate key") && strings.Contains(errStr, "email") {
			return user.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Update domain entity with generated ID
	u.ID = dbModel.ID
	u.CreatedAt = dbModel.CreatedAt
	u.UpdatedAt = dbModel.UpdatedAt

	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var dbModel models.UserModel
	err := r.db.DB.WithContext(ctx).Where("email = ?", email).First(&dbModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return toUserEntity(&dbModel), nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	var dbModel models.UserModel
	err := r.db.DB.WithContext(ctx).First(&dbModel, "id = ?", userID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return toUserEntity(&dbModel), nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]*user.User, error) {
	var dbModels []models.UserModel
	err := r.db.DB.WithContext(ctx).Find(&dbModels).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	users := make([]*user.User, len(dbModels))
	for i, dbModel := range dbModels {
		users[i] = toUserEntity(&dbModel)
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	u.UpdatedAt = time.Now()

	result := r.db.DB.WithContext(ctx).Model(&models.UserModel{}).
		Where("id = ?", u.ID).
		Updates(map[string]interface{}{
			"full_name":    u.FullName,
			"phone_number": u.PhoneNumber,
			"address":      u.Address,
			"updated_at":   u.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	result := r.db.DB.WithContext(ctx).Model(&models.UserModel{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password_hashed": passwordHash,
			"updated_at":      time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update password: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).Delete(&models.UserModel{}, "id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) CreatePasswordResetToken(ctx context.Context, token *user.PasswordResetToken) error {
	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.Used = false

	dbModel := toPasswordResetTokenModel(token)
	if err := r.db.DB.WithContext(ctx).Create(dbModel).Error; err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	token.ID = dbModel.ID
	token.CreatedAt = dbModel.CreatedAt

	return nil
}

func (r *UserRepository) GetPasswordResetToken(ctx context.Context, token string) (*user.PasswordResetToken, error) {
	var dbModel models.PasswordResetTokenModel
	err := r.db.DB.WithContext(ctx).
		Where("token = ? AND used = false AND expires_at > NOW()", token).
		First(&dbModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, user.ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reset token: %w", err)
	}

	return toPasswordResetTokenEntity(&dbModel), nil
}

func (r *UserRepository) MarkTokenAsUsed(ctx context.Context, tokenID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Model(&models.PasswordResetTokenModel{}).
		Where("id = ?", tokenID).
		Update("used", true)

	return result.Error
}

// Helper functions to convert between domain entities and database models

func toUserModel(u *user.User) *models.UserModel {
	return &models.UserModel{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		PasswordHashed: u.PasswordHashed,
		FullName:       u.FullName,
		PhoneNumber:    u.PhoneNumber,
		Role:           u.Role,
		Address:        u.Address,
		IsActive:       u.IsActive,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func toUserEntity(m *models.UserModel) *user.User {
	return &user.User{
		ID:             m.ID,
		Username:       m.Username,
		Email:          m.Email,
		PasswordHashed: m.PasswordHashed,
		FullName:       m.FullName,
		PhoneNumber:    m.PhoneNumber,
		Role:           m.Role,
		Address:        m.Address,
		IsActive:       m.IsActive,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func toPasswordResetTokenModel(t *user.PasswordResetToken) *models.PasswordResetTokenModel {
	return &models.PasswordResetTokenModel{
		ID:        t.ID,
		UserID:    t.UserID,
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
		Used:      t.Used,
		CreatedAt: t.CreatedAt,
	}
}

func toPasswordResetTokenEntity(m *models.PasswordResetTokenModel) *user.PasswordResetToken {
	return &user.PasswordResetToken{
		ID:        m.ID,
		UserID:    m.UserID,
		Token:     m.Token,
		ExpiresAt: m.ExpiresAt,
		Used:      m.Used,
		CreatedAt: m.CreatedAt,
	}
}
