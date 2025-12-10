package middleware

import (
	"cargo-tracker/internal/logger"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware logs HTTP requests and responses with structured logging.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()

		requestID := GetRequestID(c)
		log := logger.WithRequestID(requestID)

		log.Info("Incoming request",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", ip),
			zap.String("user_agent", userAgent),
		)

		// Log request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get response status
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Log response
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", ip),
			zap.String("user_agent", userAgent),
			zap.Int("status_code", statusCode),
			zap.Duration("latency", latency),
		}

		if errorMessage != "" {
			fields = append(fields, zap.String("error", errorMessage))
		}

		switch {
		case statusCode >= 500:
			log.Error("Request completed with server error", fields...)
		case statusCode >= 400:
			log.Warn("Request completed with client error", fields...)
		default:
			log.Info("Request completed successfully", fields...)
		}
	}
}
