package service

import (
	"context"
	"errors"
	"fmt"
	"logistics-quality-monitor/internal/config"
	"logistics-quality-monitor/internal/logger"
	"logistics-quality-monitor/internal/user/model"
	"logistics-quality-monitor/internal/user/repository"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserService struct {
	repo   *repository.UserRepository
	config *config.Config
}

func NewService(repo *repository.UserRepository, cfg *config.Config) *UserService {
	return &UserService{
		repo:   repo,
		config: cfg,
	}
}

func (s *UserService) Register(ctx context.Context, request *model.RegisterRequest) (*model.AuthResponse, error) {
	if err := utils.ValidateStruct(request); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := utils.ValidatePassword(request.Password); err != nil {
		return nil, appErrors.NewAppError("WEAK_PASSWORD", err.Error(), nil)
	}

	existingUser, err := s.repo.GetUserByEmail(ctx, request.Email)
	if err != nil && !errors.Is(err, appErrors.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		logger.Warn("Registration attempt with existing email",
			zap.String("email", request.Email),
			zap.String("event", "registration_failed_duplicate_email"),
		)
		return nil, appErrors.ErrUserAlreadyExists
	}

	hashedPassword, err := utils.HashPassword(request.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Username:       request.Username,
		Email:          request.Email,
		PasswordHashed: hashedPassword,
		FullName:       request.FullName,
		PhoneNumber:    request.PhoneNumber,
		Role:           request.Role,
		Address:        request.Address,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	tokenPair, err := utils.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		s.config.JWT.Secret,
		s.config.JWT.ExpiryHours,
		s.config.JWT.RefreshExpiryHours,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	logger.Info("User registered successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("username", user.Username),
		zap.String("role", user.Role),
		zap.String("event", "user_registered"),
	)

	return &model.AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (s *UserService) Login(ctx context.Context, request *model.LoginRequest) (*model.AuthResponse, error) {
	if err := utils.ValidateStruct(request); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	user, err := s.repo.GetUserByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, appErrors.ErrUserNotFound) {
			logger.Warn("Login attempt with non-existent email",
				zap.String("email", request.Email),
				zap.String("event", "user_not_found"),
			)
			return nil, appErrors.ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		logger.Warn("Login attempt for inactive user",
			zap.String("user_id", user.ID.String()),
			zap.String("email", user.Email),
			zap.String("event", "login_failed_inactive_user"),
		)
		return nil, appErrors.ErrUserInactive
	}

	if !utils.CheckPassword(user.PasswordHashed, request.Password) {
		logger.Warn("Login attempt with invalid password",
			zap.String("user_id", user.ID.String()),
			zap.String("email", user.Email),
			zap.String("event", "login_failed_invalid_password"),
		)
		return nil, appErrors.ErrInvalidCredentials
	}

	tokenPair, err := utils.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		s.config.JWT.Secret,
		s.config.JWT.ExpiryHours,
		s.config.JWT.RefreshExpiryHours,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	logger.Info("User logged in successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("role", user.Role),
		zap.String("event", "login_success"),
	)

	return &model.AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (s *UserService) ForgotPassword(ctx context.Context, request *model.ForgotPasswordRequest) error {
	if err := utils.ValidateStruct(request); err != nil {
		return appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	user, err := s.repo.GetUserByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, appErrors.ErrUserNotFound) {
			logger.Info("Password reset requested for non-existent email",
				zap.String("email", request.Email),
				zap.String("event", "password_reset_requested_non_existent_email"),
			)
			return nil
		}
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	resetToken := &model.PasswordResetToken{
		UserID:    user.ID,
		Token:     utils.GenerateResetToken(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.repo.CreatePasswordResetToken(ctx, resetToken); err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	logger.Info("Password reset token generated",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("token_id", resetToken.ID.String()),
		zap.Time("expires_at", resetToken.ExpiresAt),
		zap.String("event", "password_reset_token_generated"),
	)

	// TODO: Send email with reset link
	// For now, just log the token (in production, send via email)
	logger.Debug("Password reset token details",
		zap.String("email", user.Email),
		zap.String("reset_token", resetToken.Token),
		zap.String("reset_link", fmt.Sprintf("https://yourdomain.com/reset-password?token=%s", resetToken.Token)),
		zap.String("event", "password_reset_token_details"),
	)

	return nil
}

func (s *UserService) ResetPassword(ctx context.Context, request *model.ResetPasswordRequest) error {
	if err := utils.ValidateStruct(request); err != nil {
		return appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := utils.ValidatePassword(request.NewPassword); err != nil {
		return appErrors.NewAppError("WEAK_PASSWORD", err.Error(), nil)
	}

	resetToken, err := s.repo.GetPasswordResetToken(ctx, request.Token)
	if err != nil {
		logger.Warn("Password reset attempt with invalid token",
			zap.String("token", request.Token),
			zap.String("event", "password_reset_failed_invalid_token"),
		)
		return err
	}
	if resetToken.Used {
		return appErrors.NewAppError("INVALID_TOKEN", "Token has already been used", nil)
	}
	if time.Now().After(resetToken.ExpiresAt) {
		return appErrors.NewAppError("INVALID_TOKEN", "Token has expired", nil)
	}

	hashedPassword, err := utils.HashPassword(request.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, resetToken.UserID, hashedPassword); err != nil {
		return err
	}

	if err := s.repo.MarkTokenAsUsed(ctx, resetToken.ID); err != nil {
		logger.Error("Failed to mark password reset token as used",
			zap.String("user_id", resetToken.UserID.String()),
			zap.String("token_id", resetToken.ID.String()),
			zap.Error(err),
		)
	}

	logger.Info("Password reset successfully",
		zap.String("user_id", resetToken.UserID.String()),
		zap.String("token_id", resetToken.ID.String()),
		zap.String("event", "password_reset_success"),
	)

	return nil
}

func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, request *model.ChangePasswordRequest) error {
	if err := utils.ValidateStruct(request); err != nil {
		return appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := utils.ValidatePassword(request.NewPassword); err != nil {
		return appErrors.NewAppError("WEAK_PASSWORD", err.Error(), nil)
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if !utils.CheckPassword(user.PasswordHashed, request.OldPassword) {
		logger.Warn("Password change attempt with invalid old password",
			zap.String("user_id", user.ID.String()),
			zap.String("email", user.Email),
			zap.String("event", "password_change_failed_invalid_old_password"),
		)
		return appErrors.ErrInvalidCredentials
	}

	hashedPassword, err := utils.HashPassword(request.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		return err
	}

	logger.Info("Password changed successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("event", "password_change_success"),
	)

	return nil
}

func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, request *model.UpdateProfileRequest) (*model.UserResponse, error) {
	if err := utils.ValidateStruct(request); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if request.FullName != nil {
		user.FullName = *request.FullName
	}
	if request.PhoneNumber != nil {
		user.PhoneNumber = request.PhoneNumber
	}
	if request.Address != nil {
		user.Address = request.Address
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}

func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	claims, err := utils.ValidateToken(refreshToken, s.config.JWT.Secret)
	if err != nil {
		logger.Warn("Token refresh attempt with invalid token",
			zap.String("event", "token_refresh_failed_invalid_token"),
			zap.Error(err),
		)
		return nil, appErrors.ErrInvalidToken
	}

	tokenPair, err := utils.GenerateTokenPair(
		claims.UserID,
		claims.Email,
		claims.Role,
		s.config.JWT.Secret,
		s.config.JWT.ExpiryHours,
		s.config.JWT.RefreshExpiryHours,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	logger.Debug("Token refresh successfully",
		zap.String("user_id", claims.UserID.String()),
		zap.String("email", claims.Email),
		zap.String("event", "token_refresh_success"),
	)

	return tokenPair, nil
}
