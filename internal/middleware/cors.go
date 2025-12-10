package middleware

import (
	"cargo-tracker/internal/config"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware(cfg *config.CORSConfig) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     cfg.AllowedMethods,
		AllowHeaders:     cfg.AllowedHeaders,
		ExposeHeaders:    cfg.ExposedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           time.Duration(cfg.MaxAge),
	}

	return cors.New(corsConfig)
}
