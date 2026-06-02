package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterIncomeEntryRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewIncomeEntryHandler(db)
	api.POST("/income-entries", handler.Create)
	api.GET("/income-entries", handler.List)
	api.PUT("/income-entries/:id", handler.Update)
	api.DELETE("/income-entries/:id", handler.Delete)
}
