package config

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	SMTP      SMTPConfig
	RateLimit RateLimitConfig
	CORS      CORSConfig
}

type ServerConfig struct {
	Port        string
	Host        string
	Environment string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret             string
	ExpiryHours        int
	RefreshExpiryHours int
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type RateLimitConfig struct {
	GeneralRPS   float64 // Requests per second for general endpoints
	GeneralBurst int     // Burst size for general endpoints
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AddConfigPath(".")
	if homeDir, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(homeDir)
	}
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		log.Printf("Warning: config file not found: %v. Falling back to environment variables only.", err)
	}

	config := &Config{
		Server: ServerConfig{
			Port:        viper.GetString("SERVER_PORT"),
			Host:        viper.GetString("SERVER_HOST"),
			Environment: viper.GetString("ENVIRONMENT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			DBName:   viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
		},
		JWT: JWTConfig{
			Secret:             viper.GetString("JWT_SECRET"),
			ExpiryHours:        viper.GetInt("JWT_EXPIRY_HOURS"),
			RefreshExpiryHours: viper.GetInt("JWT_REFRESH_EXPIRY_HOURS"),
		},
		SMTP: SMTPConfig{
			Host:     viper.GetString("SMTP_HOST"),
			Port:     viper.GetInt("SMTP_PORT"),
			User:     viper.GetString("SMTP_USER"),
			Password: viper.GetString("SMTP_PASSWORD"),
			From:     viper.GetString("SMTP_FROM"),
		},
		RateLimit: RateLimitConfig{
			GeneralRPS:   viper.GetFloat64("RATE_LIMIT_GENERAL_RPS"),
			GeneralBurst: viper.GetInt("RATE_LIMIT_GENERAL_BURST"),
		},
		CORS: CORSConfig{
			AllowedOrigins:   viper.GetStringSlice("CORS_ALLOWED_ORIGINS"),
			AllowedMethods:   viper.GetStringSlice("CORS_ALLOWED_METHODS"),
			AllowedHeaders:   viper.GetStringSlice("CORS_ALLOWED_HEADERS"),
			ExposedHeaders:   viper.GetStringSlice("CORS_EXPOSED_HEADERS"),
			AllowCredentials: viper.GetBool("CORS_ALLOW_CREDENTIALS"),
			MaxAge:           viper.GetInt("CORS_MAX_AGE"),
		},
	}

	return config, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
