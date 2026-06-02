package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterStockMovementRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewStockMovementHandler(db)
	api.POST("/stock-movements", handler.Create)
	api.GET("/stock-movements", handler.List)
}
