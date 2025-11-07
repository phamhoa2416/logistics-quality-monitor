package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"logistics-quality-monitor/internal/auth/models"
	"logistics-quality-monitor/internal/auth/repository"
	"logistics-quality-monitor/internal/config"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo   *repository.Repository
	config *config.Config
}

func NewService(repo *repository.Repository, cfg *config.Config) *Service {
	return &Service{
		repo:   repo,
		config: cfg,
	}
}

func (s *Service) Register(ctx context.Context, request *models.RegisterRequest) (*models.AuthResponse, error) {
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
		return nil, appErrors.ErrUserAlreadyExists
	}

	hashedPassword, err := utils.HashPassword(request.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
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

	return &models.AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (s *Service) Login(ctx context.Context, request *models.LoginRequest) (*models.AuthResponse, error) {
	if err := utils.ValidateStruct(request); err != nil {
		return nil, appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	user, err := s.repo.GetUserByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, appErrors.ErrUserNotFound) {
			return nil, appErrors.ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, appErrors.ErrUserInactive
	}

	if !utils.CheckPassword(user.PasswordHashed, request.Password) {
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

	return &models.AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (s *Service) ForgotPassword(ctx context.Context, request *models.ForgotPasswordRequest) error {
	if err := utils.ValidateStruct(request); err != nil {
		return appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	user, err := s.repo.GetUserByEmail(ctx, request.Email)
	if err != nil {
		log.Printf("Password reset requested for non-existent email: %s", request.Email)
		return nil
	}

	resetToken := &models.PasswordResetToken{
		UserID:    user.ID,
		Token:     utils.GenerateResetToken(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.repo.CreatePasswordResetToken(ctx, resetToken); err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	// TODO: Send email with reset link
	// For now, just log the token (in production, send via email)
	log.Printf("Password reset token for %s: %s", user.Email, resetToken.Token)
	log.Printf("Reset link: http://%s:%s/auth/reset-password?token=%s",
		s.config.Server.Host, s.config.Server.Port, resetToken.Token)

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, request *models.ResetPasswordRequest) error {
	if err := utils.ValidateStruct(request); err != nil {
		return appErrors.NewAppError("VALIDATION_ERROR", "Invalid input", err)
	}

	if err := utils.ValidatePassword(request.NewPassword); err != nil {
		return appErrors.NewAppError("WEAK_PASSWORD", err.Error(), nil)
	}

	resetToken, err := s.repo.GetPasswordResetToken(ctx, request.Token)
	if err != nil {
		return err
	}

	hashedPassword, err := utils.HashPassword(request.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, resetToken.UserID, hashedPassword); err != nil {
		return err
	}

	if err := s.repo.MarkTokenAsUsed(ctx, resetToken.ID); err != nil {
		log.Printf("Failed to mark token as used: %v", err)
	}

	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, request *models.ChangePasswordRequest) error {
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
		return appErrors.ErrInvalidCredentials
	}

	hashedPassword, err := utils.HashPassword(request.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.repo.UpdatePassword(ctx, userID, hashedPassword)
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, request *models.UpdateProfileRequest) (*models.UserResponse, error) {
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

func (s *Service) RefreshToken(refreshToken string) (*utils.TokenPair, error) {
	claims, err := utils.ValidateToken(refreshToken, s.config.JWT.Secret)
	if err != nil {
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

	return tokenPair, nil
}
