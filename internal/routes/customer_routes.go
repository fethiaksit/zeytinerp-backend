package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterCustomerRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewCustomerHandler(db)
	api.POST("/customers", handler.Create)
	api.GET("/customers", handler.List)
	api.GET("/customers/:id", handler.Get)
	api.PUT("/customers/:id", handler.Update)
	api.DELETE("/customers/:id", handler.Delete)
	api.GET("/customers/:id/balance", handler.Balance)
}
