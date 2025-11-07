package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"logistics-quality-monitor/internal/auth/models"
	"logistics-quality-monitor/internal/database"
	appErrors "logistics-quality-monitor/pkg/errors"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *database.Database
}

func NewRepository(db *database.Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
        INSERT INTO users (id, username, email, password, full_name, phone_number, role, address, is_active, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        RETURNING id, created_at, updated_at
    `

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	err := r.db.DB.QueryRowContext(
		ctx,
		query,
		user.ID,
		user.Username,
		user.Email,
		user.Password,
		user.FullName,
		user.PhoneNumber,
		user.Role,
		user.Address,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" {
			return appErrors.ErrUserAlreadyExists
		}

		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
        SELECT id, username, email, password, full_name, phone_number, role, address, is_active, created_at, updated_at
        FROM users
        WHERE email = $1
    `

	user := &models.User{}
	err := r.db.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.FullName,
		&user.PhoneNumber,
		&user.Role,
		&user.Address,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `
        SELECT id, username, email, password, full_name, phone_number, role, address, is_active, created_at, updated_at
        FROM users
        WHERE id = $1
    `

	user := &models.User{}
	err := r.db.DB.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.FullName,
		&user.PhoneNumber,
		&user.Role,
		&user.Address,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
        UPDATE users
        SET full_name = $1, phone_number = $2, address = $3, updated_at = $4
        WHERE id = $5
    `

	user.UpdatedAt = time.Now()

	result, err := r.db.DB.ExecContext(
		ctx,
		query,
		user.FullName,
		user.PhoneNumber,
		user.Address,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return appErrors.ErrUserNotFound
	}

	return nil
}

func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, password string) error {
	query := `
        UPDATE users
        SET password = $1, updated_at = $2
        WHERE id = $3
    `

	result, err := r.db.DB.ExecContext(ctx, query, password, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return appErrors.ErrUserNotFound
	}

	return nil
}

func (r *Repository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	createTableQuery := `
        CREATE TABLE IF NOT EXISTS password_reset_tokens (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            token VARCHAR(255) UNIQUE NOT NULL,
            expires_at TIMESTAMPTZ NOT NULL,
            used BOOLEAN DEFAULT false,
            created_at TIMESTAMPTZ DEFAULT now()
        )
    `

	_, err := r.db.DB.ExecContext(ctx, createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create password_reset_tokens table: %w", err)
	}

	query := `
        INSERT INTO password_reset_tokens (id, user_id, token, expires_at, used, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at
    `

	token.ID = uuid.New()
	token.CreatedAt = time.Now()
	token.Used = false

	err = r.db.DB.QueryRowContext(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.Token,
		token.ExpiresAt,
		token.Used,
		token.CreatedAt,
	).Scan(&token.ID, &token.CreatedAt)

	return err
}

func (r *Repository) GetPasswordResetToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	query := `
        SELECT id, user_id, token, expires_at, used, created_at
        FROM password_reset_tokens
        WHERE token = $1 AND used = false AND expires_at > NOW()
    `

	resetToken := &models.PasswordResetToken{}
	err := r.db.DB.QueryRowContext(ctx, query, token).Scan(
		&resetToken.ID,
		&resetToken.UserID,
		&resetToken.Token,
		&resetToken.ExpiresAt,
		&resetToken.Used,
		&resetToken.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reset token: %w", err)
	}

	return resetToken, nil
}

func (r *Repository) MarkTokenAsUsed(ctx context.Context, tokenID uuid.UUID) error {
	query := `UPDATE password_reset_tokens SET used = true WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, tokenID)
	return err
}
