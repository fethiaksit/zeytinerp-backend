package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"http://zeytinerp.herevemarket.com",
			"https://zeytinerp.herevemarket.com",
			"http://localhost:5173",
		}
	}
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}

func HasWildcardOrigin(allowedOrigins []string) bool {
	for _, origin := range allowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			return true
		}
	}
	return false
}
