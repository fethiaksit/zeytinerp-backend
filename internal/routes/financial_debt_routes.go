package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterFinancialDebtRoutes(api *gin.RouterGroup, db *gorm.DB) {
	debtHandler := handlers.NewFinancialDebtHandler(db)
	api.POST("/financial-debts", debtHandler.Create)
	api.GET("/financial-debts", debtHandler.List)
	api.GET("/financial-debts/:id", debtHandler.Get)
	api.PUT("/financial-debts/:id", debtHandler.Update)
	api.DELETE("/financial-debts/:id", debtHandler.Delete)
	api.GET("/financial-debts/:id/balance", debtHandler.Balance)
	api.GET("/financial-debts/:id/summary", debtHandler.Summary)
	api.POST("/financial-debts/:id/installments", debtHandler.CreateInstallment)
	api.GET("/financial-debts/:id/installments", debtHandler.ListInstallments)
	api.POST("/financial-debts/:id/installments/bulk", debtHandler.BulkCreateInstallments)
	api.PUT("/financial-installments/:id", debtHandler.UpdateInstallment)
	api.DELETE("/financial-installments/:id", debtHandler.DeleteInstallment)
	api.GET("/financial-alerts", debtHandler.Alerts)

	paymentHandler := handlers.NewFinancialDebtPaymentHandler(db)
	api.POST("/financial-debt-payments", paymentHandler.Create)
	api.GET("/financial-debt-payments", paymentHandler.List)
	api.DELETE("/financial-debt-payments/:id", paymentHandler.Delete)
}
