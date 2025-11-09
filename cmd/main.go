//package main
//
//import (
//	"context"
//	"errors"
//	"log"
//	"logistics-quality-monitor/internal/logger"
//	"logistics-quality-monitor/internal/routes"
//	"net"
//	"net/http"
//	"os"
//	"os/signal"
//	"syscall"
//	"time"
//
//	"go.uber.org/zap"
//	"logistics-quality-monitor/internal/config"
//	"logistics-quality-monitor/internal/database"
//)
//
//func main() {
//	cfg, err := config.Load()
//	if err != nil {
//		os.Stderr.WriteString("Failed to load configuration: " + err.Error() + "\n")
//		os.Exit(1)
//	}
//
//	env := cfg.Server.Environment
//	if env == "" {
//		env = "development"
//	}
//	if err := logger.Init(env); err != nil {
//		os.Stderr.WriteString("Failed to initialize logger: " + err.Error() + "\n")
//		os.Exit(1)
//	}
//	defer logger.Sync()
//
//	logger.Info("Starting application",
//		zap.String("environment", env),
//	)
//
//	if cfg.Database.Host == "" || cfg.Database.DBName == "" {
//		logger.Fatal("Database configuration is missing. Please set DB_HOST and DB_NAME environment variables.")
//	}
//	if cfg.JWT.Secret == "" {
//		logger.Fatal("JWT secret is missing. Please set JWT_SECRET environment variable.")
//	}
//
//	db, err := database.NewDatabase(cfg)
//	if err != nil {
//		logger.Fatal("Failed to connect to database: %v", zap.Error(err))
//	}
//	defer func() {
//		if err := db.Close(); err != nil {
//			logger.Error("Failed to close database connection: %v", zap.Error(err))
//		}
//	}()
//
//	router := routes.SetupRoutes(cfg, db)
//
//	host := cfg.Server.Host
//	if host == "" {
//		host = "0.0.0.0"
//	}
//	port := cfg.Server.Port
//	if port == "" {
//		port = "8080"
//	}
//	addr := net.JoinHostPort(host, port)
//
//	server := &http.Server{
//		Addr:         addr,
//		Handler:      router,
//		ReadTimeout:  15 * time.Second,
//		WriteTimeout: 15 * time.Second,
//		IdleTimeout:  60 * time.Second,
//	}
//
//	// Start goroutine
//	go func() {
//		logger.Info("Server starting",
//			zap.String("address", addr),
//		)
//		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
//			logger.Fatal("Failed to start server", zap.Error(err))
//		}
//	}()
//
//	// Wait for interrupt signal to gracefully shut down the server
//	quit := make(chan os.Signal, 1)
//	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
//	<-quit
//	logger.Info("Shutdown Server ...")
//
//	// Graceful shutdown with timeout
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	if err := server.Shutdown(ctx); err != nil {
//		logger.Fatal("Failed to shut down server", zap.Error(err))
//	}
//
//	log.Println("Server exited properly")
//}

package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"logistics-quality-monitor/internal/config"
	"logistics-quality-monitor/internal/logger"
	"logistics-quality-monitor/internal/middleware"
	"logistics-quality-monitor/server/delivery/http/handler"
	"logistics-quality-monitor/server/infrastructure/database/postgres"
	"logistics-quality-monitor/server/usecase/user"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	// Initialize infrastructure
	db, _ := postgres.NewDB(cfg)
	defer db.Close()

	// Create repository implementations (infrastructure layer)
	userRepo := postgres.NewUserRepository(db)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(db)

	// Create use case (depends on domain interfaces)
	userService := user.NewService(userRepo, refreshTokenRepo, cfg)

	// Create handler (depends on use case)
	userHandler := handler.NewUserHandler(userService)

	// Start token cleanup job
	//cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	//defer cleanupCancel()
	//go userService.StartTokenCleanupJob(cleanupCtx, 1*time.Hour)

	// Setup routes
	router := gin.Default()
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
				userHandler.RegisterAdminRoutes(admin)
			}
		}
	}

	// Start server...
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
