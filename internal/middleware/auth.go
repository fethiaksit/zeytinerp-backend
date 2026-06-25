package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"market-erp-backend/internal/services"
)

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			authFail(c, "authorization token is required")
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			authFail(c, "invalid or expired token")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		if token == "" {
			authFail(c, "authorization token is required")
			return
		}

		claims, err := services.VerifyJWT(jwtSecret, token)
		if err != nil {
			authFail(c, "invalid or expired token")
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func authFail(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error":   message,
	})
}
