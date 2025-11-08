package service

import (
	"context"
	"logistics-quality-monitor/internal/logger"
	"time"

	"go.uber.org/zap"
)

func (s *UserService) StartTokenCleanupJob(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Token cleanup job started",
		zap.Duration("interval", interval),
	)

	s.cleanupExpiredTokens(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Token cleanup job stopped")
			return
		case <-ticker.C:
			s.cleanupExpiredTokens(ctx)
		}
	}
}

func (s *UserService) cleanupExpiredTokens(ctx context.Context) {
	olderThan := 24 * time.Hour
	if err := s.refreshRepo.DeleteExpiredTokens(ctx, olderThan); err != nil {
		logger.Error("Failed to delete expired tokens", zap.Error(err))
		return
	}

	logger.Debug("Expired tokens cleaned up successfully",
		zap.Duration("older_than", olderThan),
	)
}
