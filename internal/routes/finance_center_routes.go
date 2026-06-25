package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterFinanceCenterRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewFinanceCenterHandler(db)
	group := api.Group("/finance-center")
	group.GET("/summary", handler.Summary)
	group.GET("/history", handler.History)
	group.GET("/money-flow", handler.MoneyFlow)
	group.GET("/debt-distribution", handler.DebtDistribution)
	group.GET("/cashflow", handler.Cashflow)
	group.GET("/alerts", handler.Alerts)
}
