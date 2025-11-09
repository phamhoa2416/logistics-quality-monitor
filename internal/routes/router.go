package routes

import (
	"context"
	"github.com/gin-gonic/gin"
	"logistics-quality-monitor/internal/config"
	"logistics-quality-monitor/internal/database"
	"logistics-quality-monitor/internal/logger"
	"logistics-quality-monitor/internal/middleware"
	userHandler "logistics-quality-monitor/internal/user/handler"
	userRepository "logistics-quality-monitor/internal/user/repository"
	userService "logistics-quality-monitor/internal/user/service"
	"net/http"
	"time"

	deviceHandler "logistics-quality-monitor/internal/device/handler"
	deviceRepository "logistics-quality-monitor/internal/device/repository"
	deviceService "logistics-quality-monitor/internal/device/service"
)

func SetupRoutes(cfg *config.Config, db *database.Database) *gin.Engine {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Add middleware in order: request ID, logging, security headers, CORS, request size limit, general rate limit
	router.Use(gin.Recovery())
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.CORSMiddleware(&cfg.CORS))
	router.Use(middleware.RequestSizeLimitMiddleware(10 << 20))
	router.Use(middleware.RateLimitMiddleware(cfg.RateLimit.GeneralRPS, cfg.RateLimit.GeneralBurst))

	router.GET("/health", func(c *gin.Context) {
		if err := db.Health(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"message": "Database connection failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "Service is running",
		})
	})

	userRepo := userRepository.NewRepository(db)
	refreshRepo := userRepository.NewRefreshTokenRepository(db)
	userSvc := userService.NewService(userRepo, refreshRepo, cfg)
	userHdl := userHandler.NewHandler(userSvc)

	deviceRepo := deviceRepository.NewRepository(db)
	deviceSvc := deviceService.NewService(deviceRepo, *userRepo, cfg)
	deviceHdl := deviceHandler.NewHandler(deviceSvc)

	// Start token cleanup job
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go userSvc.StartTokenCleanupJob(cleanupCtx, 1*time.Hour)

	v1 := router.Group("/api/v1")
	{
		userHdl.RegisterRoutes(v1)
		deviceHdl.RegisterRoutes(v1)
		//deviceHdl.RegisterAdminRoutes(v1)

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			userHdl.RegisterProfileRoutes(protected)
			protected.POST("/revoke", userHdl.RevokeToken)

			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly())
			{
				userHdl.RegisterAdminRoutes(admin)
				deviceHdl.RegisterAdminRoutes(admin)
			}
		}
	}

	logger.Info("All routes initialized")
	return router
}
