package main

import (
	"context"
	"errors"
	"log"
	"logistics-quality-monitor/internal/logger"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"logistics-quality-monitor/internal/config"
	"logistics-quality-monitor/internal/database"
	"logistics-quality-monitor/internal/middleware"
	"logistics-quality-monitor/internal/user/handler"
	"logistics-quality-monitor/internal/user/repository"
	"logistics-quality-monitor/internal/user/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("Failed to load configuration: " + err.Error() + "\n")
		os.Exit(1)
	}

	env := cfg.Server.Environment
	if env == "" {
		env = "development"
	}
	if err := logger.Init(env); err != nil {
		os.Stderr.WriteString("Failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting application",
		zap.String("environment", env),
	)

	if cfg.Database.Host == "" || cfg.Database.DBName == "" {
		logger.Fatal("Database configuration is missing. Please set DB_HOST and DB_NAME environment variables.")
	}
	if cfg.JWT.Secret == "" {
		logger.Fatal("JWT secret is missing. Please set JWT_SECRET environment variable.")
	}

	db, err := database.NewDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection: %v", zap.Error(err))
		}
	}()

	userRepository := repository.NewRepository(db)
	refreshTokenRepository := repository.NewRefreshTokenRepository(db)
	userService := service.NewService(userRepository, refreshTokenRepository, cfg)
	userHandler := handler.NewHandler(userService)

	// Start token cleanup job
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go userService.StartTokenCleanupJob(cleanupCtx, 1*time.Hour)

	// Set Gin mode based on environment
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

	v1 := router.Group("/api/v1")
	{
		userHandler.RegisterRoutes(v1)

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			userHandler.RegisterProfileRoutes(protected)
			protected.POST("/revoke", userHandler.RevokeToken)

			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly())
			{
				admin.GET("/users", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{
						"message": "Admin users list",
					})
				})
			}
		}
	}

	host := cfg.Server.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}
	addr := net.JoinHostPort(host, port)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start goroutine
	go func() {
		logger.Info("Server starting",
			zap.String("address", addr),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutdown Server ...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Failed to shutdown server", zap.Error(err))
	}

	log.Println("Server exited properly")
}
