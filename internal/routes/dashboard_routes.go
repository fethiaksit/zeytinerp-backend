package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterDashboardRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewDashboardHandler(db)
	api.GET("/dashboard", handler.Get)
	api.GET("/dashboard/monthly", handler.Monthly)
	api.GET("/reports", handler.Reports)
	api.GET("/reports/monthly", handler.ReportsMonthly)
}
