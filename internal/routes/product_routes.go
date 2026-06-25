package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterProductRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewProductHandler(db)
	api.POST("/products", handler.Create)
	api.GET("/products", handler.List)
	api.GET("/products/:id", handler.Get)
	api.PUT("/products/:id", handler.Update)
	api.DELETE("/products/:id", handler.Delete)
	api.GET("/products/:id/stock", handler.Stock)
}
