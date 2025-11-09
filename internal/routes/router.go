package routes

import (
	_ "context"
	"github.com/gin-gonic/gin"
	"logistics-quality-monitor/internal/config"
	userHandler "logistics-quality-monitor/internal/delivery/http/handler"
	"logistics-quality-monitor/internal/infrastructure/database/postgres"
	"logistics-quality-monitor/internal/logger"
	"logistics-quality-monitor/internal/middleware"
	"logistics-quality-monitor/internal/usecase/user"
	"net/http"
	_ "time"

	_ "logistics-quality-monitor/internal/device/handler"
	_ "logistics-quality-monitor/internal/device/repository"
	_ "logistics-quality-monitor/internal/device/service"
)

func SetupRoutes(cfg *config.Config, db *postgres.DB) *gin.Engine {
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

	// Create repository implementations (infrastructure layer)
	userRepo := postgres.NewUserRepository(db)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(db)

	// Create use case (depends on domain interfaces)
	userService := user.NewService(userRepo, refreshTokenRepo, cfg)

	// Create handler (depends on use case)
	userHdl := userHandler.NewUserHandler(userService)

	//deviceRepo := deviceRepository.NewRepository(db)
	//deviceSvc := deviceService.NewService(deviceRepo, *userRepo, cfg)
	//deviceHdl := deviceHandler.NewHandler(deviceSvc)
	//
	//// Start token cleanup job
	//cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	//defer cleanupCancel()
	//go userSvc.StartTokenCleanupJob(cleanupCtx, 1*time.Hour)

	v1 := router.Group("/api/v1")
	{
		userHdl.RegisterRoutes(v1)
		//deviceHdl.RegisterRoutes(v1)
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
				//deviceHdl.RegisterAdminRoutes(admin)
			}
		}
	}

	logger.Info("All routes initialized")
	return router
}
