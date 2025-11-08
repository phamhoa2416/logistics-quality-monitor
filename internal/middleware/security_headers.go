package middleware

import "github.com/gin-gonic/gin"

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()

		// Prevent MIME type sniffing
		headers.Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks
		headers.Set("X-Frame-Options", "DENY")

		// Enable XSS protection
		headers.Set("X-XSS-Protection", "1; mode=block")

		// Set referrer policy
		headers.Set("Referrer-Policy", "no-referrer")

		// Content Security Policy
		headers.Set("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}
