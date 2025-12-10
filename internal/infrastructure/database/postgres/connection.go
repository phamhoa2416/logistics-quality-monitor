package postgres

import (
	"cargo-tracker/internal/config"
	"cargo-tracker/internal/logger"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

func NewDB(cfg *config.Config) (*DB, error) {
	dsn := cfg.Database.DSN()

	var gormLogLevel gormLogger.LogLevel
	if cfg.Server.Environment == "production" {
		gormLogLevel = gormLogger.Warn
	} else {
		gormLogLevel = gormLogger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error getting sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Database.Host),
		zap.String("database", cfg.Database.DBName),
		zap.Int("max_open_connections", 25),
		zap.Int("max_idle_connections", 5),
	)

	return &DB{DB: db}, nil
}

func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *DB) Health() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
