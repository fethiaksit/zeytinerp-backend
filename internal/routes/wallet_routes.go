package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterWalletRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewWalletHandler(db)

	api.GET("/wallet/summary", handler.Summary)
	api.GET("/wallet/transactions", handler.ListTransactions)
	api.POST("/wallet/transactions", handler.CreateTransaction)
	api.DELETE("/wallet/transactions/:id", handler.DeleteTransaction)
}
