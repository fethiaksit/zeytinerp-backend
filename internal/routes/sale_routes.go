package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterSaleRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewSaleHandler(db)

	api.POST("/sales", handler.Create)
	api.GET("/sales/:id", handler.Get)
	api.GET("/pos/sales/:id", handler.Get)
}
