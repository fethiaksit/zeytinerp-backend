package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterEmployeeRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewEmployeeHandler(db)
	api.POST("/employees", handler.Create)
	api.GET("/employees", handler.List)
	api.GET("/employees-balances", handler.Balances)
	api.GET("/employees/:id", handler.Get)
	api.PUT("/employees/:id", handler.Update)
	api.DELETE("/employees/:id", handler.Delete)
	api.GET("/employees/:id/balance", handler.Balance)
}
