package routes

import (
	"cargo-tracker/internal/config"
	"cargo-tracker/internal/delivery/http/handler"
	"cargo-tracker/internal/infrastructure/database/postgres"
	"cargo-tracker/internal/logger"
	"cargo-tracker/internal/middleware"
	"cargo-tracker/internal/usecase/device"
	"cargo-tracker/internal/usecase/shipment"
	"cargo-tracker/internal/usecase/user"
	_ "context"
	"net/http"
	_ "time"

	"github.com/gin-gonic/gin"
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

	userRepository := postgres.NewUserRepository(db)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(db)
	userService := user.NewService(userRepository, refreshTokenRepo, cfg)
	userHandler := handler.NewUserHandler(userService)

	deviceRepository := postgres.NewDeviceRepository(db)
	deviceService := device.NewService(deviceRepository, userRepository)
	deviceHandler := handler.NewDeviceHandler(deviceService)

	shipmentRepository := postgres.NewShipmentRepository(db)
	shipmentService := shipment.NewService(shipmentRepository, userRepository, deviceRepository)
	shipmentHandler := handler.NewShipmentHandler(shipmentService)

	//// Start token cleanup job
	//cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	//defer cleanupCancel()
	//go userSvc.StartTokenCleanupJob(cleanupCtx, 1*time.Hour)

	v1 := router.Group("/api/v1")
	{
		userHandler.RegisterRoutes(v1)
		deviceHandler.RegisterRoutes(v1)
		shipmentHandler.RegisterRoutes(v1)

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			userHandler.RegisterProfileRoutes(protected)
			protected.POST("/revoke", userHandler.RevokeToken)

			// Customer routes
			customer := protected.Group("")
			customer.Use(middleware.RoleMiddleware("customer"))
			{
				shipmentHandler.RegisterCustomerRoutes(customer)
			}

			// Provider routes
			provider := protected.Group("")
			provider.Use(middleware.RoleMiddleware("provider"))
			{
				shipmentHandler.RegisterProviderRoutes(provider)
			}

			// Shipper routes
			shipper := protected.Group("")
			shipper.Use(middleware.RoleMiddleware("shipper"))
			{
				shipmentHandler.RegisterShipperRoutes(shipper)
			}

			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly())
			{
				userHandler.RegisterAdminRoutes(admin)
				deviceHandler.RegisterAdminRoutes(admin)
			}
		}
	}

	logger.Info("All routes initialized")
	return router
}
