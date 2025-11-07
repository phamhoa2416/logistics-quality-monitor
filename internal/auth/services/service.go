package services

import (
	"context"
	"fmt"
	"logistics-quality-monitor/internal/auth/models"
	"logistics-quality-monitor/internal/auth/repository"
	"logistics-quality-monitor/internal/config"
	appErrors "logistics-quality-monitor/pkg/errors"
	"logistics-quality-monitor/pkg/utils"
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

	existingUser, _ := s.repo.GetUserByEmail(ctx, request.Email)
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
