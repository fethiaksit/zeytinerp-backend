package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterCategoryRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewCategoryHandler(db)
	api.GET("/categories", handler.List)
	api.POST("/categories", handler.Create)
	api.PUT("/categories/:id", handler.Update)
}
