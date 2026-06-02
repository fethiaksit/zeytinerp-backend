package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterCashReportRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewCashReportHandler(db)
	api.POST("/daily-cash-reports", handler.Create)
	api.GET("/daily-cash-reports", handler.List)
	api.GET("/daily-cash-reports/:id", handler.Get)
	api.PUT("/daily-cash-reports/:id", handler.Update)
	api.DELETE("/daily-cash-reports/:id", handler.Delete)

	api.POST("/cash-reports", handler.Create)
	api.GET("/cash-reports", handler.List)
	api.GET("/cash-reports/:id", handler.Get)
	api.PUT("/cash-reports/:id", handler.Update)
	api.DELETE("/cash-reports/:id", handler.Delete)
}
