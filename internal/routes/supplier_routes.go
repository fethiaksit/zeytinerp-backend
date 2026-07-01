package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterSupplierRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewSupplierHandler(db)
	api.GET("/firms/today-visits", handler.TodayVisits)
	api.GET("/firms/visits", handler.Visits)
	api.POST("/suppliers", handler.Create)
	api.GET("/suppliers", handler.List)
	api.GET("/suppliers-balances", handler.Balances)
	api.GET("/suppliers-currency-totals", handler.CurrencyTotals)
	api.GET("/suppliers/:id", handler.Get)
	api.PUT("/suppliers/:id", handler.Update)
	api.DELETE("/suppliers/:id", handler.Delete)
	api.GET("/suppliers/:id/balance", handler.Balance)
}
