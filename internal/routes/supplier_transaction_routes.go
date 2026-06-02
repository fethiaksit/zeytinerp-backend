package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterSupplierTransactionRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewSupplierTransactionHandler(db)
	api.POST("/supplier-transactions", handler.Create)
	api.GET("/supplier-transactions", handler.List)
	api.DELETE("/supplier-transactions/:id", handler.Delete)
}
