package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterBankWalletRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewBankWalletHandler(db)

	api.GET("/bank-accounts", handler.ListAccounts)
	api.POST("/bank-accounts", handler.CreateAccount)
	api.GET("/bank-accounts/:id", handler.GetAccount)
	api.PUT("/bank-accounts/:id", handler.UpdateAccount)
	api.DELETE("/bank-accounts/:id", handler.DeleteAccount)

	api.GET("/bank-accounts/:id/transactions", handler.ListTransactions)
	api.POST("/bank-accounts/:id/transactions", handler.CreateTransaction)
	api.DELETE("/bank-transactions/:id", handler.DeleteTransaction)

	api.GET("/bank-wallet/summary", handler.Summary)
	api.GET("/bank-wallet/daily-summary", handler.DailySummary)
	api.GET("/bank-wallet/monthly-summary", handler.MonthlySummary)
}
