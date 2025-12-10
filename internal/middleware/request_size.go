package middleware

import (
	"cargo-tracker/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	DefaultMaxRequestSize = 10 << 20
)

// RequestSizeLimitMiddleware limits the size of incoming requests to maxSize bytes.
func RequestSizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	if maxSize <= 0 {
		maxSize = DefaultMaxRequestSize
	}

	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			utils.ErrorResponse(c, http.StatusRequestEntityTooLarge, "Request body too large")
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}
