package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterCustomerTransactionRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewCustomerTransactionHandler(db)
	api.POST("/customer-transactions", handler.Create)
	api.GET("/customer-transactions", handler.List)
	api.DELETE("/customer-transactions/:id", handler.Delete)
}
