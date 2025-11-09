package main

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"log"
	"logistics-quality-monitor/internal/config"
	"logistics-quality-monitor/internal/infrastructure/database/postgres"
	"logistics-quality-monitor/internal/logger"
	"logistics-quality-monitor/internal/routes"
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
	defer func(db *postgres.DB) {
		err := db.Close()
		if err != nil {
			logger.Fatal("Failed to close DB", zap.Error(err))
		}
	}(db)

	// Start token cleanup job
	//cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	//defer cleanupCancel()
	//go userService.StartTokenCleanupJob(cleanupCtx, 1*time.Hour)

	// Setup routes
	router := routes.SetupRoutes(cfg, db)

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
