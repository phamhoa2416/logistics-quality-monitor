package user

import (
	"context"
	"errors"
	"fmt"
	"logistics-quality-monitor/internal/config"
	domainUser "logistics-quality-monitor/internal/domain/user"
	"logistics-quality-monitor/internal/logger"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service implements user use cases
type Service struct {
	userRepo         domainUser.Repository
	refreshTokenRepo domainUser.RefreshTokenRepository
	config           *config.Config
}

// NewService creates a new user service
func NewService(
	userRepo domainUser.Repository,
	refreshTokenRepo domainUser.RefreshTokenRepository,
	cfg *config.Config,
) *Service {
	return &Service{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		config:           cfg,
	}
}

func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := utils.ValidatePassword(req.Password); err != nil {
		return nil, appErrors.NewAppError("WEAK_PASSWORD", err.Error(), nil)
	}

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, domainUser.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		logger.Warn("Registration attempt with existing email",
			zap.String("email", req.Email),
			zap.String("event", "registration_failed_duplicate_email"),
		)
		return nil, appErrors.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create domain entity
	user := &domainUser.User{
		Username:       req.Username,
		Email:          req.Email,
		PasswordHashed: hashedPassword,
		FullName:       req.FullName,
		PhoneNumber:    req.PhoneNumber,
		Role:           req.Role,
		Address:        req.Address,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Save user
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate tokens
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

	// Store refresh token
	refreshToken := &domainUser.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(time.Duration(s.config.JWT.RefreshExpiryHours) * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	logger.Info("User registered successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("username", user.Username),
		zap.String("role", user.Role),
		zap.String("event", "user_registered"),
	)

	return &AuthResponse{
		User:         ToUserResponse(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domainUser.ErrUserNotFound) {
			logger.Warn("Login attempt with non-existent email",
				zap.String("email", req.Email),
				zap.String("event", "user_not_found"),
			)
			return nil, appErrors.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if user is active
	if !user.IsActive {
		logger.Warn("Login attempt for inactive user",
			zap.String("user_id", user.ID.String()),
			zap.String("email", user.Email),
			zap.String("event", "login_failed_inactive_user"),
		)
		return nil, appErrors.ErrUserInactive
	}

	// Verify password
	if !utils.CheckPassword(user.PasswordHashed, req.Password) {
		logger.Warn("Login attempt with invalid password",
			zap.String("user_id", user.ID.String()),
			zap.String("email", user.Email),
			zap.String("event", "login_failed_invalid_password"),
		)
		return nil, appErrors.ErrInvalidCredentials
	}

	// Generate tokens
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

	// Store refresh token
	refreshToken := &domainUser.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(time.Duration(s.config.JWT.RefreshExpiryHours) * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	logger.Info("User logged in successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("role", user.Role),
		zap.String("event", "login_success"),
	)

	return &AuthResponse{
		User:         ToUserResponse(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (s *Service) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	if err := utils.ValidateStruct(req); err != nil {
		return appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domainUser.ErrUserNotFound) {
			logger.Info("Password reset requested for non-existent email",
				zap.String("email", req.Email),
				zap.String("event", "password_reset_requested_non_existent_email"),
			)
			return nil // Don't reveal if user exists
		}
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	resetToken := &domainUser.PasswordResetToken{
		UserID:    user.ID,
		Token:     utils.GenerateResetToken(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Used:      false,
		CreatedAt: time.Now(),
	}

	if err := s.userRepo.CreatePasswordResetToken(ctx, resetToken); err != nil {
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
	logger.Debug("Password reset token details",
		zap.String("email", user.Email),
		zap.String("reset_token", resetToken.Token),
		zap.String("reset_link", fmt.Sprintf("https://yourdomain.com/reset-password?token=%s", resetToken.Token)),
		zap.String("event", "password_reset_token_details"),
	)

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	if err := utils.ValidateStruct(req); err != nil {
		return appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		return appErrors.NewAppError("WEAK_PASSWORD", err.Error(), nil)
	}

	resetToken, err := s.userRepo.GetPasswordResetToken(ctx, req.Token)
	if err != nil {
		logger.Warn("Password reset attempt with invalid token",
			zap.String("token", req.Token),
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

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, resetToken.UserID, hashedPassword); err != nil {
		return err
	}

	if err := s.userRepo.MarkTokenAsUsed(ctx, resetToken.ID); err != nil {
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

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error {
	if err := utils.ValidateStruct(req); err != nil {
		return appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		return appErrors.NewAppError("WEAK_PASSWORD", err.Error(), nil)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !utils.CheckPassword(user.PasswordHashed, req.OldPassword) {
		logger.Warn("Password change attempt with invalid old password",
			zap.String("user_id", user.ID.String()),
			zap.String("email", user.Email),
			zap.String("event", "password_change_failed_invalid_old_password"),
		)
		return appErrors.ErrInvalidCredentials
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		return err
	}

	logger.Info("Password changed successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
		zap.String("event", "password_change_success"),
	)

	return nil
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return ToUserResponse(user), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) (*UserResponse, error) {
	if err := utils.ValidateStruct(req); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.PhoneNumber != nil {
		user.PhoneNumber = req.PhoneNumber
	}
	if req.Address != nil {
		user.Address = req.Address
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return ToUserResponse(user), nil
}

func (s *Service) GetAllUsers(ctx context.Context) ([]*UserResponse, error) {
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var responses []*UserResponse
	for _, user := range users {
		responses = append(responses, ToUserResponse(user))
	}

	return responses, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return err
	}

	logger.Info("User deleted successfully",
		zap.String("user_id", userID.String()),
		zap.String("event", "user_deleted"),
	)

	return nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	// Validate JWT token
	claims, err := utils.ValidateToken(refreshToken, s.config.JWT.Secret)
	if err != nil {
		logger.Warn("Token refresh attempt with invalid token",
			zap.String("event", "token_refresh_failed_invalid_token"),
			zap.Error(err),
		)
		return nil, appErrors.ErrInvalidToken
	}

	// Check if refresh token exists in DB
	dbToken, err := s.refreshTokenRepo.GetByToken(ctx, refreshToken)
	if err != nil {
		logger.Warn("Token refresh attempt with non-existent or invalid token",
			zap.String("user_id", claims.UserID.String()),
			zap.String("event", "token_refresh_failed_token_not_found"),
		)
		return nil, appErrors.ErrInvalidToken
	}

	// Verify token belongs to the user
	if dbToken.UserID != claims.UserID {
		logger.Warn("Token refresh attempt with mismatched user ID",
			zap.String("token_user_id", dbToken.UserID.String()),
			zap.String("claim_user_id", claims.UserID.String()),
			zap.String("event", "token_refresh_failed_user_mismatch"),
		)
		return nil, appErrors.ErrInvalidToken
	}

	// Revoke the old refresh token
	if err := s.refreshTokenRepo.Revoke(ctx, dbToken.ID); err != nil {
		logger.Error("Failed to revoke refresh token",
			zap.String("token_id", dbToken.ID.String()),
			zap.Error(err),
		)
	}

	// Generate new token pair
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

	// Store the new refresh token
	newRefreshToken := &domainUser.RefreshToken{
		UserID:    claims.UserID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(time.Duration(s.config.JWT.RefreshExpiryHours) * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.refreshTokenRepo.Create(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	logger.Debug("Token refresh successfully",
		zap.String("user_id", claims.UserID.String()),
		zap.String("email", claims.Email),
		zap.String("old_token_id", dbToken.ID.String()),
		zap.String("new_token_id", newRefreshToken.ID.String()),
		zap.String("event", "token_refresh_success"),
	)

	return tokenPair, nil
}

func (s *Service) RevokeToken(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	dbToken, err := s.refreshTokenRepo.GetByToken(ctx, refreshToken)
	if err != nil {
		return appErrors.ErrInvalidToken
	}

	if dbToken.UserID != userID {
		return appErrors.ErrInvalidToken
	}

	if err := s.refreshTokenRepo.Revoke(ctx, dbToken.ID); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	logger.Info("Refresh token revoked successfully",
		zap.String("user_id", userID.String()),
		zap.String("token_id", dbToken.ID.String()),
		zap.String("event", "token_revoked"),
	)

	return nil
}

func (s *Service) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	if err := s.refreshTokenRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke all tokens for user: %w", err)
	}

	logger.Info("All refresh tokens revoked for user",
		zap.String("user_id", userID.String()),
		zap.String("event", "all_tokens_revoked"),
	)

	return nil
}
