package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterEmployeeTransactionRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewEmployeeTransactionHandler(db)
	api.POST("/employee-transactions", handler.Create)
	api.GET("/employee-transactions", handler.List)
	api.DELETE("/employee-transactions/:id", handler.Delete)
}
