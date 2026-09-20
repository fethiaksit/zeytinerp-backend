package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "forbidden: missing role claims",
			})
			return
		}

		role, _ := roleVal.(string)
		for _, allowed := range allowedRoles {
			if strings.EqualFold(role, allowed) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "forbidden: insufficient permissions",
		})
	}
}

func RequireAdmin() gin.HandlerFunc {
	return RequireRole("admin")
}
