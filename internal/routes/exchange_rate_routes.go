package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterExchangeRateRoutes(group *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewExchangeRateHandler(db)
	group.GET("/api/exchange-rates/latest", handler.Latest)
}
