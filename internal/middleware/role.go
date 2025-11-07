package middleware

import (
	"logistics-quality-monitor/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			utils.ErrorResponse(c, http.StatusForbidden, "Role not found in context")
			c.Abort()
			return
		}

		userRole := role.(string)

		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		utils.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions")
		c.Abort()
	}
}

func AdminOnly() gin.HandlerFunc {
	return RoleMiddleware("admin")
}

func ShipperOnly() gin.HandlerFunc {
	return RoleMiddleware("shipper")
}

func ProviderOnly() gin.HandlerFunc {
	return RoleMiddleware("provider")
}

func CustomerOnly() gin.HandlerFunc {
	return RoleMiddleware("customer")
}
